package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func CheckForAccess(c *gin.Context, patient map[string]interface{}) error {
	pharmacistId := c.GetString("code")
	pharmacist, err := FetchPharmacistByCode(c, pharmacistId)
	if err != nil {
		log.Println("Error from fetchPatientByCode: ", err)
		return err
	}
	pharmacistHosId, ok := pharmacist["createdBy"].(string)
	if !ok {
		log.Println("Unable to fetch createdBy from patient ")
		return errors.New(util.UNABLE_TO_FETCH_CREATED_BY_FROM_PATIENT)
	}
	hospitalIdFromPatient, ok := patient["hospitalId"].(string)
	if !ok {
		log.Println("Unable to fetch hospitalId field from patient")
		return errors.New(util.UNABLE_TO_FETCH_HOSPITAL_ID_FROM_PATIENT)
	}
	if pharmacistHosId != hospitalIdFromPatient {
		log.Println("This pharmacist doesnot have access")
		return errors.New(util.PHARMACIST_DOESNOT_HAVE_ACCESS_TO_BILL)
	}
	return nil
}
func GetLatestAppointmentIDFromPatient(patient map[string]interface{}) (string, error) {
	rawApp, ok := patient["appointments"]
	if !ok || rawApp == nil {
		log.Println("Unable to find key Appointments from patient")
		return "", errors.New(util.UNABLE_TO_FIND_APPOINTMENTS_IN_PATIENT)
	}
	log.Printf("appointments type : %T", rawApp)
	var app []interface{}
	switch v := rawApp.(type) {
	case primitive.A:
		app = []interface{}(v)
	case []interface{}:
		app = v
	default:
		return "", errors.New(util.UNSUPPORTED_APPOINTMENTS_TYPE)
	}
	if len(app) == 0 {
		return "", errors.New(util.APPOINTMENT_FIELD_IS_EMPTY)
	}

	appointmentId, ok := app[len(app)-1].(string)
	if !ok {
		return "", errors.New(util.UNABLE_TO_FETCH_LATEST_APPOINTMENT)
	}
	return appointmentId, nil
}
func FetchTestsFromMedicalRecord(medicalRecord map[string]interface{}) ([]string, error) {
	var tests []string
	rawTests, exists := medicalRecord["testList"]
	if !exists || rawTests == nil {
		log.Println("No testList found in medicalRecord")
		return nil, errors.New(util.UNABLE_TO_FIND_TEST_LISTS_IN_MEDICAL_RECORD)
	}
	log.Printf("The type of rawTests: %T ", rawTests)
	var testList []interface{}

	switch v := rawTests.(type) {
	case []interface{}:
		testList = v
	case primitive.A:
		testList = []interface{}(v)
	default:
		return nil, errors.New(util.UNSUPPORTED_TEST_LIST)
	}
	for _, t := range testList {
		val, ok := t.(string)
		if !ok {
			log.Println("Unable to fetch test from testList")
			return nil, errors.New(util.UNABLE_TO_FETCH_TEST_FROM_TEST_LIST)
		}
		tests = append(tests, val)
	}
	return tests, nil
}
func GenerateBillForTests(c *gin.Context, medicalRecord map[string]interface{}) ([]map[string]interface{}, int, error) {

	tests, err := FetchTestsFromMedicalRecord(medicalRecord)
	if err != nil {
		log.Println("Error from fetchTestsFromMedicalRecord: ", err)
		return nil, 0, err
	}
	log.Println("tests: ", tests)
	var billTests []map[string]interface{}
	var incTestPrice int
	for _, t := range tests {
		test, err := FetchTestByCode(c, t)
		if err != nil {
			log.Println("Error from fetchTestByCode: ", err)
			return nil, 0, err
		}
		billTest := make(map[string]interface{})
		priceVal, ok := test["price"].(string)
		if !ok {
			log.Println("Unable to get price from single test")
			return nil, 0, errors.New(util.UNABLE_TO_FETCH_PRICE_FROM_TEST)
		}
		price, _ := strconv.Atoi(priceVal)
		billTest["testId"] = t
		billTest["price"] = priceVal
		incTestPrice = incTestPrice + price
		billTests = append(billTests, billTest)
	}
	return billTests, incTestPrice, nil
}
func FetchPrescriptionIdFromMedicalRecord(medicalRecord map[string]interface{}) (string, error) {
	prescriptionId, ok := medicalRecord["prescriptionId"].(string)
	if !ok {
		log.Println("Unable to fetch prescription field from medicalRecord")
		return "", errors.New(util.UNABLE_TO_FETCH_PRESCRIPTION_FROM_MEDICAL_RECORD)
	}
	return prescriptionId, nil
}
func ExtractMedicines(prescription map[string]interface{}) ([]interface{}, error) {
	medicineRaw, ok := prescription["medicines"]
	if !ok {
		log.Println("Unable to fetch medicines from medicineRaw")
		return nil, errors.New(util.UNABLE_TO_FETCH_MEDICINES_FROM_PRESCRIPTION)
	}
	var medicines []interface{}
	switch v := medicineRaw.(type) {
	case primitive.A:
		medicines = []interface{}(v)
	case []interface{}:
		medicines = v
	default:
		return nil, errors.New(util.UNSUPPORTED_MEDICINE_TYPE_FROM_PRESCRIPTION)
	}
	return medicines, nil

}
func FetchFieldsFromMedicine(c *gin.Context, medicineId string) (int, int, int, error) {
	var val int
	medicineFetched, err := FetchMedicineByCode(c, medicineId)
	if err != nil {
		log.Println("Error from fetchMedicineByCode: ", err)
		return val, val, val, err
	}

	pricePerStripVal, ok := medicineFetched["pricePerStrip"].(string)
	if !ok {
		log.Println("Unable to fetch pricePerStrip from medicineFetched")
		return val, val, val, errors.New(util.UNABLE_TO_FETCH_PRICE_PER_STRIP)
	}
	pricePerStrip, _ := strconv.Atoi(pricePerStripVal)

	log.Printf("tabletsPerStip %T", medicineFetched["tabletsPerStrip"])
	tabletsPerStripVal, ok := medicineFetched["tabletsPerStrip"]
	if !ok {
		log.Println("unable to fetch tabletsPerStrip")
		return val, val, val, errors.New(util.UNABLE_TO_FETCH_TABLETS_PER_STRIP)
	}
	tabletsPerStripInt, ok := tabletsPerStripVal.(string)
	if !ok {
		log.Println("Type assertion from tabletsPerStrip")
		return val, val, val, errors.New(util.TABLETS_PER_STRIP_MUST_BE_VALID_TYPE)
	}
	tabletsPerStrip, _ := strconv.Atoi(tabletsPerStripInt)
	log.Println("tabletsPerStrip ", tabletsPerStrip)

	totalNoOfTabletsVal, ok := medicineFetched["totalNoOfTablets"]
	if !ok {
		log.Println("unable to fetch totalNoOfTablets")
		return val, val, val, errors.New(util.UNABLE_TO_FETCH_TOTAL_NO_OF_TABLETS)
	}
	totalNoOfTabletsInt, ok := totalNoOfTabletsVal.(string)
	if !ok {
		log.Println("Type assertion from totalNoOfTablets")
		return val, val, val, errors.New(util.TOTAL_NO_OF_TABLETS_MUST_BE_VALID_TYPE)
	}
	totalNoOfTablets, _ := strconv.Atoi(totalNoOfTabletsInt)
	log.Println("totalNoOfTablets: ", totalNoOfTablets)
	return pricePerStrip, tabletsPerStrip, totalNoOfTablets, nil
}
func FetchMedicineFieldsFromPrescription(medicine map[string]interface{}) (string, int, error) {

	medicineId, ok := medicine["medicineId"].(string)
	if !ok {
		log.Println("Unable to fetch medicineId or type assertion")
		return "", 0, errors.New("Unable to fetch medicineId")
	}

	dosagePerFrequencyVal, ok := medicine["dosagePerFrequency"].(string)
	if !ok {
		log.Println("Unable to fetch dosagePerFrequency or type assertion")
		return "", 0, errors.New(util.UNABLE_TO_FETCH_DOSAGE_PER_FREQUENCY)
	}
	dosagePerFrequency, _ := strconv.Atoi(dosagePerFrequencyVal)

	noOfDaysVal, ok := medicine["noOfDays"].(string)
	if !ok {
		log.Println("Unable to fetch noOfDays")
		return "", 0, errors.New(util.UNABLE_TO_FETCH_NO_OF_DAYS)
	}
	noOfDays, _ := strconv.Atoi(noOfDaysVal)

	log.Printf("frequncy type %T", medicine["frequency"])
	freq := medicine["frequency"].(map[string]interface{})
	timesPerDay := 0
	if freq["morning"].(bool) {
		timesPerDay++
	}
	if freq["afternoon"].(bool) {
		timesPerDay++
	}
	if freq["night"].(bool) {
		timesPerDay++
	}
	totalTablets := dosagePerFrequency * timesPerDay * noOfDays
	return medicineId, totalTablets, nil
}

// fetching precription to dispense medines for the patinets
func fetchPrescriptionFromMedicalRecord(
	c *gin.Context,
	medicalRecord map[string]interface{},
) (map[string]interface{}, error) {

	prescriptionId, err := FetchPrescriptionIdFromMedicalRecord(medicalRecord)
	if err != nil {
		log.Println("Error fetching prescription ID:", err)
		return nil, err
	}

	prescription, err := FetchPrescriptionByCode(c, prescriptionId)
	if err != nil {
		log.Println("Error fetching prescription:", err)
		return nil, err
	}

	return prescription, nil
}

// calcuate and updates the medicine stock if we needed and give the data we want update the medicines after dispenesed
func calculateAndUpdateMedicine(
	c *gin.Context,
	medicineId string,
	requiredTablets int,
	pricePerStrip int,
	tabletsPerStrip int,
	availableTablets int,
) (map[string]interface{}, int, error) {

	singleMedicine := make(map[string]interface{})
	costPerTablet := pricePerStrip / tabletsPerStrip
	remainingTablets := availableTablets - requiredTablets

	singleMedicine["medicineId"] = medicineId
	singleMedicine["requiredTablets"] = strconv.Itoa(requiredTablets)
	singleMedicine["costPerTablet"] = strconv.Itoa(costPerTablet)
	singleMedicine["totalNoOfTablets"] = strconv.Itoa(availableTablets)

	if remainingTablets < 0 {
		singleMedicine["isDispensed"] = false
		singleMedicine["pricePerMedicine"] = "0"
		return singleMedicine, 0, nil
	}

	price := requiredTablets * costPerTablet
	singleMedicine["isDispensed"] = true
	singleMedicine["pricePerMedicine"] = strconv.Itoa(price)

	if err := updateMedicineStock(
		c,
		medicineId,
		remainingTablets,
		tabletsPerStrip,
	); err != nil {
		return nil, 0, err
	}

	return singleMedicine, price, nil
}

// medicne stock after dispensed gets updated here
func updateMedicineStock(
	c *gin.Context,
	medicineId string,
	remainingTablets int,
	tabletsPerStrip int,
) error {

	update := map[string]interface{}{
		"noOfstrips":       strconv.Itoa(remainingTablets / tabletsPerStrip),
		"totalNoOfTablets": strconv.Itoa(remainingTablets),
	}

	_, err := UpdateMedicines(c, medicineId, update)
	if err != nil {
		log.Println("Unable to update medicine stock:", err)
		return errors.New("unable to update totalNoOfTablets")
	}

	return nil
}

// here it calcualtes and update each and evry single medicine we mentioned in the medical record (Prescription one)
func processSingleMedicine(
	c *gin.Context,
	m interface{},
) (map[string]interface{}, int, error) {

	medicine, ok := m.(map[string]interface{})
	if !ok {
		return nil, 0, errors.New(util.UNABLE_TO_FETCH_MEDICINE_FROM_MEDICINE)
	}

	medicineId, requiredTablets, err := FetchMedicineFieldsFromPrescription(medicine)
	if err != nil {
		return nil, 0, err
	}

	pricePerStrip, tabletsPerStrip, totalTablets, err :=
		FetchFieldsFromMedicine(c, medicineId)
	if err != nil {
		return nil, 0, err
	}

	return calculateAndUpdateMedicine(
		c,
		medicineId,
		requiredTablets,
		pricePerStrip,
		tabletsPerStrip,
		totalTablets,
	)
}

func GenerateBillForMedicines(
	c *gin.Context,
	medicalRecord map[string]interface{},
) ([]map[string]interface{}, int, error) {

	prescription, err := fetchPrescriptionFromMedicalRecord(c, medicalRecord)
	if err != nil {
		return nil, 0, err
	}

	medicines, err := ExtractMedicines(prescription)
	if err != nil {
		return nil, 0, err
	}

	var (
		billMedicines   []map[string]interface{}
		totalBillAmount int
	)

	for _, m := range medicines {
		item, price, err := processSingleMedicine(c, m)
		if err != nil {
			return nil, 0, err
		}

		billMedicines = append(billMedicines, item)
		totalBillAmount += price
	}

	return billMedicines, totalBillAmount, nil
}

/*
* Validate user inputs first
* Verify whether the pharmacist can create bill for that medicalRecord
* Check the fields and Generate a code and then createdBy
* Fetch tenantId from context
* Include tenantId
* Combine all the remaining data and prepare it
* Get tests and prescriptionId from the medicalRecord
* Generate a bill of cost per medicines and cost per tests and return the amount for all of them
* Save to db and cache
 */
func CreateBill(c *gin.Context, patientId string) (string, error) {
	patient, err := FetchPatientByCode(c, patientId)
	if err != nil {
		log.Println("Error from fetchPatientByCode: ", err)
		return "", err
	}
	err = CheckForAccess(c, patient)
	if err != nil {
		log.Println("Error from CheckForAccess: ", err)
		return "", err
	}

	latestApp, err := GetLatestAppointmentIDFromPatient(patient)
	if err != nil {
		log.Println("Error from getLatestAppointmentIDFromPatient: ", err)
		return "", err
	}
	log.Println("latestAppId: ", latestApp)
	appointment, err := FetchAppointmentByCode(c, latestApp)
	if err != nil {
		log.Println("Error from fetchAppointmentByCode: ", err)
		return "", err
	}
	medicalId, exists := appointment["medicalId"].(string)
	if !exists {
		log.Println("Unable to fetch medicalId from appointment")
		return "", errors.New(util.UNABLE_TO_FETCH_MEDICAL_ID_FROM_APPOINTMENT)
	}
	medicalRecord, err := FetchMedicalRecordByCode(c, medicalId)
	if err != nil {
		log.Println("Error from fetchMedicalRecordByCode: ", err)
		return "", err
	}
	billTests, incTestPrice, err := GenerateBillForTests(c, medicalRecord)
	if err != nil {
		log.Println("Error from generateBillForTests: ", err)
		return "", err
	}
	billMedicines, incMedicinePrice, err := GenerateBillForMedicines(c, medicalRecord)
	if err != nil {
		log.Println("Error from generateBillFromMedicines: ", err)
		return "", err
	}
	bill := bson.M{}
	bill["medicines"] = billMedicines
	bill["tests"] = billTests
	bill["amountForTests"] = strconv.Itoa(incTestPrice)
	bill["amountForMedicine"] = strconv.Itoa(incMedicinePrice)
	bill["amount"] = strconv.Itoa(incMedicinePrice + incTestPrice)
	code, err := common.GenerateEmpCode(util.BillCollection)
	if err != nil {
		log.Println("Error from generateEmpCode: ", err)
		return "", err
	}
	bill["code"] = code
	updMedicalRecord := make(map[string]interface{})
	updMedicalRecord["billId"] = code
	_, err = UpdateMedicalRecord(c, medicalId, updMedicalRecord)
	if err != nil {
		log.Println("Error from updateMedicalRecord: ", err)
		return "", err
	}

	pharmacistId, err := common.GetFromContext[string](c, "code")
	if err != nil {
		log.Println("Error from GetFromContext: ", err)
		return "", err
	}
	bill["tenantId"] = medicalRecord["tenantId"].(string)
	bill["hospitalId"] = medicalRecord["hospitalId"].(string)
	bill["prescripitonId"] = medicalRecord["prescriptionId"].(string)
	bill["medicalId"] = medicalId
	bill["patientId"] = patientId
	bill["createdBy"] = pharmacistId
	bill["updatedBy"] = pharmacistId
	bill["createdAt"] = time.Now()
	bill["updatedAt"] = time.Now()
	collection := db.OpenCollections(util.BillCollection)
	inserted, err := db.CreateOne(c, collection, bill)
	if err != nil {
		log.Println("Error from createOne: ", err)
		return "", err
	}
	log.Println("inserted: ", inserted.InsertedID)
	key := util.BillKey + code
	err = redis.SetCache(c, key, bill)
	if err != nil {
		log.Println("Error while setting cache")
	}
	return "created successfully", nil
}

/*
* Get bill for the given billId
* Get tenantId,code,collection,isSuperAdmin from the context
* Check who can access
* Fetch from access, based on the accessibility
* if exists return
* If not exists fetch from database
* Return from database and set in cache
 */
func FetchBillByCode(c *gin.Context, billId string) (map[string]interface{}, error) {
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

	key := util.BillKey + billId
	if cached, exists, err := common.CheckCacheAccess(c, key, collFromContext, userData, tenantId, code, isSuperAdmin); exists {
		return cached, err
	}
	coll := db.OpenCollections(util.BillCollection)
	filter := bson.M{"code": billId}
	result := make(map[string]interface{})

	err = db.FindOne(c, coll, filter, &result)
	if err != nil {
		log.Println("Error from findOne: ", err)
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

func BuildBillingData(c *gin.Context, patient map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	patientName := patient["name"].(string)
	patientID := patient["code"].(string)
	patientemail := patient["email"].(string)
	age := toInt(patient["age"])
	gender := patient["gender"].(string)
	admissionDate := getString(patient["admissionDate"])
	phone := getString(patient["phoneNo"])

	rawIDs, ok := patient["appointments"].([]interface{})
	if !ok || len(rawIDs) == 0 {
		return nil, errors.New("no appointment IDs found")
	}

	appointmentID := rawIDs[len(rawIDs)-1].(string)
	appointment, err := FetchAppointmentByCode(c, appointmentID)
	if err != nil {
		return nil, err
	}
	//   appointment["isPr"]

	hospital, err := FetchHospitalByCode(c, appointment["hospitalId"].(string))
	if err != nil {
		return nil, err
	}
	//medicalRecordFetching

	medicalRecord, err := FetchMedicalRecordByCode(c, appointment["medicalId"].(string))
	if err != nil {
		return nil, err
	}
	billingRecord, err := FetchBillByCode(c, medicalRecord["billId"].(string))
	if err != nil {
		return nil, err
	}
	totalTestsVal := billingRecord["amountForTests"]
	totalmedicineBillVal := billingRecord["amountForMedicine"]

	totaltestbill, err := strconv.Atoi(totalTestsVal.(string))
	if err != nil {
		return nil, err
	}

	totalmedbill, err := strconv.Atoi(totalmedicineBillVal.(string))
	if err != nil {
		return nil, err
	}

	grandTotal := totaltestbill + totalmedbill

	logo, _ := ImageToBase64("https://healthhub360.s3.ap-southeast-2.amazonaws.com/smalllogo.jpg")
	qr, _ := ImageToBase64("https://healthhub360.s3.ap-southeast-2.amazonaws.com/qrcode.png")
	upiId := "paytmqr5r0hgo@ptys"
	name := "Kadambala Aditya"
	upistring := BuildUPIString(upiId, name, grandTotal)

	qrCode, _ := GenerateQRCode(upistring)

	link, err := CreateRazorpayPaymentLink(grandTotal, patientName, patientemail, phone)
	if err != nil {
		return nil, err
	}
	url, err := GenerateQRCode(link)
	if err != nil {
		return nil, err
	}
	result["HospitalLogo"] = template.URL(logo)
	result["Barcode"] = template.URL(qr)

	result["HospitalName"] = hospital["name"]
	result["HospitalAddress"] = hospital["address"]
	result["HospitalContact"] = hospital["phoneNo"]

	result["PatientName"] = patientName
	result["PatientID"] = patientID
	result["Age"] = age
	result["Gender"] = gender
	result["Phone"] = phone
	result["AdmissionDate"] = admissionDate

	result["totalTests"] = totaltestbill
	result["totalMeds"] = totalmedbill
	result["grandTotal"] = grandTotal

	result["AccountNumber"] = "41330117270"
	result["IFSC"] = "SBIN0011224"
	result["Bank"] = "STATE BANK OF INDIA"
	result["QRLink"] = template.URL(url)
	result["DynamicQRLink"] = template.URL(qrCode)
	return result, nil
}

func GenerateBillingPDF(data map[string]interface{}, htmlPath string, pdfPath string) error {
	tmpl, err := template.ParseFiles("./templates/billing.html")
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.WriteFile(htmlPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	cmd := exec.Command(
		"wkhtmltopdf",
		"--enable-local-file-access",
		"--load-error-handling", "ignore",
		htmlPath,
		pdfPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}

	return nil
}

/*
The Main Function starts here
*/
func GenerateBillingReport(c *gin.Context, patientCode string) ([]string, error) {
	patient, err := FetchPatientByCode(c, patientCode)
	if err != nil {
		return nil, err
	}

	data, err := BuildBillingData(c, patient)
	if err != nil {
		return nil, err
	}

	name := patient["name"].(string)

	htmlPath := fmt.Sprintf("bill_%s.html", patientCode)
	pdfPath := fmt.Sprintf("%s_bill.pdf", name)

	err = GenerateBillingPDF(data, htmlPath, pdfPath)
	if err != nil {
		return nil, err
	}

	return []string{pdfPath}, nil
}
func CreateRazorpayPaymentLink(amount int, name, email, phone string) (string, error) {

	// Razorpay Test Credentials (replace with your LIVE keys in production)
	key := "rzp_test_Rp2NNdbZg1oaD3"
	secret := "gIgHjpFH3Dag1PR97gXyGtOb"

	url := "https://api.razorpay.com/v1/payment_links"

	payload := map[string]interface{}{
		"amount":      amount * 100, // Razorpay expects paise
		"currency":    "INR",
		"description": fmt.Sprintf("Hospital Bill Payment for %s", name),

		"customer": map[string]interface{}{
			"name":    name,
			"email":   email,
			"contact": phone,
		},

		"notify": map[string]bool{
			"sms":   true,
			"email": true,
		},

		"reminder_enable": true,
		"callback_method": "get",
	}

	// Convert payload to JSON
	body, _ := json.Marshal(payload)

	// Create request
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.SetBasicAuth(key, secret)
	req.Header.Set("Content-Type", "application/json")

	//Perform request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	// Read response
	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	fmt.Println("Razorpay Response:", result)

	// Extract short_url
	if link, ok := result["short_url"].(string); ok {
		return link, nil
	}

	// Razorpay error message
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		return "", fmt.Errorf("razorpay error: %v", errObj["description"])
	}

	return "", errors.New("unknown razorpay error, no short_url returned")
}

func GenerateQRCode(data string) (string, error) {
	qrURL := "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=" + data

	resp, err := http.Get(qrURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	qrBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	base64QR := base64.StdEncoding.EncodeToString(qrBytes)

	return "data:image/png;base64," + base64QR, nil
}
func BuildUPIString(upiID, name string, amount int) string {
	encodedName := url.QueryEscape(name)

	return fmt.Sprintf(
		"upi://pay?pa=%s&pn=%s&am=%d&cu=INR&tn=Hospital+Bill",
		upiID,
		encodedName,
		amount,
	)
}

func DeleteBillByCode(c *gin.Context, billId string) (string, error) {
	collection := db.OpenCollections(util.BillCollection)
	pharmacistId, ok := c.Get("code")
	if !ok {
		return "", errors.New("unable to fetch code from context")
	}
	filter := bson.M{
		"code": billId,
	}

	log.Println(filter)
	result := make(map[string]interface{})
	err := db.FindOne(c, collection, filter, &result)
	if err != nil {
		log.Println("Error from the findOne function:", err)
		return "", err

	}
	if pharmacistId.(string) != result["createdBy"].(string) {
		log.Println("This user doesnot have access")
		return "", errors.New("This user doesnot have access")
	}
	deleted, err := db.DeleteOne(c, collection, filter)
	if err != nil {
		log.Println("Error from the deleteOne function: ", err)
		return "", err
	}
	log.Println("Deleted: ", deleted.DeletedCount)
	key := util.BillKey + billId
	err = redis.DeleteCache(c, key)
	if err != nil {
		log.Println("Error from deleteCache:", err)
		return "", err
	}
	msg := fmt.Sprintf("User %s deleted successfuly ", billId)
	return msg, nil
}
