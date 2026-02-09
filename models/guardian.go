package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Guardian struct {
	ID        primitive.ObjectID `json:"id" bson:"id"`
	Code      string             `json:"code" bson:"code"`
	RefID     string             `json:"refID" bson:"refID"`
	Name      string             `json:"guardianName" bson:"guardianName"`
	Mail      string             `json:"guardianMail" bson:"guardianMail"`
	Phone     string             `json:"guardianPhone" bson:"guardianPhone"`
	GovtID    string             `json:"guardianGovtID" bson:"guardianGovtID"`
	Password  string             `json:"password,omitempty" bson:"password,omitempty"`
	Signature string             `json:"signature,omitempty" bson:"signature,omitempty"`
	OTP       string             `json:"otp,omitempty" bson:"otp,omitempty"`
	Token     string             `json:"token,omitempty" bson:"token,omitempty"`
	IsActive  bool               `json:"isActive" bson:"isActive"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy string             `json:"updatedBy" bson:"updatedBy"`
}
