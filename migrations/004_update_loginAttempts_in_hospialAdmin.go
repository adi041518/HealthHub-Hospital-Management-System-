package migrations

import (
	"context"
	"log"
	"strconv"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	"github.com/KanapuramVaishnavi/Core/util"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateLoginAttemptsInHospitalAdmin() {
	ctx := context.Background()
	coll := util.HospitalCollection
	cursor, err := db.DB.Collection(coll).Find(ctx, bson.M{})
	if err != nil {
		log.Println("Error while find all documents in hospitalCollection")
		log.Fatal("Error while finding all documents in hospital")
	}
	for cursor.Next(ctx) {
		hospital := make(map[string]interface{})
		err := cursor.Decode(hospital)
		if err != nil {
			log.Println("Error while decoding the hospital ")
			log.Fatal("Error while decoding the hospital")
		}
		loginAttemptsVal := hospital["loginAttempts"].(int32)
		loginAttempts := strconv.Itoa(int(loginAttemptsVal))
		updated, err := db.DB.Collection(coll).UpdateOne(ctx, bson.M{"code": hospital["code"]}, bson.M{"$set": bson.M{"loginAttempts": loginAttempts}})
		if err != nil {
			log.Println("Error while updating the document")
			log.Fatal("Error while updating the document")
		}
		log.Println("updatedCount: ", updated.ModifiedCount)
	}
	log.Println("Updated the type of loginAttempts in hospital")
}
