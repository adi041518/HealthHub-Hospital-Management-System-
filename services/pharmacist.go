package services

import (
	"errors"
	"fmt"
	"log"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
* Validate user inputs first
* Fetch collection name from the roleCode given
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from context
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* Save to db and cache
* Send mail
 */
func CreatePharmacist(ctx *gin.Context, body map[string]interface{}) error {
	err := common.ValidateUserInput(body)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return err
	}
	collection, err := common.FetchCollectionFromRoleDoc(ctx, body["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return err
	}
	code, CreatedBy, err := common.CheckerAndGenerateUserCodes(ctx, collection, body["email"].(string), body["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return err
	}

	otp, err := common.GenerateAndHashOTP(body)
	if err != nil {
		log.Println("Error from GenerateAndHashOTP", err)
		return err
	}
	log.Println("otp:", otp)

	tenantId, err := common.GetTenantIdFromContext(ctx)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return err
	}
	log.Println("tenantId from context: ", tenantId)

	if err := common.PrepareUser(body, code, CreatedBy, tenantId); err != nil {
		log.Println("Error from PrepareUser", err)
		return err
	}
	key := util.PharamacistKey + code
	err = redis.SetCache(ctx, key, body)
	if err != nil {
		log.Println("Error while caching new pharmacist: ", err)
	}
	if _, err := common.SaveUserToDB(collection, body); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	if err := common.CreateLoginRecord(ctx, collection, code, body["email"].(string), body["phoneNo"].(string), body["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}

	subject := "Your Pharmacist OTP Verification"
	mbody := fmt.Sprintf("Hello %s,\n\nYour OTP for Pharmacist verification is: %s\n\nThank you!", body["name"].(string), otp)

	err = common.SendOTPToMail(body["email"].(string), subject, mbody)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* isSuperAdmin,tenantId,collection and code from context
* Pass those fields and key fetch from cache
* If exists,check who can access(superAdmin,tenantAdmin,hospitalAdmin)
* If not found go to db search for the document
* Search the doument, check who can access pharmacist
* If comparision works then return the pharmacist
 */
func FetchPharmacistByCode(c *gin.Context, pharmacistId string) (map[string]interface{}, error) {

	coll := util.PharmacistCollection
	key := util.PharamacistKey + pharmacistId
	isSuperAdmin := c.GetBool("isSuperAdmin")
	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	ctxCollection := c.GetString("collection")
	cached := make(map[string]interface{})

	cached, exists, err := common.FetchByCodeFromCache(c, key, isSuperAdmin, tenantId, code, ctxCollection)
	if err != nil {
		log.Println("Error from FetchByCodeFromCache: ", err)
		return nil, err
	}
	if exists && cached != nil {
		return cached, nil
	}

	result := make(map[string]interface{})
	collection := db.OpenCollections(coll)
	log.Println("Error from getCache:", err)
	filter := bson.M{
		"code": pharmacistId,
	}

	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return nil, err
	}
	err = common.HasAccess(isSuperAdmin, ctxCollection, tenantId, code, result)
	if err != nil {
		log.Println("Error from HasAccess: ", err)
		return nil, err
	}

	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache")
		return nil, err
	}

	return result, nil
}

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfPharmacist
* Return them
 */
func FetchAllPharmacist(c *gin.Context) ([]interface{}, error) {
	code := c.GetString("code")
	log.Println("code from context: ", code)
	ctxCollection := c.GetString("collection")
	log.Println("collection from context: ", ctxCollection)
	isSuperAdmin := c.GetBool("isSuperAdmin")
	log.Println("isSuperAdmin from context: ", isSuperAdmin)

	filter := make(map[string]interface{})
	if isSuperAdmin {
		filter = bson.M{}
	} else if ctxCollection == util.TenantCollection {
		filter = bson.M{
			"tenantId": code,
		}
	} else if ctxCollection == util.HospitalCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New(util.INVALID_USER_TO_ACCESS)
	}
	collection := db.OpenCollections(util.PharmacistCollection)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdatePharmacist(c *gin.Context, data map[string]interface{}, pharmacistId string) (string, error) {
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := common.TrimIfExists(data, f); err != nil {
			log.Println("Error from trimIfExists: ", err)
			return "", err
		}
	}
	if err := common.HandleDOB(data); err != nil {
		return "", err
	}

	collection := db.OpenCollections(util.PharmacistCollection)
	err := common.CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}

	code := c.GetString("code")
	updateFilter := common.BuildUpdateFilter(data, code)
	filter := bson.M{
		"code": pharmacistId,
	}
	pharmacist := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &pharmacist)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return "", err
	}
	log.Println(pharmacist)
	val := pharmacist["createdBy"].(string)
	log.Println(val)
	if code != val {
		log.Println("This hospitalAdmin does not have access to update")
		return "", errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS)
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return "", err
	}

	log.Println(res.ModifiedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	key := util.PharamacistKey + pharmacistId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old pharmacist cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated pharmacist:", err)
	}

	return "Updated Successfully", nil
}

/*
* Build filter to search based on doctorId
* If found with the field createdBy from the result document found from document found
* Compare code from context and createdBy, if it works well go for the delete
* If not ,no another hospital admin can have access to delete it
 */
func DeletePharmacist(c *gin.Context, pharmacistId string) (string, error) {
	collection := db.OpenCollections(util.PharmacistCollection)
	hospitalCodeRaw, ok := c.Get("code")
	if !ok {
		log.Println("Unable to fetch code from the context")
		return "", errors.New(util.UNABLE_TO_FETCH_CODE_FROM_CONTEXT)
	}
	hospitalId, ok := hospitalCodeRaw.(string)
	if !ok {
		return "", errors.New("Unable to get hospitalCode from the context")
	}

	filter := bson.M{
		"code": pharmacistId,
	}
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	val := result["createdBy"].(string)
	if val != hospitalId {
		log.Println("This hospital admin doesnot have access")
		return "", errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS)
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.PharamacistKey + pharmacistId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache: ", err)
	}
	msg := fmt.Sprintf("The doctor %s deleted", pharmacistId)
	return msg, nil
}
