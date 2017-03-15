// Package service holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package service

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/kolide"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewService creates a new service from the config struct
func NewService(ds kolide.Datastore, resultStore kolide.QueryResultStore, logger kitlog.Logger, kolideConfig config.KolideConfig, mailService kolide.MailService, c clock.Clock, checker kolide.LicenseChecker) (kolide.Service, error) {
	var svc kolide.Service

	statusWriter := osqueryLogFile(kolideConfig.Osquery.StatusLogFile, logger)
	resultWriter := osqueryLogFile(kolideConfig.Osquery.ResultLogFile, logger)

	svc = service{
		ds:             ds,
		resultStore:    resultStore,
		logger:         logger,
		config:         kolideConfig,
		clock:          c,
		licenseChecker: checker,

		osqueryStatusLogWriter: statusWriter,
		osqueryResultLogWriter: resultWriter,
		mailService:            mailService,
	}
	svc = validationMiddleware{svc, ds}
	return svc, nil
}

// osqueryLogFile creates a log file for osquery status/result logs
// the logFile can be rotated by sending a `SIGHUP` signal to kolide.
func osqueryLogFile(path string, appLogger kitlog.Logger) io.Writer {
	osquerydLogger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    500, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
	}
	appLogger = kitlog.With(appLogger, "component", "osqueryd-logger")

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for {
			<-sig //block on signal
			if err := osquerydLogger.Rotate(); err != nil {
				appLogger.Log("err", err)
			}
		}
	}()

	return osquerydLogger
}

type service struct {
	ds             kolide.Datastore
	resultStore    kolide.QueryResultStore
	logger         kitlog.Logger
	config         config.KolideConfig
	clock          clock.Clock
	licenseChecker kolide.LicenseChecker

	osqueryStatusLogWriter io.Writer
	osqueryResultLogWriter io.Writer

	mailService kolide.MailService
}

func (s service) SendEmail(mail kolide.Email) error {
	return s.mailService.SendEmail(mail)
}

func (s service) Clock() clock.Clock {
	return s.clock
}
