package controllers

import (
	"HealthHub360/services"

	"log"
	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Role(router *gin.Engine) {
	role := router.Group("/role")
	{
		role.POST("create", authorization.Authorize("role", "create"), CreateRole)
		role.POST("/update/:roleCode", authorization.Authorize("role", "update"), UpdateRole)
		role.GET("/fetch/:roleCode", authorization.Authorize("role", "view"), FetchRoleById)
		role.DELETE("/delete/:roleCode", authorization.Authorize("role", "delete"), DeleteRole)
	}
}

/*
* Take the json format
* Pass to prepareData
* Move to services with the parameter context and map[string]interface
 */
func CreateRole(c *gin.Context) {

	var roleData map[string]interface{}

	if err := c.BindJSON(&roleData); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	insertedRole, err := services.CreateRole(c, roleData)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}

	c.JSON(http.StatusOK, util.SuccessResponse(insertedRole))
}

/*
Here It reads all the roles of the user
*/
func ReadRoles(c *gin.Context) {
	data, err := services.ReadRoles(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(data))
}

/*
Here it updates the role of the role by taking the code from param
*/
func UpdateRole(c *gin.Context) {
	roleCode := c.Param("roleCode")

	var body map[string]interface{}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}

	updated, err := services.UpdateRole(c, roleCode, body)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}

	c.JSON(200, util.SuccessResponse(updated))
}

/*
Fetch role by its id using rolecode given in param
*/
func FetchRoleById(c *gin.Context) {
	roleCode := c.Param("roleCode")
	var body map[string]interface{}
	body, err := services.FetchRoleById(c, roleCode)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse(body))
}

/*
Here it deletes the role of bty taking the roleid as a
param and perform the delete operation
*/
func DeleteRole(c *gin.Context) {
	log.Println("Adi")
	roleCode := c.Param("roleCode")
	err := services.DeleteRole(c, roleCode)
	if err != nil {
		c.JSON(400, util.FailedResponse(err))
		return
	}
	c.JSON(200, util.SuccessResponse("Deleted successfully"))
}
