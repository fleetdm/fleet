package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAddFMASvc struct {
	fleet.Service
	err error
}

func (s fakeAddFMASvc) AddFleetMaintainedApp(ctx context.Context, _ *uint, _ uint, _, _, _, _ string, _ bool, _ bool, _, _, _ []string) (uint, error) {
	return 0, s.err
}

func TestAddFleetMaintainedAppEndpointErrorTranslation(t *testing.T) {
	ctx := t.Context()
	// inner error mirrors the real wrap chain from DownloadInstaller
	wrap := func(sentinel error) error {
		return ctxerr.Wrap(ctx, ctxerr.Wrapf(ctx, sentinel, "reading installer %q contents", "https://cdn/app.pkg"), "downloading app installer")
	}

	cases := []struct {
		name    string
		in      error
		wantMsg string // expected GatewayError.Message
		want504 bool
	}{
		{"canceled", wrap(context.Canceled), fleet.AddMaintainedAppCanceledErrMsg, true},
		{"deadline", wrap(context.DeadlineExceeded), fleet.AddMaintainedAppTimeoutErrMsg, true},
		{"other", wrap(errors.New("boom")), "", false}, // passthrough, untouched
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := addFleetMaintainedAppEndpoint(ctx, &addFleetMaintainedAppRequest{}, fakeAddFMASvc{err: tc.in})
			require.NoError(t, err)
			gotErr := resp.(*addFleetMaintainedAppResponse).Err
			require.Error(t, gotErr)

			if tc.want504 {
				var ge *fleet.GatewayError
				require.ErrorAs(t, gotErr, &ge)
				require.Equal(t, http.StatusGatewayTimeout, ge.StatusCode())
				require.Equal(t, tc.wantMsg, ge.Message)
				// GatewayError has no Unwrap, so errors.Is can't reach the sentinel
				// and EncodeError's 499 branch won't fire.
				require.NotErrorIs(t, gotErr, context.Canceled)
				require.NotErrorIs(t, gotErr, context.DeadlineExceeded)
			} else {
				var ge *fleet.GatewayError
				require.NotErrorAs(t, gotErr, &ge) // untouched passthrough
			}
		})
	}
}

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
