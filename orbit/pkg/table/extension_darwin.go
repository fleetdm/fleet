//go:build darwin

package table

import (
	"github.com/kolide/osquery-go"
	"github.com/kolide/osquery-go/plugin/table"

	"github.com/macadmins/osquery-extension/tables/mdm"
)

func platformTables() []osquery.OsqueryPlugin {
	var plugins []osquery.OsqueryPlugin
	plugins = append(plugins, table.NewPlugin("mdm", mdm.MDMInfoColumns(), mdm.MDMInfoGenerate))
	return plugins
}
