package datastore

import (
	"github.com/Sirupsen/logrus"
	"github.com/kolide/kolide-ose/kolide"
)

const (
	// TODO @marpaia fix/document default values
	defaultSessionKeySize  int     = 24
	defaultSessionLifespan float64 = 10
	defaultMaxAttempts     int     = 15
)

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *dbOptions) error

type dbOptions struct {
	// maxAttempts configures the number of retries to connect to the DB
	maxAttempts     int
	db              kolide.Datastore
	debug           bool // gorm debug
	logger          *logrus.Logger
	sessionKeySize  int
	sessionLifespan float64
}

// SessionKeySize configures the session key size
func SessionKeySize(keySize int) DBOption {
	return func(o *dbOptions) error {
		o.sessionKeySize = keySize
		return nil
	}
}

// SessionLifespan sets a custom session lifespan
func SessionLifespan(lifespan float64) DBOption {
	return func(o *dbOptions) error {
		o.sessionLifespan = lifespan
		return nil
	}
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
