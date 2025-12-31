package fleet

import (
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/jmoiron/sqlx"
)

// DBLock represents a database transaction lock information as returned
// by datastore.DBLocks.
type DBLock struct {
	WaitingTrxID   string  `db:"waiting_trx_id" json:"waiting_trx_id"`
	WaitingThread  uint64  `db:"waiting_thread" json:"waiting_thread"`
	WaitingQuery   *string `db:"waiting_query" json:"waiting_query,omitempty"`
	BlockingTrxID  string  `db:"blocking_trx_id" json:"blocking_trx_id"`
	BlockingThread uint64  `db:"blocking_thread" json:"blocking_thread"`
	BlockingQuery  *string `db:"blocking_query" json:"blocking_query,omitempty"`
}

// DBReader is an interface that defines the methods required for reads.
type DBReader interface {
	sqlx.QueryerContext
	sqlx.PreparerContext

	Close() error
	Rebind(string) string
}

// DBReadTx is an alias for common_mysql.DBReadTx.
type DBReadTx = common_mysql.DBReadTx
