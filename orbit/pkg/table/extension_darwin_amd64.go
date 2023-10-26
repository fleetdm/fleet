//go:build darwin && amd64

package table

import "github.com/osquery/osquery-go"

// stub for amd64 platforms
func appendTables(plugins []osquery.OsqueryPlugin) []osquery.OsqueryPlugin {
	return plugins
}
