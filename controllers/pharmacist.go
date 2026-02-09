package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Pharmacist(router *gin.Engine) {
	pharma := router.Group("/pharmacist")
	{
		pharma.POST("/create", authorization.Authorize("pharmacist", "create"), CreatePharmacist)
		pharma.GET("/fetch/:code", authorization.Authorize("pharmacist", "view"), FetchPharmacistByCode)
		pharma.GET("/fetchAll", authorization.Authorize("pharmacist", "view"), FetchAllPharmacist)
		pharma.PATCH("/update/:code", authorization.Authorize("pharmacist", "update"), UpdatePharmacist)
		pharma.DELETE("/delete/:code", authorization.Authorize("pharmacist", "delete"), DeletePharmacistByCode)
	}
}

func CreatePharmacist(ctx *gin.Context) {
	var body map[string]interface{}
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	err = services.CreatePharmacist(ctx, body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))

}
func FetchPharmacistByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchPharmacistByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

func FetchAllPharmacist(c *gin.Context) {
	doc, err := services.FetchAllPharmacist(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}

func UpdatePharmacist(c *gin.Context) {
	var data map[string]interface{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	pharmacistId := c.Param("code")
	msg, err := services.UpdatePharmacist(c, data, pharmacistId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}

func DeletePharmacistByCode(c *gin.Context) {
	pharmacistId := c.Param("code")
	msg, err := services.DeletePharmacist(c, pharmacistId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
