package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Appointment struct {
	ID              primitive.ObjectID `json:"id" bson:"id"`
	Code            string             `json:"code" bson:"code"`
	PatientId       string             `json:"patientID" bson:"patientID"`
	DoctorId        string             `json:"doctorID" bson:"doctorID"`
	HospitalId      string             `json:"hospitalId" bson:"hospitalId"`
	MedicalRecordId string             `json:"medicalRecordId" bson:"medicalRecordId"`
	Slot            time.Time          `json:"slot" bson:"slot"`
	Status          string             `json:"status" bson:"status"`
	CreatedAt       time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy       string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt       time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy       string             `json:"updatedBy" bson:"updatedBy"`
}
type Patient struct {
	ID          primitive.ObjectID `json:"id" bson:"id"`
	Code        string             `json:"code" bson:"code"`
	Name        string             `json:"name" bson:"name"`
	Mail        string             `json:"mail" bson:"mail"`
	Phone       string             `json:"phoneNo" bson:"phoneNo"`
	Age         int                `json:"age" bson:"age"`
	Gender      string             `json:"gender" bson:"gender"`
	Password    string             `json:"password,omitempty" bson:"password,omitempty"`
	Token       string             `json:"token,omitempty" bson:"token,omitempty"`
	IsActive    bool               `json:"isActive" bson:"isActive"`
	Appointment []Appointment      `json:"appointment" bson:"appointment"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy   string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy   string             `json:"updatedBy" bson:"updatedBy"`
}
