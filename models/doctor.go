package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Doctor struct {
	ID            primitive.ObjectID `json:"id" bson:"id"`
	Code          string             `json:"code" bson:"code"`
	Name          string             `json:"name" bson:"name"`
	Mail          string             `json:"mail" bson:"mail"`
	Department    string             `json:"department" bson:"department"`
	Availability  []time.Time        `json:"availability" bson:"availability"`
	PhoneNo       string             `json:"phoneNo" bson:"phoneNo"`
	Password      string             `json:"password,omitempty" bson:"password,omitempty"`
	Token         string             `json:"token,omitempty" bson:"token,omitempty"`
	TenantId      string             `json:"tenantId" bson:"tenantId"`
	LoginAttempts int                `json:"loginAttempts" bson:"loginAttempts"`
	Reset         bool               `json:"reset" bson:"reset"`
	IsBlocked     bool               `json:"isBlocked" bson:"isBlocked"`
	IsActive      bool               `json:"isActive" bson:"isActive"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy     string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy     string             `json:"updatedBy" bson:"updatedBy"`
}
