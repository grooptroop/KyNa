package service

import (
	"context"
	"fmt"
	"log"
	"os/exec"
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
}

type DeleteMachineInput struct {
	ID       int64
	Username string
	Name     string
}

func (s *MachineService) ListMachines(ctx context.Context, username string) ([]model.UserMachine, error) {
	return s.repo.ListByUsername(ctx, username)
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

	m := &model.UserMachine{
		Username:        in.Username,
		Name:            in.Name,
		Mode:            mode,
		ServiceKind:     serviceKind,
		Status:          model.MachineStatusPending,
		ResourcesPreset: resources,
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

	svcEnabled := true
	svcType := "LoadBalancer"
	ingressEnabled := false

	switch serviceKind {
	case "web":
		svcEnabled = true
		svcType = "LoadBalancer"
		svcPort = containerPort
		ingressEnabled = false
	case "api":
		svcEnabled = true
		svcType = "ClusterIP"
		svcPort = containerPort
		ingressEnabled = false
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

	if in.Image != "" {
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

	releaseName := fmt.Sprintf("machine-%s-%s", m.Username, m.Name)
	ns := m.Username

	args := []string{
		"install",
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

	cmd := exec.CommandContext(ctx, "helm", args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("helm install for machine failed: %v, output: %s", err, string(out))
		return m, nil
	}

	svcName := fmt.Sprintf("hello-%s-%s", m.Username, m.Name)

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
		m.Status = model.MachineStatusReady
		if err := s.repo.UpdateStatusAndIP(ctx, m.ID, m.Status, m.ExternalIP); err != nil {
			log.Printf("failed to update machine status/ip in db: %v", err)
		}
	}

	// проверка CrashLoopBackOff
	crash, _ := s.checkPodCrashLoop(ctx, m.Username, m.Name)
	if crash {
		m.Status = model.MachineStatusFailed
		if err := s.repo.UpdateStatusAndIP(ctx, m.ID, m.Status, m.ExternalIP); err != nil {
			log.Printf("failed to update machine status to failed: %v", err)
		}
	}

	log.Printf("DEBUG: MachineService.CreateMachine end id=%d status=%s ip=%v",
		m.ID, m.Status, m.ExternalIP)

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
	labelSelector := fmt.Sprintf("app=hello-%s-%s", username, name)

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
		// не считаем это фатальной ошибкой: просто говорим, что краша не знаем
		return false, nil
	}

	reason := strings.TrimSpace(string(out))
	return reason == "CrashLoopBackOff", nil
}
