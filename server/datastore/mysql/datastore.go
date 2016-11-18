package mysql

import (
	"fmt"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/go-kit/kit/log"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/kolide-ose/server/config"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pressly/goose"
)

const (
	defaultSelectLimit = 1000
)

// Datastore is an implementation of kolide.Datastore interface backed by
// MySQL
type Datastore struct {
	db     *sqlx.DB
	logger log.Logger
	clock  clock.Clock
}

// New creates an MySQL datastore.
func New(dbConnectString string, c clock.Clock, opts ...DBOption) (*Datastore, error) {

	options := &dbOptions{
		maxAttempts: defaultMaxAttempts,
		logger:      log.NewNopLogger(),
	}

	for _, setOpt := range opts {
		setOpt(options)
	}

	db, err := sqlx.Open("mysql", dbConnectString)
	if err != nil {
		return nil, err
	}

	var dbError error
	for attempt := 0; attempt < options.maxAttempts; attempt++ {
		dbError = db.Ping()
		if dbError == nil {
			// we're connected!
			break
		}
		sleep := time.Duration(attempt)
		options.logger.Log("mysql", fmt.Sprintf(
			"could not connect to db: %v, sleeping %v", dbError, sleep))
		time.Sleep(sleep * time.Second)
	}

	if dbError != nil {
		return nil, dbError
	}

	ds := &Datastore{db, options.logger, c}

	return ds, nil

}

func (d *Datastore) Name() string {
	return "mysql"
}

// Migrate creates database
func (d *Datastore) Migrate() error {

	goose.SetDialect("mysql")

	if err := goose.Run("up", d.db.DB, "."); err != nil {
		return err
	}

	return nil

}

// Drop removes database
func (d *Datastore) Drop() error {
	goose.SetDialect("mysql")

	for {
		version, err := goose.EnsureDBVersion(d.db.DB)
		if err != nil {
			return err
		}

		if version == 0 {
			d.db.Exec("DROP TABLE IF EXISTS `goose_db_version`;")
			return nil
		}

		if err = goose.Run("down", d.db.DB, "."); err != nil {
			return err
		}
	}

}

// Close frees resources associated with underlying mysql connection
func (d *Datastore) Close() error {
	return d.db.Close()
}

func (d *Datastore) log(msg string) {
	d.logger.Log("comp", d.Name(), "msg", msg)
}

func appendListOptionsToSQL(sql string, opts kolide.ListOptions) string {
	if opts.OrderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == kolide.OrderDescending {
			direction = "DESC"
		}

		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, opts.OrderKey, direction)
	}
	// REVIEW: If caller doesn't supply a limit apply a default limit of 1000
	// to insure that an unbounded query with many results doesn't consume too
	// much memory or hang
	if opts.PerPage == 0 {
		opts.PerPage = defaultSelectLimit
	}

	sql = fmt.Sprintf("%s LIMIT %d", sql, opts.PerPage)

	offset := opts.PerPage * opts.Page

	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}

	return sql
}

// GetMysqlConnectionString returns a MySQL connection string using the
// provided configuration.
func GetMysqlConnectionString(conf config.MysqlConfig) string {
	return fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
		conf.Username,
		conf.Password,
		conf.Address,
		conf.Database,
	)
}
