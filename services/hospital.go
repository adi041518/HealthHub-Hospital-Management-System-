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
* Validate inputs
* Fetch collection from roleDoc
* Check if user with same email or phoneNo exists
* GenerateOtp and bcrypt it
* Fill the extra fields to insert into the input fields
* Set in cache as well as in db
* Create login record in login collection
* Send otp to the provided mail
 */
func CreateHospital(c *gin.Context, data map[string]interface{}) error {
	if err := common.ValidateUserInput(data); err != nil {
		log.Println("Error from ValidateUserInput", err)
		return err
	}
	collection, err := common.FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return err
	}
	code, createdBy, err := common.CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserCodes:", err)
		return err
	}
	log.Println(code)
	otp, err := common.GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return err
	}
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken: ", err)
		return err
	}
	log.Println("tenantId from context: ", tenantId)

	if err = common.PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return err
	}
	key := util.HospitalKey + code
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
	subject := "Your Hospital OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for hospital verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = common.SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the update and search filters
 */
func UpdateHospital(c *gin.Context, data map[string]interface{}, hospitalId string) error {
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := common.TrimIfExists(data, f); err != nil {
			log.Println("Error from trimIfExists: ", err)
			return err
		}
	}
	if err := common.HandleDOB(data); err != nil {
		log.Println("Error from handlDOB: ", err)
		return err
	}
	collection := db.OpenCollections(util.HospitalCollection)
	err := common.CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return err
	}
	tenantId := c.GetString("code")
	updateFilter := common.BuildUpdateFilter(data, tenantId)
	filter := bson.M{
		"code": hospitalId,
	}
	value := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &value)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return err
	}
	log.Println(value)
	val := value["createdBy"].(string)
	if val != tenantId {
		log.Println("This tenant doesnot have access")
		return errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}

	log.Println("modifiedCount: ", res.ModifiedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return err
	}
	key := util.HospitalKey + hospitalId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	// Set new cache entry
	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return nil
}

/*
* Check for whether the hospitalAmdin exists or not
* SuperAdmin,tenant only have access to fetch hospitalAdmin
* Check in cache,if exists return hospitalAdmin
* If not exists,search in database and set in cache
 */
func FetchHospitalByCode(c *gin.Context, hospitalId string) (map[string]interface{}, error) {
	coll := util.HospitalCollection

	key := util.HospitalKey + hospitalId
	log.Println("Cache key: ", key)
	isSuperAdmin, err := common.GetFromContext[bool](c, "isSuperAdmin")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return nil, err
	}
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken ", err)
		return nil, err
	}
	log.Println("tenantId from token: ", tenantId)

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	tenantIdCache, ok := cached["tenantId"].(string)
	if !ok {
		fmt.Println("createdBy not found or invalid")
		return nil, errors.New(util.UNABLE_TO_FETCH_TENANT_ID)
	}
	if err == nil && exists {
		if !isSuperAdmin {
			if tenantIdCache != tenantId {
				log.Println("Error from the tenant which is tenant doesnot have access")
				return nil, errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
			}
		}
		log.Println("From cache")
		return cached, nil
	}
	result := make(map[string]interface{})
	filter := bson.M{
		"code": hospitalId,
	}
	collection := db.OpenCollections(coll)
	log.Println("Filter: ", filter)
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the FindOne function,err")
		return nil, err
	}
	if !isSuperAdmin {
		tenantIdFromColl := result["tenantId"].(string)
		if tenantIdFromColl != tenantId {
			log.Println("This tenant does not have access to fetch")
			return nil, errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
		}
	}

	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from the setCache:", err)
	}

	return result, nil
}

/*
* Fetch all hospitals from database
* Fetch hospitals can be only viewed by either superAdmin nor Tenant
* Return all hospitals for the filter
 */
func FetchAllHospital(c *gin.Context) ([]interface{}, error) {
	collection := db.OpenCollections(util.HospitalCollection)
	code := c.GetString("code")
	log.Println("code from context: ", code)
	ctxCollection := c.GetString("collection")
	log.Println("collection from context: ", ctxCollection)
	isSuperAdmin := c.GetBool("isSuperAdmin")
	log.Println("isSuperAdmin from context: ", isSuperAdmin)
	filter := make(map[string]interface{})
	if isSuperAdmin {
		filter = bson.M{}
	} else if !isSuperAdmin && ctxCollection == util.TenantCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("Invalid user to access ")
		return nil, errors.New(util.INVALID_USER_TO_ACCESS)
	}
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	log.Println("hospitals: ", doc)
	return doc, nil
}

/*
* Fetch hospital based on provided id
* If exists ,only tenna can delete the hospitalAdmin
* Delete from database as well as in cache also
 */
func DeleteHospitalByCode(c *gin.Context, hospitalId string) (string, error) {
	collection := db.OpenCollections(util.HospitalCollection)
	tenantId, ok := c.Get("code")
	if !ok {
		return "", errors.New(util.UNABLE_TO_FETCH_CODE_FROM_CONTEXT)
	}
	filter := bson.M{
		"code": hospitalId,
	}
	log.Println(filter)
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the findOne function:", err)
		return "", err
	}
	if tenantId.(string) != result["createdBy"].(string) {
		log.Println("User doesnot have access")
		return "", errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
	}
	_, err = db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	key := util.HospitalKey + hospitalId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", hospitalId)
	return msg, nil
}
