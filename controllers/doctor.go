package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Doctor(router *gin.Engine) {
	doctor := router.Group("/doctor")
	doctor.POST("/create", authorization.Authorize("doctor", "create"), CreateDoctor)
	doctor.PUT("/update/:code", authorization.Authorize("doctor", "update"), UpdateDoctor)
	doctor.GET("/fetch/:code", authorization.Authorize("doctor", "view"), FetchDoctorByCode)
	doctor.GET("/fetchAll", authorization.Authorize("doctor", "view"), FetchAllDoctors)
	doctor.DELETE("/delete/:code", authorization.Authorize("doctor", "delete"), DeleteDoctor)
}

/*
* Bind JSON
* And Pass to the service
 */
func CreateDoctor(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateDoctor(c, data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(response))
}

/*
* Get code from params
* Bind the fields which are need to be updated
* Pass to the service
 */
func UpdateDoctor(c *gin.Context) {
	doctorId := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdateDoctor(c, data, doctorId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}

/*
* Extract code and tenantId from the context
* Pass the code and tenantId to the services
 */
func FetchDoctorByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchDoctorByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

/*
* Extract tenantId from the context
* Pass tenantId to the services
 */
func FetchAllDoctors(c *gin.Context) {
	result, err := services.FetchAllDoctors(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(result))
}

/*
* Extract code from the parameter
* Pass the code to the service
 */
func DeleteDoctor(c *gin.Context) {
	code := c.Param("code")
	data, err := services.DeleteDoctor(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
