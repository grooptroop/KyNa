package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/http/middleware"
	"github.com/grooptroop/KyNa/Go_site/internal/model"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
)

type MachineHandler struct {
	svc *service.MachineService
}

func NewMachineHandler(svc *service.MachineService) *MachineHandler {
	return &MachineHandler{svc: svc}
}

func (h *MachineHandler) ListMachines(c *gin.Context) {
	username := middleware.CurrentUsername(c)
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	machines, err := h.svc.ListMachines(c.Request.Context(), username)
	if err != nil {
		log.Printf("FAILED TO LIST MACHINES: %v", err)
		c.String(http.StatusInternalServerError, "failed to list machines")
		return
	}

	data := struct {
		Username string
		Machines []model.UserMachine
	}{
		Username: username,
		Machines: machines,
	}

	c.HTML(http.StatusOK, "machines.tmpl", data)
}

func (h *MachineHandler) ShowCreateForm(c *gin.Context) {
	username := middleware.CurrentUsername(c)
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	c.HTML(http.StatusOK, "machine_new.tmpl", nil)
}

func (h *MachineHandler) CreateMachine(c *gin.Context) {
	username := middleware.CurrentUsername(c)
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "invalid form")
		return
	}

	serviceKind := c.PostForm("service_kind")
	name := c.PostForm("name")
	version := c.PostForm("version")
	resourcesPreset := c.PostForm("resources_preset")
	image := c.PostForm("image")

	containerPortStr := c.PostForm("container_port")
	servicePortStr := c.PostForm("service_port")

	containerPort, _ := strconv.Atoi(containerPortStr)
	servicePort, _ := strconv.Atoi(servicePortStr)

	if serviceKind == "" {
		c.String(http.StatusBadRequest, "service type is required (choose a card)")
		return
	}
	if name == "" {
		c.String(http.StatusBadRequest, "name is required")
		return
	}

	if resourcesPreset == "" {
		resourcesPreset = "small"
	}

	log.Printf(
		"CREATE MACHINE: user=%s service=%s name=%s version=%s resources=%s image=%s cport=%d sport=%d",
		username, serviceKind, name, version, resourcesPreset, image, containerPort, servicePort,
	)

	enableIngress := c.PostForm("enable_ingress") == "on"
	ingressHost := c.PostForm("ingress_host")

	accessScope := c.PostForm("access_scope")

	if accessScope == "public" && serviceKind == "api" {
		enableIngress = c.PostForm("enable_ingress") == "on"
		if enableIngress {
			ingressHost = strings.TrimSpace(c.PostForm("ingress_host"))
			if ingressHost == "" {
				enableIngress = false
			}
		}
	}

	in := service.CreateMachineInput{
		Username:        username,
		Name:            name,
		ServiceKind:     serviceKind,
		Version:         version,
		ResourcesPreset: resourcesPreset,
		Image:           image,
		ContainerPort:   containerPort,
		ServicePort:     servicePort,
		AccessScope:     accessScope,
		EnableIngress:   enableIngress,
		IngressHost:     ingressHost,
	}

	_, err := h.svc.CreateMachine(c.Request.Context(), in)
	if err != nil {
		log.Printf("CREATE MACHINE ERROR: %v", err)
		c.String(http.StatusInternalServerError, "failed to create machine")
		return
	}

	c.Redirect(http.StatusSeeOther, "/me/machines")
}

func (h *MachineHandler) DeleteMachine(c *gin.Context) {
	username := middleware.CurrentUsername(c)
	if username == "" {
		c.Redirect(http.StatusSeeOther, "/auth/login")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	in := service.DeleteMachineInput{
		ID:       id,
		Username: username,
	}

	if err := h.svc.DeleteMachine(c.Request.Context(), in); err != nil {
		log.Printf("DELETE MACHINE ERROR: %v", err)
		c.String(http.StatusInternalServerError, "failed to delete machine")
		return
	}

	c.Redirect(http.StatusSeeOther, "/me/machines")
}

func (h *MachineHandler) AdminUserMachines(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	machines, err := h.svc.ListMachines(c.Request.Context(), username)
	if err != nil {
		log.Printf("FAILED TO LIST MACHINES FOR USER %s: %v", username, err)
		c.String(http.StatusInternalServerError, "failed to list machines")
		return
	}

	data := struct {
		Username string
		Machines []model.UserMachine
	}{
		Username: username,
		Machines: machines,
	}

	c.HTML(http.StatusOK, "admin_user_machines.tmpl", data)
}

func (h *MachineHandler) AdminDeleteMachine(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid id")
		return
	}

	in := service.DeleteMachineInput{
		ID:       id,
		Username: username,
	}

	if err := h.svc.DeleteMachine(c.Request.Context(), in); err != nil {
		log.Printf("ADMIN DELETE MACHINE ERROR: %v", err)
		c.String(http.StatusInternalServerError, "failed to delete machine")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/users/"+username+"/machines")
}
