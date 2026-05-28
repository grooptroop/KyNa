package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/http/middleware"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) ShowRegister(c *gin.Context) {
	c.HTML(http.StatusOK, "register.tmpl", nil)
}

func (h *AuthHandler) Register(c *gin.Context) {
	in := service.RegisterInput{
		Username: c.PostForm("username"),
		Email:    c.PostForm("email"),
		Password: c.PostForm("password"),
	}

	_, err := h.svc.Register(c.Request.Context(), in)
	if err != nil {
		log.Printf("FAILED TO REGISTER: %v", err)
		c.String(http.StatusBadRequest, "failed to register user")
		return
	}

	c.Redirect(http.StatusFound, "/auth/login")
}

func (h *AuthHandler) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "login.tmpl", nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	in := service.LoginInput{
		Username: c.PostForm("username"),
		Password: c.PostForm("password"),
	}

	_, sessionID, err := h.svc.Login(c.Request.Context(), in)
	if err != nil {
		log.Printf("FAILED TO LOGIN: %v", err)
		c.String(http.StatusUnauthorized, "invalid username or password")
		return
	}

	c.SetCookie(
		middleware.SessionCookieName,
		sessionID,
		3600*24,
		"/",
		"",
		false,
		true,
	)

	c.Redirect(http.StatusFound, "/me/machines")
}

func (h *AuthHandler) Logout(c *gin.Context) {
	cookie, err := c.Cookie(middleware.SessionCookieName)
	if err == nil && cookie != "" {
		h.svc.Logout(cookie)
	}

	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/auth/login")
}
