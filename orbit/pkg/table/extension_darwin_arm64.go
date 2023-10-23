//go:build darwin && arm64

package table

import (
	// ARM64 Kolide tables
	appicons "github.com/kolide/launcher/pkg/osquery/tables/app-icons"

	"github.com/osquery/osquery-go"
)

func appendARM64Tables(plugins []osquery.OsqueryPlugin) []osquery.OsqueryPlugin {
	plugins = append(plugins,
		// ARM64 Kolide tables
		appicons.AppIcons(),
	)
	return plugins
}
