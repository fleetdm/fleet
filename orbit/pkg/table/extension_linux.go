//go:build windows

package table

import (
	"github.com/kolide/launcher/pkg/osquery/tables/xconf"
	"github.com/kolide/launcher/pkg/osquery/tables/xrdb"
	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Kolide tables
		xconf.TablePlugin(nil, kolideLogger), // table name is "kolide_xconf"
		xrdb.TablePlugin(nil, kolideLogger),  // table name is "kolide_xrdb"
	}
}
