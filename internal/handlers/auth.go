package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/database"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/models"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func Signup(c *gin.Context) {
	utils.ErrorResponse(c, 400, "Signup is handled by Auth0")
}

func Login(c *gin.Context) {
	utils.ErrorResponse(c, 400, "Login is handled by Auth0")
}

func Me(c *gin.Context) {
	userID := c.GetString("userId")

	collection := database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"auth0Id": userID}).Decode(&user)
	if err != nil {
		utils.SuccessResponse(c, 200, gin.H{
			"auth0Id": userID,
			"role":    c.GetString("role"),
		})
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"_id":     primitive.NewObjectID(),
		"auth0Id": user.Auth0ID,
		"name":    user.Name,
		"email":   user.Email,
		"role":    user.Role,
	})
}
