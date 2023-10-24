//go:build !darwin && !windows && !linux

package table

import "github.com/osquery/osquery-go"

func PlatformTables() []osquery.OsqueryPlugin { return nil }
