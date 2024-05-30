//go:build !darwin && !windows && !linux

// Currently (2021/10/26) this file is not needed. However, keeping this around for potential
// expansion to other OSs.

package table

import "github.com/osquery/osquery-go"

func PlatformTables(_ PluginOpts) []osquery.OsqueryPlugin { return nil }
