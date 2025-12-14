package common_mysql

import "github.com/jmoiron/sqlx"

// DBReadTx provides a minimal interface for read-only transactions.
type DBReadTx interface {
	sqlx.QueryerContext
	sqlx.PreparerContext

	Rebind(string) string
}
