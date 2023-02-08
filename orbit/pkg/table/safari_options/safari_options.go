//go:build darwin
// +build darwin

package safari_options

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/osquery/osquery-go/plugin/table"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("user_name"), // required
		table.TextColumn("Show_full_url_in_smart_search_field"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	userName := ""
	if constraints, ok := queryContext.Constraints["userName"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				userName = constraint.Expression
			}
		}
	}
	if userName == "" {
		return nil, errors.New("missing userName")
	}

	//cmd := exec.CommandContext(ctx, "/usr/bin/defaults", "read", "/Users/sharonkatz/Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari", "ShowFullURLInSmartSearchField")
	cmd := exec.CommandContext(ctx, "/usr/bin/defaults", "/Users/"+userName+"/Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari", "ShowFullURLInSmartSearchField")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	return []map[string]string{{
		"userName":                            rightName,
		"Show_full_url_in_smart_search_field": string(out),
	}}, nil
}
