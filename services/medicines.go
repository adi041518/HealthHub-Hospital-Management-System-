package services

import (
	"errors"
	"log"
	"strconv"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
* Validate the input fields
* Normalize the expiryDate
* Get pharmacistId from the context
* Bind the data with some more fields
* Check whether the medicines withe same name already exists in db
* Create in db
* Set in cache
 */
func CreateMedicines(c *gin.Context, data map[string]interface{}) (string, error) {
	fields := []string{"name", "dosage", "expiryDate", "noOfStrips", "tabletsPerStrip", "pricePerStrip"}
	for _, value := range fields {
		err := common.GetTrimmedString(data, value)
		if err != nil {
			log.Println("Error from getTrimmedString")
			return "", err
		}
	}
	dateStr, err := common.NormalizeDate(data["expiryDate"].(string))
	if err != nil {
		log.Println("Error from normalizeDate: ", err)
		return "", err
	}
	data["expiryDate"] = dateStr
	pharmacistId := c.GetString("code")
	noOfStripsVal, ok := data["noOfStrips"].(string)
	if !ok {
		log.Println("Unable to get noOfStrips")
		return "", errors.New(util.UNABLE_TO_FETCH_NO_OF_STRIPS)
	}
	noOfStrips, _ := strconv.Atoi(noOfStripsVal)
	tabletsPerStripVal, ok := data["tabletsPerStrip"].(string)
	if !ok {
		log.Println("Unable to get tabletsPerStrips")
		return "", errors.New(util.UNABLE_TO_FETCH_TABLETS_PER_STRIP)
	}
	tabletsPerStrip, _ := strconv.Atoi(tabletsPerStripVal)
	data["createdBy"] = pharmacistId
	totalNoOfTabletsVal := noOfStrips * tabletsPerStrip
	totalNoOfTablets := strconv.Itoa(totalNoOfTabletsVal)
	data["totalNoOfTablets"] = totalNoOfTablets
	coll := util.MedicineCollection
	collection := db.OpenCollections(coll)

	filter := bson.M{
		"name": data["name"],
	}
	medicine := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, medicine)
	if !errors.Is(err, mongo.ErrNoDocuments) {
		log.Println("Medicine with same name already exists: ", err)
		return "", errors.New(util.MEDICINE_ALREADY_EXISTS_WITH_THIS_NAME)
	}
	code, err := common.GenerateEmpCode(util.MedicineCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	data["code"] = code
	pharmaColl := util.PharmacistCollection
	pharmaCollection := db.OpenCollections(pharmaColl)
	pharmacist := make(map[string]interface{})
	pFilter := bson.M{
		"code": pharmacistId,
	}
	err = db.FindOne(c, pharmaCollection, pFilter, pharmacist)
	if err != nil {
		log.Println("Error from findOne function: ", err)
		return "", err
	}
	data["tenantId"] = pharmacist["tenantId"].(string)
	data["hospitalId"] = pharmacist["createdBy"].(string)
	log.Println("MEDICINE CODE:", code)

	inserted, err := db.CreateOne(c, collection, data)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("Inserted: ", inserted.InsertedID)
	key := util.MedicinesKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error from setCache: ", err)
		return "", err
	}
	return "Successfully created", nil
}

/*
* Get medicine for the given medicineId
* Get tenantId,code,collection,isSuperAdmin from the context
* Check who can access
* Fetch from access, based on the accessibility
* if exists return
* If not exists fetch from database
* Return from database and set in cache
 */
func FetchMedicineByCode(c *gin.Context, medicineId string) (map[string]interface{}, error) {

	key := util.MedicinesKey + medicineId
	tenantId := c.GetString("tenantId")
	code := c.GetString("code")
	collFromContext := c.GetString("collection")
	isSuperAdmin := c.GetBool("isSuperAdmin")

	collectionFromContext := db.OpenCollections(collFromContext)
	userData := make(map[string]interface{})
	err := db.FindOne(c, collectionFromContext, bson.M{"code": code}, userData)
	if err != nil {
		log.Println("Error from findOne while fetching user: ", err)
		return nil, err
	}

	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}

	coll := db.OpenCollections(util.MedicineCollection)
	filter := bson.M{"code": medicineId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne while fetching medicines: ", err)
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
* Search for listOfMedicines
* Return them
 */
func FetchAllMedicines(c *gin.Context) ([]interface{}, error) {
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
	} else if ctxCollection == util.PharmacistCollection {
		collection := db.OpenCollections(util.PharmacistCollection)
		pharmacist := make(map[string]interface{})
		err := db.FindOne(c, collection, bson.M{"code": code}, pharmacist)
		if err != nil {
			log.Println("Error from findOne: ", err)
			return nil, err
		}
		filter = bson.M{
			"hospitalId": pharmacist["createdBy"].(string),
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
	collection := db.OpenCollections(util.MedicineCollection)
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
* Update this medicine by pharmacist, who has access only match hospitalId's of pharmacist and medicines
* Fetch updated document
* Delete from cache, set in Cache
 */
func UpdateMedicines(c *gin.Context, medicineId string, data map[string]interface{}) (string, error) {
	pharmacistId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	fields := []string{"name", "dosage", "expiryDate"}
	for _, field := range fields {
		err := common.TrimIfExists(data, field)
		if err != nil {

			log.Println("Error from trimIfExists: ", err)
			return "", err
		}
	}
	intFields := []string{"noOfStrips", "tabletsPerStrip"}
	for _, field := range intFields {
		number, ok := data[field].(float64)
		if ok {
			data[field] = int(number)
		}
	}
	coll := util.MedicineCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code": medicineId,
	}
	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from findOne while fetching medicine:", err)
		return "", err
	}
	pharmacist := make(map[string]interface{})
	pharmaCollection := db.OpenCollections(util.PharmacistCollection)
	pharmaFilter := bson.M{
		"code": pharmacistId,
	}
	err = db.FindOne(c, pharmaCollection, pharmaFilter, pharmacist)
	if err != nil {
		log.Println("Error from findOne while fetching pharmacist: ", err)
		return "", err
	}
	if pharmacist["createdBy"].(string) != result["hospitalId"].(string) {
		log.Println("This pharmacist doesnot have access")
		return "", errors.New(util.PHARMACIST_DOESNOT_HAVE_ACCESS)
	}
	update := bson.M{
		"$set": data,
	}
	updated, err := db.UpdateOne(c, collection, filter, update)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return "", err
	}
	log.Println("updated medicine count: ", updated.ModifiedCount)
	updatedMedicine := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, updatedMedicine)
	if err != nil {
		log.Println("Error from findOne:", err)
		return "", err
	}
	key := util.MedicinesKey + medicineId
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old medicine cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated medicine:", err)
	}
	return "Updated successfully", nil
}

/*
* Build filter to search based on medicineId
* If found, fetch field createdBy from the result document found
* Compare code from context and createdBy, if it works well go for the delete
* If not, no another pharmacist can have access to delete it
 */
func DeleteMedicine(c *gin.Context, medicineId string) (string, error) {
	pharmacistId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from getFromContext: ", err)
		return "", err
	}
	coll := util.MedicineCollection
	collection := db.OpenCollections(coll)
	filter := bson.M{
		"code":      medicineId,
		"createdBy": pharmacistId,
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
