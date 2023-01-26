//go:build darwin
// +build darwin

package logging_and_auditing

import (
	"context"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"strings"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.IntegerColumn("security_auditing_flags"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {

	flagsLine, err := tbl_common.FindLineInFileContainingString("/etc/security/audit_control", "flags")
	if err != nil {
		return []map[string]string{}, err
	}
	flags := strings.Replace(flagsLine, "flags:", "", 1)

	return []map[string]string{
		{"security_auditing_flags": flags},
	}, nil
}
