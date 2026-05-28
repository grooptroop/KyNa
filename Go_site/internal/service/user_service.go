package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
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

type UserHandler struct {
	svc *UserService
}

func NewUserHandler(svc *UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListUsers(c.Request.Context())
	if err != nil {
		log.Printf("FAILED TO LIST USERS: %v", err)
		c.String(http.StatusInternalServerError, "failed to list users")
		return
	}

	c.HTML(http.StatusOK, "users.tmpl", gin.H{
		"Users": users,
	})
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

	if s.helmChartDir == "" {
		return u, nil
	}

	releaseName := fmt.Sprintf("user-%s", u.Username)

	cmd := exec.CommandContext(
		ctx,
		"helm",
		"install",
		releaseName,
		s.helmChartDir,
		"--set", fmt.Sprintf("username=%s", u.Username),
		"--set", fmt.Sprintf("domain=%s", u.Domain),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return u, fmt.Errorf("helm install failed: %v, output: %s", err, string(out))
	}
	svcName := fmt.Sprintf("hello-%s", u.Username)
	ns := u.Username

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
		log.Printf("kubectl get svc external ip failed: %v, output: %s", ipErr, string(ipOut))
	}

	if externalIP != "" {
		u.ExternalIP = &externalIP
		u.Status = model.StatusReady
		if err := s.repo.UpdateStatusAndIP(ctx, u.Username, u.Status, u.ExternalIP); err != nil {
			log.Printf("failed to update status/ip in db: %v", err)
		}
	}

	return u, nil
}

func (s *UserService) DeleteUser(ctx context.Context, username string) error {
	if err := s.repo.DeleteByUsername(ctx, username); err != nil {
		return err
	}

	releaseName := fmt.Sprintf("user-%s", username)

	cmd := exec.CommandContext(
		ctx,
		"helm",
		"uninstall",
		releaseName,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("helm uninstall failed: %v, output: %s", err, string(out))

	}

	nsCmd := exec.CommandContext(
		ctx,
		"kubectl",
		"delete",
		"namespace",
		username,
		"--ignore-not-found=true",
	)

	nsOut, nsErr := nsCmd.CombinedOutput()
	if nsErr != nil {
		log.Printf("kubectl delete namespace failed: %v, output: %s", nsErr, string(nsOut))
	}

	return nil
}
