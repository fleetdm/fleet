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
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/logwriter"
	"github.com/kolide/fleet/server/sso"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewService creates a new service from the config struct
func NewService(ds kolide.Datastore, resultStore kolide.QueryResultStore,
	logger kitlog.Logger, kolideConfig config.KolideConfig, mailService kolide.MailService,
	c clock.Clock, sso sso.SessionStore) (kolide.Service, error) {
	var svc kolide.Service
	statusWriter, err := osqueryLogFile(kolideConfig.Osquery.StatusLogFile, logger, kolideConfig.Osquery.EnableLogRotation)
	if err != nil {
		return nil, err
	}
	resultWriter, err := osqueryLogFile(kolideConfig.Osquery.ResultLogFile, logger, kolideConfig.Osquery.EnableLogRotation)
	if err != nil {
		return nil, err
	}

	svc = service{
		ds:          ds,
		resultStore: resultStore,
		logger:      logger,
		config:      kolideConfig,
		clock:       c,

		osqueryStatusLogWriter: statusWriter,
		osqueryResultLogWriter: resultWriter,
		mailService:            mailService,
		ssoSessionStore:        sso,
	}
	svc = validationMiddleware{svc, ds, sso}
	return svc, nil
}

// osqueryLogFile creates a log file for osquery status/result logs
// the logFile can be rotated by sending a `SIGHUP` signal to kolide if
// enableRotation is true
func osqueryLogFile(path string, appLogger kitlog.Logger, enableRotation bool) (io.Writer, error) {
	if enableRotation {
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
		return osquerydLogger, nil
	}
	// no log rotation
	return logwriter.New(path)
}

type service struct {
	ds          kolide.Datastore
	resultStore kolide.QueryResultStore
	logger      kitlog.Logger
	config      config.KolideConfig
	clock       clock.Clock

	osqueryStatusLogWriter io.Writer
	osqueryResultLogWriter io.Writer

	mailService     kolide.MailService
	ssoSessionStore sso.SessionStore
}

func (s service) SendEmail(mail kolide.Email) error {
	return s.mailService.SendEmail(mail)
}

func (s service) Clock() clock.Clock {
	return s.clock
}

type validationMiddleware struct {
	kolide.Service
	ds              kolide.Datastore
	ssoSessionStore sso.SessionStore
}
