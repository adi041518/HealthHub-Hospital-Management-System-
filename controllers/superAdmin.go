package controllers

import (
	"HealthHub360/services"

	"log"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func SuperAdmin(router *gin.Engine) {
	superAdmin := router.Group("/superAdmin")
	{
		superAdmin.GET("/fetch/:superAdminId", authorization.Authorize("superAdmin", "view"), FetchSuperAdminByCode)
		superAdmin.PUT("/update/:superAdminId", authorization.Authorize("superAdmin", "update"), UpdateSuperAdmin)
	}
}

func CreateSuperAdmin(ctx *gin.Context) {
	var user map[string]interface{}
	err := ctx.BindJSON(&user)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	err = services.CreateSuperAdmin(ctx, user)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse("Created successfully"))
}

func FetchSuperAdminByCode(ctx *gin.Context) {
	superAdminId := ctx.Param("superAdminId")
	user, err := services.FetchSuperAdminByCode(ctx, superAdminId)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	ctx.JSON(200, util.SuccessResponse(user))
}

func UpdateSuperAdmin(ctx *gin.Context) {
	var data map[string]interface{}
	if err := ctx.BindJSON(&data); err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	superAdminId := ctx.Param("superAdminId")
	err := services.UpdateSuperAdmin(ctx, superAdminId, data)
	if err != nil {
		ctx.JSON(400, util.FailedResponse(err))
		return
	}
	log.Println("done done done ")
	ctx.JSON(200, util.SuccessResponse("Updated SUCCESSFULLY"))
}
