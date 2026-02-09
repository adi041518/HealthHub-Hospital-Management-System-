package migrations

import (
	"context"
	"log"
	"strconv"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	"github.com/KanapuramVaishnavi/Core/util"
	"go.mongodb.org/mongo-driver/bson"
)

func ChangeLoginAttemptsType() {
	ctx := context.Background()
	tenantColl := util.TenantCollection
	cursor, _ := db.DB.Collection(tenantColl).Find(ctx, bson.M{})
	for cursor.Next(ctx) {
		tenant := make(map[string]interface{})
		cursor.Decode(&tenant)
		loginAttemptsType := tenant["loginAttempts"].(int32)
		loginAttempts := strconv.Itoa(int(loginAttemptsType))
		db.DB.Collection(tenantColl).UpdateOne(ctx, bson.M{"code": tenant["code"]},
			bson.M{"$set": bson.M{
				"loginAttempts": loginAttempts,
				"updatedAt":     time.Now(),
			}})
	}
	log.Println("Changed the type of loginAttempts")
}
