package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.svc.ListAdminUsers(c.Request.Context())
	if err != nil {
		log.Printf("FAILED TO LIST USERS: %v", err)
		c.String(http.StatusInternalServerError, "failed to list users")
		return
	}

	c.HTML(http.StatusOK, "users.tmpl", gin.H{
		"Users": users,
	})
}

type createUserForm struct {
	Username string `form:"username" binding:"required"`
	Domain   string `form:"domain" binding:"required"`
	Mode     string `form:"mode" binding:"required"`
}

func (h *UserHandler) ShowCreateForm(c *gin.Context) {
	c.HTML(http.StatusOK, "user_new.tmpl", nil)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var form createUserForm
	if err := c.ShouldBind(&form); err != nil {
		c.String(http.StatusBadRequest, "invalid form")
		return
	}

	u, err := h.svc.CreateUser(c.Request.Context(), service.CreateUserInput{
		Username: form.Username,
		Domain:   form.Domain,
		Mode:     form.Mode,
	})
	if err != nil {
		// ВОТ ЭТО добавь, чтобы увидеть реальную ошибку
		log.Printf("failed to create user: %v", err)

		c.String(http.StatusInternalServerError, "failed to create user")
		return
	}

	log.Printf("user created: %+v", u)

	c.Redirect(http.StatusSeeOther, "/users")
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	if err := h.svc.DeleteUser(c.Request.Context(), username); err != nil {
		log.Printf("failed to delete user %s: %v", username, err)
		c.String(http.StatusInternalServerError, "failed to delete user")
		return
	}

	c.Redirect(http.StatusSeeOther, "/users")
}
