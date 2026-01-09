package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/handlers"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/middleware"
)

func AttendanceRoutes(r *gin.Engine) {
	r.POST("/attendance/start", middleware.AuthMiddleware(), handlers.StartAttendance)
	r.GET("/class/:id/my-attendance", middleware.AuthMiddleware(), handlers.GetMyAttendance)
}
