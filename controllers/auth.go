package controllers

import (
	"HealthHub360/services"
	"log"
	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Auth(router *gin.Engine) {

	router.POST("/login", Login)
	router.POST("/reset-password", authorization.JWTAuth(), ResetPassword)
	router.POST("/forgot-password", ForgotPassword)
}

/*
* Here binding happens with the respective fields if any error return error
* And if no error moves to services
 */
func Login(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.Login(c, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

/*
* Bind the reset fields and if any error return error
* If no error, pass to the services
 */
func ResetPassword(c *gin.Context) {
	var body map[string]interface{}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	msg, err := services.ResetPassword(c, body)
	if err != nil {
		log.Println("ResetPasswordGeneric error:", err)
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

/*
* Bind the forget fields and if any error return error
* If no error, pass to the services
 */
func ForgotPassword(c *gin.Context) {

	var req map[string]interface{}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	msg, err := services.ForgotPassword(c, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
