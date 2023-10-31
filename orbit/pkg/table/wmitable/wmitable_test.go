//go:build windows
// +build windows

// based on github.com/kolide/launcher/pkg/osquery/tables
package wmitable

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueries(t *testing.T) {
	t.Parallel()

	wmiTable := Table{logger: log.NewNopLogger()}

	tests := []struct {
		name        string
		class       string
		properties  []string
		namespace   string
		whereClause string
		minRows     int
		keyNames    []string
		noData      bool
		err         bool
	}{
		{
			name:       "simple operating system query",
			class:      "Win32_OperatingSystem",
			properties: []string{"name,version"},
			minRows:    1,
		},
		{
			name:       "queries with non-string types",
			class:      "Win32_OperatingSystem",
			properties: []string{"InstallDate,primary"},
			minRows:    1,
		},
		{
			name:       "multiple operating system query",
			class:      "Win32_OperatingSystem",
			properties: []string{"name", "version"},
			minRows:    1,
		},
		{
			name:       "wmi properties with an array",
			class:      "Win32_SystemEnclosure",
			properties: []string{"ChassisTypes"},
			minRows:    1,
			keyNames:   []string{"0"}, // arrays come back with the position as the key
		},
		{
			name:       "process query",
			class:      "WIN32_process",
			properties: []string{"Caption,CommandLine,CreationDate,Name,Handle,ReadTransferCount"},
			minRows:    10,
		},
		{
			name:       "bad class name",
			class:      "Win32_OperatingSystem;",
			properties: []string{"name,version"},
			err:        true,
		},
		{
			name:       "bad properties",
			class:      "Win32_OperatingSystem",
			properties: []string{"name,ver;sion"},
			err:        true,
		},

		{
			name:       "bad namespace",
			class:      "Win32_OperatingSystem",
			properties: []string{"name,version"},
			namespace:  `root\unknown\namespace`,
			noData:     true,
		},
		{
			name:       "different namespace",
			class:      "MSKeyboard_PortInformation",
			properties: []string{"ConnectorType,FunctionKeys,Indicators"},
			namespace:  `root\wmi`,
			minRows:    3,
		},
		{
			name:       "different namespace, double slash",
			class:      "MSKeyboard_PortInformation",
			properties: []string{"ConnectorType,FunctionKeys,Indicators"},
			namespace:  `root\\wmi`,
			minRows:    3,
		},
		{
			name:        "where clause non-existent file",
			class:       "CIM_DataFile",
			properties:  []string{"name", "hidden"},
			whereClause: `name = 'c:\\does\\not\\exist'`,
			noData:      true,
		},
		{
			name:        "where clause",
			class:       "CIM_DataFile",
			properties:  []string{"name", "hidden"},
			whereClause: `name = 'c:\\windows\\system32\\notepad.exe'`,
			minRows:     1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// making this parallel causing the unit test to occasionally fail

			mockQC := tablehelpers.MockQueryContext(map[string][]string{
				"class":       {tt.class},
				"properties":  tt.properties,
				"namespace":   {tt.namespace},
				"whereclause": {tt.whereClause},
			})

			rows, err := wmiTable.generate(context.TODO(), mockQC)

			if tt.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.noData {
				assert.Empty(t, rows, "Expected no results")
				return
			}

			// It's hard to model what we expect to get
			// from a random windows machine. So let's
			// just check for non-empty data.
			assert.GreaterOrEqual(t, len(rows), tt.minRows, "Expected minimum rows")
			for _, row := range rows {
				// this has gone through dataflatten. Test for various expected results
				require.Contains(t, row, "class", "class column")
				require.Equal(t, tt.class, row["class"], "class name is equal")

				for _, columnName := range []string{"fullkey", "parent", "key", "value"} {
					require.Contains(t, row, columnName, "%s column", columnName)
					assert.NotEmpty(t, tt.class, row[columnName], "%s column not empty", columnName)
				}

				if tt.keyNames != nil {
					assert.Contains(t, tt.keyNames, row["key"], "key is in keyNames")
				}
			}
		})
	}
}
