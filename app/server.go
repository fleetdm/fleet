package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gin-gonic/contrib/ginrus"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/sessions"
	"github.com/spf13/viper"
	"gopkg.in/go-playground/validator.v8"
)

var validate = validator.New(&validator.Config{TagName: "validate", FieldNameTag: "json"})

// Get the SMTP connection pool from the context, or panic
func GetSMTPConnectionPool(c *gin.Context) SMTPConnectionPool {
	return c.MustGet("SMTPConnectionPool").(SMTPConnectionPool)
}

func SMTPConnectionPoolMiddleware(pool SMTPConnectionPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("SMTPConnectionPool", pool)
		c.Next()
	}
}

// Get the database connection from the context, or panic
func GetDB(c *gin.Context) datastore.Datastore {
	return c.MustGet("DB").(datastore.Datastore)
}

func DatabaseMiddleware(db datastore.Datastore) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("DB", db)
		c.Next()
	}
}

// UnauthorizedError emits a response that is appropriate in the event that a
// request is received by a user which is not authorized to carry out the
// requested action
func UnauthorizedError(c *gin.Context) {
	errors.ReturnError(
		c,
		errors.NewWithStatus(
			http.StatusUnauthorized,
			"Unauthorized",
			"Unauthorized",
		))
}

// NotFoundRequestError emits a response that is appropriate in the event that
// a request is received for a resource which is not found
func NotFoundRequestError(c *gin.Context) {
	errors.ReturnError(
		c,
		errors.NewWithStatus(
			http.StatusNotFound,
			"Not Found",
			"Not Found",
		))
}

// Create a new server for testing purposes with no routes attached
// func createEmptyTestServer(db datastore.Datastore) *gin.Engine {
// 	server := gin.New()
// 	server.Use(DatabaseMiddleware(db))
// 	server.Use(SessionBackendMiddleware)
// 	return server
// }

// NewSessionManager allows you to get a SessionManager instance for a given
// web request. Unless you're interacting with login, logout, or core auth
// code, this should be abstracted by the ViewerContext pattern.
func NewSessionManager(c *gin.Context) *sessions.SessionManager {
	return &sessions.SessionManager{
		Request: c.Request,
		Backend: GetSessionBackend(c),
		Writer:  c.Writer,
	}
}

// Unmarshal JSON from the gin context into a struct
func parseJSON(c *gin.Context, obj interface{}) error {
	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(obj); err != nil {
		return errors.NewFromError(err, http.StatusBadRequest, "JSON parse error")
	}
	return nil
}

// Parse JSON into a struct with json.Unmarshal, followed by validation with
// the validator library.
func ParseAndValidateJSON(c *gin.Context, obj interface{}) error {
	if err := parseJSON(c, obj); err != nil {
		return err
	}

	return validate.Struct(obj)
}

func NotFound(c *gin.Context) {
	errors.ReturnError(
		c,
		errors.NewWithStatus(
			http.StatusNotFound,
			"Not found",
			fmt.Sprintf("Route not found for request: %+v", c.Request),
		))
}

// CreateServer creates a gin.Engine HTTP server and configures it to be in a
// state such that it is ready to serve HTTP requests for the kolide application
func CreateServer(backend sessions.SessionBackend, db datastore.Datastore, pool SMTPConnectionPool, w io.Writer, resultHandler OsqueryResultHandler, statusHandler OsqueryStatusHandler) *gin.Engine {
	server := gin.New()
	server.Use(DatabaseMiddleware(db))
	server.Use(SMTPConnectionPoolMiddleware(pool))
	server.Use(SessionBackendMiddleware(backend))

	sessions.Configure(&sessions.SessionConfiguration{
		CookieName:     "KolideSession",
		JWTKey:         viper.GetString("auth.jwt_key"),
		SessionKeySize: viper.GetInt("session.key_size"),
		Lifespan:       viper.GetFloat64("session.expiration_seconds"),
	})

	// TODO: The following loggers are not synchronized with each other or
	// logrus.StandardLogger() used through the rest of the codebase. As
	// such, their output may become intermingled.
	// See https://github.com/Sirupsen/logrus/issues/391

	// Ginrus middleware logs details about incoming requests using the
	// logrus WithFields
	requestLogger := logrus.New()
	requestLogger.Out = w
	server.Use(ginrus.Ginrus(requestLogger, time.RFC3339, false))

	// Recovery middleware recovers from panic(), returning a 500 response
	// code and printing the panic information to the log
	recoveryLogger := logrus.New()
	recoveryLogger.WriterLevel(logrus.ErrorLevel)
	recoveryLogger.Out = w
	server.Use(gin.RecoveryWithWriter(recoveryLogger.Writer()))

	// Set the 404 route
	server.NoRoute(NotFound)

	// Kolide react entrypoint
	server.HTMLRender = loadTemplates("react.tmpl")
	server.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "react.tmpl", gin.H{})
	})
	// Kolide assets
	server.Use(static.Serve("/assets", NewBinaryFileSystem("/build")))

	v1 := server.Group("/api/v1")

	// Kolide application API endpoints
	kolide := v1.Group("/kolide")

	kolide.POST("/login", Login)
	kolide.GET("/logout", Logout)

	kolide.POST("/user", GetUser)
	kolide.PUT("/user", CreateUser)
	kolide.PATCH("/user", ModifyUser)

	kolide.PATCH("/user/password", ChangeUserPassword)
	kolide.POST("/user/password/reset", ResetUserPassword)
	kolide.DELETE("/user/password/reset", DeletePasswordResetRequest)
	kolide.POST("/user/password/reset/verify", VerifyPasswordResetRequest)
	kolide.PATCH("/user/admin", SetUserAdminState)
	kolide.PATCH("/user/enabled", SetUserEnabledState)

	kolide.POST("/user/sessions", GetInfoAboutSessionsForUser)
	kolide.DELETE("/user/sessions", DeleteSessionsForUser)

	kolide.DELETE("/session", DeleteSession)
	kolide.POST("/session", GetInfoAboutSession)

	// osquery API endpoints
	osq := v1.Group("/osquery")

	osqueryHandler := OsqueryHandler{
		ResultHandler: resultHandler,
		StatusHandler: statusHandler,
	}

	osq.POST("/enroll", OsqueryEnroll)
	osq.POST("/config", OsqueryConfig)
	osq.POST("/log", osqueryHandler.OsqueryLog)
	osq.POST("/distributed/read", OsqueryDistributedRead)
	osq.POST("/distributed/write", OsqueryDistributedWrite)

	return server
}
