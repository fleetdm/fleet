package main

import (
	"github.com/gin-gonic/gin"
)

func attachRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")

	// osquery API endpoints
	osquery := v1.Group("/osquery")
	osquery.POST("/enroll", OsqueryEnroll)
	osquery.POST("/config", OsqueryConfig)
	osquery.POST("/log", OsqueryLog)
	osquery.POST("/distributed/read", OsqueryDistributedRead)
	osquery.POST("/distributed/write", OsqueryDistributedWrite)
}

func createServer() *gin.Engine {
	server := gin.New()
	attachRoutes(server)
	return server
}
