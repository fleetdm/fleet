//go:build linux

package table

import (
	"github.com/kolide/launcher/pkg/osquery/tables/cryptsetup"
	"github.com/osquery/osquery-go"
)

func PlatformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Kolide extensions.
		cryptsetup.TablePlugin(kolideLogger),
	}
}
