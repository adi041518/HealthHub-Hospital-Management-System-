package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Patient(router *gin.Engine) {
	patient := router.Group("/patient")
	{
		patient.POST("/create", authorization.Authorize("patient", "create"), CreatePatient)
		patient.GET("/fetch/:patientId", authorization.Authorize("patient", "view"), FetchPatientByCode)
		patient.PATCH("/update/:patientId", authorization.Authorize("patient", "update"), UpdatePatientByCode)
		patient.GET("/fetchAll", authorization.Authorize("patient", "view"), FetchAllPatients)
		patient.DELETE("/delete/:patientId", authorization.Authorize("patient", "delete"), DeletePatient)
	}
}

func CreatePatient(c *gin.Context) {
	data := make(map[string]interface{})
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.CreatePatient(c, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}

func FetchPatientByCode(c *gin.Context) {
	patientId := c.Param("patientId")
	patient, err := services.FetchPatientByCode(c, patientId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(patient))
}

func UpdatePatientByCode(c *gin.Context) {
	patientId := c.Param("patientId")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdatePatientByCode(c, patientId, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func FetchAllPatients(c *gin.Context) {
	patients, err := services.FetchAllPatients(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(patients))
}

func DeletePatient(c *gin.Context) {
	patientId := c.Param("patientId")
	msg, err := services.DeletePatient(c, patientId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
