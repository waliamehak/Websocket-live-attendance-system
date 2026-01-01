package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Attendance struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	ClassID   primitive.ObjectID `bson:"classId" json:"classId"`
	StudentID primitive.ObjectID `bson:"studentId" json:"studentId"`
	Status    string             `bson:"status" json:"status"`
}
