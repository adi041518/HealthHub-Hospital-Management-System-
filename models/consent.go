package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Consent struct {
	ID                primitive.ObjectID `json:"id" bson:"id"`
	Code              string             `json:"code" bson:"code"`
	ConsentType       string             `json:"consentType" bson:"consentType"` // General,Surgery,DataSharing
	ConsentPermission string             `json:"consentPermission" bson:"consentPermission"`
	RefID             string             `json:"refID" bson:"refID"` // PatientID
	CreatedAt         time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy         string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt         time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy         string             `json:"updatedBy" bson:"updatedBy"`
}
