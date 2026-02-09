package migrations

import (
	"context"
	"log"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	"github.com/KanapuramVaishnavi/Core/util"
	"go.mongodb.org/mongo-driver/bson"
)

func RemovePharamcistIdFromBill() {
	ctx := context.Background()
	bill := util.BillCollection
	updated, err := db.DB.Collection(bill).UpdateMany(ctx, bson.M{}, bson.M{"$unset": bson.M{"pharmacistId": ""}})
	if err != nil {
		log.Fatal("Unable to remove pharmacistId from bill")
	}
	log.Printf("%d pharmacistId field deleted successfully in bill", updated.ModifiedCount)
}
