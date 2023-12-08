//go:build darwin && arm64

package table

import (
	// ARM64 Kolide tables
	appicons "github.com/fleetdm/fleet/v4/orbit/pkg/table/app-icons"

	"github.com/osquery/osquery-go"
)

func appendTables(plugins []osquery.OsqueryPlugin) []osquery.OsqueryPlugin {
	plugins = append(plugins,
		// arm64 tables
		appicons.AppIcons(),
	)
	return plugins
}
