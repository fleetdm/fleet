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
	config config.MysqlConfig
}

// New creates an MySQL datastore.
func New(config config.MysqlConfig, c clock.Clock, opts ...DBOption) (*Datastore, error) {
	options := &dbOptions{
		maxAttempts: defaultMaxAttempts,
		logger:      log.NewNopLogger(),
	}

	for _, setOpt := range opts {
		setOpt(options)
	}

	db, err := sqlx.Open("mysql", generateMysqlConnectionString(config))
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
		interval := time.Duration(attempt) * time.Second
		options.logger.Log("mysql", fmt.Sprintf(
			"could not connect to db: %v, sleeping %v", dbError, interval))
		time.Sleep(interval)
	}

	if dbError != nil {
		return nil, dbError
	}

	ds := &Datastore{
		db:     db,
		logger: options.logger,
		clock:  c,
		config: config,
	}

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

// Initialize preload data needed by the application
func (d *Datastore) Initialize() error {
	if err := d.createBuiltinLabels(); err != nil {
		return err
	}
	return nil
}

// Drop removes database
func (d *Datastore) Drop() error {
	tables := []struct {
		Name string `db:"TABLE_NAME"`
	}{}

	sql := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?;
	`

	if err := d.db.Select(&tables, sql, d.config.Database); err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		return tx.Rollback()
	}

	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DROP TABLE %s;", table.Name))
		if err != nil {
			return tx.Rollback()
		}
	}
	_, err = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
	if err != nil {
		return tx.Rollback()
	}
	return tx.Commit()
}

// HealthCheck returns an error if the MySQL backend is not healthy.
func (d *Datastore) HealthCheck() error {
	_, err := d.db.Exec("select 1")
	return err
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

// generateMysqlConnectionString returns a MySQL connection string using the
// provided configuration.
func generateMysqlConnectionString(conf config.MysqlConfig) string {
	return fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8&parseTime=true&loc=UTC",
		conf.Username,
		conf.Password,
		conf.Address,
		conf.Database,
	)
}

func (d *Datastore) createBuiltinLabels() error {
	// Nuke built in labels and recreate them
	_, err := d.db.Exec("DELETE from labels WHERE label_type = ?", kolide.LabelTypeBuiltIn)
	if err != nil {
		return err
	}

	labels := []kolide.Label{
		{
			Name:      "All Hosts",
			Query:     "select 1;",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "darwin",
			Name:      "Mac OS X",
			Query:     "select 1 from osquery_info where build_platform = 'darwin';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "ubuntu",
			Name:      "Ubuntu Linux",
			Query:     "select 1 from osquery_info where build_platform = 'ubuntu';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "centos",
			Name:      "CentOS Linux",
			Query:     "select 1 from osquery_info where build_platform = 'centos';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
		{
			Platform:  "windows",
			Name:      "MS Windows",
			Query:     "select 1 from osquery_info where build_platform = 'windows';",
			LabelType: kolide.LabelTypeBuiltIn,
		},
	}

	for _, label := range labels {
		_, err = d.NewLabel(&label)
		if err != nil {
			return err
		}
	}

	return nil

}
