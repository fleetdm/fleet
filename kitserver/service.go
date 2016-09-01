// Package kitserver holds the implementation of the kolide service interface and the HTTP endpoints
// for the API
package kitserver

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/kolide"
)

// configuration defaults
// TODO move to main?
const (
	defaultBcryptCost  int    = 12
	defaultSaltKeySize int    = 24
	defaultCookieName  string = "KolideSession"
)

// NewService creates a new service from the config struct
func NewService(config ServiceConfig) (kolide.Service, error) {
	var svc kolide.Service
	svc = service{
		ds:                  config.Datastore,
		logger:              config.Logger,
		saltKeySize:         config.SaltKeySize,
		bcryptCost:          config.BcryptCost,
		jwtKey:              config.JWTKey,
		cookieName:          config.SessionCookieName,
		OsqueryEnrollSecret: config.OsqueryEnrollSecret,
		OsqueryNodeKeySize:  config.OsqueryNodeKeySize,
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

	OsqueryEnrollSecret string
	OsqueryNodeKeySize  int
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
	OsqueryEnrollSecret string
	OsqueryNodeKeySize  int
}
