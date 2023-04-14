//go:build !darwin && !windows

package table

import "github.com/osquery/osquery-go"

func PlatformTables(_ *Runner) []osquery.OsqueryPlugin { return nil }
