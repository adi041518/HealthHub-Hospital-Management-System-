package services

import (
	"context"
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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
* Get code from the context
* Return code
 */
func getReceptionistID(c *gin.Context) (interface{}, error) {
	code, ok := c.Get("code")
	if !ok {
		log.Println("Unable to get receptionist code from context")
		return nil, errors.New(util.UNABLE_TO_FETCH_CODE_FROM_CONTEXT)
	}
	return code, nil
}

/*
* Validate the fields that came from request
 */
func validateAppointmentInput(data map[string]interface{}) error {
	fields := []string{"patientId", "reason", "symptoms", "date", "time"}
	for _, f := range fields {
		if err := common.GetTrimmedString(data, f); err != nil {
			log.Println("Error from getTrimmedString:", err)
			return err
		}
	}
	return nil
}

/*
* Search for receptionist and doctor
* Compare both createdBy and give access for the receptionist to view doctor
*
 */
func CheckForPrivileges(c *gin.Context, receptionistId, doctorId string) (map[string]interface{}, error) {

	receptionist, err := FetchReceptionistByCode(c, receptionistId)
	if err != nil {
		log.Println("Error from FetchReceptionistByCode:", err)
		return nil, err
	}
	recepHosCode, ok := receptionist["createdBy"].(string)
	if !ok {
		log.Println("Error getting hospitalCode from Receptionist ")
		return nil, errors.New(util.UNABLE_TO_FETCH_CREATED_BY_FROM_RECEPTIONIST)
	}

	doctor, err := FetchDoctorByCode(c, doctorId)
	if err != nil {
		log.Println("Error from FetchDoctorByCode: ", err)
		return nil, err
	}
	docHosCode, ok := doctor["createdBy"].(string)
	if !ok {
		log.Println("Error getting hospitalCode from doctor ")
		return nil, errors.New(util.UNABLE_TO_GET_HOSPITAL_ID_FROM_DOCTOR)
	}

	if recepHosCode != docHosCode {
		log.Println("This receptionist doesnot have access to view this doctor")
		return nil, errors.New(util.RECEPTIONIST_DOESNOT_HAVE_ACCESS_TO_VIEW_DOCTOR)
	}
	return doctor, nil
}

/*
* Move to doctorAvailabilitySlots
* Get the filter and search for document
* Validate isWeeklyOff and isLeave fields
* Return the found document
 */
func fetchDoctorSlot(c context.Context, coll *mongo.Collection, filter bson.M) (map[string]interface{}, error) {
	doc := make(map[string]interface{})
	err := db.FindOne(c, coll, filter, doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New(util.NO_TIME_SLOT_AVAILABLE_FOR_THIS_DATE)
		}
		return nil, err
	}
	if off, _ := doc["isWeeklyOff"].(bool); off {
		return nil, errors.New(util.DOCTOR_WEEKLY_OFF)
	}
	if leave, _ := doc["isLeave"].(bool); leave {
		return nil, errors.New(util.DOCTOR_IS_ON_LEAVE)
	}
	return doc, nil
}

func ExtractSlots(doc map[string]interface{}) ([]map[string]interface{}, error) {
	slotsList := []map[string]interface{}{}
	switch raw := doc["slots"].(type) {
	case primitive.A: // slot is primitive array
		for _, v := range raw {
			slotsList = append(slotsList, v.(map[string]interface{}))
		}

	case []interface{}: // normal array
		for _, v := range raw {
			slotsList = append(slotsList, v.(map[string]interface{}))
		}

	default:
		return nil, errors.New("Invalid slot type found in DB")
	}
	return slotsList, nil
}

func ValidateSlot(slotsList []map[string]interface{}, timeGiven string) error {
	slotFound := false
	for _, slot := range slotsList {
		if slot["start"].(string) == timeGiven {
			slotFound = true
			if !slot["isAvailable"].(bool) {
				return errors.New(util.SLOT_UNAVAILABLE)
			}
			if slot["isBooked"].(bool) {
				return errors.New(util.SLOT_ALREADY_BOOKED)
			}
			break
		}
	}

	if !slotFound {
		return errors.New("Slot does not exist for this doctor")
	}
	return nil
}

/*
* Search for the slots in the given document
* Based on the given time filter it
* Then update several fields if match found
* Update the doctorAvailability slots with the search filter as well as update filter
 */
func checkAndBookSlot(ctx context.Context, slotColl *mongo.Collection, doc map[string]interface{}, timeGiven, patientId string) error {

	slotsList, err := ExtractSlots(doc)
	if err != nil {
		return err
	}

	if err = ValidateSlot(slotsList, timeGiven); err != nil {
		return err
	}
	update := bson.M{
		"$set": bson.M{
			"slots.$.patientId":   patientId,
			"slots.$.isAvailable": false,
			"slots.$.isBooked":    true,
		},
	}
	filter := bson.M{
		"doctorId":    doc["doctorId"],
		"hospitalId":  doc["hospitalId"],
		"date":        doc["date"],
		"slots.start": timeGiven,
	}
	_, err = db.UpdateOne(ctx, slotColl, filter, update)
	if err != nil {
		log.Println("Error while updating slots availability when match found: ", err)
	}
	return err
}

/*
* Generate medicalRecord code
* Generate new medicalDocument
* Insert new document in the medicalRecord db
 */
func createMedicalRecord(c *gin.Context, data map[string]interface{}, doctorId string, hospitalId string, nurseId string, createdBy string, tenantId string) (string, error) {
	medicalCode, err := common.GenerateEmpCode(util.MedicalRecordCollection)
	if err != nil {
		log.Println("Error while generating medicalRecord code: ", err)
		return "", err
	}

	medicalDoc := bson.M{
		"code":          medicalCode,
		"doctorId":      doctorId,
		"nurseId":       nurseId,
		"hospitalId":    hospitalId,
		"patientId":     data["patientId"],
		"tenantId":      tenantId,
		"appointmentId": data["code"],
		"reason":        data["reason"],
		"createdBy":     createdBy,
		"updatedBy":     createdBy,
		"createdAt":     time.Now(),
		"updatedAt":     time.Now(),
	}
	_, err = common.GenerateAndHashOTP(data)
	if err != nil {
		log.Println("Error from GeneraeAndHashOTP:", err)
		return "", err
	}
	coll := util.MedicalRecordCollection
	collection := db.OpenCollections(coll)
	key := util.MedicalRecordKey + medicalCode
	err = redis.SetCache(c, key, medicalDoc)
	if err != nil {
		log.Println("Error while caching new medicalRecord : ", err)
	}
	_, err = db.CreateOne(c, collection, medicalDoc)
	if err != nil {
		log.Println("Error while creating createMedicalRecord: ", err)
		return "", err
	}
	return medicalCode, err
}

type AppointmentInput struct {
	Data         map[string]interface{}
	DoctorID     string
	HospitalID   string
	NurseID      string
	AppCode      string
	MedicalCode  string
	CreatedBy    string
	DateModified string
}

/*
* BuildAppointment which is newOne
* Get all the fields
* return the new appointment
 */
func buildAppointment(input AppointmentInput) map[string]interface{} {
	return map[string]interface{}{
		"code":       input.AppCode,
		"date":       input.DateModified,
		"time":       input.Data["time"],
		"doctorId":   input.DoctorID,
		"nurseId":    input.NurseID,
		"hospitalId": input.HospitalID,
		"medicalId":  input.MedicalCode,
		"tenantId":   input.Data["tenantId"],
		"createdBy":  input.CreatedBy,
	}
}
func ExtractAppointments(patient map[string]interface{}) ([]string, error) {
	var appointments []string
	raw, exists := patient["appointments"]
	if !exists || raw == nil {
		appointments = []string{}
	} else {
		val, ok := raw.(primitive.A)
		log.Println("val: ", val)
		if !ok {
			log.Println("Unable to fetch appointments")
			return nil, errors.New(util.UNABLE_TO_FIND_APPOINTMENTS_IN_PATIENT)
		}
		for _, a := range val {
			if str, ok := a.(string); ok {
				appointments = append(appointments, str)
			}
		}
	}
	return appointments, nil
}

func ValidateLatestAppointment(c *gin.Context, appointments []string) error {

	if len(appointments) > 0 {
		latestAppointmentsId := appointments[len(appointments)-1]
		appointment, err := FetchAppointmentByCode(c, latestAppointmentsId)
		if err != nil {
			log.Println("Error from fetchAppointmentByCode: ", appointment)
			return err
		}
		isProcessing, ok := appointment["isProcessing"].(bool)
		if !ok {
			log.Println("isProcessing field unable to fetch from appointment")
			return errors.New(util.UNABLE_TO_FETCH_IS_PROCCESSING_FIELD)
		}
		if isProcessing {
			log.Println("Latestappointment is still processing,cannot create one more appointment")
			return errors.New(util.PATIENT_IS_STILL_PROCESSING)
		}
	}
	return nil
}

/*
* Get appointments from the patient
* Update appointmnets with new appointmentId
* Refresh the cache
 */
func PatientUpdate(c *gin.Context, data map[string]interface{}, appCode, patientId string) error {
	patCollection := db.OpenCollections(util.PatientCollection)

	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		log.Println("Error from fetchPatientByCode: ", err)
		return err
	}
	log.Println("Patient: ", patient)
	log.Printf("patient appointments type %T", patient["appointments"])
	appointments, err := ExtractAppointments(patient)
	if err != nil {
		return err
	}
	if err := ValidateLatestAppointment(c, appointments); err != nil {
		return err
	}
	appointments = append(appointments, appCode)
	patientUpdate := bson.M{
		"$set": bson.M{
			"appointments": appointments,
		},
	}
	log.Println("Appointments:", appointments)
	patientFilter := bson.M{
		"code": patientId,
	}
	updated, err := db.UpdateOne(c, patCollection, patientFilter, patientUpdate)
	if err != nil {
		log.Println("Error from UpdateOne: ", err)
		return err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updPatient := make(map[string]interface{})
	err = db.FindOne(c, patCollection, patientFilter, updPatient)
	if err != nil {
		log.Println("Error from FindOne function: ", err)
		return err
	}
	key := util.PatientKey + patientId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error while deleting patient from cache: ", err)
	}
	err = redis.SetCache(c, key, updPatient)
	if err != nil {
		log.Println("Error while caching updated patient: ", err)
	}
	return nil
}

/*
* GetReceptionistID from context
* Validate the input fields
* Normalize the date
* Check whether receptionist have access to create appointment for the doctorId
* DoctorAvailability check for weeklyOff and weekend and get slots
* Check the slot and book slot and update several fields
* Build appointment
* Update patient by appointment
 */

func CreateAppointment(c *gin.Context, doctorId string, nurseId string, data map[string]interface{}) (string, error) {

	receptionistId, err := getReceptionistID(c)
	if err != nil {
		log.Println("Error from getReceptionistID: ", err)
		return "", err
	}
	log.Println("CreatedBy:", receptionistId.(string))

	if err := validateAppointmentInput(data); err != nil {
		log.Println("Error from validateAppointmentInput: ", err)
		return "", err
	}

	dateModified, err := common.NormalizeDate(data["date"].(string))
	if err != nil {
		log.Println("Error from NormalizeDate: ", err)
		return "", err
	}

	doctor, err := CheckForPrivileges(c, receptionistId.(string), doctorId)
	if err != nil {
		log.Println("Error from FetchReceptionist: ", err)
		return "", err
	}
	slotColl := db.OpenCollections(util.DoctorTimeSlotCollection)
	docSlotFilter := bson.M{
		"doctorId":   doctorId,
		"hospitalId": doctor["createdBy"].(string),
		"date":       dateModified,
	}
	log.Println("filter: ", docSlotFilter)
	doc, err := fetchDoctorSlot(c, slotColl, docSlotFilter)
	if err != nil {
		log.Println("Error from fetchDoctorSlot:", err)
		return "", err
	}
	appCode, err := common.GenerateEmpCode(util.AppointmentCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}

	timeGiven := data["time"].(string)
	if err := checkAndBookSlot(c, slotColl, doc, timeGiven, data["patientId"].(string)); err != nil {
		log.Println("Error fron checkAndBookSlot: ", err)
		return "", err
	}
	data["code"] = appCode
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken", err)
		return "", err
	}
	data["tenantId"] = tenantId
	medicalCode, err := createMedicalRecord(c, data, doctorId, doctor["createdBy"].(string), nurseId, receptionistId.(string), tenantId)
	if err != nil {
		return "", err
	}

	hospitalId := doctor["createdBy"].(string)
	newApp := buildAppointment(AppointmentInput{
		Data:         data,
		DoctorID:     doctorId,
		HospitalID:   hospitalId,
		NurseID:      nurseId,
		AppCode:      appCode,
		MedicalCode:  medicalCode,
		CreatedBy:    receptionistId.(string),
		DateModified: dateModified,
	})
	patientErr := PatientUpdate(c, data, appCode, data["patientId"].(string))
	if patientErr != nil {
		log.Println("Error from patientUpdate: ", patientErr)
		return "", patientErr
	}

	collection := db.OpenCollections(util.AppointmentCollection)
	inserted, err := db.CreateOne(c, collection, newApp)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.AppointmentKey + appCode
	cacheErr := redis.SetCache(c, key, newApp)
	if cacheErr != nil {
		log.Println("Error from setCache : ", cacheErr)
		return "", cacheErr
	}

	return "created Successfully", nil
}

/*
* Get appointment for the given appointmentId
* Get tenantId,code,collection,isSuperAdmin from the context
* Check who can access
* Fetch from access, based on the accessibility
* if exists return
* If not exists fetch from database
* Return from database and set in cache
 */
func FetchAppointmentByCode(c *gin.Context, appointmentId string) (map[string]interface{}, error) {

	key := util.AppointmentKey + appointmentId

	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne(while fetching user): ", err)
		return nil, err
	}

	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(util.AppointmentCollection)
	filter := bson.M{"code": appointmentId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne(while fetching appointments): ", err)
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

/*
* Make a filter
* According to the user,the filter condition changes
* Search for listOfAppointments
* Return them
 */
func FetchAllAppointment(c *gin.Context) ([]interface{}, error) {
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
			log.Println("Error from findOne(while fetching receptionist): ", err)
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
	collection := db.OpenCollections(util.AppointmentCollection)
	doc, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from FindAll", err)
		return nil, err
	}
	return doc, nil
}

/*
* Build filter to search based on appointmentId
* If found, fetch field createdBy from the result document found
* Compare code from context and createdBy, if it works well go for the delete
* If not, no another receptionist can have access to delete it
 */
func DeleteAppointmentByCode(c *gin.Context, appointmentId string) (string, error) {
	collection := db.OpenCollections(util.AppointmentCollection)
	receptionistId, ok := c.Get("code")
	if !ok {
		return "", errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"code": appointmentId,
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
		return "", errors.New(util.INVALID_USER_TO_ACCESS)
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.AppointmentKey + appointmentId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", appointmentId)
	return msg, nil
}

func PrepareUpdateMetadata(c *gin.Context, data map[string]interface{}) {
	code := c.GetString("code")
	data["updatedBy"] = code
	data["updatedAt"] = time.Now()
}
func ValidateReceptionistAccess(appointment map[string]interface{}, code string) error {
	receptionist, ok := appointment["createdBy"].(string)
	if !ok {
		log.Println("Error while checking the value is present in it or not")
		return errors.New(util.UNABLE_TO_FETCH_CREATED_BY_FROM_APPOINTMENT)
	}
	if receptionist != code {
		log.Println("This receptionist doesnot have access to update the appointment")
		return errors.New(util.RECEPTIONIST_DOESNOT_HAVE_ACCESS_TO_UPDATE_APPOINTMENT)
	}
	return nil
}
func ValidateDoctorAccess(c *gin.Context, appointment map[string]interface{}, code string) error {
	doctor, err := FetchDoctorByCode(c, code)
	if err != nil {
		log.Println("Error from fetchPharmacistByCode: ", err)
		return err
	}
	hospitalIdFromApp, ok := appointment["hospitalId"].(string)
	if !ok {
		log.Println("Unable to get hospitalId from appointment")
		return errors.New(util.UNABLE_TO_FETCH_HOSPITAL_ID_FROM_APPOINTMENT)
	}
	if hospitalIdFromApp != doctor["createdBy"].(string) {
		log.Println("This doctor doesnot have access to update appointment")
		return errors.New(util.DOCTOR_DOESNOT_HAVE_ACCESS_TO_UPDATE_APPOINTMENT)
	}
	return nil
}
func ValidateUpdateAccess(c *gin.Context, appointment map[string]interface{}) error {

	code := c.GetString("code")
	collFromContext := c.GetString("collection")

	if collFromContext == util.ReceptionistCollection {
		return ValidateReceptionistAccess(appointment, code)
	}

	if collFromContext == util.DoctorCollection {
		return ValidateDoctorAccess(c, appointment, code)
	}

	return nil
}

func UpdateAppointment(c *gin.Context, appColl *mongo.Collection, appointmentId string, data map[string]interface{}) (map[string]interface{}, error) {
	filter := bson.M{
		"code": appointmentId,
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, appColl, filter, update)
	if err != nil {
		log.Println("Error while updating medicalRecord:", err)
		return nil, err
	}
	log.Println("Updated: ", updated.ModifiedCount)
	updatedAppointment := make(map[string]interface{})
	err = db.FindOne(c, appColl, filter, updatedAppointment)
	if err != nil {
		log.Println("Error from findOne after updating appointment:", err)
		return nil, err
	}
	return updatedAppointment, nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the search filters and update fields
* Update this appointment by either receptionist nor doctor
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateAppointmentByCode(c *gin.Context, appointmentId string, data map[string]interface{}) (string, error) {
	PrepareUpdateMetadata(c, data)

	appColl := db.OpenCollections(util.AppointmentCollection)
	err := common.CheckForEmailAndPhoneNo(c, appColl, data)
	if err != nil {
		log.Println("Error from checkForEmailAndPhoneNo: ", err)
		return "", err
	}
	appointment, err := FetchAppointmentByCode(c, appointmentId)
	if err != nil {
		log.Println("Error while fetching appointment(FindOne)", err)
		return "", err
	}

	if err := ValidateUpdateAccess(c, appointment); err != nil {
		return "", err
	}
	updatedAppointment, err := UpdateAppointment(c, appColl, appointmentId, data)
	if err != nil {
		log.Println("Error from updateAppointment: ", err)
		return "", err
	}

	code := c.GetString("code")
	key := util.ReceptionistKey + code

	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old appointment cache:", err)
	}

	if err := redis.SetCache(c, key, updatedAppointment); err != nil {
		log.Println("Failed caching updated appoitment:", err)
	}
	return "updated", nil
}
