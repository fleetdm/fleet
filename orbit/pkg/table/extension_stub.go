//go:build !darwin && !windows

package table

import "github.com/osquery/osquery-go"

func PlatformTables() []osquery.OsqueryPlugin { return nil }
