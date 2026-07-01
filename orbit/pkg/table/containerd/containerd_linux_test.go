//go:build linux

package containerd

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/stretchr/testify/require"
)

func TestSocketPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		constraints map[string][]string
		expected    string
	}{
		{
			name:        "return legacy default",
			constraints: nil,
			expected:    defaultSocketPath,
		},
		{
			name:        "return explicit socket",
			constraints: map[string][]string{socketPathCol: {"/run/k3s/containerd/containerd.sock"}},
			expected:    "/run/k3s/containerd/containerd.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, resolveSocketPath(tablehelpers.MockQueryContext(tt.constraints)))
		})
	}
}
