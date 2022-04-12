//go:build !darwin

package table

import "github.com/osquery/osquery-go"

func platformTables() []osquery.OsqueryPlugin { return nil }
