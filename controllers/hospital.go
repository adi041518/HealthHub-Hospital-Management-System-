package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Hospital(router *gin.Engine) {
	hospital := router.Group("/hospital")
	{
		hospital.POST("/create", authorization.Authorize("hospital", "create"), HospitalCreate)
		hospital.PUT("/update/:code", authorization.Authorize("hospital", "update"), UpdateHospital)
		hospital.GET("/fetch/:code", authorization.Authorize("hospital", "view"), FetchHospitalByCode)
		hospital.GET("/fetchAll", authorization.Authorize("hospital", "view"), FetchAllHospital)
		hospital.DELETE("/delete/:code", authorization.Authorize("hospital", "delete"), DeleteHospitalByCode)
	}
}
func HospitalCreate(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	if err := services.CreateHospital(c, data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("created successfully"))
}
func UpdateHospital(c *gin.Context) {
	hospitalId := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	if err := services.UpdateHospital(c, data, hospitalId); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("updated successfully"))
}
func FetchHospitalByCode(c *gin.Context) {
	code := c.Param("code")
	doc, err := services.FetchHospitalByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}
func FetchAllHospital(c *gin.Context) {
	doc, err := services.FetchAllHospital(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}
func DeleteHospitalByCode(c *gin.Context) {
	code := c.Param("code")
	msg, err := services.DeleteHospitalByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
