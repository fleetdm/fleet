package service

import (
	"github.com/fleetdm/fleet/server/kolide"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// logging middleware logs the service actions
type loggingMiddleware struct {
	kolide.Service
	logger kitlog.Logger
}

// NewLoggingService takes an existing service and adds a logging wrapper
func NewLoggingService(svc kolide.Service, logger kitlog.Logger) kolide.Service {
	return loggingMiddleware{Service: svc, logger: logger}
}

// loggerDebug returns the the info level if there error is non-nil, otherwise defaulting to the debug level.
func (mw loggingMiddleware) loggerDebug(err error) kitlog.Logger {
	if err != nil {
		return level.Info(mw.logger)
	}
	return level.Debug(mw.logger)
}

// loggerInfo returns the info level
func (mw loggingMiddleware) loggerInfo(err error) kitlog.Logger {
	return level.Info(mw.logger)
}
