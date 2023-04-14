//go:build !darwin && !windows

package table

import "github.com/osquery/osquery-go"

func PlatformTables(_ string) []osquery.OsqueryPlugin { return nil }
