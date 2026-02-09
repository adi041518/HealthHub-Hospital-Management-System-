package controllers

import (
	"HealthHub360/services"

	"net/http"

	authorization "github.com/KanapuramVaishnavi/Core/config/authorization"
	util "github.com/KanapuramVaishnavi/Core/util"

	"github.com/gin-gonic/gin"
)

func Medicines(router *gin.Engine) {
	medicines := router.Group("/medicines")
	{
		medicines.POST("/create", authorization.Authorize("medicine", "create"), CreateMedicines)
		medicines.GET("/fetch/:medicineCode", authorization.Authorize("medicine", "view"), FetchMedicineByCode)
		medicines.GET("/fetchAll", authorization.Authorize("medicine", "view"), FetchAllMedicines)
		medicines.PATCH("/update/:medicineCode", authorization.Authorize("medicine", "update"), UpdateMedicines)
		medicines.DELETE("/delete/:medicineCode", authorization.Authorize("medicine", "delete"), DeleteMedicine)
	}
}
func CreateMedicines(c *gin.Context) {
	var data map[string]interface{}
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	msg, err := services.CreateMedicines(c, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func FetchMedicineByCode(c *gin.Context) {
	medicineId := c.Param("medicineCode")
	medicine, err := services.FetchMedicineByCode(c, medicineId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicine))
}

func FetchAllMedicines(c *gin.Context) {
	medicines, err := services.FetchAllMedicines(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return
	}
	c.JSON(http.StatusOK, util.SuccessResponse(medicines))
}
func UpdateMedicines(c *gin.Context) {
	medicineId := c.Param("medicineCode")
	data := make(map[string]interface{})
	if err := c.BindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
	}
	msg, err := services.UpdateMedicines(c, medicineId, data)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return

	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}

func DeleteMedicine(c *gin.Context) {
	medicineId := c.Param("medicineCode")
	msg, err := services.DeleteMedicine(c, medicineId)
	if err != nil {
		c.JSON(http.StatusBadRequest, util.FailedResponse(err))
		return

	}
	c.JSON(http.StatusOK, util.SuccessResponse(msg))
}
