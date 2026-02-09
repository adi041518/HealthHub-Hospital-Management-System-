package migrations

import (
	"context"
	"log"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	"github.com/KanapuramVaishnavi/Core/util"
	"go.mongodb.org/mongo-driver/bson"
)

func AddPharmacistIdField() {
	ctx := context.Background()
	bill := util.BillCollection
	result, err := db.DB.Collection(bill).UpdateMany(
		ctx,
		bson.M{"pharmacistId": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"pharmacistId": ""}},
	)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}
	log.Printf("Migration applied: %d documents updated\n", result.ModifiedCount)
}
