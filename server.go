package main

import (
	"github.com/gin-gonic/gin"
)

// ServerError is a helper which accepts a string error and returns a map in
// format that is required by gin.Context.JSON
func ServerError(e string) *map[string]interface{} {
	return &map[string]interface{}{
		"error": e,
	}
}

// DatabaseError emits a response that is appropriate in the event that a
// database failure occurs, a record is not found in the database, etc
func DatabaseError(c *gin.Context) {
	c.JSON(500, ServerError("Database error"))
}

// UnauthorizedError emits a response that is appropriate in the event that a
// request is received by a user which is not authorized to carry out the
// requested action
func UnauthorizedError(c *gin.Context) {
	c.JSON(401, ServerError("Unauthorized"))
}

func createTestServer() *gin.Engine {
	server := gin.New()
	server.Use(TestingDatabaseMiddleware)
	return server
}

// CreateServer creates a gin.Engine HTTP server and configures it to be in a
// state such that it is ready to serve HTTP requests for the kolide application
func CreateServer() *gin.Engine {
	server := gin.New()
	server.Use(ProductionDatabaseMiddleware)

	v1 := server.Group("/api/v1")

	// Kolide application API endpoints
	kolide := v1.Group("/kolide")
	kolide.Use(SessionMiddleware)
	kolide.Use(JWTRenewalMiddleware)

	kolide.POST("/login", Login)
	kolide.GET("/logout", Logout)

	kolide.GET("/user", GetUser)
	kolide.PUT("/user", CreateUser)
	kolide.PATCH("/user", ModifyUser)
	kolide.DELETE("/user", DeleteUser)

	kolide.PATCH("/user/password", ResetUserPassword)
	kolide.PATCH("/user/admin", SetUserAdminState)
	kolide.PATCH("/user/enabled", SetUserEnabledState)

	// osquery API endpoints
	osquery := v1.Group("/osquery")
	osquery.POST("/enroll", OsqueryEnroll)
	osquery.POST("/config", OsqueryConfig)
	osquery.POST("/log", OsqueryLog)
	osquery.POST("/distributed/read", OsqueryDistributedRead)
	osquery.POST("/distributed/write", OsqueryDistributedWrite)

	return server
}
