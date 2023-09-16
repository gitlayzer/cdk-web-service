package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

var (
	ServiceMode = os.Getenv("GIN_MODE")
)

func init() {
	if ServiceMode == "" {
		ServiceMode = gin.DebugMode
	}
}

func Handler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"msg": "Hello This is cdk-web-service",
	})
}

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "true",
	})
}

func main() {
	gin.SetMode(ServiceMode)

	r := gin.Default()

	r.GET("/", Handler)
	r.GET("/health", Health)

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
