package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddFleetMaintainedAppDecodeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantTeamID *uint
		wantErr    string
	}{
		{
			name:       "fleet_id accepted",
			body:       `{"fleet_id": 42, "fleet_maintained_app_id": 1}`,
			wantTeamID: ptr.Uint(42),
		},
		{
			name:       "team_id still accepted",
			body:       `{"team_id": 7, "fleet_maintained_app_id": 1}`,
			wantTeamID: ptr.Uint(7),
		},
		{
			name:       "neither provided",
			body:       `{"fleet_maintained_app_id": 1}`,
			wantTeamID: nil,
		},
		{
			name:    "both provided is an error",
			body:    `{"team_id": 1, "fleet_id": 2, "fleet_maintained_app_id": 1}`,
			wantErr: `Specify only one of "team_id" or "fleet_id"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/", io.NopCloser(bytes.NewBufferString(tt.body)))
			require.NoError(t, err)

			result, err := addFleetMaintainedAppRequest{}.DecodeRequest(context.Background(), r)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			req := result.(*addFleetMaintainedAppRequest)
			if tt.wantTeamID == nil {
				assert.Nil(t, req.TeamID)
			} else {
				require.NotNil(t, req.TeamID)
				assert.Equal(t, *tt.wantTeamID, *req.TeamID)
			}
			// FleetID should always be nil after normalization
			assert.Nil(t, req.FleetID)
		})
	}
}
