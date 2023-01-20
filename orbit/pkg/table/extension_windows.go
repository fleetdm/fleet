//go:build windows

package table

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/mdm"
	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func platformTables() []osquery.OsqueryPlugin {
	return []osquery.OsqueryPlugin{
		// Fleet tables
		table.NewPlugin("mdm_bridge", mdmbridge.Columns(), mdmbridge.Generate),
	}
}
