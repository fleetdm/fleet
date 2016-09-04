// Package kitserver holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package kitserver

import (
	"io"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/kolide"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// configuration defaults
// TODO move to main?
const (
	defaultBcryptCost   int    = 12
	defaultSaltKeySize  int    = 24
	defaultCookieName   string = "KolideSession"
	defaultEnrollSecret string = "xxx change me"
	defaultNodeKeySize  int    = 24
)

// NewService creates a new service from the config struct
func NewService(config ServiceConfig) (kolide.Service, error) {
	var svc kolide.Service

	logFile := func(path string) io.Writer {
		return &lumberjack.Logger{
			Filename:   path,
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28, //days
		}
	}

	svc = service{
		ds:                      config.Datastore,
		logger:                  config.Logger,
		saltKeySize:             config.SaltKeySize,
		bcryptCost:              config.BcryptCost,
		jwtKey:                  config.JWTKey,
		cookieName:              config.SessionCookieName,
		osqueryEnrollSecret:     config.OsqueryEnrollSecret,
		osqueryNodeKeySize:      config.OsqueryNodeKeySize,
		osqueryStatusLogWriter:  logFile(config.OsqueryStatusLogPath),
		osqueryResultsLogWriter: logFile(config.OsqueryResultsLogPath),
	}
	svc = validationMiddleware{svc}
	return svc, nil
}

type service struct {
	ds     kolide.Datastore
	logger kitlog.Logger

	saltKeySize int
	bcryptCost  int

	jwtKey     string
	cookieName string

	osqueryEnrollSecret     string
	osqueryNodeKeySize      int
	osqueryStatusLogWriter  io.Writer
	osqueryResultsLogWriter io.Writer
}

// ServiceConfig holds the parameters for creating a Service
type ServiceConfig struct {
	Datastore kolide.Datastore
	Logger    kitlog.Logger

	// password config
	SaltKeySize int
	BcryptCost  int

	// session config
	JWTKey            string
	SessionCookieName string

	// osquery config
	OsqueryEnrollSecret   string
	OsqueryNodeKeySize    int
	OsqueryStatusLogPath  string
	OsqueryResultsLogPath string
}
