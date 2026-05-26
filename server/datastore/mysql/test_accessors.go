package mysql

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// CheckAndModifyMysqlConfig is an exported wrapper around the package's
// internal config validator; used by mysqltest helpers. It must not be used
// from production code other than the existing mysql package internals.
func CheckAndModifyMysqlConfig(conf *config.MysqlConfig) error {
	return checkAndModifyConfig(conf)
}

// TestPrimaryDB exposes the primary *sqlx.DB for use by test helpers in the
// mysqltest package. It must not be used from production code.
func (ds *Datastore) TestPrimaryDB() *sqlx.DB {
	return ds.primary
}

// TestReplica returns the replica fleet.DBReader for use by test helpers.
// It must not be used from production code.
func (ds *Datastore) TestReplica() fleet.DBReader {
	return ds.replica
}

// TestSetReplica overrides the replica for use by test helpers. It must not
// be used from production code.
func (ds *Datastore) TestSetReplica(r fleet.DBReader) {
	ds.replica = r
}

// TestSetReadReplicaConfig overrides the read replica MySQL config for use
// by test helpers. It must not be used from production code.
func (ds *Datastore) TestSetReadReplicaConfig(cfg *common_mysql.MysqlConfig) {
	ds.readReplicaConfig = cfg
}

// TestLogger returns the datastore's logger for use by test helpers.
func (ds *Datastore) TestLogger() *slog.Logger {
	return ds.logger
}

// TestWriter returns the writer DB for use by test helpers.
func (ds *Datastore) TestWriter(ctx context.Context) *sqlx.DB {
	return ds.writer(ctx)
}

// TestReader returns the reader DB for use by test helpers.
func (ds *Datastore) TestReader(ctx context.Context) fleet.DBReader {
	return ds.reader(ctx)
}

// TestEncrypt encrypts data with the datastore's server private key for use
// by test helpers. It must not be used from production code.
func (ds *Datastore) TestEncrypt(data []byte) ([]byte, error) {
	return encrypt(data, ds.serverPrivateKey)
}
