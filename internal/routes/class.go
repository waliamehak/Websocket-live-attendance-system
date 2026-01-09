package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/handlers"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/middleware"
)

func ClassRoutes(r *gin.Engine) {
	r.POST("/class", middleware.AuthMiddleware(), handlers.CreateClass)
	r.POST("/class/:id/add-student", middleware.AuthMiddleware(), handlers.AddStudent)
	r.GET("/class/:id", middleware.AuthMiddleware(), handlers.GetClass)
	r.GET("/students", middleware.AuthMiddleware(), handlers.GetStudents)
}
