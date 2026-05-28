package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/grooptroop/KyNa/Go_site/internal/service"
)

const (
	SessionCookieName = "session_id"
	ContextUserKey    = "currentUser"
)

func AuthMiddleware(sessions *service.SessionStore, require bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(SessionCookieName)
		if err != nil || cookie == "" {
			if require {
				c.Redirect(http.StatusFound, "/auth/login")
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if username, ok := sessions.Get(cookie); ok {
			c.Set(ContextUserKey, username)
			c.Next()
			return
		}

		if require {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func CurrentUsername(c *gin.Context) string {
	v, exists := c.Get(ContextUserKey)
	if !exists {
		return ""
	}
	username, _ := v.(string)
	return username
}
