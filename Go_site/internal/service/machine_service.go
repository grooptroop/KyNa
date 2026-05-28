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
	Username    string
	Name        string
	Mode        string
	ServiceKind string
	Version     string
}

func (s *MachineService) ListMachines(ctx context.Context, username string) ([]model.UserMachine, error) {
	return s.repo.ListByUsername(ctx, username)
}

func (s *MachineService) CreateMachine(ctx context.Context, in CreateMachineInput) (*model.UserMachine, error) {
	log.Printf("DEBUG: MachineService.CreateMachine start username=%q name=%q mode=%q",
		in.Username, in.Name, in.Mode)

	if in.Username == "" {
		return nil, fmt.Errorf("MachineService.CreateMachine: empty username")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("MachineService.CreateMachine: empty name")
	}
	if in.Mode == "" {
		in.Mode = "app"
	}

	m := &model.UserMachine{
		Username: in.Username,
		Name:     in.Name,
		Mode:     in.Mode,
		Status:   model.MachineStatusPending,
	}

	// 1. сначала пишем в БД
	if err := s.repo.Create(ctx, m); err != nil {
		log.Printf("CREATE MACHINE DB ERROR: %v", err)
		return nil, err
	}

	log.Printf("DEBUG: Machine saved to DB id=%d username=%s name=%s",
		m.ID, m.Username, m.Name)

	// 2. если Helm выключен — выходим
	if s.helmChartDir == "" {
		return m, nil
	}

	// 3. деплой Helm
	releaseName := fmt.Sprintf("machine-%s-%s", m.Username, m.Name)
	ns := m.Username

	cmd := exec.CommandContext(
		ctx,
		"helm",
		"install",
		releaseName,
		s.helmChartDir,
		"--namespace", ns,
		"--set", fmt.Sprintf("username=%s", m.Username),
		"--set", fmt.Sprintf("name=%s", m.Name),
		"--set", fmt.Sprintf("mode=%s", m.Mode),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("helm install for machine failed: %v, output: %s", err, string(out))
		// не падаем, БД уже обновлена
		return m, nil
	}

	// 4. получаем external IP
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

	log.Printf("DEBUG: MachineService.CreateMachine end id=%d status=%s ip=%v",
		m.ID, m.Status, m.ExternalIP)

	return m, nil
}

type DeleteMachineInput struct {
	ID       int64
	Username string
	Name     string // опционально, если не хотим делать доп. SELECT
}

// DeleteMachine удаляет Helm-релиз и запись в user_machines
func (s *MachineService) DeleteMachine(ctx context.Context, in DeleteMachineInput) error {
	if in.Username == "" {
		return fmt.Errorf("DeleteMachine: empty username")
	}
	if in.ID == 0 && in.Name == "" {
		return fmt.Errorf("DeleteMachine: empty id and name")
	}

	name := in.Name

	// Если имя не передали, найдём по id
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

	// 1. helm uninstall
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
			// можно не падать, а продолжить, если хотим всё равно удалить запись
		}
	}

	// 2. удалить запись в user_machines
	if err := s.repo.DeleteByID(ctx, in.ID, in.Username); err != nil {
		return err
	}

	log.Printf("DEBUG: Machine deleted id=%d username=%s name=%s", in.ID, in.Username, name)

	return nil
}
