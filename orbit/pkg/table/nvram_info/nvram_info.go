//go:build darwin
// +build darwin

package nvram_info

import (
	"context"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"strings"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("amfi_enabled"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	amfiEnabled, err := getAMFIEnabled(ctx)

	return []map[string]string{
		{"amfi_enabled": amfiEnabled},
	}, err
}

func getAMFIEnabled(ctx context.Context) (amfiEnabled string, err error) {
	res, err := tbl_common.RunCommand(ctx, "/usr/sbin/nvram", "-p")
	amfiEnabled = ""
	if err == nil {
		amfiEnabled = "0"
		if !strings.Contains(res, "amfi_get_out_of_my_way=1") {
			amfiEnabled = "1"
		}
	}
	return amfiEnabled, err
}
