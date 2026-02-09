package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Tenant struct {
	ID            primitive.ObjectID `json:"id" bson:"id"`
	Code          string             `json:"code" bson:"code"`
	RoleCode      string             `json:"roleCode" bson:"roleCode"`
	Name          string             `json:"name" bson:"name"`
	Mail          string             `json:"mail" bson:"mail"`
	PhoneNo       string             `json:"phoneNo" bson:"phoneNo"`
	Password      string             `json:"password,omitempty" bson:"password"`
	Token         string             `json:"token,omitempty" bson:"token"`
	LoginAttempts int                `json:"loginAttempts" bson:"loginAttempts"`
	Reset         bool               `json:"reset" bson:"reset"`
	IsBlocked     bool               `json:"isBlocked" bson:"isBlocked"`
	IsActive      bool               `json:"isActive" bson:"isActive"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy     string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy     string             `json:"updatedBy" bson:"updatedBy"`
}
