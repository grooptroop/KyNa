package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/http/handlers"
	"github.com/grooptroop/KyNa/Go_site/internal/http/middleware"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
)

func Register(
	r *gin.Engine,
	userHandler *handlers.UserHandler,
	machineHandler *handlers.MachineHandler,
	authHandler *handlers.AuthHandler,
	sessions *service.SessionStore,
) {
	r.GET("/auth/register", authHandler.ShowRegister)
	r.POST("/auth/register", authHandler.Register)
	r.GET("/auth/login", authHandler.ShowLogin)
	r.POST("/auth/login", authHandler.Login)
	r.GET("/auth/logout", authHandler.Logout)

	authRequired := middleware.AuthMiddleware(sessions, true)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/me/machines")
	})

	admin := r.Group("/admin")
	admin.Use(authRequired)
	{
		admin.GET("/users", userHandler.ListUsers)
		admin.GET("/users/new", userHandler.ShowCreateForm)
		admin.POST("/users", userHandler.CreateUser)
		admin.POST("/users/:username/delete", userHandler.DeleteUser)
		admin.GET("/users/:username/machines", machineHandler.AdminUserMachines)
		admin.POST("/users/:username/machines/:id/delete", machineHandler.AdminDeleteMachine)
	}

	me := r.Group("/me")
	me.Use(authRequired)
	{
		me.GET("/machines", machineHandler.ListMachines)
		me.GET("/machines/new", machineHandler.ShowCreateForm)
		me.POST("/machines", machineHandler.CreateMachine)
		me.POST("/machines/:id/delete", machineHandler.DeleteMachine)
		me.GET("/machines/json", machineHandler.ListMachinesJSON)

		me.GET("/machines/:id/edit", machineHandler.ShowEditForm)
		me.POST("/machines/:id", machineHandler.UpdateMachine)
		me.GET("/machines/history", machineHandler.ShowUserHistory)
	}

}
