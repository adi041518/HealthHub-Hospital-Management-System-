package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	redis "github.com/KanapuramVaishnavi/Core/config/redis"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var RCBE string = "roleCode cannot be empty"

func ValidateRoleData(data map[string]interface{}) (string, []map[string]interface{}, error) {
	roleName, err := getStringField(data, "roleName", true)
	if err != nil {
		return "", nil, err
	}
	roleName = strings.ToUpper(roleName)

	privilegesRaw, exists := data["privileges"]
	if !exists {
		return "", nil, errors.New("privileges are required")
	}

	privList, ok := privilegesRaw.([]interface{})
	if !ok || len(privList) == 0 {
		return "", nil, errors.New("privileges cannot be empty")
	}

	privileges := make([]map[string]interface{}, 0, len(privList))
	moduleSet := make(map[string]bool)

	for i, p := range privList {
		priv, err := validatePrivilege(p, i, moduleSet)
		if err != nil {
			return "", nil, err
		}
		privileges = append(privileges, priv)
	}

	return roleName, privileges, nil
}

// Helper: Get a string field and validate emptiness
func getStringField(data map[string]interface{}, key string, required bool) (string, error) {
	valRaw, exists := data[key]
	if required && !exists {
		return "", fmt.Errorf("%s is required", key)
	}
	val, ok := valRaw.(string)
	if !ok || strings.TrimSpace(val) == "" {
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	return strings.TrimSpace(val), nil
}

// Helper: Validate single privilege
func validatePrivilege(p interface{}, index int, moduleSet map[string]bool) (map[string]interface{}, error) {
	item, ok := p.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid privilege at index %d", index)
	}

	module, err := getStringField(item, "module", true)
	if err != nil {
		return nil, fmt.Errorf("%s at index %d", err.Error(), index)
	}

	if moduleSet[module] {
		return nil, fmt.Errorf("duplicate module found: %s", module)
	}
	moduleSet[module] = true
	item["module"] = module

	accessRaw, exists := item["access"]
	if !exists {
		return nil, fmt.Errorf("access is required for module %s", module)
	}

	accessListRaw, ok := accessRaw.([]interface{})
	if !ok || len(accessListRaw) == 0 {
		return nil, fmt.Errorf("access list cannot be empty for module %s", module)
	}

	accessList := make([]string, 0, len(accessListRaw))
	for _, a := range accessListRaw {
		str, ok := a.(string)
		if !ok || strings.TrimSpace(str) == "" {
			return nil, fmt.Errorf("invalid access value for module %s", module)
		}
		accessList = append(accessList, strings.TrimSpace(str))
	}
	item["access"] = accessList

	return item, nil
}

func PrepareRole(c *gin.Context, data map[string]interface{}, roleName string, privileges []map[string]interface{}) (map[string]interface{}, error) {
	data["roleName"] = roleName
	data["privileges"] = privileges

	isNew, err := CheckIfRoleNameExists(context.Background(), roleName)
	if err != nil {
		return nil, err
	}
	if !isNew {
		return nil, errors.New("roleName already exists")
	}

	roleCode, err := common.GenerateEmpCode(util.RoleCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return nil, fmt.Errorf("failed to generate roleCode: %v", err)
	}

	data["roleCode"] = roleCode
	createdBy := "SYSTEM"
	if roleName != util.SuperAdminCollection {
		codeFromContext, err := common.GetFromContext[string](c, "code")
		if err != nil {
			log.Println("Error from getFromContext: ", err)
			return nil, err
		}
		createdBy = codeFromContext
	}
	data["CreatedBy"] = createdBy
	data["UpdatedBy"] = createdBy
	data["CreatedAt"] = time.Now()
	data["UpdatedAt"] = time.Now()

	return data, nil
}

/*
* Check is RoleName is empty
* check if any document present in db with same RoleName
* Check if previlege is empty,Check if module is empty,Check if access length is 0
* Set remaining feilds
* CreateOne create the document in the respective db provided
 */

func CreateRole(c *gin.Context, data map[string]interface{}) (string, error) {
	roleName, privileges, err := ValidateRoleData(data)
	if err != nil {
		return "", err
	}

	data, err = PrepareRole(c, data, roleName, privileges)
	if err != nil {
		return "", err
	}
	collection := db.OpenCollections(util.RoleCollection)
	_, err = db.CreateOne(c, collection, data)
	if err != nil {
		return "", fmt.Errorf("failed to insert role: %v", err)
	}

	key := util.RoleKey + data["roleCode"].(string)
	if err := redis.SetCache(c, key, data); err != nil {
		return "", fmt.Errorf("failed to cache role: %v", err)
	}
	log.Println("data: ", data)
	return "Role created successfully", nil
}

/*
* Check if the collection consists of document with the filter
* If any document not fund nor db error throw error
* If not find the document and check with the field if already exists return false
* Return true only when the document not fund
 */
func CheckIfRoleNameExists(c context.Context, roleName string) (bool, error) {
	collectionStr := util.RoleCollection
	collection := db.OpenCollections(collectionStr)
	filter := bson.M{
		"roleName": roleName,
	}
	result := make(map[string]interface{})

	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		if err == mongo.ErrNoDocuments || strings.Contains(err.Error(), "no matching document found") {
			return true, nil
		}
		return false, fmt.Errorf("database error: %v", err)
	}

	if value, exists := result["roleName"]; exists && value == roleName {
		log.Printf("Role name '%s' already exists\n", roleName)
		return false, errors.New(util.ROLE_NAME_ALREADY_EXISTS)
	}
	return true, nil
}

// /*
// * Check if previleges exists or not
// * Check if the module data is present or not
// * Check if the access length is more than 0 or not
// * Return false for the above conditions
// * only return true when previleges field is fine
//  */
// func CheckIfPrivilegesIsEmpty(c context.Context, previleges []map[string]interface{}) (bool, error) {
// 	for _, p := range previleges {

// 		val, exists := p["module"]
// 		if !exists {
// 			return false, errors.New(util.MODULE_NOT_PROVIDED)
// 		}

// 		module, ok := val.(string)
// 		if !ok || strings.TrimSpace(module) == "" {
// 			return false, errors.New(util.MODULE_NOT_PROVIDED)
// 		}

// 		val, exists = p["access"]
// 		if !exists {
// 			return false, errors.New(util.ACCESS_NOT_PROVIDED)
// 		}

// 		access, ok := val.([]string)
// 		if !ok || len(access) == 0 {
// 			return false, errors.New(util.ACCESS_NOT_PROVIDED)
// 		}
// 	}
// 	return true, nil
// }

// /*
// *  Check if the same module present in the array
//  */
// func CheckDuplicateModules(privileges []map[string]interface{}) error {
// 	moduleSet := make(map[string]bool)

// 	for _, p := range privileges {
// 		module, _ := p["module"].(string)
// 		moduleClean := strings.TrimSpace(module)

// 		if moduleClean == "" {
// 			continue
// 		}

// 		if moduleSet[moduleClean] {
// 			return fmt.Errorf("duplicate module found: %s", moduleClean)
// 		}

// 		moduleSet[moduleClean] = true
// 	}

// 	return nil
// }

// /*
// * Take map[string]interface
// * Do type assertion for each of them
// * Do generate the roleCode
// * Convert the []interface{} to the []map[string]interface{}
//  */
// func PrepareRoleData(c *gin.Context, roleData map[string]interface{}) (map[string]interface{}, error) {

// 	err := getTrimmedString(roleData, "roleName")
// 	if err != nil {
// 		log.Println("Error from getTrimmedString:", err)
// 		return nil, err
// 	}
// 	if v, ok := roleData["roleName"].(string); ok {
// 		roleData["roleName"] = v
// 	}
// 	roleName := strings.ToUpper(roleData["roleName"].(string))
// 	exists, err := CheckIfRoleNameExists(c, roleName)
// 	if !exists {
// 		log.Println("Error from the CheckIdRoleNameExists")
// 		return nil, err
// 	}
// 	collection := "role"
// 	roleCode, err := GenerateEmpCode(collection)
// 	if err != nil {
// 		log.Println("Error while generating code", err)
// 	}
// 	roleData["roleCode"] = roleCode
// 	v, ok := roleData["privileges"].([]interface{})
// 	if !ok {

// 		log.Println("Privileges field not found")
// 		return nil, errors.New("Privileges field not found")
// 	}
// 	privs := make([]map[string]interface{}, 0)

// 	for _, item := range v {
// 		if m, ok := item.(map[string]interface{}); ok {
// 			if moduleVal, exists := m["module"]; exists {
// 				m["module"] = module
// 			}
// 			if accessRaw, exists := m["access"].([]interface{}); exists {
// 				accessList := make([]string, 0)
// 				for _, a := range accessRaw {
// 					if s, ok := a.(string); ok {
// 						accessList = append(accessList, s)
// 					}
// 				}
// 				m["access"] = accessList
// 			}

// 			privs = append(privs, m)
// 		}
// 	}
// 	roleData["privileges"] = privs

// 	return roleData, nil
// }

/*
It returns the roles present in the database
*/
func ReadRoles(ctx *gin.Context) ([]interface{}, error) {
	var results []interface{}
	collection := db.OpenCollections("role")
	results, err := db.FindAll(ctx, collection, nil, nil)
	if err != nil {
		return []interface{}{}, err
	}
	return results, nil
}

/*
parsePrivileges converts []interface{} into []map[string]interface{}
and ensures access list is []string.
*/
func parsePrivileges(raw []interface{}) []map[string]interface{} {
	privs := make([]map[string]interface{}, 0, len(raw))

	for _, item := range raw {
		if m, ok := toMap(item); ok {
			if module, ok := getStringNow(m, "module"); ok {
				m["module"] = module
			}
			if accessRaw, ok := m["access"].([]interface{}); ok {
				m["access"] = parseAccessList(accessRaw)
			}
			privs = append(privs, m)
		}
	}

	return privs
}

// Helper: Get string value from map
func getStringNow(m map[string]interface{}, key string) (string, bool) {
	val, ok := m[key].(string)
	if ok {
		return val, true
	}
	return "", false
}

// Helper: Convert interface{} to map[string]interface{}
func toMap(i interface{}) (map[string]interface{}, bool) {
	m, ok := i.(map[string]interface{})
	return m, ok
}

// Helper: Convert []interface{} to []string
func parseAccessList(raw []interface{}) []string {
	list := make([]string, 0, len(raw))
	for _, a := range raw {
		if str, ok := a.(string); ok {
			list = append(list, str)
		}
	}
	return list
}

/*
parseUpdateFields extracts and normalizes update fields
before applying them to the DB.
*/
func parseUpdateFields(updateData map[string]interface{}) (bson.M, error) {

	update := bson.M{}

	if v, ok := updateData["roleName"].(string); ok && strings.TrimSpace(v) != "" {
		update["roleName"] = strings.ToUpper(v)
	}

	if rawPrivs, ok := updateData["privileges"].([]interface{}); ok {
		update["privileges"] = parsePrivileges(rawPrivs)
	}

	return update, nil
}

/*
updateRoleInDB applies update fields to an existing role document.
*/
func updateRoleInDB(c *gin.Context, roleCode string, update bson.M) error {
	collection := db.OpenCollections("role")
	_, err := db.UpdateOne(c, collection, bson.M{"roleCode": roleCode}, update)
	if err != nil {
		return fmt.Errorf("update failed: %v", err)
	}
	return nil
}

/*
UpdateRole handles updating an existing role:
- Validates updates
- Applies update to DB
- Refreshes Redis cache
*/
func UpdateRole(c *gin.Context, roleCode string, updateData map[string]interface{}) (map[string]interface{}, error) {

	if strings.TrimSpace(roleCode) == "" {
		return nil, errors.New(RCBE)
	}

	updateFields, err := parseUpdateFields(updateData)
	if err != nil {
		return nil, err
	}

	if len(updateFields) == 0 {
		return nil, errors.New("no valid fields to update")
	}

	updateFields["UpdatedAt"] = time.Now()
	updateFields["UpdatedBy"] = "SYSTEM"

	if err := updateRoleInDB(c, roleCode, bson.M{"$set": updateFields}); err != nil {
		return nil, err
	}

	collection := db.OpenCollections("role")
	var updated map[string]interface{}
	err = db.FindOne(c, collection, bson.M{"roleCode": roleCode}, &updated)
	if err != nil {
		return nil, err
	}
	key := util.TestKey + roleCode
	if err := redis.DeleteCache(c, key); err != nil {
		log.Println("Failed deleting old tenant cache:", err)
	}

	if err := redis.SetCache(c, key, updated); err != nil {
		log.Println("Failed caching updated tenant:", err)
	}

	return updated, nil
}

/*
FetchRoleById retrieves a role by its roleCode.
Steps:
1. Check Redis cache
2. If missing, fetch from MongoDB
3. Repopulate cache
*/
func FetchRoleById(c *gin.Context, roleCode string) (map[string]interface{}, error) {

	if strings.TrimSpace(roleCode) == "" {
		return nil, errors.New(RCBE)
	}
	key := util.RoleKey + roleCode

	var cached map[string]interface{}
	found, err := redis.GetCache(c, key, &cached)
	if err == nil && found {
		return cached, nil
	}

	collection := db.OpenCollections(util.RoleCollection)
	filter := bson.M{"roleCode": roleCode}
	role := make(map[string]interface{})
	err = db.FindOne(c, collection, filter, role)
	if err != nil {
		return nil, errors.New("role not found")
	}

	return role, nil
}

/*
DeleteRole removes a role from DB and clears its cache entry.
*/
func DeleteRole(c *gin.Context, roleCode string) error {

	if strings.TrimSpace(roleCode) == "" {
		return errors.New(RCBE)
	}

	collection := db.OpenCollections("role")

	filter := bson.M{"roleCode": roleCode}

	var existing map[string]interface{}
	err := db.FindOne(c, collection, filter, &existing)
	if err != nil {
		return errors.New("role not found")
	}

	_, err = db.DeleteOne(c, collection, filter)
	if err != nil {
		return err
	}
	key := util.RoleKey + roleCode
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error while deleting role: ", err)
	}

	return nil
}
