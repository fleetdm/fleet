package service

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kolide/fleet/server/kolide"
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

// loggerDebug returns the debug level
func (mw loggingMiddleware) loggerDebug(err error) kitlog.Logger {
	return level.Debug(mw.logger)
}

// loggerInfo returns the info level
func (mw loggingMiddleware) loggerInfo(err error) kitlog.Logger {
	return level.Info(mw.logger)
}
