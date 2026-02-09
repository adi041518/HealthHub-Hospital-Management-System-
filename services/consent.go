package services

import (
	"errors"
	"log"
	"strconv"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

/*
* Bind the remaining data with the data
* Update the remaining field
 */
func PrepareConsentData(data map[string]interface{}, code string, consentCode string, patientId string, tenantId string, hospitalId string) map[string]interface{} {
	data["code"] = consentCode
	data["patientId"] = patientId
	data["tenantId"] = tenantId
	data["hospitalId"] = hospitalId
	data["createdBy"] = code
	data["updatedBy"] = code
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()
	return data
}
func VerifyConsentBelowAge(c *gin.Context, patientId string, data map[string]interface{}) error {
	collection := db.OpenCollections(util.ConsentVerificationCollection)
	filter := bson.M{
		"patientId": patientId,
	}
	consentVerification := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, consentVerification)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return err
	}
	otpFromConsentVerification, ok := consentVerification["otp"].(string)
	if !ok {
		log.Println("Unable to fetch otp from the particular document")
		return errors.New(util.UNABLE_TO_FETCH_OTP_FROM_DOCUMENT)
	}
	fields := []string{"password", "patientId"}
	for _, field := range fields {
		err := common.GetTrimmedString(data, field)
		if err != nil {
			log.Println("Error from getTrimmedString: ", err)
			return err
		}
	}
	if otpFromConsentVerification != data["password"].(string) {
		log.Println("Incorrect password")
		return errors.New(util.INCORRECT_PASSWORD)
	}

	data["isConsentVerified"] = true
	return nil
}

/*
* Fetch murse based on the code
* Fetch medicalRecord based on the providedId
* Fetch patient based on patientId present in medicalRecord
* Check for the age and and whose age is <18
* Check for the consentVerification and otp and validate with data(otp)
* Remaining fields should be mapped
* Generate code and create consent and set in cache
* UpdateMedicalRecord with the consentId
 */
func CreateConsent(c *gin.Context, data map[string]interface{}, medicalRecordId string) (string, error) {
	log.Println("data: ", data)
	code := c.GetString("code")
	tenantId := c.GetString("tenantId")
	nurse, err := FetchNurseByCode(c, code)
	if err != nil {
		log.Println("Error from fetchNurseByCode: ", err)
		return "", err
	}

	collection := db.OpenCollections(util.ConsentCollection)

	medicalRecord, err := FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		log.Println("Error from fetchMedicalRecordByCode: ", err)
		return "", err
	}

	patientId := medicalRecord["patientId"].(string)
	log.Println("patientId: ", patientId)
	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		return "", err
	}

	age, _ := strconv.Atoi(patient["age"].(string))
	log.Println("age: ", age)
	if age > 18 {
		data["isConsentVerified"] = true
	} else {
		err := VerifyConsentBelowAge(c, patientId, data)
		if err != nil {
			log.Println("Error from verifyConsentBelowAge: ", err)
			return "", err
		}

	}
	consentCode, err := common.GenerateEmpCode(util.ConsentCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	log.Println("nurse: ", nurse)
	PrepareConsentData(data, code, consentCode, patientId, tenantId, nurse["createdBy"].(string))
	inserted, err := db.CreateOne(c, collection, data)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.ConsentKey + consentCode
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	medicalRecordNew := make(map[string]interface{})
	medicalRecordNew["consentId"] = consentCode
	err = UpdateMedicalRecordByNurse(c, medicalRecordId, medicalRecordNew)
	if err != nil {
		log.Println("Error from updateMedicalRecordByNurse: ", err)
		return "", err
	}
	return "created successfully", nil
}

/*
* If superAdmin return nil
* If tenantId verify with the record tenantId and userTenantId
* If patientCollection, verify the code and patientId stored in the record
* If GuardianCollection, match code and guardianId from the document
* If any user wants to fetch ,check fo user createdBy and hospitalId in the consent record
 */
func CanAccessConsent(userData, record map[string]interface{}, tenantId string, code string, collFromContext string, isSuperAdmin bool) error {
	log.Println("record: ", record)

	if isSuperAdmin {
		return nil
	}

	if collFromContext == util.TenantCollection {
		if record["tenantId"].(string) != tenantId {
			return errors.New(util.TENANT_DOESNOT_HAVE_ACCESS)
		}
		return nil
	}

	if collFromContext == util.HospitalCollection {
		if record["hospitalId"].(string) != code {
			return errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS)
		}
		return nil
	}

	if collFromContext == util.PatientCollection {
		if record["patientId"].(string) != code {
			return errors.New(util.PATIENT_DOESNOT_HAVE_ACCESS)
		}
	}

	if collFromContext == util.GuardianCollection {
		if record["guardianId"].(string) != code {
			return errors.New(util.GUARDIAN_DOESNOT_HAVE_ACCESS)
		}
	}
	log.Println("userData: ", userData["createdBy"].(string))
	log.Println("hospitalId: ", record["hospitalId"].(string))
	if userData["createdBy"].(string) != record["hospitalId"].(string) {
		return errors.New(util.INVALID_USER_TO_ACCESS)
	}

	return nil
}

/*
* Fetch the consent from cache and check who can access
* If validations are checked and verified return cache document
* If not return nil
 */
func CheckConsentCacheAccess(c *gin.Context, key string, collFromContext string, userData map[string]interface{}, tenantId, code string, isSuperAdmin bool) (map[string]interface{}, bool, error) {

	cached := make(map[string]interface{})
	exists, err := redis.GetCache(c, key, &cached)
	if err != nil || !exists {
		return nil, false, nil
	}

	if err := CanAccessConsent(userData, cached, tenantId, code, collFromContext, isSuperAdmin); err != nil {
		return nil, true, err
	}

	return cached, true, nil
}

/*
* Fetch code,tenantId,collection,isSuperAdmin from the context
* Check who can access if exists and user have privileges return cache
* If not in cache search in dataBase
* If data was not in database return error or else return nil
 */
func FetchConsentByCode(c *gin.Context, consentId string) (map[string]interface{}, error) {

	key := util.ConsentKey + consentId
	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne to fetch user: ", err)
		return nil, err
	}

	if cached, exists, err := CheckConsentCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists && cached != nil {
		return cached, err
	}

	coll := db.OpenCollections(util.ConsentCollection)
	filter := bson.M{"code": consentId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne while fetching consent : ", err)
		return nil, err
	}

	if err := CanAccessConsent(userData, result, tenantId, code, collFromContext, isSuperAdmin); err != nil {
		return nil, err
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}

	return result, nil
}

/*
* Modify filter based on the user collection
* Search in database and then if error doesnot occurs return all the consents
* If errors occur return it
 */
func FetchAllConsents(c *gin.Context) ([]interface{}, error) {
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
	} else if ctxCollection == util.NurseCollection {
		filter = bson.M{
			"createdBy": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New(util.INVALID_USER_TO_ACCESS)
	}
	collection := db.OpenCollections(util.ConsentCollection)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* Search for the particular consent exists in db
* Check who can delete consent and delete in database and delete in cache
 */
func DeleteConsent(c *gin.Context, consentId string) (string, error) {
	nurseId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	coll := util.ConsentCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":      consentId,
		"createdBy": nurseId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return "", err
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from deleteOne: ", err)
		return "", err
	}
	log.Println("DeletedCount: ", deleted.DeletedCount)
	return "Deleted successfully", nil
}
