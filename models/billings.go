package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Billing struct {
	ID            primitive.ObjectID `json:"id" bson:"id"`
	Code          string             `json:"code" bson:"code"`
	AppointmentID string             `json:"appointmentID" bson:"appointmentID"`
	PatientID     string             `json:"patientID" bson:"patientID"`
	ServiceCharge float64            `json:"serviceCharge" bson:"serviceCharge"`
	Amount        float64            `json:"amount" bson:"amount"`
	Status        string             `json:"status" bson:"status"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy     string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy     string             `json:"updatedBy" bson:"updatedBy"`
}
