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

// keep track of active session in memory
type ActiveSession struct {
	ClassID    string
	StartedAt  string
	Attendance map[string]string // studentId -> status
}

var activeSession *ActiveSession

type StartAttendanceRequest struct {
	ClassID string `json:"classId" binding:"required"`
}

func StartAttendance(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" {
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

	// check if class exists and teacher owns it
	collection := database.DB.Collection("classes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var class models.Class
	err = collection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		utils.ErrorResponse(c, 404, "Class not found")
		return
	}

	teacherID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	if class.TeacherID != teacherID {
		utils.ErrorResponse(c, 403, "Forbidden, not class teacher")
		return
	}

	// start new session
	activeSession = &ActiveSession{
		ClassID:    req.ClassID,
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		Attendance: make(map[string]string),
	}

	utils.SuccessResponse(c, 200, gin.H{
		"classId":   activeSession.ClassID,
		"startedAt": activeSession.StartedAt,
	})
}

func GetMyAttendance(c *gin.Context) {
	role := c.GetString("role")
	if role != "student" {
		utils.ErrorResponse(c, 403, "Forbidden, student access required")
		return
	}

	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid class ID")
		return
	}

	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	// check if student is enrolled
	collection := database.DB.Collection("classes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var class models.Class
	err = collection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		utils.ErrorResponse(c, 404, "Class not found")
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

	// check db for persisted attendance
	attendanceCollection := database.DB.Collection("attendance")
	var attendance models.Attendance
	err = attendanceCollection.FindOne(ctx, bson.M{
		"classId":   classID,
		"studentId": userID,
	}).Decode(&attendance)

	if err != nil {
		// not found in db yet
		utils.SuccessResponse(c, 200, gin.H{
			"classId": classID.Hex(),
			"status":  nil,
		})
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"classId": classID.Hex(),
		"status":  attendance.Status,
	})
}
