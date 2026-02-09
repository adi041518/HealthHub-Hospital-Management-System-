package main

import (
	"HealthHub360/jobs"
	"HealthHub360/routes"
	"log"

	server "github.com/KanapuramVaishnavi/Core/server"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	startServer = server.Start
	isTest      = false
)

func main() {
	run()
}

func run() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error in loading the ENV")
	}

	defaultopts := server.GetDefaultOptions()

	options := server.Options{
		CacheEnabled:     defaultopts.CacheEnabled,
		MongoEnabled:     defaultopts.MongoEnabled,
		WebServerEnabled: defaultopts.WebServerEnabled,
		WebServerPort:    defaultopts.WebServerPort,

		JobsEnabled: !isTest,
		JobsHandler: func() {
			if isTest {
				return
			}
			jobs.SeedDoctorLeaves()
			jobs.StartDailyScheduler()
		},

		WebServerPreHandler: func(r *gin.Engine) {
			if isTest {
				return
			}
			r.Use(cors.New(cors.Config{
				AllowOrigins:     []string{"*"},
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
				AllowCredentials: true,
			}))
			routes.Routes(r)
		},

		//MigrationEnabled: !isTest,
		/*MigrationHandler: func() {
			if isTest {
				return
			}
			migrations.AddPharmacistIdField()
			migrations.ChangeLoginAttemptsType()
			migrations.RemovePharamcistIdFromBill()
			migrations.UpdateLoginAttemptsInHospitalAdmin()
		},*/
	}
	startServer(options)
}
