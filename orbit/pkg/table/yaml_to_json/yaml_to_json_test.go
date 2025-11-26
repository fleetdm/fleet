package yaml_to_json

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateFunc(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		wantErr      bool
		validateJSON bool
	}{
		{
			name: "simple YAML",
			yamlContent: `name: test
version: 1.0
features:
  - feature1
  - feature2`,
			wantErr:      false,
			validateJSON: true,
		},
		{
			name: "complex nested YAML",
			yamlContent: `server:
  host: localhost
  port: 8080
  ssl:
    enabled: true
    cert: /path/to/cert
database:
  type: postgres
  connection:
    host: db.example.com
    port: 5432`,
			wantErr:      false,
			validateJSON: true,
		},
		{
			name: "YAML with different types",
			yamlContent: `string: hello
number: 42
float: 3.14
bool: true
null_value: null`,
			wantErr:      false,
			validateJSON: true,
		},
		{
			name:        "invalid YAML",
			yamlContent: "{ invalid: [[[",
			wantErr:     true,
		},
		{
			name:         "simple key-value",
			yamlContent:  "key: value",
			wantErr:      false,
			validateJSON: true,
		},
		{
			name: "array at root",
			yamlContent: `- item1
- item2
- item3`,
			wantErr:      false,
			validateJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryContext := table.QueryContext{
				Constraints: map[string]table.ConstraintList{
					"yaml": {
						Constraints: []table.Constraint{
							{
								Operator:   table.OperatorEquals,
								Expression: tt.yamlContent,
							},
						},
					},
				},
			}

			results, err := GenerateFunc(context.Background(), queryContext)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, results, 1)
			require.Equal(t, tt.yamlContent, results[0]["yaml"])

			if tt.validateJSON {
				// Validate that the output is valid JSON
				var jsonData interface{}
				err := json.Unmarshal([]byte(results[0]["json"]), &jsonData)
				require.NoError(t, err, "output should be valid JSON")
			}
		})
	}
}

func TestGenerateFuncMissingYamlConstraint(t *testing.T) {
	// Test without yaml constraint
	queryContext := table.QueryContext{
		Constraints: map[string]table.ConstraintList{},
	}

	results, err := GenerateFunc(context.Background(), queryContext)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing yaml column constraint")
	require.Nil(t, results)
}
