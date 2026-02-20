package fleetd_macadmins_extensions

import (
	"context"
	"runtime/debug"

	"github.com/osquery/osquery-go/plugin/table"
)

const macadminsModulePath = "github.com/macadmins/osquery-extension"

// TablePlugin returns an osquery table plugin that reports which version of
// macadmins/osquery-extension fleetd was built with.
func TablePlugin() *table.Plugin {
	columns := []table.ColumnDefinition{
		table.TextColumn("version"),
	}
	return table.NewPlugin("fleetd_macadmins_extensions", columns, generate)
}

func generate(_ context.Context, _ table.QueryContext) ([]map[string]string, error) {
	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == macadminsModulePath {
				version = dep.Version
				if dep.Replace != nil {
					version = dep.Replace.Version
				}
				break
			}
		}
	}
	return []map[string]string{{"version": version}}, nil
}
