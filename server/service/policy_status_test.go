package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestGetPolicyStatus(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	policyID := uint(1)
	expectedRuns := []fleet.GetPolicyStatusPolicyRun{
		{
			HostID:              1,
			HostName:            "host1",
			NewStatus:           false,
			ConsecutiveFailures: 1,
			CreatedAt:           time.Now(),
		},
	}
	expectedMeta := &fleet.PaginationMetadata{
		HasNextResults:     false,
		HasPreviousResults: false,
		TotalResults:       1,
	}

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id}}, nil
	}
	ds.GetPolicyStatusFunc = func(ctx context.Context, pid uint, filter fleet.TeamFilter, req fleet.GetPolicyStatusRequest) ([]fleet.GetPolicyStatusPolicyRun, int, *fleet.PaginationMetadata, error) {
		require.Equal(t, policyID, pid)
		require.NotNil(t, filter.User)
		return expectedRuns, 1, expectedMeta, nil
	}

	// Create an admin viewer context to bypass auth checks
	adminViewer := viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}

	t.Run("Requires premium license", func(t *testing.T) {
		ctx = viewer.NewContext(ctx, adminViewer)
		ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierFree})
		_, err := svc.GetPolicyStatus(ctx, &fleet.Policy{PolicyData: fleet.PolicyData{ID: policyID}}, fleet.GetPolicyStatusRequest{})
		require.NoError(t, err) // Service method doesn't check license
	})

	t.Run("Endpoint enforces premium license", func(t *testing.T) {
		ctx = viewer.NewContext(ctx, adminViewer)
		ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierFree})
		_, err := getPolicyStatusEndpoint(ctx, &fleet.GetPolicyStatusRequest{PolicyID: policyID}, svc)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
	})

	t.Run("Success", func(t *testing.T) {
		ctx = viewer.NewContext(ctx, adminViewer)
		ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		respRaw, err := getPolicyStatusEndpoint(ctx, &fleet.GetPolicyStatusRequest{PolicyID: policyID}, svc)
		require.NoError(t, err)

		resp, ok := respRaw.(*fleet.GetPolicyStatusResponse)
		require.True(t, ok)
		require.NoError(t, resp.Err)
		require.Equal(t, 1, resp.Count)
		require.Equal(t, expectedMeta, resp.Meta)
		require.Equal(t, expectedRuns, resp.Runs)
	})
}
