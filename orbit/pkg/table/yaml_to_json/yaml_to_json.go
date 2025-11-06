package yaml_to_json

import (
	"context"
	"errors"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/osquery/osquery-go/plugin/table"
)

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("yaml"),
		table.TextColumn("json"),
	}
}

func GenerateFunc(_ context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	yamlContent := ""
	if constraints, ok := queryContext.Constraints["yaml"]; ok {
		for _, constraint := range constraints.Constraints {
			if constraint.Operator == table.OperatorEquals {
				yamlContent = constraint.Expression
			}
		}
	}
	if yamlContent == "" {
		return nil, errors.New("missing yaml column constraint; e.g. WHERE yaml = 'key: value'")
	}

	jsonData, err := yaml.YAMLToJSON([]byte(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON: %w", err)
	}

	return []map[string]string{{
		"yaml": yamlContent,
		"json": string(jsonData),
	}}, nil
}
