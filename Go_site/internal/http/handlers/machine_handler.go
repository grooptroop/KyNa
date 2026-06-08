package handlers

import (
	"io"
	"log"
	"net/http"
	"os"
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

func (h *MachineHandler) ListMachinesJSON(c *gin.Context) {
	username := middleware.CurrentUsername(c)
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	items, err := h.svc.ListMachinesJSON(c.Request.Context(), username)
	if err != nil {
		log.Printf("FAILED TO LIST MACHINES JSON: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list machines"})
		return
	}

	c.JSON(http.StatusOK, items)
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

	if err := c.Request.ParseMultipartForm(2 << 20); err != nil { // 2MiB более чем достаточно для .tar пути
		c.String(http.StatusBadRequest, "invalid form: "+err.Error())
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

	var imageTarPath string
	fileHeader, err := c.FormFile("image_tar")
	if err == nil && fileHeader != nil && fileHeader.Size > 0 {

		file, err := fileHeader.Open()
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to open uploaded tar: "+err.Error())
			return
		}
		defer file.Close()

		tmpFile, err := os.CreateTemp("", "uploaded-image-*.tar")
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to create temp file: "+err.Error())
			return
		}
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, file); err != nil {
			c.String(http.StatusInternalServerError, "failed to save uploaded tar: "+err.Error())
			return
		}

		imageTarPath = tmpFile.Name()
		log.Printf("uploaded docker tar saved as %s (%d bytes)", imageTarPath, fileHeader.Size)
	}

	log.Printf(
		"CREATE MACHINE: user=%s service=%s name=%s version=%s resources=%s image=%s cport=%d sport=%d",
		username, serviceKind, name, version, resourcesPreset, image, containerPort, servicePort,
	)

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
		ImageTarPath:    imageTarPath,
	}

	_, err = h.svc.CreateMachine(c.Request.Context(), in)
	if err != nil {
		log.Printf("CREATE MACHINE ERROR: %v", err)
		c.String(http.StatusInternalServerError, "failed to create machine")
		return
	}

	c.Redirect(http.StatusSeeOther, "/me/machines")
}

func (h *MachineHandler) UpdateMachine(c *gin.Context) {
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

	if err := c.Request.ParseMultipartForm(2 << 20); err != nil {
		c.String(http.StatusBadRequest, "invalid form: "+err.Error())
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
		c.String(http.StatusBadRequest, "service type is required")
		return
	}
	if name == "" {
		c.String(http.StatusBadRequest, "name is required")
		return
	}

	if resourcesPreset == "" {
		resourcesPreset = "small"
	}

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

	var imageTarPath string
	fileHeader, err := c.FormFile("image_tar")
	if err == nil && fileHeader != nil && fileHeader.Size > 0 {
		file, err := fileHeader.Open()
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to open uploaded tar: "+err.Error())
			return
		}
		defer file.Close()

		tmpFile, err := os.CreateTemp("", "uploaded-image-*.tar")
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to create temp file: "+err.Error())
			return
		}
		defer tmpFile.Close()

		if _, err := io.Copy(tmpFile, file); err != nil {
			c.String(http.StatusInternalServerError, "failed to save uploaded tar: "+err.Error())
			return
		}

		imageTarPath = tmpFile.Name()
		log.Printf("uploaded docker tar saved as %s (%d bytes)", imageTarPath, fileHeader.Size)
	}

	log.Printf(
		"UPDATE MACHINE: id=%d user=%s service=%s name=%s version=%s resources=%s image=%s cport=%d sport=%d",
		id, username, serviceKind, name, version, resourcesPreset, image, containerPort, servicePort,
	)

	in := service.UpdateMachineInput{
		ID:              id,
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
		ImageTarPath:    imageTarPath,
	}

	if err := h.svc.UpdateMachine(c.Request.Context(), in); err != nil {
		log.Printf("UPDATE MACHINE ERROR: %v", err)
		c.String(http.StatusInternalServerError, "failed to update machine")
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

func (h *MachineHandler) ShowEditForm(c *gin.Context) {
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

	m, err := h.svc.GetMachine(c.Request.Context(), username, id)
	if err != nil {
		log.Printf("FAILED TO GET MACHINE: %v", err)
		c.String(http.StatusNotFound, "machine not found")
		return
	}

	data := struct {
		Username string
		Machine  *model.UserMachine
		Mode     string
	}{
		Username: username,
		Machine:  m,
		Mode:     "edit",
	}

	c.HTML(http.StatusOK, "machine_new.tmpl", data)
}
