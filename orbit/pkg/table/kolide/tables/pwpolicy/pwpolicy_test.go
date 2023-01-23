//go:build darwin
// +build darwin

package pwpolicy

import (
	"context"
	"os/exec"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/kolide/tables/tablehelpers"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		file        string
		queryClause []string
		len         int
		err         bool
	}{
		{
			name: "no data, just languages",
			file: path.Join("testdata", "empty.output"),
			len:  41,
		},
		{
			file: path.Join("testdata", "test1.output"),
			len:  148,
		},
		{
			file:        path.Join("testdata", "test1.output"),
			queryClause: []string{"policyCategoryAuthentication"},
			len:         8,
		},
	}

	for _, tt := range tests {
		tt := tt
		testTable := &Table{
			logger: log.NewNopLogger(),
			execCC: execFaker(tt.file),
		}

		testName := tt.file + "/" + tt.name
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			mockQC := tablehelpers.MockQueryContext(map[string][]string{
				"query": tt.queryClause,
			})

			rows, err := testTable.generate(context.TODO(), mockQC)

			if tt.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, tt.len, len(rows))
		})
	}
}

func execFaker(filename string) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "/bin/cat", filename)
	}
}
