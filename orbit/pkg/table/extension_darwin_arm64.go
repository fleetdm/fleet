//go:build darwin && arm64

package table

import (
	// ARM64 Kolide tables
	appicons "github.com/kolide/launcher/pkg/osquery/tables/app-icons"
	"github.com/kolide/launcher/pkg/osquery/tables/macos_software_update"

	"github.com/osquery/osquery-go"
)

func appendTables(plugins []osquery.OsqueryPlugin) []osquery.OsqueryPlugin {
	plugins = append(plugins,
		// ARM64 Kolide tables
		appicons.AppIcons(),
		macos_software_update.MacOSUpdate(),
		macos_software_update.RecommendedUpdates(kolideLogger),
	)
	return plugins
}
