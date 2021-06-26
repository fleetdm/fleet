package service

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// logging middleware logs the service actions
type loggingMiddleware struct {
	fleet.Service
	logger kitlog.Logger
}

// NewLoggingService takes an existing service and adds a logging wrapper
func NewLoggingService(svc fleet.Service, logger kitlog.Logger) fleet.Service {
	return loggingMiddleware{Service: svc, logger: logger}
}

// loggerDebug returns the the info level if there error is non-nil, otherwise defaulting to the debug level.
func (mw loggingMiddleware) loggerDebug(err error) kitlog.Logger {
	logger := mw.logger
	if e, ok := err.(fleet.ErrWithInternal); ok {
		logger = kitlog.With(logger, "internal", e.Internal())
	}
	if err != nil {
		return level.Info(logger)
	}
	return level.Debug(logger)
}

// loggerInfo returns the info level
func (mw loggingMiddleware) loggerInfo(err error) kitlog.Logger {
	logger := mw.logger
	if e, ok := err.(fleet.ErrWithInternal); ok {
		logger = kitlog.With(logger, "internal", e.Internal())
	}
	return level.Info(logger)
}
