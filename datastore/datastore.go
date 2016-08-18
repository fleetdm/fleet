// Package datastore implements Kolide's interactions with the database backend
package datastore

import (
	"github.com/Sirupsen/logrus"
	"github.com/kolide/kolide-ose/errors"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/kolide/kolide-ose/sessions"
)

type Datastore interface {
	kolide.UserStore
	kolide.HostStore
	kolide.CampaignStore
	Drop() error
	Migrate() error
}

type dbOptions struct {
	maxAttempts int
	db          Datastore
	debug       bool // gorm debug
	logger      *logrus.Logger
}

// DBOption is used to pass optional arguments to a database connection
type DBOption func(o *dbOptions) error

// Logger adds a logger to the datastore
func Logger(l *logrus.Logger) DBOption {
	return func(o *dbOptions) error {
		o.logger = l
		return nil
	}
}

// LimitAttempts sets number of maximum connection attempts
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
func datastore(db Datastore) DBOption {
	return func(o *dbOptions) error {
		o.db = db
		return nil
	}
}

// New creates a Datastore with a database connection
// Use DBOption to pass optional arguments
func New(driver, conn string, opts ...DBOption) (Datastore, error) {
	opt := &dbOptions{
		maxAttempts: 15, // default attempts
	}
	for _, option := range opts {
		if err := option(opt); err != nil {
			return nil, errors.DatabaseError(err)
		}
	}

	// check if datastore is already present
	if opt.db != nil {
		return opt.db, nil
	}

	var db Datastore
	switch driver {
	case "gorm":
		db, err := openGORM("mysql", conn, opt.maxAttempts)
		if err != nil {
			return nil, errors.DatabaseError(err)
		}
		// configure logger
		if opt.logger != nil {
			db.SetLogger(opt.logger)
			db.LogMode(opt.debug)
		}
		ds := gormDB{DB: db}
		if err := ds.Migrate(); err != nil {
			return nil, errors.DatabaseError(err)
		}
		return ds, nil
	}
	return db, nil
}

// NewSessionBackend creates a new session from a datastore
// this function nis temporary
func NewSessionBackend(ds Datastore) sessions.SessionBackend {
	db := ds.(gormDB)
	return &sessions.GormSessionBackend{DB: db.DB}
}
