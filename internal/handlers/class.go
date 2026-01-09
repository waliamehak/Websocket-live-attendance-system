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

type CreateClassRequest struct {
	ClassName string `json:"className" binding:"required"`
}

type AddStudentRequest struct {
	StudentID string `json:"studentId" binding:"required"`
}

func CreateClass(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" {
		utils.ErrorResponse(c, 403, "Forbidden, teacher access required")
		return
	}

	var req CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request schema")
		return
	}

	teacherID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))

	class := models.Class{
		ID:         primitive.NewObjectID(),
		ClassName:  req.ClassName,
		TeacherID:  teacherID,
		StudentIDs: []primitive.ObjectID{},
	}

	collection := database.DB.Collection("classes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, class)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to create class")
		return
	}

	utils.SuccessResponse(c, 201, gin.H{
		"_id":        class.ID,
		"className":  class.ClassName,
		"teacherId":  class.TeacherID,
		"studentIds": class.StudentIDs,
	})
}

func AddStudent(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" {
		utils.ErrorResponse(c, 403, "Forbidden, teacher access required")
		return
	}

	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid class ID")
		return
	}

	var req AddStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "Invalid request schema")
		return
	}

	studentID, err := primitive.ObjectIDFromHex(req.StudentID)
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid student ID")
		return
	}

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

	userCollection := database.DB.Collection("users")
	var student models.User
	err = userCollection.FindOne(ctx, bson.M{"_id": studentID, "role": "student"}).Decode(&student)
	if err != nil {
		utils.ErrorResponse(c, 404, "Student not found")
		return
	}

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": classID},
		bson.M{"$addToSet": bson.M{"studentIds": studentID}},
	)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to add student")
		return
	}

	err = collection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to fetch updated class")
		return
	}

	utils.SuccessResponse(c, 200, gin.H{
		"_id":        class.ID,
		"className":  class.ClassName,
		"teacherId":  class.TeacherID,
		"studentIds": class.StudentIDs,
	})
}

func GetClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, 400, "Invalid class ID")
		return
	}

	userID, _ := primitive.ObjectIDFromHex(c.GetString("userId"))
	role := c.GetString("role")

	collection := database.DB.Collection("classes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var class models.Class
	err = collection.FindOne(ctx, bson.M{"_id": classID}).Decode(&class)
	if err != nil {
		utils.ErrorResponse(c, 404, "Class not found")
		return
	}

	isTeacher := role == "teacher" && class.TeacherID == userID
	isEnrolled := false
	for _, sid := range class.StudentIDs {
		if sid == userID {
			isEnrolled = true
			break
		}
	}

	if !isTeacher && !isEnrolled {
		utils.ErrorResponse(c, 403, "Forbidden, not class teacher")
		return
	}

	type StudentResponse struct {
		ID    primitive.ObjectID `json:"_id"`
		Name  string             `json:"name"`
		Email string             `json:"email"`
	}

	var students []StudentResponse
	userCollection := database.DB.Collection("users")

	for _, studentID := range class.StudentIDs {
		var user models.User
		err := userCollection.FindOne(ctx, bson.M{"_id": studentID}).Decode(&user)
		if err == nil {
			students = append(students, StudentResponse{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
			})
		}
	}

	utils.SuccessResponse(c, 200, gin.H{
		"_id":       class.ID,
		"className": class.ClassName,
		"teacherId": class.TeacherID,
		"students":  students,
	})
}

func GetStudents(c *gin.Context) {
	role := c.GetString("role")
	if role != "teacher" {
		utils.ErrorResponse(c, 403, "Forbidden, teacher access required")
		return
	}

	collection := database.DB.Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{"role": "student"})
	if err != nil {
		utils.ErrorResponse(c, 500, "Failed to fetch students")
		return
	}
	defer cursor.Close(ctx)

	type StudentResponse struct {
		ID    primitive.ObjectID `json:"_id"`
		Name  string             `json:"name"`
		Email string             `json:"email"`
	}

	var students []StudentResponse
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err == nil {
			students = append(students, StudentResponse{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
			})
		}
	}

	utils.SuccessResponse(c, 200, students)
}
