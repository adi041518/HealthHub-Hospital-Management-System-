package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Nurse(router *gin.Engine) {
	nurse := router.Group("/nurse")
	{
		nurse.POST("/create", authorization.Authorize("nurse", "create"), CreateNurse)
		nurse.PUT("/update/:code", authorization.Authorize("nurse", "update"), UpdateNurse)
		nurse.GET("/fetch/:code", authorization.Authorize("nurse", "view"), FetchNurseByCode)
		nurse.GET("/fetchAll", authorization.Authorize("nurse", "view"), FetchAllNurses)
		nurse.DELETE("/delete/code", authorization.Authorize("nurse", "delete"), DeleteNurseByCode)
	}
}

/*
* Bind JSON
* And Pass to the service
 */
func CreateNurse(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateNurse(c, data)
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
func UpdateNurse(c *gin.Context) {
	code := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdateNurse(c, data, code)
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
func FetchNurseByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchNurseByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

/*
This will help to fetch all the nurses of the tenant of the param given
*/
func FetchAllNurses(c *gin.Context) {
	doc, err := services.FetchAllNurses(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}

/*
This will help to delete  the specific nurse of the code in the nurse collection
*/
func DeleteNurseByCode(c *gin.Context) {
	nurseid := c.Param("code")
	msg, err := services.DeleteNurseByCode(c, nurseid)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
