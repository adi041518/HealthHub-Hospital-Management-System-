package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Nurse struct {
	ID        primitive.ObjectID `json:"id" bson:"id"`
	Code      string             `json:"code" bson:"code"`
	Name      string             `json:"name" bson:"name"`
	Mail      string             `json:"mail" bson:"mail"`
	PhoneNo   string             `json:"phoneNo" bson:"phoneNo"`
	Password  string             `json:"password,omitempty" bson:"password,omitempty"`
	Token     string             `json:"token,omitempty" bson:"token,omitempty"`
	IsActive  bool               `json:"isActive" bson:"isActive"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	CreatedBy string             `json:"createdBy" bson:"createdBy"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy string             `json:"updatedBy" bson:"updatedBy"`
}
