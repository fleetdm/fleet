package mysql

import "github.com/go-kit/kit/log"

const defaultMaxAttempts int = 15

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *dbOptions) error

type dbOptions struct {
	// maxAttempts configures the number of retries to connect to the DB
	maxAttempts int
	logger      log.Logger
}

// Logger adds a logger to the datastore
func Logger(l log.Logger) DBOption {
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
