package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(c, 401, "Unauthorized, token missing or invalid")
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(authHeader)

		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			utils.ErrorResponse(c, 401, "Unauthorized, token missing or invalid")
			c.Abort()
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
