package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Prescription struct {
	ID            primitive.ObjectID `json:"id" bson:"id"`
	Code          string             `json:"code" bson:"code"`
	AppointmentID string             `json:"appointmentID" bson:"appointmentID"`
	PatientID     string             `json:"patientID" bson:"patientID"`
	Medicines     []string           `json:"medicines" bson:"medicines"`
	Dosage        map[string]string  `json:"dosage" bson:"dosage"`
	Limit         []string           `json:"limit" bson:"limit"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy     string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy     string             `json:"updatedBy" bson:"updatedBy"`
}
