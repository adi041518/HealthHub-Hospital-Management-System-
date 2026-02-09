package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const ERR_WHILE_FETCHING_TENANT string = "Error from findOne while fetching tenant: "

/*
CreateTenant handles creating a Tenant user.
It validates email/phone, generates employee code, fetches roleCode,
prepares the data, and inserts the record into MongoDB.
*/
func CreateTenant(c *gin.Context, data map[string]interface{}) error {

	if err := common.ValidateUserInput(data); err != nil {
		log.Println("Error from validateUserInput:", err)
		return err
	}
	collection, err := common.FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return err
	}
	code, CreatedBy, err := common.CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return err
	}
	otp, err := common.GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GenerateAndHashOTP", err)
		return err
	}
	log.Println("otp:", otp)
	tenantId := code
	if err := common.PrepareUser(data, code, CreatedBy, tenantId); err != nil {
		log.Println("Error from PrepareUser", err)
		return err
	}

	key := util.TenantKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from SetCache:", err)
		return errors.New("Error from setCache")
	}
	if _, err := common.SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := common.CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}

	subject := "Your Tenant OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for Tenant verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = common.SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Get tenant from cache,if exists return tenant
* If not exists,fetch tenant from dataBase
* Set in Cache
 */
func FetchTenantByCode(c *gin.Context, tenantId string) (map[string]interface{}, error) {
	superAdminId := c.GetString("code")
	collFromContext := c.GetString("collection")
	if collFromContext != util.SuperAdminCollection {
		log.Println("This user doesnot have access")
		return nil, errors.New("This user doesnot have access")
	}
	coll := util.TenantCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":      tenantId,
		"createdBy": superAdminId,
	}
	key := util.TenantKey + tenantId
	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	if err != nil && exists {
		return cached, nil
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println(ERR_WHILE_FETCHING_TENANT, err)
		return nil, err
	}
	err = redis.SetCache(c, key, result)
	log.Println("Unable to set in cache")
	return result, nil
}

/*
It returns an array of documnets
where it matches with the filter given with it and perform
the Find all Function
*/
func FetchAllTenants(c *gin.Context) ([]interface{}, error) {
	collection := db.OpenCollections(util.TenantCollection)
	results, err := db.FindAll(c, collection, nil, nil)
	if err != nil {
		return []interface{}{}, err
	}
	log.Println("tenants are", results)
	return results, nil
}

/*
UpdateTenantByCode updates an existing Tenant document using its unique tenant code.

Workflow:
1. Validate tenant code input
2. Check if tenant exists
3. Parse updateData and prepare update fields
4. Normalize DOB if provided
5. Apply updates to MongoDB
6. Refresh cache (delete old → write new)
7. Return updated tenant document
*/
func UpdateTenantByCode(c *gin.Context, tenantId string, data map[string]interface{}) (string, error) {

	updateFields, err := parseTenantUpdateFields(c, data)
	if err != nil {
		log.Println("Error from parseTenantUpdateFields: ", err)
		return "", err
	}
	collection := db.OpenCollections(util.TenantCollection)
	err = common.CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}
	filter := bson.M{
		"code": tenantId,
	}
	tenant := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &tenant)
	if err != nil {
		log.Println(ERR_WHILE_FETCHING_TENANT, err)
		return "", err
	}
	err = updateTenantInDB(tenantId, updateFields)
	if err != nil {
		log.Println("Error from updateTenantInDB: ", err)
		return "", err
	}
	updatedTenant := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &updatedTenant)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	key := util.TenantKey + tenantId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, updatedTenant); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return "Updated successfully", nil
}

/*
parseTenantUpdateFields extracts and validates the update fields from updateData.
It also normalizes DOB and sets metadata like updatedAt and updatedBy.
*/
func parseTenantUpdateFields(c *gin.Context, updateData map[string]interface{}) (bson.M, error) {

	update := bson.M{}

	if v, ok := updateData["name"].(string); ok && strings.TrimSpace(v) != "" {
		update["name"] = v
	}

	if v, ok := updateData["email"].(string); ok && strings.TrimSpace(v) != "" {
		update["email"] = v
	}

	if v, ok := updateData["phoneNo"].(string); ok && strings.TrimSpace(v) != "" {
		update["phoneNo"] = v
	}

	if v, ok := updateData["dob"].(string); ok && strings.TrimSpace(v) != "" {
		modDob, err := common.NormalizeDate(v)
		if err != nil {
			return nil, err
		}
		update["dob"] = modDob
	}

	if len(update) == 0 {
		return nil, errors.New(util.NO_FIELDS_PROVIDED_TO_UPDATE)
	}

	update["updatedAt"] = time.Now()
	update["updatedBy"] = c.GetString("code")

	return update, nil
}

/*
updateTenantInDB applies the parsed updates to the tenant document in MongoDB.
*/
func updateTenantInDB(code string, update bson.M) error {

	collection := db.OpenCollections(util.TenantCollection)
	filter := bson.M{"code": code}

	res, err := db.UpdateOne(context.Background(), collection, filter, bson.M{"$set": update})
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return err
	}
	log.Println(res.ModifiedCount)
	return nil
}

/*
It deletes the document which matches the code given in the
tenant where it used delete one function
*/
func DeleteTenantByCode(c *gin.Context, tenantId string) error {
	superAdmin := c.GetString("code")
	if tenantId == "" {
		return errors.New("tenant code required")
	}
	collection := db.OpenCollections(util.TenantCollection)
	filter := bson.M{"code": tenantId}
	res := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, res)
	if err != nil {
		log.Println(ERR_WHILE_FETCHING_TENANT, err)
		return err
	}
	if superAdmin != res["createdBy"].(string) {
		log.Println("User doesnot have access")
		return errors.New(util.SUPER_ADMIN_DOESNOT_HAVE_ACCESS)
	}
	delete, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		return err
	}
	key := util.TenantKey + tenantId
	log.Println("Tenant cache key: ", key)
	err = redis.DeleteCache(c, key)
	if err != nil {
		return err
	}
	log.Println(delete.DeletedCount)
	return nil
}
