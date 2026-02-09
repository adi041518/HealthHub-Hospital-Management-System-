package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	db "github.com/KanapuramVaishnavi/Core/config/db"
	jwt "github.com/KanapuramVaishnavi/Core/config/jwt"
	common "github.com/KanapuramVaishnavi/Core/coreServices"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

/*
* Check is the emailExists,phoneExists,codeExists or not
* If non of these three exists then throw error
* If any of the field provided and the value is empty or type assertion then throw error
 */

func validateLoginInput(data map[string]interface{}) error {
	_, emailExists := data["email"]
	_, phoneExists := data["phoneNo"]
	_, codeExists := data["code"]

	if !emailExists && !phoneExists && !codeExists {
		return errors.New(util.PLEASE_PROVIDE_EMAIL_OR_PHONE_OR_CODE)
	}
	_, passwordExists := data["password"]
	if !passwordExists {
		return errors.New(util.PASSWORD_NOT_PROVIDED)
	}
	if passwordExists {
		err := common.GetTrimmedString(data, "password")
		if err != nil {
			log.Println("error from getTrimmed string for password:", err)
			return errors.New(util.PASSWORD_NOT_PROVIDED)
		}
	}
	if emailExists {
		err := common.GetTrimmedString(data, "email")
		if err != nil {
			log.Println("error from getTrimmed string for email:", err)
			return errors.New(util.EMAIL_NOT_PROVIDED)
		}
	}
	if phoneExists {
		err := common.GetTrimmedString(data, "phoneNo")
		if err != nil {
			log.Println("error from getTrimmed string for phoneNo:", err)
			return errors.New(util.PHONE_NUMBER_NOT_PROVIDED)
		}
	}
	if codeExists {
		err := common.GetTrimmedString(data, "code")
		if err != nil {
			log.Println("error from getTrimmed string for code:", err)
			return errors.New(util.CODE_NOT_PROVIDED)
		}
	}
	return nil
}

/*
* Create Filter to find the document in db
 */
func buildLoginFilter(data map[string]interface{}) bson.M {
	filter := bson.M{}

	if v, ok := data["email"].(string); ok && v != "" {
		filter["email"] = v
	}
	if v, ok := data["phoneNo"].(string); ok && v != "" {
		filter["phoneNo"] = v
	}
	if v, ok := data["code"].(string); ok && v != "" {
		filter["code"] = v
	}

	return filter
}

/*
* Pass the fiter and find which document gets matches with the filter
 */
func FetchUser(ctx context.Context, filter bson.M) (map[string]interface{}, error) {
	collection := db.OpenCollections("LOGIN")
	result := make(map[string]interface{})

	err := db.FindOne(ctx, collection, filter, &result)
	if err != nil {
		return nil, errors.New("user not found in login collection")
	}

	return result, nil
}

/*
* Fetch user based on the collection and code
* Using findOne search for it
 */
func FetchUserByRole(ctx context.Context, collectionName string, code string) (map[string]interface{}, error) {
	collection := db.OpenCollections(collectionName)
	result := make(map[string]interface{})

	err := db.FindOne(ctx, collection, bson.M{"code": code}, &result)
	if err != nil {
		return nil, fmt.Errorf("no user found in %s collection", collectionName)
	}
	return result, nil
}

/*
* Check for the otp expiry
* check with time which is now with the otpExpiry
 */
func ValidateOTPExpiry(userDoc map[string]interface{}) error {

	otpExpiryRaw, ok := userDoc["otpExpiry"]
	if !ok {
		return errors.New("OTP expiry not found. Please complete verification")
	}

	var expiryTime time.Time

	switch v := otpExpiryRaw.(type) {
	case primitive.DateTime:
		expiryTime = v.Time()
	case time.Time:
		expiryTime = v
	default:
		return errors.New("invalid otpExpiry format in user collection")
	}

	if time.Now().After(expiryTime) {
		return errors.New("OTP expired. Please request a new OTP")
	}

	return nil
}

/*
* If match found then compare the input password and then the password found from the filtered document
 */
func verifyPassword(dbPassword string, inputPassword string) error {
	if strings.TrimSpace(dbPassword) == "" {
		return errors.New("stored password missing or invalid")
	}

	err := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(inputPassword))
	if err != nil {
		return errors.New("Password mismatch")
	}

	return nil
}

/*
* If login fails then set in cache the key count
* Increment the count for the key
* Set key value for 10mins
 */
// func IncrementLoginAttempts(code string) (int, error) {
// 	key := "LOGIN_FAIL:" + code

// 	attempts, err := redis.Rdb.Incr(context.Background(), key).Result()
// 	if err != nil {
// 		log.Println("Unable to increment login count")
// 		return 0, err
// 	}

//		log.Println("Current attempts:", attempts)
//		return int(attempts), nil
//	}
var loginAttempts = make(map[string]int)
var mu sync.Mutex

func IncrementLoginAttempts(code string) int {
	mu.Lock()
	defer mu.Unlock()

	loginAttempts[code]++
	attempts := loginAttempts[code]

	log.Println("Current attempts for", code, ":", attempts)
	return attempts
}

/*
* Pass the attempts
* And update the document with the token generated
 */
func UpdateUserAttempts(ctx context.Context, collectionName string, code string, attempts int) error {
	collection := db.OpenCollections(collectionName)

	filter := bson.M{"code": code}
	update := bson.M{"$set": bson.M{"loginAttempts": attempts}}

	_, err := db.UpdateOne(ctx, collection, filter, update)
	return err
}

func LoginAttempts(c *gin.Context, code string, collection, dbPassword, inputPassword string) error {

	passErr := verifyPassword(dbPassword, inputPassword)
	if passErr != nil {
		attempts := IncrementLoginAttempts(code)
		if err := UpdateUserAttempts(c, collection, code, attempts); err != nil {
			log.Println("Error while updating the attempts in collection")
			return err
		}
		if attempts >= 3 {
			// Disable account in MongoDB
			updated, err := db.UpdateOne(context.Background(),
				db.OpenCollections(collection),
				bson.M{"code": code},
				bson.M{"$set": bson.M{"isBlocked": true}},
			)
			log.Println("Error while updating the collection for isActive field")
			log.Println("Updating: ", updated.ModifiedCount)
			return err
		}

		log.Println("Error from IncrementLoginattempts")
		return passErr
	}
	return nil
}

func TokenGeneration(userDoc map[string]interface{}, code string, email string, collection string) (string, error) {

	roleCode := userDoc["roleCode"].(string)
	tenantId := ""
	isSuperAdmin := false
	if collection == util.SuperAdminCollection {
		tenantId = ""
		isSuperAdmin = true
	} else {
		tenantId = userDoc["tenantId"].(string)
		isSuperAdmin = false
	}
	log.Println("Login isSuperAdmin:", isSuperAdmin)
	log.Println("Login tenantId:", tenantId)
	token, err := jwt.GenerateJWT(code, email, roleCode, collection, tenantId, isSuperAdmin)
	if err != nil {
		log.Println("Error while generating the token: ", err)
		return "", err
	}
	return token, nil
}

/*
* Pass the token
* And update the document with the token generated
 */
func UpdateUserByToken(ctx context.Context, collectionName string, code string, token string) error {
	collection := db.OpenCollections(collectionName)

	filter := bson.M{"code": code}
	update := bson.M{"$set": bson.M{"token": token, "isActive": true}}

	_, err := db.UpdateOne(ctx, collection, filter, update)
	return err
}

/*
* Validate super admin inputs first
* Build the filter to find the document
* Fetch superAdmin
* Verify Password
* GenerateJWT
* UpdateToken
 */
func Login(c *gin.Context, data map[string]interface{}) (map[string]interface{}, error) {
	if err := validateLoginInput(data); err != nil {
		log.Println("error from validation input for the login")
		return nil, err
	}

	filter := buildLoginFilter(data)
	loginDoc, err := FetchUser(context.Background(), filter)
	if err != nil {
		log.Println("error from the fetchUser function:", err)
		return nil, err
	}

	inputPassword := data["password"].(string)
	dbPassword := loginDoc["password"].(string)
	collection := loginDoc["collection"].(string)
	code := loginDoc["code"].(string)
	email := loginDoc["email"].(string)

	userDoc, err := FetchUserByRole(c, collection, code)
	if err != nil {
		log.Println("Error from FetchUserByRole", err)
		return nil, err
	}

	err = LoginAttempts(c, code, collection, dbPassword, inputPassword)
	if err != nil {
		log.Println("Error from attempts:", err)
		return nil, err
	}
	if userDoc["reset"] == true {
		if err := ValidateOTPExpiry(userDoc); err != nil {
			return nil, err
		}
		log.Println("while password is otp")
	}

	token, err := TokenGeneration(userDoc, code, email, collection)
	if err != nil {
		log.Println("Error from tokenGeneration: ", err)
		return nil, err
	}

	if err := UpdateUserByToken(c, collection, code, token); err != nil {
		log.Println("Error while updating the collection with token field")
		return nil, err
	}
	user, err := FetchUserByRole(c, collection, code)
	if err != nil {
		log.Println("Error from FetchUserByRole", err)
		return nil, err
	}
	log.Println("patient : ", user)
	return user, nil
}

/*
* Extract token info like code, and collection
 */
func ExtractTokenInfo(c *gin.Context) (collection string, code string, err error) {
	collectionVal, ok := c.Get("collection")
	if !ok {
		return "", "", errors.New("invalid token: collection missing")
	}
	log.Println("collection from token:", collectionVal)
	collection, ok = collectionVal.(string)
	if !ok || collection == "" {
		return "", "", errors.New("invalid token: collection invalid")
	}

	codeVal, ok := c.Get("code")
	if !ok {
		return "", "", errors.New("invalid token: code missing")
	}

	log.Println("code from token:", codeVal)
	code, ok = codeVal.(string)
	if !ok || code == "" {
		return "", "", errors.New("invalid token: code invalid")
	}
	superAdminVal, ok := c.Get("isSuperAdmin")
	IsSuperAdmin, ok := superAdminVal.(bool)
	log.Println("isSuperAdmin from context", IsSuperAdmin)
	tenantIdVal, ok := c.Get("tenantId")
	tenantId, ok := tenantIdVal.(string)
	log.Println("tenantId from context: ", tenantId)
	return collection, code, nil
}

/*
* Validate input field
 */
func ValidatePasswordInput(body map[string]interface{}) (string, string, error) {
	_, npExists := body["newPassword"]
	_, cpExists := body["confirmPassword"]

	if !npExists {
		return "", "", errors.New("newPassword required")
	}
	if !cpExists {
		return "", "", errors.New("confirmPassword required")
	}
	err := common.GetTrimmedString(body, "newPassword")
	if err != nil {
		log.Println("Error from getTrimmedString:", err)
		return "", "", errors.New("invalid newPassword")
	}
	err = common.GetTrimmedString(body, "confirmPassword")
	if err != nil {
		log.Println("Error from getTrimmedString:", err)
		return "", "", errors.New("invalid confirmPassword")
	}
	newPassword := body["newPassword"].(string)
	confirmPassword := body["confirmPassword"].(string)
	if newPassword != confirmPassword {
		return "", "", errors.New("newPassword and confirmPassword do not match")
	}
	log.Println(newPassword)
	log.Println(confirmPassword)
	return newPassword, confirmPassword, nil
}

/*
* Update password in main collection as well as in the login collection
 */
func UpdatePasswordInCollections(c *gin.Context, collectionName string, code string, hashedPassword string) error {
	filter := bson.M{"code": code}

	coll := db.OpenCollections(collectionName)
	update := bson.M{
		"$set": bson.M{
			"password":  hashedPassword,
			"reset":     false,
			"updatedAt": time.Now(),
			"updatedBy": code,
		},
	}

	_, err := db.UpdateOne(c, coll, filter, update)
	if err != nil {
		log.Println("Error updating password in main collection:", err)
		return err
	}

	loginColl := db.OpenCollections("LOGIN")
	_, err = db.UpdateOne(c, loginColl, filter, bson.M{
		"$set": bson.M{"password": hashedPassword},
	})

	if err != nil {
		log.Println("Warning: failed to update login collection password:", err)
	}

	return nil
}

/*
* Check if length less than 7 return error
* Must have upperCase,number,special
* If any of them gives error return error
 */
func validatePasswordRules(password string) error {

	if len(password) < 7 {
		return errors.New("password must be at least 7 characters long")
	}

	// At least one uppercase
	hasUpper := false
	// At least one number
	hasNumber := false
	// At least one special character
	hasSpecial := false

	specialChars := "!@#$%^&*()-_=+[]{}|;:',.<>?/`~"

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		case strings.ContainsRune(specialChars, ch):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	if !(hasUpper && hasNumber && hasSpecial) {
		return errors.New("password must include: one uppercase, one number, and one special character")
	}

	return nil
}

/*
* Generate a bcrypt based on the password given
 */
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

/*
* Extract token and get code and collection
* Validate the input fields given
* Validate set of rules for the password
* Hash the password
* Now update the password in the collection with the hashCode
* Update in login collection
 */
func ResetPassword(c *gin.Context, body map[string]interface{}) (string, error) {
	collection, code, err := ExtractTokenInfo(c)
	if err != nil {
		log.Println("Error from extractToken Info from resetPassword")
		return "", err
	}
	newPassword, _, err := ValidatePasswordInput(body)
	if err != nil {
		log.Println("Error from validateToken Info from resetPassword")
		return "", err
	}

	if err := validatePasswordRules(newPassword); err != nil {
		log.Println("error from validate password in validatePasswordRules")
		return "", err
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		log.Println("Error from hashedPassword")
		return "", errors.New("failed to hash new password")
	}
	log.Println(hashedPassword)

	if err := UpdatePasswordInCollections(c, collection, code, hashedPassword); err != nil {
		log.Println("Error from UpdatePasswordInCollection")
		return "", errors.New("failed to update password")
	}

	return "Password reset successful", nil
}

/*
* Check is the emailExists,phoneExists or not
* If non of these two exists then throw error
* If any of the field provided and the value is empty or type assertion then throw error
 */
func validateForgetInput(data map[string]interface{}) error {
	_, emailExists := data["email"]
	_, phoneExists := data["phoneNo"]

	if !emailExists && !phoneExists {
		return errors.New(util.PLEASE_PROVIDE_EMAIL_OR_PHONE)
	}

	if emailExists {
		err := common.GetTrimmedString(data, "email")
		if err != nil {
			log.Println("Error from the getTrimmed string:", err)
			return errors.New(util.EMAIL_NOT_PROVIDED)
		}
	}

	if phoneExists {
		err := common.GetTrimmedString(data, "phoneNo")
		if err != nil {
			log.Println("Error from the getTrimmed string:", err)
			return errors.New(util.PHONE_NUMBER_NOT_PROVIDED)
		}
	}
	return nil
}

/*
* Create Filter to find the document in db
 */
func buildForgetFilter(data map[string]interface{}) bson.M {
	filter := bson.M{}

	if v, ok := data["email"].(string); ok && v != "" {
		filter["email"] = v
	}
	if v, ok := data["phoneNo"].(string); ok && v != "" {
		filter["phoneNo"] = v
	}
	log.Println(filter)
	return filter
}

/*
* Validate input fields
* Build a filter to find in mainCollection and login collection
* Fetch login document from the login collection
* Fetch mainCollection document
* Generate otp and expiry and update the hashed otp in the login collection and main collection
* Send mail to the particular mail
 */

func ForgotPassword(c *gin.Context, data map[string]interface{}) (string, error) {

	if err := validateForgetInput(data); err != nil {
		log.Println("error from validation input for the login")
		return "", err
	}

	filter := buildForgetFilter(data)

	loginDoc, err := FetchUser(context.Background(), filter)
	if err != nil {
		log.Println("error from the fetchUser function:", err)
		return "", err
	}

	email := loginDoc["email"].(string)
	collection := loginDoc["collection"].(string)

	mainCollection := db.OpenCollections(collection)
	user := make(map[string]interface{})
	err = db.FindOne(context.Background(), mainCollection, filter, user)
	if err != nil {
		log.Println("No document found in collection")
		return "", err
	}

	// Generate OTP
	otp := common.GenerateOTP()
	log.Println(otp)
	expiry := time.Now().Add(10 * time.Minute)
	hashedPassword, err := HashPassword(otp)
	if err != nil {
		log.Println("Error from hashedPassword")
		return "", errors.New("failed to hash new password")
	}
	log.Println(hashedPassword)

	update := bson.M{
		"$set": bson.M{
			"reset":     true,
			"password":  hashedPassword,
			"otpExpiry": expiry,
		},
	}
	loginUpdate := bson.M{
		"$set": bson.M{
			"password": hashedPassword,
		},
	}
	_, err = db.UpdateOne(context.Background(), mainCollection, filter, update)
	if err != nil {
		return "", errors.New("Failed to store OTP")
	}

	loginCollection := db.OpenCollections("LOGIN")
	_, err = db.UpdateOne(context.Background(), loginCollection, filter, loginUpdate)
	if err != nil {
		return "", errors.New("Failed to store OTP")
	}

	subject := "Your forget OTP Verification"
	body := fmt.Sprintf("Hello ,\n\nYour reset OTP for verification is: %s\n\nThank you!", otp)

	err = common.SendOTPToMail(email, subject, body)
	if err != nil {
		log.Println("OTP email failed:", err)
		return "", errors.New("failed to send OTP email")
	}

	return "forget mail sent successfully", nil
}
