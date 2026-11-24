package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/stretchr/testify/assert"
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

// A removed key's error message comes from pkg/removednames, so every removedkey tag on a request type needs an entry
// there.
func TestRemovedKeyTagsRegistered(t *testing.T) {
	for _, typ := range []any{fleet.AppConfig{}, fleet.TeamPayload{}, fleet.TeamSpec{}, fleet.MDMAppleSetupPayload{}} {
		rules := endpointer.ExtractRemovedKeyRules(typ)
		require.NotEmpty(t, rules, "%T", typ)
		for _, rule := range rules {
			_, ok := removednames.Find(removednames.KindJSONKey, rule.Key, "")
			assert.True(t, ok, "%T: %q has no entry in pkg/removednames", typ, rule.Key)
		}
	}
}
