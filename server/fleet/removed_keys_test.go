package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A removed key's error message comes from pkg/removednames, so every removedkey tag on a request type needs an entry
// there.
func TestRemovedKeyTagsRegistered(t *testing.T) {
	for _, typ := range []any{AppConfig{}, TeamPayload{}, TeamSpec{}, MDMAppleSetupPayload{}} {
		rules := endpointer.ExtractRemovedKeyRules(typ)
		require.NotEmpty(t, rules, "%T", typ)
		for _, rule := range rules {
			_, ok := removednames.Find(removednames.KindJSONKey, rule.Key, "")
			assert.True(t, ok, "%T: %q has no entry in pkg/removednames", typ, rule.Key)
		}
	}
}
