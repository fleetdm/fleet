// based on github.com/kolide/launcher/pkg/osquery/tables
package firefox_preferences

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		filePaths               []string
		expectedResultsFilePath string
		query                   string
	}{
		{
			name: "no path",
		},
		{
			name:                    "single path",
			filePaths:               []string{path.Join("testdata", "prefs.js")},
			expectedResultsFilePath: "testdata/output.single_path.json",
		},
		{
			name:                    "single path with query",
			filePaths:               []string{path.Join("testdata", "prefs.js")},
			expectedResultsFilePath: "testdata/output.single_path_with_query.json",
			query:                   "app.normandy.first_run",
		},
		{
			name:                    "multiple paths",
			filePaths:               []string{path.Join("testdata", "prefs.js"), path.Join("testdata", "prefs2.js")},
			expectedResultsFilePath: "testdata/output.multiple_paths.json",
		},
		{
			name:                    "file with bad data",
			filePaths:               []string{path.Join("testdata", "prefs3.js")},
			expectedResultsFilePath: "testdata/output.file_with_bad_data.json",
		},
	}

	table := Table{logger: zerolog.Nop()}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			constraints := make(map[string][]string)
			constraints["path"] = tt.filePaths
			if tt.query != "" {
				constraints["query"] = append(constraints["query"], tt.query)
			}

			got, _ := table.generate(context.Background(), tablehelpers.MockQueryContext(constraints))

			var want []map[string]string

			if tt.expectedResultsFilePath != "" {
				wantBytes, err := os.ReadFile(tt.expectedResultsFilePath)
				require.NoError(t, err)

				err = json.Unmarshal(wantBytes, &want)
				require.NoError(t, err)
			}

			assert.ElementsMatch(t, want, got)
		})
	}
}
