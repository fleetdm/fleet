package integrationtest

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestAdditionalMetadata(t *testing.T) {
	t.Run("saves additional data about the host if present", func(t *testing.T) {
		ctx := t.Context()
		eh := ctxerr.MockHandler{}
		ctx = ctxerr.NewContext(ctx, eh)

		h := &fleet.Host{Platform: "test_platform", OsqueryVersion: "5.0"}
		hctx := host.NewContext(ctx, h)
		// Register the host as an error attribute provider
		hctx = ctxerr.AddErrorAttributeProvider(hctx, &host.HostAttributeProvider{Host: h})
		err := ctxerr.New(hctx, "with host context")

		// Use the public LogFields() method to verify the metadata
		var ferr *ctxerr.FleetError
		require.ErrorAs(t, err, &ferr)
		fields := ferr.LogFields()

		// The fields should contain host info and timestamp
		fieldMap := make(map[string]any)
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			require.True(t, ok, "expected string key")
			fieldMap[key] = fields[i+1]
		}

		require.Contains(t, fieldMap, "host")
		hostData, ok := fieldMap["host"].(map[string]any)
		require.True(t, ok, "expected host to be a map")
		require.Equal(t, "test_platform", hostData["platform"])
		require.Equal(t, "5.0", hostData["osquery_version"])
	})

	t.Run("saves additional data about the viewer if present", func(t *testing.T) {
		ctx := t.Context()
		eh := ctxerr.MockHandler{}
		ctx = ctxerr.NewContext(ctx, eh)

		v := viewer.Viewer{Session: &fleet.Session{ID: 1}, User: &fleet.User{SSOEnabled: true}}
		vctx := viewer.NewContext(ctx, v)
		// Register the viewer as an error attribute provider
		vctx = ctxerr.AddErrorAttributeProvider(vctx, &v)
		err := ctxerr.New(vctx, "with viewer context")

		// Use the public LogFields() method to verify the metadata
		var ferr *ctxerr.FleetError
		require.ErrorAs(t, err, &ferr)
		fields := ferr.LogFields()

		// The fields should contain viewer info and timestamp
		fieldMap := make(map[string]any)
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			require.True(t, ok, "expected string key")
			fieldMap[key] = fields[i+1]
		}

		require.Contains(t, fieldMap, "viewer")
		viewerData, ok := fieldMap["viewer"].(map[string]any)
		require.True(t, ok, "expected viewer to be a map")
		require.Equal(t, true, viewerData["is_logged_in"])
		require.Equal(t, true, viewerData["sso_enabled"])
	})
}
