package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func VerifyHasAccess(c *gin.Context, doctorId string, medicalRecordId string) (map[string]interface{}, error) {

	medicalRecord, err := FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		log.Println("Error from fetchMedicalRecordByCode: ", err)
		return nil, err
	}
	doctorIdFromMedicalRecord, exists := medicalRecord["doctorId"].(string)
	if !exists {
		log.Println("doctorId doesnot exists in medicalRecord")
		return nil, errors.New(util.UNABLE_TO_FETCH_DOCTOR_ID_FROM_MEDICAL_RECORD)
	}
	if doctorId != doctorIdFromMedicalRecord {
		log.Println("User doesnot have access")
		return nil, errors.New(util.DOCTOR_DOESNOT_HAVE_ACCESS_TO_CREATE_PRESCRIPTION)
	}
	return medicalRecord, nil
}
func ValidateMedicines(rawMedicines []interface{}) error {

	for _, m := range rawMedicines {
		medicine, ok := m.(map[string]interface{})
		if !ok {
			return errors.New("invalid medicine format")
		}

		err := ValidateMedicineFields(medicine)
		if err != nil {
			log.Println("Error from validateMedicineFields: ", err)
			return err
		}
	}
	return nil
}
func ValidateMedicineFields(medicine map[string]interface{}) error {
	fields := []string{"medicineId", "instructions", "dosagePerFrequency", "noOfDays"}
	for _, field := range fields {
		err := common.GetTrimmedString(medicine, field)
		if err != nil {
			return err
		}
	}
	frequency, ok := medicine["frequency"].(map[string]interface{})
	if !ok {
		log.Println("Frequency must be an object")
		return errors.New(util.FREQUENCY_MUST_BE_AN_OBJECT)
	}

	boolFields := []string{"morning", "afternoon", "night"}
	for _, bf := range boolFields {
		val, ok := frequency[bf].(bool)
		if !ok {
			return errors.New("frequency field missing: " + bf)
		}
		frequency[bf] = val
	}
	return nil
}

/*
* Validate user inputs first
* Verify whether the doctor can create prescription for that medicalRecord
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from context
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* Update medicalRecord with the new prescription
* Update Appointment with isProcessing fields
* Save to db and cache
 */
func CreatePrescription(c *gin.Context, data map[string]interface{}, medicalRecordId string) (string, error) {

	doctorId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext(doctorId): ", err)
		return "", err
	}

	medicalRecord, err := VerifyHasAccess(c, doctorId, medicalRecordId)
	if err != nil {
		log.Println("Error from VerifyHasAccess: ", err)
		return "", err
	}
	rawMedicines, ok := data["medicines"].([]interface{})
	if !ok {
		log.Println("Medicines field must be list of interface")
		return "", errors.New(util.MEDICINES_MUST_BE_ARRAY)
	}
	err = common.GetTrimmedString(data, "diagnosis")
	if err != nil {
		log.Println("Error from getTrimmedString: ", err)
		return "", err
	}
	err = ValidateMedicines(rawMedicines)
	if err != nil {
		log.Println("Error from validateMedicines: ", err)
		return "", err
	}

	tenantId, err := common.GetFromContext[string](c, "tenantId")
	if err != nil {
		log.Println("Error from getFromContext(tenantId): ", err)
		return "", err
	}

	coll := util.PrescriptionCollection
	prescriptionCode, err := common.GenerateEmpCode(coll)
	if err != nil {
		return "", err
	}
	doctor, err := FetchDoctorByCode(c, doctorId)
	if err != nil {
		log.Println("Error from fetchDoctorByCode: ", err)
		return "", err
	}
	data["code"] = prescriptionCode
	data["hospitalId"] = doctor["createdBy"].(string)
	data["tenantId"] = tenantId
	data["createdBy"] = doctorId
	data["updatedBy"] = doctorId
	data["createdAt"] = time.Now()
	data["updatedAt"] = time.Now()

	collection := db.OpenCollections(coll)
	_, err = db.CreateOne(c, collection, data)
	if err != nil {
		return "", err
	}

	medRecDoc := make(map[string]interface{})
	medRecDoc["prescriptionId"] = prescriptionCode
	_, err = UpdateMedicalRecord(c, medicalRecordId, medRecDoc)
	if err != nil {
		log.Println("Error from updateMedicalRecord: ", err)
		return "", err
	}
	updAppointment := bson.M{
		"isProcessing": false,
	}

	_, err = UpdateAppointmentByCode(c, medicalRecord["appointmentId"].(string), updAppointment)
	if err != nil {
		log.Println("Update(isProcessing) field for the latestAppointment: ", err)
		return "", err
	}
	key := util.PrescriptionKey + prescriptionCode
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error while caching new prescription: ", err)
	}
	return "Created successfully", nil
}

/*
* Get prescription from the given prescriptionId
* Get tenantId,code,collection,isSuperAdmin from the context
* Check who can access
* Fetch from access, based on the accessibility
* if exists return
* If not exists fetch from database
* Return from database and set in cache
 */
func FetchPrescriptionByCode(c *gin.Context, prescriptionId string) (map[string]interface{}, error) {

	key := util.PrescriptionKey + prescriptionId

	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne the user: ", err)
		return nil, err
	}

	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(util.PrescriptionCollection)
	filter := bson.M{"code": prescriptionId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne the prescription: ", err)
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
* Make a filter
* According to the user,the filter condition changes
* Search for listOfPrescription
* Return them
 */
func FetchAllPresciptions(c *gin.Context) ([]interface{}, error) {
	coll := util.PrescriptionCollection
	collection := db.OpenCollections(coll)
	doctorId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return nil, err
	}
	filter := bson.M{
		"updatedBy": doctorId,
	}
	prescriptions, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from findAll: ", err)
		return nil, err
	}
	return prescriptions, nil
}

/*
* Validate data and trim and update
* Update some fields accordingly
 */
func ValidateUpdatePrescriptionData(data map[string]interface{}, doctorId string) (map[string]interface{}, error) {

	Fields := []string{"instructions", "dosagePerFrequency", "noOfDays"}
	for _, field := range Fields {
		err := common.TrimIfExists(data, field)
		if err != nil {
			log.Println("Error from trimIfExists: ", err)
			return nil, err
		}
	}
	if freq, exists := data["frequency"]; exists {
		f, okay := freq.(map[string]interface{})
		if !okay {
			log.Println("frequency field must be object")
			return nil, errors.New(util.FREQUENCY_MUST_BE_AN_OBJECT)
		}
		fields := []string{"morning", "afternoon", "night"}
		for _, field := range fields {
			if val, ok := f[field]; ok {
				boolean, ok := val.(bool)
				if !ok {
					log.Printf("Frequency %s must be true/false", field)
					return nil, fmt.Errorf("Frequency %s must be true/false", field)
				}
				f[field] = boolean
			}
		}
		data["frequency"] = f
	}
	data["updatedBy"] = doctorId
	data["updatedAt"] = time.Now()
	return data, nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Update this prescription
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdatePrescription(c *gin.Context, prescriptionId string, medicineId string, data map[string]interface{}) (string, error) {
	doctorId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	err = common.TrimIfExists(data, "diagnosis")
	if err != nil {
		log.Println("Error from trimIfExists: ", err)
		return "", err
	}
	data, err = ValidateUpdatePrescriptionData(data, doctorId)
	if err != nil {
		log.Println("Error from validateUpdatePrescriptionData: ", err)
		return "", err
	}
	coll := util.PrescriptionCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":                 prescriptionId,
		"medicines.medicineId": medicineId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne(while fetching prescription): ", err)
		return "", err
	}
	docFromPrescriptionVal, ok := result["createdBy"].(string)
	if !ok {
		log.Println("createdBy(doctor) field doesnot exists in prescription")
		return "", errors.New(util.UNABLE_TO_FETCH_CREATED_BY_FROM_PRESCRIPTION)
	}
	if doctorId != docFromPrescriptionVal {
		log.Println("This doctor doesnot have access")
		return "", errors.New(util.DOCTOR_DOESNOT_HAVE_ACCESS_TO_UPDATE)
	}
	updateFields := bson.M{}
	for key, value := range data {
		updateFields["medicines.$."+key] = value
	}
	update := bson.M{
		"$set": updateFields,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne: ", err)
		return "", err
	}
	log.Println("Updated prescription count: ", updated.ModifiedCount)
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne: ", err)
		return "", err
	}
	key := util.PrescriptionKey + prescriptionId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache: ", err)
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache: ", err)
	}
	return "updated successfully", nil
}

/*
* Build filter to search based on prescriptionId
* If found with the field createdBy from the result document found
* Compare code from context and createdBy, if it works well go for the delete
* If not no another doctor can have access to delete it
 */
func DeletePrescriptionByCode(c *gin.Context, prescripitonId string) (string, error) {
	collection := db.OpenCollections(util.PrescriptionCollection)
	doctorId, ok := c.Get("code")
	if !ok {
		return "", errors.New(util.UNABLE_TO_FETCH_CODE_FROM_CONTEXT)
	}
	filter := bson.M{
		"code": prescripitonId,
	}

	log.Println(filter)
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the findOne function:", err)
		return "", err

	}
	if doctorId.(string) != result["createdBy"].(string) {
		log.Println("This user doesnot have access")
		return "", errors.New(util.INVALID_USER_TO_ACCESS)
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.PrescriptionKey + prescripitonId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", prescripitonId)
	return msg, nil
}
