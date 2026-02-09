package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func MedicalRecord(c *gin.Engine) {
	medicalRecord := c.Group("medicalRecord")
	{
		medicalRecord.GET("/fetch/:medicalRecordId", authorization.Authorize("medicalRecord", "view"), FetchMedicalRecordByCode)
		medicalRecord.GET("/fetchAll", authorization.Authorize("medicalRecord", "view"), FetchAllMedicalRecords)
		medicalRecord.PATCH("/update/:medicalRecordId", authorization.Authorize("medicalRecord", "update"), UpdateMedicalRecord)
		medicalRecord.DELETE("/delete/:medicalRecordId", authorization.Authorize("medicalRecord", "delete"), DeleteMedicalRecordByCode)
	}
}

func FetchMedicalRecordByCode(c *gin.Context) {
	medicalRecordId := c.Param("medicalRecordId")
	medicalRecord, err := services.FetchMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(medicalRecord))
}
func FetchAllMedicalRecords(c *gin.Context) {
	medicalRecord, err := services.FetchAllMedicalRecords(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(medicalRecord))
}

func UpdateMedicalRecord(c *gin.Context) {
	medicalRecordId := c.Param("medicalRecordId")
	var data map[string]interface{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdateMedicalRecord(c, medicalRecordId, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}

func DeleteMedicalRecordByCode(c *gin.Context) {
	medicalRecordId := c.Param("medicalRecordId")
	data, err := services.DeleteMedicalRecordByCode(c, medicalRecordId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
