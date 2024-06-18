package mysql

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/log"
	"github.com/ngrok/sqlmw"
)

const (
	defaultMaxAttempts         int = 15
	defaultMinLastOpenedAtDiff     = time.Hour
)

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *dbOptions) error

type dbOptions struct {
	// maxAttempts configures the number of retries to connect to the DB
	maxAttempts         int
	logger              log.Logger
	replicaConfig       *config.MysqlConfig
	interceptor         sqlmw.Interceptor
	tracingConfig       *config.LoggingConfig
	minLastOpenedAtDiff time.Duration
	sqlMode             string
	privateKey          string
}

// Logger adds a logger to the datastore.
func Logger(l log.Logger) DBOption {
	return func(o *dbOptions) error {
		o.logger = l
		return nil
	}
}

// WithInterceptor adds the sql interceptor to the datastore.
func WithInterceptor(i sqlmw.Interceptor) DBOption {
	return func(o *dbOptions) error {
		o.interceptor = i
		return nil
	}
}

// Replica sets the configuration of the read replica for the datastore.
func Replica(conf *config.MysqlConfig) DBOption {
	return func(o *dbOptions) error {
		o.replicaConfig = conf
		return nil
	}
}

// LimitAttempts sets a the number of attempts
// to try establishing a connection to the database backend
// the default value is 15 attempts
func LimitAttempts(attempts int) DBOption {
	return func(o *dbOptions) error {
		o.maxAttempts = attempts
		return nil
	}
}

func TracingEnabled(lconfig *config.LoggingConfig) DBOption {
	return func(o *dbOptions) error {
		o.tracingConfig = lconfig
		return nil
	}
}

// WithFleetConfig provides the fleet configuration so that any config option
// that must be used in the datastore layer can be captured here.
func WithFleetConfig(conf *config.FleetConfig) DBOption {
	return func(o *dbOptions) error {
		o.minLastOpenedAtDiff = conf.Osquery.MinSoftwareLastOpenedAtDiff
		o.privateKey = conf.Server.PrivateKey
		return nil
	}
}

// SQLMode allows setting a custom sql_mode string.
func SQLMode(mode string) DBOption {
	return func(o *dbOptions) error {
		o.sqlMode = mode
		return nil
	}
}
