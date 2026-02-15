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
	"go.mongodb.org/mongo-driver/mongo"
)

func GetRoomInfo(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid class ID")
		return
	}

	userID := c.GetString("userId")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	classes := database.DB.Collection("classes")
	var class models.Class
	err = classes.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.ErrorResponse(c, 404, "Class not found")
			return
		}
		utils.ErrorResponse(c, 500, "Internal server error")
		return
	}

	isTeacher := class.TeacherID == userID
	isStudent := false
	for _, sid := range class.StudentIDs {
		if sid == userID {
			isStudent = true
			break
		}
	}

	if !isTeacher && !isStudent {
		utils.ErrorResponse(c, 403, "Forbidden, not authorized for this class")
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"classId":      classID.Hex(),
		"activeRoomId": class.ActiveRoomID,
		"isActive":     class.ActiveRoomID != "",
	})
}
