package fleetctl

import (
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// TestRemovedCommands runs every removed command, with and without the flags and arguments an old invocation may still pass.
func TestRemovedCommands(t *testing.T) {
	entries := removednames.Filter(removednames.KindCommand, 0)
	require.NotEmpty(t, entries)

	for _, entry := range entries {
		t.Run(entry.Old, func(t *testing.T) {
			app := CreateApp(nil, io.Discard, io.Discard, func(*cli.Context, error) {})
			path := strings.Fields(entry.Old)
			for _, args := range [][]string{path, append(slices.Clone(path), "--json", "extra-arg")} {
				require.EqualError(t, app.Run(args), entry.Message())
			}
		})
	}
}
