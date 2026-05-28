package service

import (
	"context"
	"log"
	"net/http"
	"os/exec"

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

func (s *UserService) ListUsers(ctx context.Context) ([]model.UserProvision, error) {
	return s.repo.List(ctx)
}

func (s *UserService) ListAdminUsers(ctx context.Context) ([]model.AdminUserView, error) {
	return s.repo.ListAdminUsers(ctx)
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

	return u, nil
}

func (s *UserService) DeleteUser(ctx context.Context, username string) error {
	if err := s.repo.DeleteByUsername(ctx, username); err != nil {
		return err
	}

	cmd := exec.CommandContext(
		ctx,
		"kubectl",
		"delete",
		"namespace",
		username,
		"--ignore-not-found=true",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("kubectl delete namespace %s failed: %v, output: %s", username, err, string(out))
	}

	return nil
}
