package mysql

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/go-kit/log"
	"github.com/ngrok/sqlmw"
)

const (
	defaultMaxAttempts         int = 15
	defaultMinLastOpenedAtDiff     = time.Hour
)

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *common_mysql.DBOptions) error

// Logger adds a Logger to the datastore.
func Logger(l log.Logger) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.Logger = l
		return nil
	}
}

// WithInterceptor adds the sql Interceptor to the datastore.
func WithInterceptor(i sqlmw.Interceptor) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.Interceptor = i
		return nil
	}
}

// Replica sets the configuration of the read replica for the datastore.
func Replica(conf *config.MysqlConfig) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.ReplicaConfig = conf
		return nil
	}
}

// LimitAttempts sets a the number of attempts
// to try establishing a connection to the database backend
// the default value is 15 attempts
func LimitAttempts(attempts int) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.MaxAttempts = attempts
		return nil
	}
}

func TracingEnabled(lconfig *config.LoggingConfig) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.TracingConfig = lconfig
		return nil
	}
}

// WithFleetConfig provides the fleet configuration so that any config option
// that must be used in the datastore layer can be captured here.
func WithFleetConfig(conf *config.FleetConfig) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.MinLastOpenedAtDiff = conf.Osquery.MinSoftwareLastOpenedAtDiff
		o.PrivateKey = conf.Server.PrivateKey
		return nil
	}
}

// SQLMode allows setting a custom sql_mode string.
func SQLMode(mode string) DBOption {
	return func(o *common_mysql.DBOptions) error {
		o.SqlMode = mode
		return nil
	}
}
