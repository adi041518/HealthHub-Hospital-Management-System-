package controllers

import (
	"HealthHub360/services"
	"log"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Bill(router *gin.Engine) {
	bill := router.Group("/bill")
	bill.POST("/create/:code", authorization.Authorize("bill", "create"), CreateBill)
	bill.GET("/fetch/:code", authorization.Authorize("bill", "view"), FetchBillByCode)
	bill.GET("/generate/:patientId", GenerateBillingReport)
	bill.DELETE("/delete/:billId", authorization.Authorize("bill", "delete"), DeleteBillByCode)
}
func CreateBill(c *gin.Context) {
	patientId := c.Param("code")
	msg, err := services.CreateBill(c, patientId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
func FetchBillByCode(c *gin.Context) {
	billId := c.Param("code")
	result, err := services.FetchBillByCode(c, billId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(result))
}

/*
* Extract code and tenantId from the context
* Pass the code and tenantId to the services
 */
func GenerateBillingReport(c *gin.Context) {
	patientId := c.Param("patientId")
	response, err := services.GenerateBillingReport(c, patientId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println(response)
	c.JSON(200, util.SuccessResponse(response))
}

func DeleteBillByCode(c *gin.Context) {
	billId := c.Param("billId")
	data, err := services.DeleteBillByCode(c, billId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}
