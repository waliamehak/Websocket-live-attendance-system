package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/database"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/routes"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if err := database.ConnectDB(mongoURI); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": true,
			"data": gin.H{
				"status": "Server is running",
			},
		})
	})

	routes.AuthRoutes(r)
	routes.ClassRoutes(r)
	routes.AttendanceRoutes(r)

	log.Printf("Server running on port %s", port)
	r.Run(":" + port)
}
