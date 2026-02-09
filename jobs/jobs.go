package jobs

import (
	db "github.com/KanapuramVaishnavi/Core/config/db"
	common "github.com/KanapuramVaishnavi/Core/coreServices"

	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"go.mongodb.org/mongo-driver/bson"
)

func StartDailyScheduler() {
	c := cron.New()

	// Runs every day at 00:05 AM
	// c.AddFunc("5 0 * * *", func()
	c.AddFunc("5 0 * * *", func() {
		log.Println("Running Daily Doctor Timeslot Scheduler...")
		RunTodayScheduler()
	})

	c.Start()
}
func RunTodayScheduler() {
	today := time.Now()
	doctors := GetAllDoctors()

	for _, d := range doctors {
		doctor, ok := d.(map[string]interface{})
		if !ok {
			log.Println("Invalid doctor record:", d)
			continue
		}
		doctorId, ok := doctor["code"].(string)
		if !ok {
			log.Println("Invalid doctorId:", doctor)
			continue
		}

		hospitalId, ok := doctor["createdBy"].(string)
		if !ok {
			log.Println("Invalid hospitalId:", doctor)
			continue
		}
		err := CreateDailySlots(context.Background(), doctorId, hospitalId, today)

		if err != nil {
			log.Println("Error generating slots for doctor:", doctorId, err)
		}
	}
}

func GetAllDoctors() []interface{} {
	coll := db.OpenCollections("DOCTOR")
	docs, err := db.FindAll(context.Background(), coll, nil, nil)
	if err != nil {
		log.Println("Error from the findAll function:", err)
	}
	return docs
}
func CreateDailySlots(ctx context.Context, doctorId string, hospitalId string, date time.Time) error {

	weekday := date.Weekday().String()

	isWeeklyOff := (weekday == "Saturday" || weekday == "Sunday")
	isLeave, _ := IsDoctorOnLeave(doctorId, date)

	slots := []map[string]interface{}{}
	if !isWeeklyOff && !isLeave {
		slots = Generate30MinSlots("10:00", "18:00")
	}
	dateStr := date.Format("02-01-2006")
	dateModified, err := common.NormalizeDate(dateStr)
	if err != nil {
		log.Println("Error while normalizing the date in creating slots: ", err)
		return err
	}
	record := bson.M{
		"doctorId":    doctorId,
		"hospitalId":  hospitalId,
		"date":        dateModified,
		"day":         weekday,
		"isWeeklyOff": isWeeklyOff,
		"isLeave":     isLeave,
		"slots":       slots,
		"createdAt":   time.Now(),
	}

	coll := db.OpenCollections("DOCTOR_TIMESLOTS")
	_, err = db.CreateOne(ctx, coll, record)
	return err
}
func IsDoctorOnLeave(doctorId string, date time.Time) (bool, error) {
	leaveColl := db.OpenCollections("DOCTOR_LEAVES")

	filter := bson.M{
		"doctorId": doctorId,
		"date":     date.Format("02-01-2006"),
	}
	count, err := leaveColl.CountDocuments(context.Background(), filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
func Generate30MinSlots(start string, end string) []map[string]interface{} {
	layout := "15:04"
	startTime, _ := time.Parse(layout, start)
	endTime, _ := time.Parse(layout, end)

	slots := []map[string]interface{}{}

	for startTime.Before(endTime) {
		slotEnd := startTime.Add(30 * time.Minute)

		slots = append(slots, map[string]interface{}{
			"start":       startTime.Format(layout),
			"end":         slotEnd.Format(layout),
			"isAvailable": true,
			"isBooked":    false,
			"patientId":   "",
		})
		startTime = slotEnd
	}
	return slots
}
func SeedDoctorLeaves() {
	coll := db.OpenCollections("DOCTOR_LEAVES")

	staticLeaves := []struct {
		DoctorId string
		Date     string
	}{
		{"D0001", "30-12-2025"},
		{"D0002", "29-12-2025"},
	}

	for _, leave := range staticLeaves {

		filter := bson.M{
			"doctorId": leave.DoctorId,
			"date":     leave.Date,
		}

		count, err := coll.CountDocuments(context.Background(), filter)
		if err != nil {
			log.Println("Error checking leave:", err)
			continue
		}

		if count == 0 {
			_, err := coll.InsertOne(context.Background(), bson.M{
				"doctorId": leave.DoctorId,
				"date":     leave.Date,
			})

			if err != nil {
				log.Println("Error inserting static leave:", err)
			}
		}
	}
}
