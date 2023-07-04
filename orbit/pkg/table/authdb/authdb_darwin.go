//go:build darwin
// +build darwin

package authdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/osquery/osquery-go/plugin/table"
	"howett.net/plist"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("right_name"), // required
		table.TextColumn("json_result"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	rightName := ""
	if constraints, ok := queryContext.Constraints["right_name"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				rightName = constraint.Expression
			}
		}
	}
	if rightName == "" {
		return nil, errors.New("missing right_name")
	}

	cmd := exec.Command("/usr/bin/security", "authorizationdb", "read", rightName)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("generate failed: %w", err)
	}

	result, err := parseAuthDBReadOutput(out)
	if err != nil {
		return nil, fmt.Errorf("parse authorizationdb read output: %w", err)
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal json result: %w", err)
	}

	return []map[string]string{{
		"right_name":  rightName,
		"json_result": string(jsonResult),
	}}, nil
}

func parseAuthDBReadOutput(out []byte) (map[string]interface{}, error) {
	var m map[string]interface{}
	if _, err := plist.Unmarshal(out, &m); err != nil {
		return nil, err
	}
	return m, nil
}
