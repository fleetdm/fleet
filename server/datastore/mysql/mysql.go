package mysql

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/go-kit/kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/mysql/migrations/data"
	"github.com/kolide/fleet/server/datastore/mysql/migrations/tables"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

const (
	defaultSelectLimit = 100000
)

// Datastore is an implementation of kolide.Datastore interface backed by
// MySQL
type Datastore struct {
	db     *sqlx.DB
	logger log.Logger
	clock  clock.Clock
	config config.MysqlConfig
}

type dbfunctions interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

func (d *Datastore) getTransaction(opts []kolide.OptionalArg) dbfunctions {
	var result dbfunctions
	result = d.db
	for _, opt := range opts {
		switch t := opt().(type) {
		case dbfunctions:
			result = t
		}
	}
	return result
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

	if config.TLSConfig != "" {
		err := registerTLS(config)
		if err != nil {
			return nil, errors.Wrap(err, "register TLS config for mysql")
		}
	}

	dsn := generateMysqlConnectionString(config)
	db, err := sqlx.Open("mysql", dsn)
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

func (d *Datastore) Begin() (kolide.Transaction, error) {
	return d.db.Beginx()
}

func (d *Datastore) Name() string {
	return "mysql"
}

func (d *Datastore) MigrateTables() error {
	if err := tables.MigrationClient.Up(d.db.DB, ""); err != nil {
		return err
	}

	return nil
}

func (d *Datastore) MigrateData() error {
	if err := data.MigrationClient.Up(d.db.DB, ""); err != nil {
		return err
	}

	return nil
}

func (d *Datastore) MigrationStatus() (kolide.MigrationStatus, error) {
	if tables.MigrationClient.Migrations == nil || data.MigrationClient.Migrations == nil {
		return 0, errors.New("unexpected nil migrations list")
	}

	lastTablesMigration, err := tables.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.New("missing tables migrations")
	}

	currentTablesVersion, err := tables.MigrationClient.GetDBVersion(d.db.DB)
	if err != nil {
		return 0, errors.New("cannot get table migration status")
	}

	lastDataMigration, err := data.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.New("missing data migrations")
	}

	currentDataVersion, err := data.MigrationClient.GetDBVersion(d.db.DB)
	if err != nil {
		return 0, errors.New("cannot get table migration status")
	}

	switch {
	case currentDataVersion == 0 && currentTablesVersion == 0:
		return kolide.NoMigrationsCompleted, nil

	case currentTablesVersion != lastTablesMigration.Version ||
		currentDataVersion != lastDataMigration.Version:
		return kolide.SomeMigrationsCompleted, nil

	default:
		return kolide.AllMigrationsCompleted, nil
	}
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

// registerTLS adds client certificate configuration to the mysql connection.
func registerTLS(config config.MysqlConfig) error {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(config.TLSCA)
	if err != nil {
		return errors.Wrap(err, "read server-ca pem")
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return errors.New("failed to append PEM.")
	}
	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
	if err != nil {
		return errors.Wrap(err, "load mysql client cert and key")
	}
	clientCert = append(clientCert, certs)
	cfg := tls.Config{
		RootCAs:      rootCertPool,
		Certificates: clientCert,
	}
	if config.TLSServerName != "" {
		cfg.ServerName = config.TLSServerName
	}
	if err := mysql.RegisterTLSConfig(config.TLSConfig, &cfg); err != nil {
		return errors.Wrap(err, "register mysql tls config")
	}
	return nil
}

// generateMysqlConnectionString returns a MySQL connection string using the
// provided configuration.
func generateMysqlConnectionString(conf config.MysqlConfig) string {
	dsn := fmt.Sprintf(
		"%s:%s@(%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC&clientFoundRows=true",
		conf.Username,
		conf.Password,
		conf.Address,
		conf.Database,
	)

	if conf.TLSConfig != "" {
		dsn = fmt.Sprintf("%s&tls=%s", dsn, conf.TLSConfig)
	}

	return dsn
}
