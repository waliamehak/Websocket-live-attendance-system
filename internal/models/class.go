package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Class struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	ClassName    string             `bson:"className" json:"className"`
	TeacherID    string             `bson:"teacherId" json:"teacherId"`
	StudentIDs   []string           `bson:"studentIds" json:"studentIds"`
	ActiveRoomID string             `bson:"activeRoomId,omitempty" json:"activeRoomId,omitempty"`
}
