// package server holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package server

import (
	"io"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
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
func NewService(ds kolide.Datastore, logger kitlog.Logger, kolideConfig config.KolideConfig) (kolide.Service, error) {
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
		ds:     ds,
		logger: logger,
		config: kolideConfig,

		osqueryStatusLogWriter:  logFile(kolideConfig.Osquery.StatusLogFile),
		osqueryResultsLogWriter: logFile(kolideConfig.Osquery.ResultLogFile),
	}
	svc = validationMiddleware{svc}
	return svc, nil
}

type service struct {
	ds     kolide.Datastore
	logger kitlog.Logger
	config config.KolideConfig

	osqueryStatusLogWriter  io.Writer
	osqueryResultsLogWriter io.Writer
}
