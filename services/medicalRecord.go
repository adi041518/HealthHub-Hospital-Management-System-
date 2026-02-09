package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const FAILED_TO_DELETE_OLD_MEDICAL_RECORD string = "Failed deleting old medicalRecord cache:"
const FAILED_CACHCING_UPDATED_MEDICAL_RECORD string = "Failed caching updated medicalRecord:"

/*
* Get medicalRecord from the given medicalRecordId
* Get tenantId,code,collection,isSuperAdmin from the context
* Check who can access
* Fetch from access, based on the accessibility
* if exists return
* If not exists fetch from database
* Return from database and set in cache
 */
func FetchMedicalRecordByCode(c *gin.Context, medicalRecordId string) (map[string]interface{}, error) {
	key := util.MedicalRecordKey + medicalRecordId

	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	log.Println("collFromContext: ", collFromContext)
	log.Println("filter: ", bson.M{"code": code})
	err := db.FindOne(c.Request.Context(), collectionFromContext, bson.M{"code": code}, &userData)
	if err != nil {
		log.Println("Error from findOne while fetching user: ", err)
		return nil, err
	}

	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(util.MedicalRecordCollection)
	filter := bson.M{"code": medicalRecordId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne while fetching medicalRecord: ", err)
		return nil, err
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

/*
* Get the code from claims which is updatedBy field
* Update based on the search filters and update fields
* Update this by nurse whose id should match with the existing medicalRecord nurseId field
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateMedicalRecordByNurse(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {

	code := c.GetString("code")
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(util.MedicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err := db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne) by nurse", err)
		return err
	}
	nurseIdVal, ok := medicalRecord["nurseId"]
	if !ok {
		log.Println("Error while checking the nurseId is present in it or not")
		return errors.New(util.CHECK_NURSE_ID_EXIST_IN_DOCUMENT)
	}
	nurseId, ok := nurseIdVal.(string)
	if !ok {
		log.Println("Error during type assertion error")
		return errors.New(util.INVALID_NURSE_ID)
	}
	if nurseId != code {
		log.Println("This nurse doesnot have access to updatethe record")
		return errors.New(util.NURSE_DOESNOT_HAVE_ACCESS_TO_UPDATE)
	}
	collection := db.OpenCollections(util.MedicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by nurse:", err)
		return err
	}
	log.Println("UpdatedCount(byNurse): ", updated.ModifiedCount)
	updatedRecord := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, &updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating medicalRecord byNurse", err)
		return err
	}
	key := util.MedicalRecordKey + medicalRecordId
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println(FAILED_TO_DELETE_OLD_MEDICAL_RECORD, err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println(FAILED_CACHCING_UPDATED_MEDICAL_RECORD, err)
	}
	return nil
}

func FetchGuardians(patient map[string]interface{}) ([]string, error) {

	var guardians []string
	switch v := patient["listOfGuardians"].(type) {
	case []interface{}:
		for _, g := range v {
			id, ok := g.(string)
			if !ok {
				return nil, errors.New("guardian id is not string")
			}
			guardians = append(guardians, id)
		}

	case primitive.A:
		for _, g := range v {
			id, ok := g.(string)
			if !ok {
				return nil, errors.New("guardian id is not string")
			}
			guardians = append(guardians, id)
		}

	default:
		return nil, errors.New("invalid listOfGuardians type")
	}
	return guardians, nil
}

func CreateConsentVerification(c *gin.Context, guardians []string, patientId string) error {
	for _, guardianId := range guardians {
		guardian, err := FetchGuardianByCode(c, guardianId)
		if err != nil {
			log.Println("Error from fetchGuardianByCode: ", err)
			return err
		}
		otp := common.GenerateOTP()
		subject := "Guardian for consent OTP Verification"
		body := fmt.Sprintf("Hello %s,\n\nYour OTP for consent verification is: %s\n\nThank you!", guardian["name"].(string), otp)
		log.Printf("Mail sent to the guardian %s and otp is %s", guardian["code"].(string), otp)
		common.SendOTPToMail(guardian["email"].(string), subject, body)
		collection := db.OpenCollections(util.ConsentVerificationCollection)
		consent := make(map[string]interface{})
		consent["otp"] = otp
		consent["patientId"] = patientId
		consent["guardianId"] = guardianId
		consent["collection"] = util.GuardianCollection
		inserted, err := db.CreateOne(c, collection, consent)
		if err != nil {
			log.Println("Error from createOne: ", err)
			return err
		}
		log.Println("Inserted document in consentVerificaton: ", inserted.InsertedID)
	}
	return nil
}
func VerifyDoctorCanAccess(medicalRecord map[string]interface{}, code string) error {
	doctorIdVal, ok := medicalRecord["doctorId"]
	if !ok {
		log.Println("Error while checking the doctorId is present in it or not")
		return errors.New(util.CHECK_DOCTOR_ID_EXIST_IN_DOCUMENT)
	}
	doctorId, ok := doctorIdVal.(string)
	if !ok {
		log.Println("Error during type assertion error")
		return errors.New(util.INVALID_DOTOR_ID)
	}
	if doctorId != code {
		log.Println("This doctor doesnot have access to update the record")
		return errors.New(util.DOCTOR_DOESNOT_HAVE_ACCESS_TO_UPDATE)
	}
	return nil
}

/*
* Get the code from claims which is updatedBy field
* Update based on the search filters and update fields
* Update by doctor, whose id should match with the existing medicalRecord doctorId field
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateMedicalRecordByDoctor(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {
	code := c.GetString("code")
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(util.MedicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err := db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne) by doctor", err)
		return err
	}
	err = VerifyDoctorCanAccess(medicalRecord, code)
	if err != nil {
		log.Println("Error from verifyDoctorCanAccess: ", err)
		return nil
	}
	patientId, ok := medicalRecord["patientId"].(string)
	if !ok {
		log.Println("Error while fetching patientId from the medicalRecord")
		return errors.New("Error while fetching patientId from the medicalRecord")
	}
	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		log.Println("Error from fetchPatientByCode:  ", err)
		return err
	}
	ageStr := patient["age"].(string)
	age, err := strconv.Atoi(ageStr)

	if age < 18 {
		guardians, err := FetchGuardians(patient)
		if err != nil {
			log.Println("Error from fetchGuardians: ", err)
			return err
		}
		if len(guardians) == 0 {
			return errors.New("no guardians found for minor patient")
		}
		err = CreateConsentVerification(c, guardians, patientId)
		if err != nil {
			log.Println("Error from createConsentVerification: ", err)
			return err
		}
	}
	collection := db.OpenCollections(util.MedicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by doctor:", err)
		return err
	}
	log.Println("UpdatedCount(medicalRecord): ", updated.ModifiedCount)
	updatedRecord := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating medicalRecord byDoctor", err)
		return err
	}
	key := util.MedicalRecordKey + medicalRecordId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println(FAILED_TO_DELETE_OLD_MEDICAL_RECORD, err)
	}

	if err := redis.SetCache(c, key, updatedRecord); err != nil {
		log.Println(FAILED_CACHCING_UPDATED_MEDICAL_RECORD, err)
	}
	return nil
}

/*
* Get the code from claims which is updatedBy field
* Update based on the search filters and update fields
* Update this by pharmacist whose id should match with the existing medicalRecord pharmacistId field
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateMedicalRecordByPharmacist(c *gin.Context, medicalRecordId string, data map[string]interface{}) error {
	code := c.GetString("code")
	pharmacist, err := FetchPharmacistByCode(c, code)
	if err != nil {
		log.Println("Error from fetchPharmacistByCode: ", err)
		return err
	}
	doctorIdFromPharmacist := pharmacist["createdBy"].(string)
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
	medicalRecordColl := db.OpenCollections(util.MedicalRecordCollection)
	mFilter := bson.M{
		"code": medicalRecordId,
	}
	medicalRecord := make(map[string]interface{})
	err = db.FindOne(c, medicalRecordColl, mFilter, &medicalRecord)
	if err != nil {
		log.Println("Error while fetching medicalRecord(FindOne) by pharmacist", err)
		return err
	}
	hospitalId, ok := medicalRecord["hospitalId"].(string)
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return errors.New(util.UNABLE_TO_FETCH_HOSPITAL_ID_FROM_MEDICAL_RECORD)
	}
	if hospitalId != doctorIdFromPharmacist {
		log.Println("This pharmacist doesnot have access to update the record")
		return errors.New(util.PHARMACIST_DOES_NOT_HAVE_ACCESS_TO_UPDATE_MEDICAL_RECORD)
	}
	collection := db.OpenCollections(util.MedicalRecordCollection)
	filter := bson.M{
		"code": medicalRecordId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord by doctor:", err)
		return err
	}
	log.Println("UpdatedCount(byPharmacist): ", updated.ModifiedCount)
	updatedRecord := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedRecord)
	if err != nil {
		log.Println("Error from findOne after updating medicalRecord by pharmacist", err)
		return err
	}
	key := util.MedicalRecordKey + medicalRecordId
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println(FAILED_TO_DELETE_OLD_MEDICAL_RECORD, err)
	}

	if err := redis.SetCache(c, key, updatedRecord); err != nil {
		log.Println(FAILED_CACHCING_UPDATED_MEDICAL_RECORD, err)
	}
	return nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Update this either by nurse,doctor,nor pharmacist nurse whose id should match with the existing medicalRecord
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateMedicalRecord(c *gin.Context, medicalRecordId string, data map[string]interface{}) (string, error) {
	collection := c.GetString("collection")
	msg := "Updated successfully"
	switch collection {
	case util.NurseCollection:
		if err := UpdateMedicalRecordByNurse(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by nurse")
		return msg, nil

	case util.DoctorCollection:
		if err := UpdateMedicalRecordByDoctor(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by doctor")
		return msg, nil
	case util.PharmacistCollection:
		if err := UpdateMedicalRecordByPharmacist(c, medicalRecordId, data); err != nil {
			return "", err
		}
		log.Println("Updated by pharmacist")
		return msg, nil
	// case receptionistCollection:
	// 	if err := UpdateMedicalRecordByReceptionist(c, medicalRecordId, data); err != nil {
	// 		return "", err
	// 	}
	// 	log.Println("Updated by receptionist")
	// 	return msg, nil
	default:
		return "", errors.New("Unauthorized role")
	}
}

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfMedicalRecord
* Return them
 */
func FetchAllMedicalRecords(c *gin.Context) ([]interface{}, error) {
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
		collection := db.OpenCollections(util.ReceptionistCollection)
		receptionist := make(map[string]interface{})
		err := db.FindOne(c, collection, bson.M{"code": code}, receptionist)
		if err != nil {
			log.Println("Error from findOne: ", err)
			return nil, err
		}
		filter = bson.M{
			"hospitalId": receptionist["createdBy"].(string),
		}
	} else if ctxCollection == util.DoctorCollection {
		filter = bson.M{
			"doctorId": code,
		}
	} else if ctxCollection == util.NurseCollection {
		filter = bson.M{
			"nurseId": code,
		}
	} else {
		log.Println("This user doesnot have access")
		return nil, errors.New(util.INVALID_USER_TO_ACCESS)
	}
	collection := db.OpenCollections(util.MedicalRecordCollection)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* Build filter to search based on medicalRecordId
* If found ,fetch field createdBy from the result document found
* Compare code from context and createdBy, if it works well go for the delete
* If not, no another receptionist can have access to delete it
 */
func DeleteMedicalRecordByCode(c *gin.Context, medicalRecordId string) (string, error) {
	collection := db.OpenCollections(util.MedicalRecordCollection)
	receptionistId, ok := c.Get("code")
	if !ok {
		return "", errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"code": medicalRecordId,
	}

	log.Println(filter)
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the findOne function:", err)
		return "", err

	}
	if receptionistId.(string) != result["createdBy"].(string) {
		log.Println("This user doesnot have access")
		return "", errors.New(util.RECEPTIONIST_DOESNOT_HAVE_ACCESS)
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.MedicalRecordKey + medicalRecordId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", medicalRecordId)
	return msg, nil
}
