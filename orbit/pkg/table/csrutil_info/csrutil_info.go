//go:build darwin
// +build darwin

package csrutil_info

import (
	"context"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"strings"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("ssv_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	SSVEnabled, err := getSSVEnabled(ctx)
	return []map[string]string{
		{"ssv_enabled": SSVEnabled},
	}, err
}

func getSSVEnabled(ctx context.Context) (SSVEnabled string, err error) {
	res, err := tbl_common.RunCommand(ctx, "/usr/bin/csrutil", "authenticated-root", "status")
	SSVEnabled = ""
	if err == nil {
		SSVEnabled = "0"
		if strings.Contains(res, "Authenticated Root status: enabled") {
			SSVEnabled = "1"
		}
	}
	return SSVEnabled, err
}
