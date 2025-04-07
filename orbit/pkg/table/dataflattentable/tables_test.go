// based on github.com/kolide/launcher/pkg/osquery/tables
package dataflattentable

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/dataflatten"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestDataFlattenTable_Animals tests the basic generation
// functionality for both plist and json parsing using the mock
// animals data.
func TestDataFlattenTablePlist_Animals(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	// Test plist parsing both the json and xml forms
	testTables := map[string]Table{
		"plist": {logger: logger, flattenFileFunc: dataflatten.PlistFile},
		"xml":   {logger: logger, flattenFileFunc: dataflatten.PlistFile},
		"json":  {logger: logger, flattenFileFunc: dataflatten.JsonFile},
	}

	tests := []struct {
		queries  []string
		expected []map[string]string
	}{
		{
			queries: []string{
				"metadata",
			},
			expected: []map[string]string{
				{"fullkey": "metadata/testing", "key": "testing", "parent": "metadata", "value": "true"},
				{"fullkey": "metadata/version", "key": "version", "parent": "metadata", "value": "1.0.1"},
			},
		},
		{
			queries: []string{
				"users/name=>*Aardvark/id",
				"users/name=>*Chipmunk/id",
			},
			expected: []map[string]string{
				{"fullkey": "users/0/id", "key": "id", "parent": "users/0", "value": "1"},
				{"fullkey": "users/2/id", "key": "id", "parent": "users/2", "value": "3"},
			},
		},
	}

	for _, tt := range tests {
		for dataType, tableFunc := range testTables {
			testFile := filepath.Join("testdata", "animals."+dataType)
			mockQC := tablehelpers.MockQueryContext(map[string][]string{
				"path":  {testFile},
				"query": tt.queries,
			})

			rows, err := tableFunc.generate(context.TODO(), mockQC)

			require.NoError(t, err)

			// delete the path and query keys, so we don't need to enumerate them in the test case
			for _, row := range rows {
				delete(row, "path")
				delete(row, "query")
			}

			// Despite being an array. data is returned unordered. Sort it.
			sort.SliceStable(tt.expected, func(i, j int) bool { return tt.expected[i]["fullkey"] < tt.expected[j]["fullkey"] })
			sort.SliceStable(rows, func(i, j int) bool { return rows[i]["fullkey"] < rows[j]["fullkey"] })

			require.EqualValues(t, tt.expected, rows, "table type %s test", dataType)
		}
	}
}

func TestDataFlattenTables(t *testing.T) {
	t.Parallel()

	logger := zerolog.Nop()

	tests := []struct {
		testTables   map[string]Table
		testFile     string
		queries      []string
		expectedRows int
		expectNoData bool
	}{
		// xml
		{
			testTables:   map[string]Table{"xml": {logger: logger, flattenFileFunc: dataflatten.XmlFile}},
			testFile:     path.Join("testdata", "simple.xml"),
			expectedRows: 6,
		},
		{
			testTables:   map[string]Table{"xml": {logger: logger, flattenFileFunc: dataflatten.XmlFile}},
			testFile:     path.Join("testdata", "simple.xml"),
			queries:      []string{"simple/Items"},
			expectedRows: 3,
		},
		{
			testTables:   map[string]Table{"xml": {logger: logger, flattenFileFunc: dataflatten.XmlFile}},
			testFile:     path.Join("testdata", "simple.xml"),
			queries:      []string{"this/does/not/exist"},
			expectNoData: true,
		},

		// ini
		{
			testTables:   map[string]Table{"ini": {logger: logger, flattenFileFunc: dataflatten.IniFile}},
			testFile:     path.Join("testdata", "secdata.ini"),
			expectedRows: 87,
		},
		{
			testTables:   map[string]Table{"ini": {logger: logger, flattenFileFunc: dataflatten.IniFile}},
			testFile:     path.Join("testdata", "secdata.ini"),
			queries:      []string{"Registry Values"},
			expectedRows: 59,
		},
		{
			testTables:   map[string]Table{"ini": {logger: logger, flattenFileFunc: dataflatten.IniFile}},
			testFile:     path.Join("testdata", "secdata.ini"),
			queries:      []string{"this/does/not/exist"},
			expectNoData: true,
		},
	}

	for testN, tt := range tests {
		tt := tt
		for tableName, testTable := range tt.testTables {
			tableName, testTable := tableName, testTable

			t.Run(fmt.Sprintf("%d/%s", testN, tableName), func(t *testing.T) {
				t.Parallel()

				mockQC := tablehelpers.MockQueryContext(map[string][]string{
					"path":  {tt.testFile},
					"query": tt.queries,
				})

				rows, err := testTable.generate(context.TODO(), mockQC)
				require.NoError(t, err)

				if tt.expectNoData {
					require.Len(t, rows, 0)
				} else {
					require.Len(t, rows, tt.expectedRows)
				}
			})
		}
	}
}
