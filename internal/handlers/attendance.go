package handlers

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/database"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/models"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/session"
	"github.com/waliamehak/WebSocket-live-attendance-system/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type StartAttendanceRequest struct {
	ClassID string `json:"classId" binding:"required"`
}

func StartAttendance(c *gin.Context) {
	if c.GetString("role") != "teacher" {
		utils.ErrorResponse(c, 403, "Forbidden, teacher access required")
		return
	}

	var req StartAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request schema")
		return
	}

	classID, err := primitive.ObjectIDFromHex(req.ClassID)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid class ID")
		return
	}

	teacherID := c.GetString("userId")

	classes := database.DB.Collection("classes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	if class.TeacherID != teacherID {
		utils.ErrorResponse(c, 403, "Forbidden, not class teacher")
		return
	}

	startedAt := time.Now().UTC().Format(time.RFC3339)
	roomID := primitive.NewObjectID().Hex()

	_, err = classes.UpdateOne(ctx, bson.M{"_id": classID}, bson.M{
		"$set": bson.M{"activeRoomId": roomID},
	})
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to create video room")
		return
	}

	session.Set(&session.ActiveSession{
		ClassID:    req.ClassID,
		StartedAt:  startedAt,
		Attendance: map[string]string{},
	})

	utils.SuccessResponse(c, 200, gin.H{
		"classId":   req.ClassID,
		"roomId":    roomID,
		"startedAt": startedAt,
	})
}

func GetMyAttendance(c *gin.Context) {
	if c.GetString("role") != "student" {
		utils.ErrorResponse(c, 403, "Forbidden, student access required")
		return
	}

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

	isEnrolled := false
	for _, sid := range class.StudentIDs {
		if sid == userID {
			isEnrolled = true
			break
		}
	}
	if !isEnrolled {
		utils.ErrorResponse(c, 403, "Forbidden, not enrolled in class")
		return
	}

	attendanceCol := database.DB.Collection("attendance")
	var attendance models.Attendance
	err = attendanceCol.FindOne(ctx, bson.M{
		"classId":   classID,
		"studentId": userID,
	}).Decode(&attendance)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.SuccessResponse(c, 200, gin.H{
				"classId": classID.Hex(),
				"status":  nil,
			})
			return
		}
		utils.ErrorResponse(c, 500, "Internal server error")
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"classId": classID.Hex(),
		"status":  attendance.Status,
	})
}
