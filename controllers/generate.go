package controllers

import (
	"HealthHub360/services"

	"github.com/gin-gonic/gin"
)

func Report(router *gin.Engine) {
	report := router.Group("/report")
	report.GET("/fetch/:code", GenerateReport)
}
func GenerateReport(c *gin.Context) {

	code := c.Param("code")
	pdfs, err := services.GenerateReport(c, code)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{
		"message": "PDFs generated successfully",
		"files":   pdfs,
	})
}
