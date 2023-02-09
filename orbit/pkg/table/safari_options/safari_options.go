//go:build darwin
// +build darwin

package safari_options

import (
	"context"
	"errors"
	"fmt"
	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("user_name"), // required
		table.TextColumn("show_full_url_in_smart_search_field"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	userName := ""
	if userName = getUserNameFromConstraints(queryContext); userName == "" {
		return nil, errors.New("missing user_name")
	}

	res, err := tbl_common.RunCommand(ctx, "/usr/bin/defaults", "read", "/Users/"+userName+"/Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari", "ShowFullURLInSmartSearchField")
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	return []map[string]string{{
		"user_name":                           userName,
		"show_full_url_in_smart_search_field": res,
	}}, nil
}

func getUserNameFromConstraints(queryContext table.QueryContext) (userName string) {
	userName = ""
	if constraints, ok := queryContext.Constraints["user_name"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				userName = constraint.Expression
			}
		}
	}
	return
}
