package datastore

import (
	"github.com/Sirupsen/logrus"
	"github.com/kolide/kolide-ose/server/kolide"
)

const (
	// TODO @marpaia fix/document default values
	defaultMaxAttempts int = 15
)

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *dbOptions) error

type dbOptions struct {
	// maxAttempts configures the number of retries to connect to the DB
	maxAttempts int
	db          kolide.Datastore
	debug       bool // gorm debug
	logger      *logrus.Logger
}

// Logger adds a logger to the datastore
func Logger(l *logrus.Logger) DBOption {
	return func(o *dbOptions) error {
		o.logger = l
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

// Debug sets the GORM debug level
func Debug() DBOption {
	return func(o *dbOptions) error {
		o.debug = true
		return nil
	}
}

// datastore allows you to pass your own datastore
// this option can be used to pass a specific testing implementation
func datastore(db kolide.Datastore) DBOption {
	return func(o *dbOptions) error {
		o.db = db
		return nil
	}
}
