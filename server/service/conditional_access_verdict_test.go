package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCAVerdictTest creates a test service with all mocks wired up for
// conditional access verdict testing. Returns the concrete *Service and the
// mock datastore. The mock proxy records whether SetComplianceStatus was
// called and with what compliant value.
func setupCAVerdictTest(t *testing.T, persistedPasses *bool) (
	*Service,
	*mock.Store,
	*bool,      // proxyCalled
	**bool,     // pushedCompliant
	chan struct{}, // setDone
) {
	t.Helper()

	const (
		teamID     = uint(1)
		caPolicyID = uint(100)
	)

	ds := new(mock.Store)

	var (
		mu              sync.Mutex
		proxyCalled     bool
		pushedCompliant *bool
		setDone         = make(chan struct{}, 1)
	)

	mockProxy := &testCAProxy{
		setComplianceStatusFunc: func(
			_ context.Context,
			_, _ string,
			_, _ string,
			_ bool,
			_, _, _ string,
			compliant bool,
			_ time.Time,
		) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
			mu.Lock()
			proxyCalled = true
			pushedCompliant = &compliant
			mu.Unlock()
			return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{
				MessageID: "msg-1",
			}, nil
		},
		getMessageStatusFunc: func(
			_ context.Context, _, _, _ string,
		) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
			setDone <- struct{}{}
			return &conditional_access_microsoft_proxy.GetMessageStatusResponse{
				MessageID: "msg-1",
				Status:    "Completed",
			}, nil
		},
	}

	cfg := config.TestConfig()
	svc, _ := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{
		ConditionalAccessMicrosoftProxy: mockProxy,
	})
	serv := ((svc.(validationMiddleware)).Service).(*Service)
	conditionalAccessSetWaitTime = 250 * time.Millisecond

	// Integration configured and setup done.
	ds.ConditionalAccessMicrosoftGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
		return &fleet.ConditionalAccessMicrosoftIntegration{
			TenantID:          "tenant-1",
			ProxyServerSecret: "secret-1",
			SetupDone:         true,
		}, nil
	}

	// Team has conditional access enabled.
	ds.TeamLiteFunc = func(ctx context.Context, id uint) (*fleet.TeamLite, error) {
		return &fleet.TeamLite{
			ID: teamID,
			Config: fleet.TeamConfigLite{
				Integrations: fleet.TeamIntegrations{
					ConditionalAccessEnabled: optjson.SetBool(true),
				},
			},
		}, nil
	}

	// Host CA status: currently non-compliant in Entra.
	ds.LoadHostConditionalAccessStatusFunc = func(ctx context.Context, hID uint) (*fleet.HostConditionalAccessStatus, error) {
		return &fleet.HostConditionalAccessStatus{
			HostID:            hID,
			DeviceID:          "entra-device-1",
			UserPrincipalName: "user@corp.com",
			Managed:           new(false),
			Compliant:         new(false), // non-compliant
			DisplayName:       "test-host",
			OSVersion:         "14.0",
		}, nil
	}

	ds.GetHostMDMFunc = func(ctx context.Context, hID uint) (*fleet.HostMDM, error) {
		return &fleet.HostMDM{Enrolled: false}, nil
	}

	ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, tID uint, platform string) ([]uint, error) {
		return []uint{caPolicyID}, nil
	}

	ds.SetHostConditionalAccessStatusFunc = func(ctx context.Context, hID uint, managed bool, compliant bool) error {
		return nil
	}

	// Persisted policy membership: the CA policy has the given pass/fail state.
	ds.GetHostPolicyMembershipForPoliciesFunc = func(ctx context.Context, hID uint, policyIDs []uint) (map[uint]*bool, error) {
		result := make(map[uint]*bool, len(policyIDs))
		for _, pid := range policyIDs {
			if pid == caPolicyID {
				result[pid] = persistedPasses
			}
		}
		return result, nil
	}

	return serv, ds, &proxyCalled, &pushedCompliant, setDone
}

// TestConditionalAccessOmissionDoesNotSpoofCompliance verifies that omitting
// CA policy results from a distributed/write submission does NOT flip the
// host's compliance status to true. The verdict must come from persisted
// policy_membership, not from the in-flight submission.
//
// See: confidential#16386
func TestConditionalAccessOmissionDoesNotSpoofCompliance(t *testing.T) {
	t.Parallel()

	const (
		hostID     = uint(1)
		teamID     = uint(1)
		otherPolID = uint(200)
	)

	// Persisted membership: CA policy is FAILING.
	serv, _, proxyCalled, _, _ := setupCAVerdictTest(t, new(false))

	// The attack: omit CA policy, only include a passing non-CA policy.
	orbitNodeKey := "orbit-key-1"
	hostTeamID := teamID
	incomingResults := map[uint]*bool{
		otherPolID: new(true),
	}

	err := serv.processConditionalAccessForNewlyFailingPolicies(
		t.Context(), hostID, &hostTeamID, &orbitNodeKey, "darwin", incomingResults,
	)
	require.NoError(t, err)

	// Give the async goroutine a moment (it should NOT fire).
	time.Sleep(1 * time.Second)

	// The proxy should NOT have been called because the computed verdict
	// (false, from persisted membership) matches the persisted Compliant=false.
	assert.False(t, *proxyCalled,
		"proxy should not be called: verdict (non-compliant) matches persisted state")
}

// TestConditionalAccessNilMembershipDoesNotSpoofCompliance verifies that when
// the persisted membership for a CA policy is nil (indeterminate / failed query),
// the host is treated as non-compliant.
func TestConditionalAccessNilMembershipDoesNotSpoofCompliance(t *testing.T) {
	t.Parallel()

	const (
		hostID     = uint(1)
		teamID     = uint(1)
		otherPolID = uint(200)
	)

	// Persisted membership: CA policy result is nil (indeterminate).
	serv, _, proxyCalled, _, _ := setupCAVerdictTest(t, nil)

	orbitNodeKey := "orbit-key-1"
	hostTeamID := teamID
	incomingResults := map[uint]*bool{
		otherPolID: new(true),
	}

	err := serv.processConditionalAccessForNewlyFailingPolicies(
		t.Context(), hostID, &hostTeamID, &orbitNodeKey, "darwin", incomingResults,
	)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	assert.False(t, *proxyCalled,
		"proxy should not be called: nil membership means non-compliant, matches persisted state")
}

// TestConditionalAccessLegitComplianceStillWorks verifies that a host that is
// genuinely compliant (persisted membership shows passing) still gets its
// status pushed when it transitions from non-compliant to compliant.
func TestConditionalAccessLegitComplianceStillWorks(t *testing.T) {
	t.Parallel()

	const (
		hostID     = uint(1)
		teamID     = uint(1)
		otherPolID = uint(200)
	)

	// Persisted membership: CA policy is PASSING.
	serv, _, proxyCalled, pushedCompliant, setDone := setupCAVerdictTest(t, new(true))

	orbitNodeKey := "orbit-key-1"
	hostTeamID := teamID
	incomingResults := map[uint]*bool{
		otherPolID: new(true),
	}

	err := serv.processConditionalAccessForNewlyFailingPolicies(
		t.Context(), hostID, &hostTeamID, &orbitNodeKey, "darwin", incomingResults,
	)
	require.NoError(t, err)

	// This time the proxy SHOULD be called because persisted membership says
	// passing (compliant=true) but persisted Entra state is Compliant=false.
	select {
	case <-setDone:
		time.Sleep(500 * time.Millisecond)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance status to be pushed")
	}

	assert.True(t, *proxyCalled, "proxy should be called for legitimate compliance change")
	require.NotNil(t, *pushedCompliant)
	assert.True(t, **pushedCompliant, "should push compliant=true for genuinely passing host")
}

// testCAProxy is a minimal mock of the ConditionalAccessMicrosoftProxy interface.
type testCAProxy struct {
	setComplianceStatusFunc func(
		ctx context.Context,
		tenantID, secret string,
		deviceID, userPrincipalName string,
		mdmEnrolled bool,
		deviceName, osName, osVersion string,
		compliant bool,
		lastCheckInTime time.Time,
	) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error)

	getMessageStatusFunc func(
		ctx context.Context,
		tenantID, secret, messageID string,
	) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error)
}

func (m *testCAProxy) Create(ctx context.Context, tenantID string) (*conditional_access_microsoft_proxy.CreateResponse, error) {
	return nil, nil
}

func (m *testCAProxy) Get(ctx context.Context, tenantID, secret string) (*conditional_access_microsoft_proxy.GetResponse, error) {
	return nil, nil
}

func (m *testCAProxy) Delete(ctx context.Context, tenantID, secret string) (*conditional_access_microsoft_proxy.DeleteResponse, error) {
	return nil, nil
}

func (m *testCAProxy) SetComplianceStatus(
	ctx context.Context,
	tenantID, secret string,
	deviceID, userPrincipalName string,
	mdmEnrolled bool,
	deviceName, osName, osVersion string,
	compliant bool,
	lastCheckInTime time.Time,
) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
	return m.setComplianceStatusFunc(ctx, tenantID, secret, deviceID, userPrincipalName, mdmEnrolled, deviceName, osName, osVersion, compliant, lastCheckInTime)
}

func (m *testCAProxy) GetMessageStatus(
	ctx context.Context, tenantID, secret, messageID string,
) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
	return m.getMessageStatusFunc(ctx, tenantID, secret, messageID)
}
