package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestCamtasiaVersionTransformer(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"26.1.0", "2026.1.0"},
		{"26.0.2", "2026.0.2"},
		{"2026.1.0", "2026.1.0"},
		{"2025.12.0", "2025.12.0"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			app := &maintained_apps.FMAManifestApp{Version: tt.in}
			out, err := CamtasiaVersionTransformer(app)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, out.Version)
		})
	}
}
