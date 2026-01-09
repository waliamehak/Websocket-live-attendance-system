package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/session"
)

func DebugRoutes(r *gin.Engine) {
	r.GET("/debug/session", func(c *gin.Context) {
		s := session.Get()
		if s == nil {
			c.JSON(200, gin.H{"session": nil})
			return
		}
		c.JSON(200, gin.H{
			"session": gin.H{
				"classId":   s.ClassID,
				"startedAt": s.StartedAt,
				"count":     len(s.Attendance),
			},
		})
	})
}
