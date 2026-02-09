package models

type Login struct {
	Code       string `json:"code" bson:"code"`
	Email      string `json:"email" bson:"email"`
	PhoneNo    string `json:"phoneNo" bson:"phoneNo"`
	Collection string `json:"collection" bson:"collection"`
	Password   string `json:"password" bson:"password"`
}
