package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Medicine struct {
	ID           primitive.ObjectID `json:"id" bson:"id"`
	Code         string             `json:"code" bson:"code"`
	MedicineName string             `json:"medicineName" bson:"medicineName"`
	DrugType     string             `json:"drugType" bson:"drugType"`
	Dosage       string             `json:"dosage" bson:"dosage"`
	NoOfStrips   int                `json:"noOfStrips" bson:"noOfStrips"`
	Required     bool               `json:"required" bson:"required"`
	CreatedAt    time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy    string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt    time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy    string             `json:"updatedBy" bson:"updatedBy"`
}
