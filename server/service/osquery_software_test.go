package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/stretchr/testify/require"
)

func TestPreservedSoftwareSources(t *testing.T) {
	aiSources := []string{"ai_clis", "mcp_servers", "ai_skills"}
	sentToMacOS := &hostDetailQueryConfig{detailQueries: map[string]osquery_utils.DetailQuery{
		"software_ai_tools": {Platforms: []string{"darwin"}},
	}}
	notSent := &hostDetailQueryConfig{detailQueries: map[string]osquery_utils.DetailQuery{}}

	for _, tc := range []struct {
		name          string
		cfg           *hostDetailQueryConfig
		platform      string
		skipped       []string
		orbitInfo     *fleet.HostOrbitInfo
		orbitInfoErr  error
		expected      []string
		expectLookup  bool
		expectLogging bool
	}{
		{
			name:     "nothing skipped",
			cfg:      sentToMacOS,
			platform: "darwin",
			skipped:  nil,
		},
		{
			name:     "a skipped query no fleetd table backs",
			cfg:      sentToMacOS,
			platform: "darwin",
			skipped:  []string{"software_vscode_extensions"},
		},
		{
			name:     "the query was not sent to the host",
			cfg:      notSent,
			platform: "darwin",
			skipped:  []string{"software_ai_tools"},
		},
		{
			name:     "the query does not run on the host's platform",
			cfg:      sentToMacOS,
			platform: "chrome",
			skipped:  []string{"software_ai_tools"},
		},
		{
			name:         "host without fleetd",
			cfg:          sentToMacOS,
			platform:     "darwin",
			skipped:      []string{"software_ai_tools"},
			orbitInfoErr: newNotFoundError(),
			expectLookup: true,
		},
		{
			name:          "orbit info cannot be read: rows are kept rather than risked",
			cfg:           sentToMacOS,
			platform:      "darwin",
			skipped:       []string{"software_vscode_extensions", "software_ai_tools"},
			orbitInfoErr:  errors.New("replica unavailable"),
			expected:      aiSources,
			expectLookup:  true,
			expectLogging: true,
		},
		{
			name:         "fleetd version cannot be parsed",
			cfg:          sentToMacOS,
			platform:     "darwin",
			skipped:      []string{"software_ai_tools"},
			orbitInfo:    &fleet.HostOrbitInfo{Version: "dev"},
			expectLookup: true,
		},
		{
			name:         "fleetd older than the table",
			cfg:          sentToMacOS,
			platform:     "darwin",
			skipped:      []string{"software_ai_tools"},
			orbitInfo:    &fleet.HostOrbitInfo{Version: "1.58.2"},
			expectLookup: true,
		},
		{
			name:          "fleetd that shipped the table",
			cfg:           sentToMacOS,
			platform:      "darwin",
			skipped:       []string{"software_vscode_extensions", "software_ai_tools"},
			orbitInfo:     &fleet.HostOrbitInfo{Version: "1.59.0"},
			expected:      aiSources,
			expectLookup:  true,
			expectLogging: true,
		},
		{
			name:          "newer fleetd",
			cfg:           sentToMacOS,
			platform:      "darwin",
			skipped:       []string{"software_ai_tools"},
			orbitInfo:     &fleet.HostOrbitInfo{Version: "1.62.1"},
			expected:      aiSources,
			expectLookup:  true,
			expectLogging: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.GetHostOrbitInfoFunc = func(ctx context.Context, hostID uint) (*fleet.HostOrbitInfo, error) {
				return tc.orbitInfo, tc.orbitInfoErr
			}
			var logs bytes.Buffer
			svc := &Service{ds: ds, logger: slog.New(slog.NewTextHandler(&logs, nil))}
			host := &fleet.Host{ID: 1, Platform: tc.platform}

			got := svc.preservedSoftwareSources(t.Context(), host, tc.cfg, tc.skipped)
			require.Equal(t, tc.expected, got)
			require.Equal(t, tc.expectLookup, ds.GetHostOrbitInfoFuncInvoked)
			if tc.expectLogging {
				require.Contains(t, logs.String(), "keeping previously reported rows")
			} else {
				require.NotContains(t, logs.String(), "keeping previously reported rows")
			}
		})
	}
}
