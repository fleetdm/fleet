package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/stretchr/testify/require"
)

func TestRemovedKeyMessage(t *testing.T) {
	// A registered name takes its message from its entry, whatever format the entry uses.
	entry, ok := removednames.Find(removednames.KindJSONKey, "team_name", "")
	require.True(t, ok)
	require.EqualError(t, &endpointer.RemovedKeyError{Key: "team_name"}, entry.Message())

	// Any other name, or a registered one in another letter case, gets the standard Fleet 5 wording.
	require.EqualError(t, &endpointer.RemovedKeyError{Key: "Require_All_Software_Macos"},
		`The "Require_All_Software_Macos" parameter is not available anymore. What to use instead: `+removednames.GuideURL)
}
