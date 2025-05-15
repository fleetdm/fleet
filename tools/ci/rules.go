//go:build ruleguard
// +build ruleguard

package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

func fmtErrorfWithoutArgs(m dsl.Matcher) {
	m.Match(`fmt.Errorf($msg)`).
		Report(`fmt.Errorf: change for errors.New($msg)`).
		Suggest(`errors.New($msg)`)
}

func createHttpClient(m dsl.Matcher) {
	m.Match(
		`http.Client{$*_}`,
		`new(http.Client)`,
		`http.Transport{$*_}`,
		`new(http.Transport)`,
	).Report(`http.Client: use fleethttp.NewClient instead`)
}

func txCheck(m dsl.Matcher) {
	m.Import("github.com/fleetdm/fleet/v4/server/datastore/mysql")
	m.Import("github.com/fleetdm/fleet/v4/server/mdm/android/mysql")
	m.Import("github.com/jmoiron/sqlx")

	isDatastoreType := func(v dsl.Var) bool {
		return v.Type.Is(`*mysql.Datastore`) || v.Type.Is(`mysql.Datastore`)
	}

	containsIllegal := func(v dsl.Var) bool {
		return v.Contains(`$ds.writer`) || v.Contains(`$ds.reader`) ||
			v.Contains(`$ds.Writer`) || v.Contains(`$ds.Reader`)
	}

	isSqlxIface := func(v dsl.Var) bool {
		return (v.Type.Is(`sqlx.ExtContext`) || v.Type.Is(`sqlx.ExecContext`))
	}

	m.Match(`$ds.withTx($_, $fn)`, `$ds.withRetryTxx($_, $fn)`, `$ds.WithRetryTxx($_, $fn)`).
		Where(isDatastoreType(m["ds"]) && containsIllegal(m["fn"])).
		Report("improper use of ds.reader or ds.writer in a transaction")

	// any Datastore method that receives a Tx (sqlx.ExtContext/sqlx.ExecContext)
	// as the first argument must use it.
	m.Match(`func ($ds *Datastore) $name($p1, $*_) $*_ { $*fn }`, `func ($ds Datastore) $name($p1, $*_) $*_ { $*fn }`).
		Where(
			isDatastoreType(m["ds"]) &&
				isSqlxIface(m["p1"]) &&
				containsIllegal(m["fn"])).
		Report("improper use of ds.reader or ds.writer in Datastore.$name")

	// any Datastore method that receives a Tx (sqlx.ExtContext/sqlx.ExecerContext)
	// as the second argument must use it (e.g. ds.method(ctx, tx, ...)).
	m.Match(`func ($ds *Datastore) $name($_, $p2, $*_) $*_ { $*fn }`, `func ($ds Datastore) $name($_, $p2, $*_) $*_ { $*fn }`).
		Where(
			isDatastoreType(m["ds"]) &&
				isSqlxIface(m["p2"]) &&
				containsIllegal(m["fn"])).
		Report("improper use of ds.reader or ds.writer in Datastore.$name")

	// any func literal that receives a Tx (sqlx.ExtContext/sqlx.ExecerContext)
	// as the first argument must use it.
	m.Match(`func($p1, $*_) $*_ { $*fn }`).
		Where(
			isSqlxIface(m["p1"]) &&
				containsIllegal(m["fn"])).
		Report("improper use of ds.reader or ds.writer in function literal")

	// TODO: https://github.com/fleetdm/fleet/pull/8621#pullrequestreview-1172676063
	// This misses the case where a call to a Datastore method is done in one of
	// those functions and that method uses the reader/writer (and does not
	// receive a Tx as argument).
	//
	// I don't think the ruleguard pattern-matching syntax supports such a case
	// (recursively check if the function and any of its callees use some field),
	// it would probably require using the lower-level go/analysis package.
}
