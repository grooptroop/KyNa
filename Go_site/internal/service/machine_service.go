package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/grooptroop/KyNa/Go_site/internal/repository"
)

type MachineService struct {
	repo         *repository.MachineRepository
	userRepo     *repository.UserRepository
	helmChartDir string
}

func NewMachineService(mRepo *repository.MachineRepository, uRepo *repository.UserRepository, helmChartDir string) *MachineService {
	return &MachineService{
		repo:         mRepo,
		userRepo:     uRepo,
		helmChartDir: helmChartDir,
	}
}

type CreateMachineInput struct {
	Username        string
	Name            string
	Mode            string
	ServiceKind     string
	Version         string
	ResourcesPreset string
	Image           string
	ContainerPort   int
	ServicePort     int
	AccessScope     string
	EnableIngress   bool
	IngressHost     string
	ImageTarPath    string
}

type UpdateMachineInput struct {
	ID              int64
	Username        string
	Name            string
	ServiceKind     string
	Version         string
	ResourcesPreset string
	Image           string
	ContainerPort   int
	ServicePort     int
	AccessScope     string
	EnableIngress   bool
	IngressHost     string
	ImageTarPath    string
}

type DeleteMachineInput struct {
	ID       int64
	Username string
	Name     string
}

func (s *MachineService) ListMachines(ctx context.Context, username string) ([]model.UserMachine, error) {
	return s.repo.ListByUsername(ctx, username)
}

type MachineJSON struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ServiceKind string  `json:"service_kind"`
	Status      string  `json:"status"`
	Resources   string  `json:"resources_preset"`
	AccessScope string  `json:"access_scope"`
	ExternalIP  *string `json:"external_ip,omitempty"`
	ClusterIP   *string `json:"cluster_ip,omitempty"`
	IngressHost *string `json:"ingress_host,omitempty"`
}

func (s *MachineService) ListMachinesJSON(ctx context.Context, username string) ([]MachineJSON, error) {
	ms, err := s.repo.ListByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	out := make([]MachineJSON, 0, len(ms))
	for _, m := range ms {
		out = append(out, MachineJSON{
			ID:          m.ID,
			Name:        m.Name,
			ServiceKind: m.ServiceKind,
			Status:      string(m.Status),
			Resources:   m.ResourcesPreset,
			AccessScope: m.AccessScope,
			ExternalIP:  m.ExternalIP,
			ClusterIP:   m.ClusterIP,
			IngressHost: m.IngressHost,
		})
	}
	return out, nil
}

func (s *MachineService) CreateMachine(ctx context.Context, in CreateMachineInput) (*model.UserMachine, error) {
	log.Printf(
		"DEBUG: MachineService.CreateMachine start username=%q name=%q mode=%q serviceKind=%q version=%q resourcesPreset=%q cport=%d sport=%d",
		in.Username, in.Name, in.Mode, in.ServiceKind, in.Version, in.ResourcesPreset, in.ContainerPort, in.ServicePort,
	)

	if in.Username == "" {
		return nil, fmt.Errorf("MachineService.CreateMachine: empty username")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("MachineService.CreateMachine: empty name")
	}

	serviceKind := in.ServiceKind
	if serviceKind == "" {
		serviceKind = "web"
	}

	mode := "app"
	if serviceKind == "worker" {
		mode = "worker"
	}

	resources := in.ResourcesPreset
	if resources == "" {
		resources = "small"
	}

	var ingressHostPtr *string
	if in.EnableIngress && in.IngressHost != "" {
		ingressHostPtr = &in.IngressHost
	}

	accessScope := in.AccessScope
	if accessScope != "internal" && accessScope != "public" {
		accessScope = "internal"
	}

	m := &model.UserMachine{
		Username:        in.Username,
		Name:            in.Name,
		Mode:            mode,
		ServiceKind:     serviceKind,
		Status:          model.MachineStatusPending,
		ResourcesPreset: resources,
		IngressHost:     ingressHostPtr,
		AccessScope:     accessScope,
		ContainerPort:   in.ContainerPort,
		ServicePort:     in.ServicePort,
	}

	if in.Image != "" {
		img := in.Image
		m.Image = &img
	}

	if err := s.repo.Create(ctx, m); err != nil {
		log.Printf("CREATE MACHINE DB ERROR: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: Machine saved to DB id=%d username=%s name=%s serviceKind=%s resources=%s",
		m.ID, m.Username, m.Name, m.ServiceKind, m.ResourcesPreset)

	if s.helmChartDir == "" {
		return m, nil
	}

	inCopy := in
	mCopy := *m

	go func() {
		bg := context.Background()
		s.provisionMachine(bg, &mCopy, inCopy, false)
	}()

	return m, nil
}

func (s *MachineService) DeleteMachine(ctx context.Context, in DeleteMachineInput) error {
	if in.Username == "" {
		return fmt.Errorf("DeleteMachine: empty username")
	}
	if in.ID == 0 && in.Name == "" {
		return fmt.Errorf("DeleteMachine: empty id and name")
	}

	name := in.Name

	if name == "" {
		machines, err := s.repo.ListByUsername(ctx, in.Username)
		if err != nil {
			return fmt.Errorf("list machines: %w", err)
		}
		var found *model.UserMachine
		for i := range machines {
			if machines[i].ID == in.ID {
				found = &machines[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("machine not found")
		}
		name = found.Name
	}

	if s.helmChartDir != "" {
		releaseName := fmt.Sprintf("machine-%s-%s", in.Username, name)
		ns := in.Username

		cmd := exec.CommandContext(
			ctx,
			"helm",
			"uninstall",
			releaseName,
			"--namespace", ns,
		)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("helm uninstall for machine failed: %v, output: %s", err, string(out))
		}
	}

	if err := s.repo.DeleteByID(ctx, in.ID, in.Username); err != nil {
		return err
	}

	log.Printf("DEBUG: Machine deleted id=%d username=%s name=%s", in.ID, in.Username, name)

	return nil
}

func (s *MachineService) checkPodCrashLoop(ctx context.Context, username, name string) (bool, error) {
	ns := username
	labelSelector := fmt.Sprintf("app=%s-%s", username, name)

	cmd := exec.CommandContext(
		ctx,
		"kubectl", "get", "pods",
		"-n", ns,
		"-l", labelSelector,
		"-o", "jsonpath={.items[0].status.containerStatuses[0].state.waiting.reason}",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("checkPodCrashLoop kubectl error: %v, out=%s", err, string(out))
		return false, nil
	}

	reason := strings.TrimSpace(string(out))
	return reason == "CrashLoopBackOff", nil
}

func loadDockerImageFromTar(ctx context.Context, tarPath string) (string, error) {
	if tarPath == "" {
		return "", fmt.Errorf("empty tar path")
	}

	cmd := exec.CommandContext(ctx, "docker", "image", "load", "-i", tarPath)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker load failed: %w, stderr: %s", err, stderr.String())
	}

	out := stdout.String()

	re := regexp.MustCompile(`Loaded image: ([^\s]+)`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		return "", fmt.Errorf("cannot parse loaded image from output: %q", out)
	}

	imageName := m[1]
	log.Printf("DEBUG: docker load output: %s", out)
	log.Printf("loaded docker image from tar %s: %s", tarPath, imageName)
	return imageName, nil
}

func (s *MachineService) provisionMachine(ctx context.Context, m *model.UserMachine, in CreateMachineInput, isUpgrade bool) {
	log.Printf("DEBUG: provisionMachine start id=%d username=%s name=%s", m.ID, m.Username, m.Name)

	serviceKind := m.ServiceKind
	resources := m.ResourcesPreset

	svcEnabled := true
	svcType := "ClusterIP"
	ingressEnabled := false

	containerPort := 80
	svcPort := 80
	if serviceKind == "api" {
		containerPort = 8080
		svcPort = 8080
	}
	if in.ContainerPort > 0 {
		containerPort = in.ContainerPort
	}
	if in.ServicePort > 0 {
		svcPort = in.ServicePort
	}

	accessScope := in.AccessScope
	if accessScope != "internal" && accessScope != "public" {
		accessScope = "internal"
	}

	cmdName := "install"
	if isUpgrade {
		cmdName = "upgrade"
	}

	switch serviceKind {
	case "web":
		svcType = "LoadBalancer"
		svcPort = containerPort
		ingressEnabled = false

	case "api":
		if accessScope == "public" {
			if in.EnableIngress && in.IngressHost != "" {
				svcType = "ClusterIP"
				ingressEnabled = true
			} else {
				svcType = "LoadBalancer"
				ingressEnabled = false
			}
		} else {
			svcType = "ClusterIP"
			ingressEnabled = false
		}
		svcPort = containerPort

	case "worker":
		svcEnabled = false
		ingressEnabled = false
	}

	cpuReq := "50m"
	memReq := "64Mi"
	cpuLimit := "200m"
	memLimit := "128Mi"

	switch resources {
	case "medium":
		cpuReq = "100m"
		memReq = "128Mi"
		cpuLimit = "300m"
		memLimit = "256Mi"
	case "large":
		cpuReq = "250m"
		memReq = "256Mi"
		cpuLimit = "500m"
		memLimit = "512Mi"
	}

	imgRepo := "nginx"
	imgTag := "stable-alpine"

	if in.ImageTarPath != "" {
		loadedImage, err := loadDockerImageFromTar(ctx, in.ImageTarPath)
		if err != nil {
			log.Printf("failed to load docker image from tar %s: %v", in.ImageTarPath, err)
			if err2 := s.repo.UpdateStatusAndIP(ctx, m.ID, model.MachineStatusFailed, nil); err2 != nil {
				log.Printf("failed to update machine status/ip in db after tar error: %v", err2)
			}
			return
		}

		parts := strings.Split(loadedImage, ":")
		if len(parts) == 2 {
			imgRepo = parts[0]
			imgTag = parts[1]
		} else {
			imgRepo = loadedImage
		}

		if err := os.Remove(in.ImageTarPath); err != nil {
			log.Printf("failed to remove temp tar %s: %v", in.ImageTarPath, err)
		}

	} else if in.Image != "" {
		parts := strings.Split(in.Image, ":")
		if len(parts) == 2 {
			imgRepo = parts[0]
			imgTag = parts[1]
		} else {
			imgRepo = in.Image
			if in.Version != "" {
				imgTag = in.Version
			}
		}
	} else {
		if in.Version != "" {
			imgTag = in.Version
		}
	}

	finalImage := fmt.Sprintf("%s:%s", imgRepo, imgTag)
	if err := s.repo.UpdateImage(ctx, m.ID, finalImage); err != nil {
		log.Printf("failed to update machine image in db: %v", err)
	} else {
		m.Image = &finalImage
	}

	releaseName := fmt.Sprintf("machine-%s-%s", m.Username, m.Name)
	ns := m.Username

	args := []string{
		cmdName,
		releaseName,
		s.helmChartDir,
		"--namespace", ns,
		"--set", fmt.Sprintf("username=%s", m.Username),
		"--set", fmt.Sprintf("name=%s", m.Name),
		"--set", fmt.Sprintf("mode=%s", m.Mode),
		"--set", fmt.Sprintf("serviceKind=%s", serviceKind),
		"--set", fmt.Sprintf("image.repository=%s", imgRepo),
		"--set", fmt.Sprintf("image.tag=%s", imgTag),
		"--set", fmt.Sprintf("containerPort=%d", containerPort),
		"--set", fmt.Sprintf("service.enabled=%t", svcEnabled),
		"--set", fmt.Sprintf("service.type=%s", svcType),
		"--set", fmt.Sprintf("service.port=%d", svcPort),
		"--set", fmt.Sprintf("ingress.enabled=%t", ingressEnabled),
		"--set", fmt.Sprintf("resources.request.cpu=%s", cpuReq),
		"--set", fmt.Sprintf("resources.request.memory=%s", memReq),
		"--set", fmt.Sprintf("resources.limits.cpu=%s", cpuLimit),
		"--set", fmt.Sprintf("resources.limits.memory=%s", memLimit),
	}
	if in.IngressHost != "" {
		args = append(args,
			"--set", fmt.Sprintf("ingress.host=%s", in.IngressHost),
		)
	}

	if isUpgrade {
		args = append(args, "--install")
	}

	log.Printf("DEBUG: [async] using image %s:%s for machine %s/%s", imgRepo, imgTag, m.Username, m.Name)
	log.Printf("DEBUG: [async] helm args: %v", args)

	cmd := exec.CommandContext(ctx, "helm", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("helm install for machine failed: %v, output: %s", err, string(out))
		if err2 := s.repo.UpdateStatusAndIP(ctx, m.ID, model.MachineStatusFailed, nil); err2 != nil {
			log.Printf("failed to update machine status/ip in db after helm error: %v", err2)
		}
		return
	}

	svcName := fmt.Sprintf("%s-%s", m.Username, m.Name)
	ns = m.Username

	clusterIPCmd := exec.CommandContext(
		ctx,
		"kubectl",
		"get",
		"svc",
		svcName,
		"-n", ns,
		"-o", "jsonpath={.spec.clusterIP}",
	)

	clusterIPOut, clusterIPErr := clusterIPCmd.CombinedOutput()
	clusterIP := strings.TrimSpace(string(clusterIPOut))
	if clusterIPErr != nil {
		log.Printf("kubectl get svc cluster ip for machine failed: %v, output: %s", clusterIPErr, string(clusterIPOut))
	}
	if clusterIP != "" {
		m.ClusterIP = &clusterIP
	}

	ipCmd := exec.CommandContext(
		ctx,
		"kubectl",
		"get",
		"svc",
		svcName,
		"-n", ns,
		"-o", "jsonpath={.status.loadBalancer.ingress[0].ip}",
	)

	ipOut, ipErr := ipCmd.CombinedOutput()
	externalIP := strings.TrimSpace(string(ipOut))
	if ipErr != nil {
		log.Printf("kubectl get svc external ip for machine failed: %v, output: %s", ipErr, string(ipOut))
	}
	if externalIP != "" {
		m.ExternalIP = &externalIP
	}

	crash, _ := s.checkPodCrashLoop(ctx, m.Username, m.Name)
	if crash {
		m.Status = model.MachineStatusFailed
	} else {
		m.Status = model.MachineStatusReady
	}

	if err := s.repo.UpdateStatusAndIP(ctx, m.ID, m.Status, m.ExternalIP); err != nil {
		log.Printf("failed to update machine status/ip in db: %v", err)
	}

	log.Printf("DEBUG: provisionMachine end id=%d status=%s external_ip=%v cluster_ip=%v",
		m.ID, m.Status, m.ExternalIP, m.ClusterIP)
}

func (s *MachineService) GetMachine(ctx context.Context, username string, id int64) (*model.UserMachine, error) {
	machines, err := s.repo.ListByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	for i := range machines {
		if machines[i].ID == id {
			return &machines[i], nil
		}
	}
	return nil, fmt.Errorf("machine not found")
}

func (s *MachineService) UpdateMachine(ctx context.Context, in UpdateMachineInput) error {
	if in.Username == "" {
		return fmt.Errorf("UpdateMachine: empty username")
	}
	if in.ID == 0 {
		return fmt.Errorf("UpdateMachine: empty id")
	}

	machines, err := s.repo.ListByUsername(ctx, in.Username)
	if err != nil {
		return fmt.Errorf("list machines: %w", err)
	}

	var m *model.UserMachine
	for i := range machines {
		if machines[i].ID == in.ID {
			m = &machines[i]
			break
		}
	}
	if m == nil {
		return fmt.Errorf("machine not found")
	}

	if in.Name != "" {
		m.Name = in.Name
	}
	if in.ServiceKind != "" {
		m.ServiceKind = in.ServiceKind
	}
	if in.ResourcesPreset != "" {
		m.ResourcesPreset = in.ResourcesPreset
	}

	accessScope := in.AccessScope
	if accessScope != "internal" && accessScope != "public" {
		accessScope = "internal"
	}
	m.AccessScope = accessScope

	var ingressHostPtr *string
	if in.EnableIngress && in.IngressHost != "" {
		ingressHostPtr = &in.IngressHost
	}
	m.IngressHost = ingressHostPtr

	if in.ContainerPort > 0 {
		m.ContainerPort = in.ContainerPort
	}

	if in.ServicePort > 0 {
		m.ServicePort = in.ServicePort
	}

	if in.Image != "" {
		img := in.Image
		m.Image = &img
	}

	m.Status = model.MachineStatusPending

	if err := s.repo.UpdateMetadata(ctx, m); err != nil {
		return fmt.Errorf("update metadata: %w", err)
	}

	if s.helmChartDir == "" {
		return nil
	}

	inCreate := CreateMachineInput{
		Username:        in.Username,
		Name:            m.Name,
		ServiceKind:     m.ServiceKind,
		Version:         in.Version,
		ResourcesPreset: m.ResourcesPreset,
		Image:           in.Image,
		ContainerPort:   in.ContainerPort,
		ServicePort:     in.ServicePort,
		AccessScope:     m.AccessScope,
		EnableIngress:   in.EnableIngress,
		IngressHost:     in.IngressHost,
		ImageTarPath:    in.ImageTarPath,
	}

	mCopy := *m
	go func() {
		bg := context.Background()
		s.provisionMachine(bg, &mCopy, inCreate, true)
	}()

	return nil
}
