package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/http/handlers"
)

func Register(r *gin.Engine, userHandler *handlers.UserHandler) {
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/users")
	})
	r.GET("/users", userHandler.ListUsers)
	r.GET("/users/new", userHandler.ShowCreateForm)
	r.POST("/users", userHandler.CreateUser)
	r.POST("/users/:username/delete", userHandler.DeleteUser)
}
