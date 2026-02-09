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

func ValidateTestInput(data map[string]interface{}) error {
	fields := []string{"testname", "price"}
	for _, f := range fields {
		if err := common.GetTrimmedString(data, f); err != nil {
			log.Println("Error from getTrimmedString:", err)
			return err
		}
	}
	return nil
}

/*
* Validate user inputs first
* Fetch collection name from the roleCode given
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from the hospital collection
* Include tenantId and generate otp and hash the otp
* Combine all the remaining data and prepare it
* Save to db and cache
* Send mail
 */
func CreateTest(c *gin.Context, data map[string]interface{}) (string, error) {
	val := ""
	err := ValidateTestInput(data)
	if err != nil {
		log.Println("Error from ValidateUserInput:", err)
		return val, err
	}
	collection := util.TestCollection
	userCodeVal, exists := c.Get("code")
	if !exists {
		log.Println("Error unable to get the code from the context")
		return "", errors.New("missing creator code")
	}
	createdBy := userCodeVal.(string)
	data["CreatedBy"] = createdBy
	data["UpdatedBy"] = createdBy
	data["CreatedAt"] = time.Now()
	data["UpdatedAt"] = time.Now()
	code, err := common.GenerateEmpCode(collection)
	if err != nil {
		log.Println("Error from GenerateEmpCode:", err)
		return "", err
	}
	data["code"] = code
	tenantId, err := common.GetTenantIdFromContext(c)
	if err != nil {
		log.Println("Error from getTenantIfFromToken: ", err)
		return val, err
	}
	log.Println("tenantId from context: ", tenantId)

	data["tenantId"] = tenantId
	key := util.TestKey + code
	err = redis.SetCache(c, key, data)
	if err != nil {
		log.Println("Error while caching new test: ", err)
	}
	if _, err := common.SaveUserToDB(collection, data); err != nil {
		log.Println("Error from the saveUserToDB:", err)
		return val, err
	}
	return "Test Creates Success", nil
}

/*
* If fields provided,trim them and append to the input data
* Get the code from claims which is createdBy field
* Update based on the update and search filters
 */
func UpdateTest(c *gin.Context, data map[string]interface{}, code string) error {
	fields := []string{"testname", "price"}
	for _, f := range fields {
		if err := common.TrimIfExists(data, f); err != nil {
			log.Println("Error from ")
			return err
		}
	}
	if err := common.HandleDOB(data); err != nil {
		return err
	}

	hospitalCode := c.GetString("code")
	updateFilter := common.BuildUpdateFilter(data, hospitalCode)
	filter := bson.M{
		"code": code,
	}
	collection := db.OpenCollections(util.TestCollection)
	value := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, value)
	if err != nil {
		log.Println("Error from the findOne function", err)
		return err
	}
	log.Println(value)
	val := value["createdBy"].(string)
	log.Println(val)
	log.Println(hospitalCode)
	if val != hospitalCode {
		log.Println("This hospital does not have access to update test")
		return errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS_TO_UPDATE_TEST)
	}
	res, err := db.UpdateOne(c, collection, filter, updateFilter)
	if err != nil {
		log.Println("Error from updateOne:", err)
		return err
	}
	log.Println(res.UpsertedCount)

	result := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, result)
	key := util.TestKey + code
	err = db.FindOne(c, collection, filter, result)
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, result); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return nil
}

/*
* Create a key to fetch from cache
* Fetch from cache if found then extract tenantId and compare with the input tenantId
* If not found go to db search for the document
* Check whether the tenantId matches with the input tenantId
* If comparision works then return the docs
 */
func FetchTestByCode(c *gin.Context, testId string) (map[string]interface{}, error) {
	coll := util.TestCollection
	key := util.TestKey + testId
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
			return nil, errors.New("cached test missing tenantId")
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
		"code": testId,
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

/*
* Make a filter
* FindAll from the above filter
 */
func FetchAllTests(c *gin.Context, tenantId string) ([]interface{}, error) {
	collection := db.OpenCollections(util.TestCollection)
	filter := bson.M{
		"tenantId": tenantId,
	}
	result, err := db.FindAll(c, collection, filter, nil)
	if err != nil {
		log.Println("Error from the findAll function: ", err)
		return nil, err
	}
	return result, nil
}

/*
* Get code from the token
* Compare code with the createdBy from the result document found from filter
* If comparision works well go for the delete
* If not return no another hospital admin can have access to delete it
 */
func DeleteTest(c *gin.Context, code string) (string, error) {
	key := util.TestKey + code
	collection := db.OpenCollections(util.TestCollection)
	hospitalCode := c.GetString("code")
	filter := bson.M{
		"code": code,
	}
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, result)
	if err != nil {
		log.Println("Error from the findOne function: ", err)
		return "", err
	}
	val := result["createdBy"].(string)
	if val != hospitalCode {
		log.Println("This hospital admin doesnot have access")
		return "", errors.New(util.HOSPITAL_ADMIN_DOESNOT_HAVE_ACCESS)
	}
	err = redis.DeleteCache(c, key)
	if err != nil {
		return "", err
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	msg := fmt.Sprintf("The test %s deleted and the count is %d", code, deleted)
	return msg, nil
}
