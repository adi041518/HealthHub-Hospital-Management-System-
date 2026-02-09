package controllers

import (
	"HealthHub360/services"
	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"
	"github.com/gin-gonic/gin"
)

func Consent(router *gin.Engine) {
	consent := router.Group("/consent")
	consent.POST("/create/:medicalRecordId", authorization.Authorize("consent", "create"), CreateConsent)
	consent.GET("/fetch/:consentId", authorization.Authorize("consent", "view"), FetchConsentByCode)
	consent.GET("/fetchAll", authorization.Authorize("consent", "view"), FetchAllConsents)
	consent.DELETE("/delete/:consentId", authorization.Authorize("consent", "delete"), DeleteConsent)
}

func CreateConsent(c *gin.Context) {
	data := make(map[string]interface{})
	if data != nil {
		if err := c.BindJSON(&data); err != nil {
			c.JSON(400, util.FailedResponse(err))
			return
		}
	}
	medicalRecordId := c.Param("medicalRecordId")
	response, err := services.CreateConsent(c, data, medicalRecordId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(response))

}

func FetchConsentByCode(c *gin.Context) {
	consentId := c.Param("consentId")
	medicine, err := services.FetchConsentByCode(c, consentId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(medicine))
}

func FetchAllConsents(c *gin.Context) {
	medicines, err := services.FetchAllConsents(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicines))
}
func DeleteConsent(c *gin.Context) {
	consentId := c.Param("consentId")
	msg, err := services.DeleteConsent(c, consentId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return

	}
	c.JSON(200, util.SuccessResponse(msg))
}
