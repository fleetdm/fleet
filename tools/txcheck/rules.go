// Package *must* be named gorules.
package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

func txCheck(m dsl.Matcher) {
	m.Import("github.com/fleetdm/fleet/v4/server/datastore/mysql")
	m.Import("github.com/jmoiron/sqlx")

	m.Match(`$ds.withTx($_, $fn)`, `$ds.withRetryTxx($_, $fn)`).
		Where(m["ds"].Type.Is(`*mysql.Datastore`) && (m["fn"].Contains(`$ds.writer`) || m["fn"].Contains(`$ds.reader`))).
		Report("improper use of ds.reader or ds.writer in a transaction")

	isExported := func(v dsl.Var) bool {
		return v.Text.Matches(`^\p{Lu}`)
	}

	// any Datastore method that receives a Tx (sqlx.ExtContext/sqlx.ExecContext)
	// as the first argument must use it.
	m.Match(`func ($ds *Datastore) $name($p1, $*_) $*_ { $*fn }`, `func ($ds Datastore) $name($p1, $*_) $*_ { $*fn }`).
		Where(
			!isExported(m["name"]) &&
				(m["ds"].Type.Is(`*mysql.Datastore`) || m["ds"].Type.Is(`mysql.Datastore`)) &&
				(m["p1"].Type.Is(`sqlx.ExtContext`) || m["p1"].Type.Is(`sqlx.ExecContext`)) &&
				(m["fn"].Contains(`$ds.writer`) || m["fn"].Contains(`$ds.reader`))).
		Report("improper use of ds.reader or ds.writer in Datastore.$name")

	// any Datastore method that receives a Tx (sqlx.ExtContext/sqlx.ExecerContext)
	// as the second argument must use it.
	m.Match(`func ($ds *Datastore) $name($_, $p2, $*_) $*_ { $*fn }`, `func ($ds Datastore) $name($_, $p2, $*_) $*_ { $*fn }`).
		Where(
			!isExported(m["name"]) &&
				(m["ds"].Type.Is(`*mysql.Datastore`) || m["ds"].Type.Is(`mysql.Datastore`)) &&
				(m["p2"].Type.Is(`sqlx.ExtContext`) || m["p2"].Type.Is(`sqlx.ExecerContext`)) &&
				(m["fn"].Contains(`$ds.writer`) || m["fn"].Contains(`$ds.reader`))).
		Report("improper use of ds.reader or ds.writer in Datastore.$name")

	// any func literal that receives a Tx (sqlx.ExtContext/sqlx.ExecerContext)
	// as the first argument must use it.
	m.Match(`func($p1, $*_) $*_ { $*fn }`).
		Where(
			(m["p1"].Type.Is(`sqlx.ExtContext`) || m["p1"].Type.Is(`sqlx.ExecerContext`)) &&
				(m["fn"].Contains(`$ds.writer`) || m["fn"].Contains(`$ds.reader`))).
		Report("improper use of ds.reader or ds.writer in function literal")
}
