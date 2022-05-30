package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestListDevicePolicies(t *testing.T) {
	ds := new(mock.Store)
	mockPolicies := []*fleet.HostPolicy{&fleet.HostPolicy{fleet.PolicyData{Name: "test-policy"}, "pass"}}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return mockPolicies, nil
	}

	ds.HostLiteFunc = func(ctx context.Context, hostID uint) (*fleet.Host, error) {
		if hostID != uint(1) {
			return nil, errors.New("test error")
		}

		return &fleet.Host{ID: hostID, TeamID: ptr.Uint(1)}, nil
	}

	t.Run("without premium license", func(t *testing.T) {
		svc := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierFree}})
		_, error := svc.ListDevicePolicies(test.UserContext(test.UserAdmin), &fleet.Host{ID: 1})
		require.ErrorIs(t, error, fleet.ErrMissingLicense)
	})

	t.Run("with premium license", func(t *testing.T) {
		svc := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

		_, error := svc.ListDevicePolicies(test.UserContext(test.UserAdmin), &fleet.Host{})
		require.Error(t, error)

		policies, error := svc.ListDevicePolicies(test.UserContext(test.UserAdmin), &fleet.Host{ID: 1})
		require.NoError(t, error)
		require.Len(t, policies, len(mockPolicies))
	})
}
