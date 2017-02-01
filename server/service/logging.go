package service

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide/server/kolide"
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
