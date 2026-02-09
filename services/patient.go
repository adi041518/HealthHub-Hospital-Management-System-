package services

import (
	"strconv"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"

	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func TrimRemainingData(data map[string]interface{}) error {

	err := common.TrimIfExists(data, "gender")
	if err != nil {
		log.Println("Error from trimIfExists", err)
		return err
	}
	err = common.TrimIfExists(data, "admissionDate")
	if err != nil {
		log.Println("Error from trimIfExists")
		return err
	}
	return nil
}

func CreatePatientAndNotify(c *gin.Context, collection string, data map[string]interface{}, code string, otp string) error {
	if _, err := common.SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return err
	}
	key := util.PatientKey + code
	err := redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Failed caching new patient: ", err)
	}
	if err := common.CreateLoginRecord(c, collection, code, data["email"].(string), data["phoneNo"].(string), data["password"].(string)); err != nil {
		log.Println("Error from the createLoginRecord", err)
		return err
	}
	subject := "Your Patient OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for patient verification is: %s\n\nThank you!", data["name"].(string), otp)

	err = common.SendOTPToMail(data["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Validate user inputs first
* Fetch collection name from the roleCode given
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from context
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* If age of patient is less than 18 ,then guardians should be there
* Validate guardian fields and add remaining data and prepare it
* If gurdians exists insert guardianId in patient document
* Save to db and cache
* Send mail
 */

func CreatePatient(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := common.ValidateUserInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}

	collection, err := common.FetchCollectionFromRoleDoc(c, data["roleCode"].(string))
	if err != nil {
		log.Println("Error from fetchRoleDocAndCollection:", err)
		return val, err
	}
	receptionistId := c.GetString("code")
	receptionist, err := FetchReceptionistByCode(c, receptionistId)
	if err != nil {
		log.Println("Error from fetchReceptionistByCode: ", err)
		return val, err
	}
	code, createdBy, err := common.CheckerAndGenerateUserCodes(c, collection, data["email"].(string), data["phoneNo"].(string))
	if err != nil {
		log.Println("Error from GenerateUserRole", err)
		return val, err
	}
	log.Println(code)

	otp, err := common.GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return val, err
	}
	log.Println(otp)
	err = TrimRemainingData(data)
	if err != nil {
		log.Println("Error from trimRemaining data: ", err)
		return val, err
	}
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIdFromToken", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)
	if err = common.PrepareUser(data, code, createdBy, tenantId); err != nil {
		log.Println("Error from prepareUser :", err)
		return val, err
	}
	age, err := common.CalculateAge(data["dob"].(string))
	if err != nil {
		log.Println("Error from CalculateAge")
		return val, err
	}
	data["age"] = strconv.Itoa(age)

	receptionist, err = FetchReceptionistByCode(c, createdBy)
	if err != nil {
		log.Println("Error from fetchReceptionistByCode: ", err)
		return val, err
	}
	log.Println("Receptionist: ", receptionist)
	log.Println("Receptionist(createdBy): ", receptionist["createdBy"].(string))
	data["hospitalId"] = receptionist["createdBy"].(string)

	listOfGuardians := []string{}
	if age < 18 {
		if listOfGuardians, err = ValidateGuardianAndCreate(c, data, listOfGuardians, createdBy, tenantId); err != nil {
			log.Println("Error from validateConsentAndCreate: ", err)
			return val, err
		}
		delete(data, "guardians")
	}
	log.Println("ListOfGuardians: ", listOfGuardians)
	data["listOfGuardians"] = listOfGuardians
	err = CreatePatientAndNotify(c, collection, data, code, otp)
	if err != nil {
		log.Println("Error from createPatientAndNotify: ", err)
		return val, err
	}

	return "created successfully", nil
}
func FetchGuardiansFromData(data map[string]interface{}) ([]interface{}, error) {
	raw, ok := data["guardians"]
	if !ok {
		log.Println("Patient is minor please provide guardians details")
		return nil, errors.New("Patient is minor please provide guardians details")
	}
	guardians, ok := raw.([]interface{})
	if !ok || len(guardians) == 0 {
		log.Println("Type assertion error for guardians field")
		return nil, errors.New("Type assertion error for guardians field")
	}
	return guardians, nil
}

func ValidateGuardianFields(guardian map[string]interface{}) (map[string]interface{}, error) {
	requiredFields := []string{"name", "dob", "phoneNo", "email", "govtId", "relation", "roleCode"}
	for _, field := range requiredFields {
		err := common.GetTrimmedString(guardian, field)
		if err != nil {
			log.Println("Error from getTrimmedString: ", err)
			return nil, err
		}

	}
	return guardian, nil
}
func VerifyGuardianAge(guardian map[string]interface{}) error {

	age, err := common.CalculateAge(guardian["dob"].(string))
	if err != nil {
		log.Println("Error from calculateAge: ", err)
		return err
	}
	if age < 18 {
		log.Println("Guardian cannot be minor")
		return errors.New(util.GUARDIAN_CANNOT_BE_MINOR)
	}

	guardian["age"] = age
	return nil
}
func persistGuardianAndNotify(c *gin.Context, guardian map[string]interface{}, otp string, guardianId string) error {

	collection := db.OpenCollections(util.GuardianCollection)
	_, err := db.CreateOne(c, collection, guardian)
	if err != nil {
		log.Println("Error while inserting into db: ", err)
		return err
	}
	key := util.GuardianKey + guardianId
	err = redis.SetCache(c, key, guardian)
	if err != nil {
		log.Println("Failed caching new  guardian: ", err)
	}
	log.Println("code: ", guardian["code"].(string))

	err = common.CreateLoginRecord(c, util.GuardianCollection, guardian["code"].(string), guardian["email"].(string), guardian["phoneNo"].(string), guardian["password"].(string))
	if err != nil {
		log.Println("Error from guardian createLoginRecord: ", err)
		return err
	}
	subject := "Guardian OTP Verification"
	body := fmt.Sprintf("Hello %s,\n\nYour OTP for guardian verification is: %s\n\nThank you!", guardian["name"].(string), otp)

	err = common.SendOTPToMail(guardian["email"].(string), subject, body)
	if err != nil {
		log.Println("OTP mail failed:", err)
		return errors.New(util.FAILED_TO_SEND_OTP)
	}
	log.Println("mail sent successfully")
	return nil
}

/*
* Get guardians from input data
* Validate gurdian fields, check for uniqueness and generate codes
* Generate otp and bcrypt it
* Store in loginCollection and also in database
* Store in Cache as well
* Send a mail to the particular guardian
 */
func ValidateGuardianAndCreate(c *gin.Context, data map[string]interface{}, listOfGuardians []string, createdBy, tenantId string) ([]string, error) {

	guardians, err := FetchGuardiansFromData(data)
	if err != nil {
		log.Println("Error from fetchGuardiansFromData: ", err)
		return nil, err
	}
	for _, g := range guardians {
		guardian, ok := g.(map[string]interface{})
		if !ok {
			log.Println("Unable to get the guardian")
			return nil, errors.New("Unable to fetch the guardian")
		}
		guardian, err = ValidateGuardianFields(guardian)
		if err != nil {
			log.Println("Error from validateGuardianField: ", err)
			return nil, err
		}
		log.Println("Updated guardian: ", guardian)
		guardianId, _, err := common.CheckerAndGenerateUserCodes(c, util.GuardianCollection, guardian["email"].(string), guardian["phoneNo"].(string))
		if err != nil {
			log.Println("Error from checkerAndGenerateUserCodes: ", err)
			return nil, err
		}
		listOfGuardians = append(listOfGuardians, guardianId)
		otp, err := common.GenerateAndHashOTP(guardian)
		if err != nil {
			log.Println("Error from generateAndHashOTP: ", err)
			return nil, err
		}
		log.Printf("guardian %s guardian OTP %s: ", guardianId, otp)
		guardian["guardianId"] = guardianId
		guardian["hospitalId"] = data["hospitalId"].(string)
		err = common.PrepareUser(guardian, guardianId, createdBy, tenantId)
		if err != nil {
			log.Println("Error from prepareUser: ", err)
			return nil, err
		}
		err = VerifyGuardianAge(guardian)
		if err := persistGuardianAndNotify(c, guardian, otp, guardianId); err != nil {
			return nil, err
		}

	}
	return listOfGuardians, nil
}

/*
* Get isSuperAdmin,tenantId,collection and code values from context
* Pass those fields and key fetch from cache
* If exists,check who can access(superAdmin,tenantAdmin,hospitalAdmin)
* If not found go to db search for the document
* Search the doument, check who can access patient
* If comparision works then return the patient
 */
func FetchPatientByCode(c *gin.Context, patientId string) (map[string]interface{}, error) {

	key := util.PatientKey + patientId

	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return nil, err
	}

	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(util.PatientCollection)
	filter := bson.M{"code": patientId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne while fetching user: ", err)
		return nil, errors.New("record not found")
	}

	if err := common.CanAccess(userData, result, tenantId, code, collFromContext, isSuperAdmin); err != nil {
		return nil, err
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}

	return result, nil

}

func ValidateAdmission(data map[string]interface{}) error {
	if admissionDateVal, ok := data["admissionDate"]; ok {
		if dateStr, ok := admissionDateVal.(string); ok {
			updatedAdmissionDate, err := common.NormalizeDate(dateStr)
			if err != nil {
				log.Println("Error from NormalizeDate:", err)
				return err
			}
			data["admissionDate"] = updatedAdmissionDate
		}
	}
	return nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdatePatientByCode(c *gin.Context, patientId string, data map[string]interface{}) (string, error) {
	val := ""
	receptionistId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return val, err
	}
	fields := []string{"name", "email", "phoneNo", "dob", "admissionDate", "gender"}
	for _, field := range fields {
		err := common.TrimIfExists(data, field)
		if err != nil {
			log.Println("Error from getTrimmedString: ", err)
			return val, err
		}
	}
	err = common.HandleDOB(data)
	if err != nil {
		log.Println("Error from handleDOB", err)
		return val, err
	}
	err = ValidateAdmission(data)
	if err != nil {
		log.Println("Error from validateAdmission: ", err)
		return val, err
	}
	coll := util.PatientCollection
	collection := db.OpenCollections(coll)
	err = common.CheckForEmailAndPhoneNo(c, collection, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}

	filter := bson.M{
		"code": patientId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne while fetching patient: ", err)
		return val, err
	}
	createdByVal, ok := result["createdBy"]
	if !ok {
		log.Println("Error whil fetching createdBy from patient")
		return val, errors.New(util.MISSING_CREATED_BY_IN_DOCUMENT)
	}
	if receptionistId != createdByVal.(string) {
		log.Println("This receptionist doesnot have access")
		return val, errors.New(util.RECEPTIONIST_DOESNOT_HAVE_ACCESS)
	}
	data["updatedBy"] = receptionistId
	data["updatedAt"] = time.Now()
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return val, err
	}
	log.Println("Updated patient: ", updated.ModifiedCount)
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne while fetching patient after update: ", err)
		return val, err
	}
	key := util.PatientKey + patientId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old patient cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated patient:", err)
	}
	return "Updated Successfully", nil
}

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfPatients
* Return them
 */
func FetchAllPatients(c *gin.Context) ([]interface{}, error) {
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
			"hospitalId": code,
		}
	} else if ctxCollection == util.ReceptionistCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New(util.INVALID_USER_TO_ACCESS)
	}
	coll := util.PatientCollection
	collection := db.OpenCollections(coll)
	patients, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findall:", err)
		return nil, err
	}
	log.Println("Patients: ", patients)
	return patients, nil
}

/*
* Build filter to search based on patientId
* If found with the field createdBy from the result document found from document found
* Compare code from context and createdBy, if it works well go for the delete
* If not no another receptionist can have access to delete it
 */
func DeletePatient(c *gin.Context, patientId string) (string, error) {
	receptionistId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	filter := bson.M{
		"code": patientId,
	}
	coll := util.PatientCollection
	collection := db.OpenCollections(coll)
	key := util.PatientKey + patientId
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function", &err)
		return "", err
	}
	if result["createdBy"].(string) != receptionistId {
		log.Println("User doesnot have access")
		return "", errors.New(util.INVALID_USER_TO_ACCESS)
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("Deleted:", deleted.DeletedCount)
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deletedCache: ", err)
	}
	return "Deleted successfully", nil
}
