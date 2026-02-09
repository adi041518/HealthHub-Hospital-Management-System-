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
func CreateDoctor(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := common.ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}
	err = common.GetTrimmedString(data, "department")
	if err != nil {
		log.Println("Error from the getTrimmedString: ", err)
		return val, err
	}
	collection, err := common.FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from FetchRoleDocAndCollection:", err)
		return val, err
	}
	code, createdBy, err := common.CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return val, err
	}
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken: ", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)

	data["tenantId"] = tenantId
	otp, err := common.GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return val, err
	}
	log.Println(otp)
	if err = common.PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	key := util.DoctorKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	if _, err := common.SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	if err := common.CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return val, err
	}
	subject := "Your Hospital OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for Hospital verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = common.SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return "", errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return "created successfully", nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateDoctor(c *gin.Context, data map[string]interface{}, doctorId string) (string, error) {
	log.Println("doctorId: ", doctorId)
	fields := []string{"name", "email", "phoneNo"}
	for _, f := range fields {
		if err := common.TrimIfExists(data, f); err != nil {
			log.Println("Error from ")
			return "", err
		}
	}
	if err := common.HandleDOB(data); err != nil {
		return "", err
	}

	collection := db.OpenCollections(util.DoctorCollection)
	err := common.CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}

	code := c.GetString("code")
	updateFilter := common.BuildUpdateFilter(data, code)
	filter := bson.M{
		"code": doctorId,
	}
	doctor := make(map[string]interface{})

	err = db.FindOne(c, collection, filter, &doctor)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	log.Println("Doctor: ", doctor)

	hospitalId := doctor["createdBy"].(string)
	log.Println("hospitalId: ", hospitalId)
	log.Println("code: ", code)
	if code != hospitalId {
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
	key := util.DoctorKey + doctorId
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old receptionist from cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated receptionist:", err)
	}

	return "Updated Successfully", nil
}

/*
* isSuperAdmin,tenantId,collection and code from context
* Pass those fields and key fetch from cache
* If exists,check who can access(superAdmin,tenantAdmin,hospitalAdmin)
* If not found go to db search for the document
* Search the doument, check who can access doctor
* If comparision works then return the doctor
 */
func FetchDoctorByCode(c *gin.Context, doctorId string) (map[string]interface{}, error) {
	coll := util.DoctorCollection
	key := util.DoctorKey + doctorId
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
		"code": doctorId,
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
		log.Println("Error from setCache: ", err)
	}

	return result, nil
}

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfDoctors
* Return them
 */
func FetchAllDoctors(c *gin.Context) ([]interface{}, error) {
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
	collection := db.OpenCollections(util.DoctorCollection)
	result, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from the findAll function: ", err)
		return nil, err
	}
	return result, nil
}

/*
* Build filter to search based on doctorId
* If found with the field createdBy from the result document found from document found
* Compare code from context and createdBy, if it works well go for the delete
* If not no another hospital admin can have access to delete it
 */
func DeleteDoctor(c *gin.Context, doctorId string) (string, error) {
	collection := db.OpenCollections(util.DoctorCollection)
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
		"code": doctorId,
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
		return "", errors.New("This hospital admin doesnot have access")
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.DoctorKey + doctorId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache: ", err)
	}
	msg := fmt.Sprintf("The doctor %s deleted", doctorId)
	return msg, nil
}
