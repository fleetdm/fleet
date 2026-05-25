package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/gocarina/gocsv"
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

func TestExportPolicyStatus(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	policyID := uint(7)
	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	runsWithAutomations := []fleet.GetPolicyStatusPolicyRun{
		{
			HostID:              1,
			HostName:            "host-fail",
			NewStatus:           false,
			ConsecutiveFailures: 3,
			CreatedAt:           now,
			AutomationExecutions: []fleet.GetPolicyStatusAutomationExecution{
				{Type: "webhook", Status: "failed", ErrorMessage: "timeout 504"},
				{Type: "jira", Status: "success", ErrorMessage: ""},
			},
		},
		{
			HostID:              2,
			HostName:            "host-pass",
			NewStatus:           true,
			ConsecutiveFailures: 0,
			CreatedAt:           now,
			// no automations
		},
	}

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		return &fleet.Policy{PolicyData: fleet.PolicyData{ID: id}}, nil
	}

	adminViewer := viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}

	t.Run("endpoint enforces premium license", func(t *testing.T) {
		freeCtx := viewer.NewContext(ctx, adminViewer)
		freeCtx = license.NewContext(freeCtx, &fleet.LicenseInfo{Tier: fleet.TierFree})
		_, err := exportPolicyStatusEndpoint(freeCtx, &exportPolicyStatusRequest{PolicyID: policyID}, svc)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
	})

	t.Run("pagination is bypassed (PerPage=0 forwarded)", func(t *testing.T) {
		var capturedReq fleet.GetPolicyStatusRequest
		ds.GetPolicyStatusFunc = func(ctx context.Context, pid uint, filter fleet.TeamFilter, req fleet.GetPolicyStatusRequest) ([]fleet.GetPolicyStatusPolicyRun, int, *fleet.PaginationMetadata, error) {
			capturedReq = req
			return nil, 0, &fleet.PaginationMetadata{}, nil
		}
		premCtx := viewer.NewContext(ctx, adminViewer)
		premCtx = license.NewContext(premCtx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		_, err := exportPolicyStatusEndpoint(premCtx, &exportPolicyStatusRequest{
			PolicyID:             policyID,
			HostNameQuery:        "myhost",
			AutomationErrorQuery: "timeout",
			RunStatus:            "policy_failed",
		}, svc)
		require.NoError(t, err)
		require.Equal(t, uint(0), capturedReq.ListOptions.PerPage)
		require.Equal(t, "myhost", capturedReq.HostNameQuery)
		require.Equal(t, "timeout", capturedReq.AutomationErrorQuery)
		require.Equal(t, "policy_failed", capturedReq.RunStatus)
	})

	t.Run("flatten: host with automations yields one row per automation", func(t *testing.T) {
		rows := flattenPolicyStatusToCSV(runsWithAutomations)
		// host-fail has 2 automations → 2 rows; host-pass has none → 1 row
		require.Len(t, rows, 3)

		// first two rows belong to host-fail
		require.Equal(t, uint(1), rows[0].HostID)
		require.Equal(t, "host-fail", rows[0].HostName)
		require.Equal(t, "failing", rows[0].Status)
		require.Equal(t, uint(3), rows[0].ConsecutiveFailures)
		require.Equal(t, "webhook", rows[0].AutomationType)
		require.Equal(t, "failed", rows[0].AutomationStatus)
		require.Equal(t, "timeout 504", rows[0].AutomationError)

		require.Equal(t, uint(1), rows[1].HostID)
		require.Equal(t, "jira", rows[1].AutomationType)
		require.Equal(t, "success", rows[1].AutomationStatus)
		require.Empty(t, rows[1].AutomationError)

		// third row belongs to host-pass with blank automation columns
		require.Equal(t, uint(2), rows[2].HostID)
		require.Equal(t, "passing", rows[2].Status)
		require.Empty(t, rows[2].AutomationType)
		require.Empty(t, rows[2].AutomationStatus)
		require.Empty(t, rows[2].AutomationError)
	})

	t.Run("GetPolicyStatus error is surfaced in response", func(t *testing.T) {
		dsErr := errors.New("db unavailable")
		ds.GetPolicyStatusFunc = func(ctx context.Context, pid uint, filter fleet.TeamFilter, req fleet.GetPolicyStatusRequest) ([]fleet.GetPolicyStatusPolicyRun, int, *fleet.PaginationMetadata, error) {
			return nil, 0, nil, dsErr
		}
		premCtx := viewer.NewContext(ctx, adminViewer)
		premCtx = license.NewContext(premCtx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		respRaw, err := exportPolicyStatusEndpoint(premCtx, &exportPolicyStatusRequest{PolicyID: policyID}, svc)
		require.NoError(t, err)
		resp, ok := respRaw.(exportPolicyStatusResponse)
		require.True(t, ok)
		require.Error(t, resp.Err)
	})

	t.Run("HijackRender writes CSV headers and valid CSV body", func(t *testing.T) {
		ds.GetPolicyStatusFunc = func(ctx context.Context, pid uint, filter fleet.TeamFilter, req fleet.GetPolicyStatusRequest) ([]fleet.GetPolicyStatusPolicyRun, int, *fleet.PaginationMetadata, error) {
			return runsWithAutomations, len(runsWithAutomations), &fleet.PaginationMetadata{}, nil
		}
		premCtx := viewer.NewContext(ctx, adminViewer)
		premCtx = license.NewContext(premCtx, &fleet.LicenseInfo{Tier: fleet.TierPremium})

		respRaw, err := exportPolicyStatusEndpoint(premCtx, &exportPolicyStatusRequest{PolicyID: policyID}, svc)
		require.NoError(t, err)

		resp, ok := respRaw.(exportPolicyStatusResponse)
		require.True(t, ok)
		require.NoError(t, resp.Err)

		w := httptest.NewRecorder()
		resp.HijackRender(premCtx, w)

		require.Equal(t, "text/csv", w.Header().Get("Content-Type"))
		require.Contains(t, w.Header().Get("Content-Disposition"), "attachment")

		var parsed []policyStatusCSVRow
		require.NoError(t, gocsv.UnmarshalBytes(w.Body.Bytes(), &parsed))
		require.Len(t, parsed, 3)
	})
}
