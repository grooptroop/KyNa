package service

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/grooptroop/KyNa/Go_site/internal/repository"
)

type UserService struct {
	repo         *repository.UserRepository
	helmChartDir string
}

func NewUserService(repo *repository.UserRepository, helmChartDir string) *UserService {
	return &UserService{
		repo:         repo,
		helmChartDir: helmChartDir,
	}
}

func (s *UserService) ListUsers(ctx context.Context) ([]model.UserProvision, error) {
	return s.repo.List(ctx)
}

type CreateUserInput struct {
	Username string
	Domain   string
	Mode     string
}

func (s *UserService) CreateUser(ctx context.Context, in CreateUserInput) (*model.UserProvision, error) {
	u := &model.UserProvision{
		Username: in.Username,
		Domain:   in.Domain,
		Mode:     in.Mode,
		Status:   model.StatusPending,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	// Если каталог чарта не задан — пропускаем развёртывание
	if s.helmChartDir == "" {
		return u, nil
	}

	// Имя релиза, например user-admin
	releaseName := fmt.Sprintf("user-%s", u.Username)

	// Команда helm install:
	// helm install user-<username> <chartDir> --set username=...,domain=...
	cmd := exec.CommandContext(
		ctx,
		"helm",
		"install",
		releaseName,
		s.helmChartDir,
		"--set", fmt.Sprintf("username=%s", u.Username),
		"--set", fmt.Sprintf("domain=%s", u.Domain),
	)

	// Важно: helm должен видеть kubeconfig, см. ниже
	out, err := cmd.CombinedOutput()
	if err != nil {
		// на практике лучше логировать out, а не терять
		return u, fmt.Errorf("helm install failed: %v, output: %s", err, string(out))
	}

	return u, nil
}
