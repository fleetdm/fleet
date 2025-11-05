//go:build darwin
// +build darwin

package testtable

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"gopkg.in/yaml.v3"
)

// Columns defines the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("path"),  // required
		table.TextColumn("key"),   // required
		table.TextColumn("value"), // result
	}
}

// Generate is called at query time to produce table rows.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// Get the path constraint
	var path string
	if constraints, ok := queryContext.Constraints["path"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				path = constraint.Expression
				break
			}
		}
	}
	if path == "" {
		return nil, errors.New("missing 'path' constraint")
	}

	// Get the key constraint
	var key string
	if constraints, ok := queryContext.Constraints["key"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				key = constraint.Expression
				break
			}
		}
	}
	if key == "" {
		return nil, errors.New("missing 'key' constraint")
	}

	// Get the YAML value
	val, err := GetYAMLValue(path, key)
	if err != nil {
		// Return an empty result but no fatal error
		return []map[string]string{{
			"path":  path,
			"key":   key,
			"value": fmt.Sprintf("error: %v", err),
		}}, nil
	}

	// Convert value to string for osquery output
	valStr := fmt.Sprintf("%v", val)

	return []map[string]string{{
		"path":  path,
		"key":   key,
		"value": valStr,
	}}, nil
}

// GetYAMLValue reads a YAML file and returns the nested key’s value.
func GetYAMLValue(path string, key string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var content map[string]interface{}
	if err := yaml.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	keys := strings.Split(key, ".")
	var current interface{} = content

	for _, k := range keys {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path invalid at '%s'", k)
		}
		current, ok = m[k]
		if !ok {
			return nil, errors.New("key not found")
		}
	}

	return current, nil
}
