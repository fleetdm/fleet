// Package *must* be named gorules.
package gorules

// We always need this import.
import "github.com/quasilyte/go-ruleguard/dsl"

func txCheck(m dsl.Matcher) {
	m.Import("github.com/fleetdm/fleet/v4/server/datastore/mysql")
	m.Import("github.com/jmoiron/sqlx")

	m.Match(`$ds.withTx($_, $fn)`, `$ds.withRetryTxx($_, $fn)`).
		Where(m["ds"].Type.Is(`*mysql.Datastore`) && (m["fn"].Contains(`$ds.writer`) || m["fn"].Contains(`$ds.reader`))).
		Report("found transaction")
}
