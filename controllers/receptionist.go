package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Receptionist(router *gin.Engine) {
	recep := router.Group("/receptionist")
	{
		recep.POST("/create", authorization.Authorize("receptionist", "create"), CreateReceptionist)
		recep.GET("/fetch/:code", authorization.Authorize("receptionist", "view"), FetchReceptionistByCode)
		recep.GET("/fetchAll", authorization.Authorize("receptionist", "view"), FetchAllReceptionist)
		recep.PATCH("/update/:code", authorization.Authorize("receptionist", "update"), UpdateReceptionist)
		recep.DELETE("/delete/:code", authorization.Authorize("receptionist", "delete"), DeleteReceptionist)
	}
}

func CreateReceptionist(ctx *gin.Context) {
	var body map[string]interface{}
	err := ctx.BindJSON(&body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	err = services.CreateReceptionist(ctx, body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))

}
func FetchReceptionistByCode(c *gin.Context) {
	code := c.Param("code")
	data, err := services.FetchReceptionistByCode(c, code)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(data))
}

func FetchAllReceptionist(c *gin.Context) {
	doc, err := services.FetchAllReceptionist(c)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(doc))
}

func UpdateReceptionist(c *gin.Context) {
	receptionistId := c.Param("code")
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdateReceptionist(c, data, receptionistId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}

func DeleteReceptionist(c *gin.Context) {
	receptionistId := c.Param("code")
	msg, err := services.DeleteReceptionist(c, receptionistId)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(msg))
}
