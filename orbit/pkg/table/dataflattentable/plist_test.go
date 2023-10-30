package dataflattentable

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kolide/launcher/pkg/dataflatten"
	"github.com/kolide/launcher/pkg/osquery/tables/tablehelpers"
	"github.com/stretchr/testify/require"
)

// TestPlist runs some real-world tests against sample plist data.
func TestPlist(t *testing.T) {
	t.Parallel()
	plistTable := Table{flattenFileFunc: dataflatten.PlistFile}

	var tests = []struct {
		paths    []string
		queries  []string
		expected []map[string]string
		err      bool
	}{
		{
			err: true,
		},
		{
			paths:   []string{filepath.Join("testdata", "NetworkInterfaces.plist")},
			queries: []string{"Interfaces/#BSD Name/SCNetworkInterfaceType/FireWire"},
			expected: []map[string]string{
				{
					"fullkey": "Interfaces/fw0/SCNetworkInterfaceType",
					"key":     "SCNetworkInterfaceType",
					"parent":  "Interfaces/fw0",
					"value":   "FireWire",
				}},
		},
		{
			paths: []string{filepath.Join("testdata", "com.apple.launchservices.secure.plist")},
			queries: []string{
				"LSHandlers/LSHandlerURLScheme=>htt*/LSHandlerRole*",
				"LSHandlers/LSHandlerContentType=>*html/LSHandlerRole*",
			},
			expected: []map[string]string{
				{"fullkey": "LSHandlers/5/LSHandlerRoleAll", "key": "LSHandlerRoleAll", "parent": "LSHandlers/5", "value": "com.choosyosx.choosy"},
				{"fullkey": "LSHandlers/6/LSHandlerRoleAll", "key": "LSHandlerRoleAll", "parent": "LSHandlers/6", "value": "com.choosyosx.choosy"},
				{"fullkey": "LSHandlers/7/LSHandlerRoleAll", "key": "LSHandlerRoleAll", "parent": "LSHandlers/7", "value": "com.choosyosx.choosy"},
				{"fullkey": "LSHandlers/8/LSHandlerRoleAll", "key": "LSHandlerRoleAll", "parent": "LSHandlers/8", "value": "com.google.chrome"},
			},
		},
	}

	for _, tt := range tests {
		mockQC := tablehelpers.MockQueryContext(map[string][]string{
			"path":  tt.paths,
			"query": tt.queries,
		})

		rows, err := plistTable.generate(context.TODO(), mockQC)
		if tt.err {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)

		// delete the path and query keys, so we don't need to enumerate them in the test case
		for _, row := range rows {
			delete(row, "path")
			delete(row, "query")
		}

		// Despite being an array. data is returned unordered. Sort it.
		sort.SliceStable(tt.expected, func(i, j int) bool { return tt.expected[i]["fullkey"] < tt.expected[j]["fullkey"] })
		sort.SliceStable(rows, func(i, j int) bool { return rows[i]["fullkey"] < rows[j]["fullkey"] })

		require.EqualValues(t, tt.expected, rows)
	}

}
