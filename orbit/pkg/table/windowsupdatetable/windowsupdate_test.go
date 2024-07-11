//go:build windows
// +build windows

// based on github.com/kolide/launcher/pkg/osquery/tables
package windowsupdatetable

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		queryFunc queryFuncType
	}{
		{name: "updates", queryFunc: queryUpdates},
		{name: "history", queryFunc: queryHistory},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := Table{
				logger:    zerolog.Nop(),
				queryFunc: tt.queryFunc,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			// ci doesn't return data, but we can, at least, check that the underlying API doesn't error.
			_, err := table.generate(ctx, tablehelpers.MockQueryContext(nil))
			require.NoError(t, err, "generate")
		})
	}
}
