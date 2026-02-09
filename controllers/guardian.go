package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Guardian(router *gin.Engine) {
	guardian := router.Group("/guardian")
	{
		guardian.PATCH("/update/:guardianId", authorization.Authorize("guardian", "update"), UpdateGuardianByCode)
		guardian.GET("/fetch/:guardianId", authorization.Authorize("guardian", "view"), FetchGuardianByCode)
		guardian.GET("/fetchAll", authorization.Authorize("guardian", "view"), FetchAllGuardians)
		guardian.DELETE("/delete/:guardianId", authorization.Authorize("guardian", "delete"), DeleteGuardian)
	}
}

func UpdateGuardianByCode(c *gin.Context) {
	guardianId := c.Param("guardianId")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.UpdateGuardianByCode(c, guardianId, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
func FetchGuardianByCode(c *gin.Context) {
	guardianId := c.Param("guardianId")
	guardian, err := services.FetchGuardianByCode(c, guardianId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(guardian))
}

func FetchAllGuardians(c *gin.Context) {
	guardians, err := services.FetchAllGuardians(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(guardians))
}

func DeleteGuardian(c *gin.Context) {
	guardianId := c.Param("guardianId")
	msg, err := services.DeleteGuardian(c, guardianId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
