package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Class struct {
	ID         primitive.ObjectID   `bson:"_id,omitempty" json:"_id,omitempty"`
	ClassName  string               `bson:"className" json:"className"`
	TeacherID  primitive.ObjectID   `bson:"teacherId" json:"teacherId"`
	StudentIDs []primitive.ObjectID `bson:"studentIds" json:"studentIds"`
}
