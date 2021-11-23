//go:build !darwin

package table

import "github.com/kolide/osquery-go"

func platformTables() []osquery.OsqueryPlugin { return nil }
