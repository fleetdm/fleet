//go:build !darwin && !windows

package table

import "github.com/osquery/osquery-go"

func platformTables() []osquery.OsqueryPlugin { return nil }
