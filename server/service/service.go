// Package service holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package service

import (
	"io"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/kolide"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewService creates a new service from the config struct
func NewService(ds kolide.Datastore, logger kitlog.Logger, kolideConfig config.KolideConfig, mailService kolide.MailService, c clock.Clock) (kolide.Service, error) {
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
		clock:  c,

		osqueryStatusLogWriter: logFile(kolideConfig.Osquery.StatusLogFile),
		osqueryResultLogWriter: logFile(kolideConfig.Osquery.ResultLogFile),
		mailService:            mailService,
	}
	svc = validationMiddleware{svc}
	return svc, nil
}

type service struct {
	ds     kolide.Datastore
	logger kitlog.Logger
	config config.KolideConfig
	clock  clock.Clock

	osqueryStatusLogWriter io.Writer
	osqueryResultLogWriter io.Writer

	mailService kolide.MailService
}
