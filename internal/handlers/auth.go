package handlers

import (
	"net/http"

	"github.com/ar4ie13/loyaltysystem/internal/apperrors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// authMiddleware used as middleware for authentication
func (h *Handlers) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			userUUID uuid.UUID
			err      error
		)

		cookie, err := c.Cookie("user_uuid")

		if err != nil || cookie == "" {
			h.zlog.Debug().Msg("user is not authorized")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": apperrors.ErrUserIsNotAuthorized.Error()})
			return
		} else {
			// Checking existing cookie
			userUUID, err = h.auth.ValidateUserUUID(cookie)
			if err != nil {
				h.zlog.Debug().Msgf("error validating user UUID: %v", err)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid cookie"})
				return
			}
		}
		// Set user UUID in the context for downstream handlers
		c.Set("user_uuid", userUUID.String())
		c.Next()
	}
}
