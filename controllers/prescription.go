package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Prescription(router *gin.Engine) {
	prescription := router.Group("/prescription")
	{
		prescription.POST("/create/:medicalRecordId", authorization.Authorize("prescription", "create"), CreatePrescription)
		prescription.GET("/fetch/:prescriptionId", authorization.Authorize("prescription", "view"), FetchPrescriptionByCode)
		prescription.GET("/fetchAll", authorization.Authorize("prescription", "view"), FetchAllPrescriptions)
		prescription.PATCH("/update/:prescriptionId/:medicineId", authorization.Authorize("prescription", "update"), UpdatePrescription)
		prescription.DELETE("/delete/:prescriptionId", authorization.Authorize("prescription", "delete"), DeletePrescriptionByCode)
	}
}
func CreatePrescription(c *gin.Context) {
	medicalRecordId := c.Param("medicalRecordId")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.CreatePrescription(c, data, medicalRecordId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func FetchPrescriptionByCode(c *gin.Context) {
	prescriptionId := c.Param("prescriptionId")
	prescription, err := services.FetchPrescriptionByCode(c, prescriptionId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(prescription))
}

func FetchAllPrescriptions(c *gin.Context) {
	prescriptions, err := services.FetchAllPresciptions(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(prescriptions))
}

func UpdatePrescription(c *gin.Context) {
	prescriptionId := c.Param("prescriptionId")
	medicineId := c.Param("medicineId")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdatePrescription(c, prescriptionId, medicineId, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func DeletePrescriptionByCode(c *gin.Context) {
	prescripitonId := c.Param("prescriptionId")
	data, err := services.DeletePrescriptionByCode(c, prescripitonId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
