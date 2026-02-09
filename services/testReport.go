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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateTestReport(c *gin.Context, patientId string) ([]string, error) {
	coll := util.TestReportCollection

	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		return nil, errors.New("error fetching patient by code")
	}
	appointmentId, err := getLatestAppointmentID(patient)
	if err != nil {
		return nil, err
	}

	latestApp, err := FetchAppointmentByCode(c, appointmentId)
	if err != nil {
		return nil, err
	}
	log.Println("latestApp: ", latestApp)
	medicalRecordId := latestApp["medicalId"].(string)
	log.Println("medicalRecordId: ", medicalRecordId)
	medicalRecord, err := FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		return nil, err
	}
	log.Println("MedicalRecord: ", medicalRecord)
	testlist, err := getMedicalRecordTestList(c, medicalRecord)
	if err != nil {
		return nil, err
	}

	rawDoctorId, ok := latestApp["doctorId"]
	if !ok {
		return nil, errors.New("doctorId missing in appointment")
	}
	doctorId, ok := rawDoctorId.(string)
	if !ok {
		return nil, errors.New("doctorId format invalid")
	}
	var reportCodes []string
	for _, testId := range testlist {
		code, err := createSingleTestReport(c, coll, testId, patientId, doctorId)
		if err != nil {
			return nil, err
		}
		reportCodes = append(reportCodes, code)
	}
	if err := updateMedicalRecordWithReports(c, medicalRecordId, reportCodes); err != nil {
		return nil, err
	}

	return reportCodes, nil
}

// func FetchTestReportsofPatient(c *gin.Context, patientId string) ([]map[string]interface{}, error) {
// 	filter := bson.M{
// 		"patientId": patientId,
// 	}
// 	coll := db.OpenCollections(testReportCollection)
// 	testReports, err := db.FindAll(c, coll)
// }

func getLatestAppointmentID(patient map[string]interface{}) (string, error) {
	rawApps, ok := patient["appointments"]
	if !ok {
		return "", errors.New("appointments missing in patient")
	}

	apps, ok := rawApps.([]interface{})
	if !ok {
		return "", errors.New("appointments format invalid")
	}
	if len(apps) == 0 {
		return "", errors.New("no appointments found")
	}

	appointmentId, ok := apps[len(apps)-1].(string)
	if !ok {
		return "", errors.New("invalid appointmentId format")
	}

	return appointmentId, nil
}

func getMedicalRecordTestList(c *gin.Context, medicalRecord map[string]interface{}) ([]string, error) {
	consentId := medicalRecord["consentId"].(string)
	consent, err := FetchConsentByCode(c, consentId)
	if err != nil {
		log.Println("Error from FetchConsentByCode: ", err)
		return nil, err
	}
	isConsentVerified, ok := consent["isConsentVerified"].(bool)
	if !ok {
		log.Println("Unable to fetch isConsentVerified from consent ")
		return nil, errors.New(util.IS_CONSENT_VERIFIED_UNABLE_TO_FETCH)
	}
	if !isConsentVerified {
		log.Println("isConsentVerified field is not approved ")
		return nil, errors.New(util.CONSENT_NOT_APPROVED)
	}
	log.Println("After")
	rawTestList, ok := medicalRecord["testList"]
	if !ok {
		return nil, errors.New("testList missing in medicalRecord")
	}

	rawList, err := normalizeMongoArray(rawTestList)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, v := range rawList {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("testId must be string")
		}
		result = append(result, s)
	}

	return result, nil
}

func normalizeMongoArray(raw interface{}) ([]interface{}, error) {
	switch v := raw.(type) {
	case primitive.A:
		return []interface{}(v), nil
	case []interface{}:
		return v, nil
	case []string:
		out := []interface{}{}
		for _, s := range v {
			out = append(out, s)
		}
		return out, nil
	case []primitive.M:
		out := []interface{}{}
		for _, m := range v {
			out = append(out, m)
		}
		return out, nil
	case []primitive.ObjectID:
		out := []interface{}{}
		for _, id := range v {
			out = append(out, id.Hex())
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported array type: %T", v)
	}
}

func createSingleTestReport(c *gin.Context, coll interface{}, testId string, patientId string, doctorId string) (string, error) {
	dummyTest, err := FetchTestByCode(c, testId)
	if err != nil {
		return "", err
	}
	testReport := map[string]interface{}{
		"testName":  testId,
		"price":     dummyTest["price"],
		"patientId": patientId,
		"doctorId":  doctorId,
	}

	collName := coll.(string)
	code, err := common.GenerateEmpCode(collName)
	if err != nil {
		return "", err
	}
	testReport["code"] = code

	key := util.TestReportKey + code
	err = redis.SetCache(c, key, testReport)
	if err != nil {
		log.Println("Error while caching new testReport: ", err)
	}
	if _, err := common.SaveUserToDB(collName, testReport); err != nil {
		return "", err
	}

	return code, nil
}

func updateMedicalRecordWithReports(c *gin.Context, medicalRecordId string, reports []string) error {
	data := map[string]interface{}{
		"testReports": reports,
	}
	return UpdateMedicalRecordByNurse(c, medicalRecordId, data)
}

func FetchTestReportsofPatientById(c *gin.Context, testReportId string) (map[string]interface{}, error) {
	coll := util.TestReportCollection
	key := util.TestReportKey + testReportId
	sa, err := common.IsSuperAdmin(c)
	if err != nil {
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
	if err == nil && exists && !sa {
		tenantIdFromCache, ok := cached["tenantId"].(string)
		if !ok {
			return nil, errors.New("cached testReport missing tenantId")
		}
		if tenantId != tenantIdFromCache {
			return nil, errors.New(util.INVALID_USER_TO_ACCESS)
		}
	}
	if err == nil && exists {
		log.Println("From cache")
		return cached, nil
	}

	result := make(map[string]interface{})
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code": testReportId,
	}
	err = db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return nil, err
	}
	if !sa {
		value := result["tenantId"].(string)
		if value != tenantId {
			return nil, errors.New(util.INVALID_USER_TO_ACCESS)
		}
	}
	err = redis.SetCache(c, key, result)
	if err != nil {
		log.Println("Error from setCache")
		return nil, err
	}

	return result, nil
}
