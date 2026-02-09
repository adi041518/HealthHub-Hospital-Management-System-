package controllers

import (
	"HealthHub360/services"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Test(router *gin.Engine) {
	test := router.Group("/test")
	{
		test.POST("/create", authorization.Authorize("test", "create"), Createtest)
		test.PUT("/update/:code", authorization.Authorize("test", "update"), Updatetest)
		test.GET("/fetch/:code", authorization.Authorize("test", "view"), FetchtestByCode)
		test.GET("/fetchAll/:tenantId", authorization.Authorize("test", "view"), FetchAlltests)
		test.DELETE("/delete/:code", authorization.Authorize("test", "delete"), Deletetest)
	}
}

/*
* Bind JSON
* And Pass to the service
 */
func Createtest(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	response, err := services.CreateTest(c, data)
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
func Updatetest(c *gin.Context) {
	code := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	if err := services.UpdateTest(c, data, code); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("updated successfully"))
}

/*
* Extract code and tenantId from the context
* Pass the code and tenantId to the services
 */
func FetchtestByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchTestByCode(c, code)
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
func FetchAlltests(c *gin.Context) {
	tenantId := c.Param("tenantId")
	result, err := services.FetchAllTests(c, tenantId)
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
func Deletetest(c *gin.Context) {
	code := c.Param("code")
	data, err := services.DeleteTest(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
