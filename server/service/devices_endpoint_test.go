package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDeviceHostEndpointScrubbing(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})

	h := &fleet.Host{
		ID:             1,
		Hostname:       "test-host",
		UUID:           "sensitive-uuid",
		HardwareSerial: "sensitive-serial",
		PrimaryMac:     "sensitive-mac",
		TeamName:       ptr.String("sensitive-team"),
		Platform:       "ios",
		MDM: fleet.MDMHostData{
			Profiles: &[]fleet.HostMDMProfile{
				{Identifier: "sensitive-profile"},
			},
		},
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return h, nil
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return h, nil
	}

	ds.GetHostIssuesLastUpdatedFunc = func(ctx context.Context, hostID uint) (time.Time, error) {
		return time.Now(), nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgLogoURL: "http://example.com/logo.png",
			},
		}, nil
	}

	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeVulnerabilities bool) error {
		return nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.ListHostUsersFunc = func(ctx context.Context, hostID uint) ([]fleet.HostUser, error) {
		return nil, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		return nil, nil
	}
	ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		return nil, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hostID uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hostID uint) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostMaintenanceWindow, error) {
		return nil, nil
	}
	ds.IsHostDiskEncryptionKeyArchivedFunc = func(ctx context.Context, hostID uint) (bool, error) {
		return false, nil
	}
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		return &fleet.HostLockWipeStatus{}, nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}

	// Inject host into context
	ctx = host.NewContext(ctx, h)
	// Inject authz context with URL-based auth method (scrubbing only happens for URL auth)
	authzCtx := &authz.AuthorizationContext{}
	authzCtx.SetAuthnMethod(authz.AuthnDeviceURL)
	ctx = authz.NewContext(ctx, authzCtx)

	req := &getDeviceHostRequest{
		Token: "test-token",
	}

	resp, err := getDeviceHostEndpoint(ctx, req, svc)
	require.NoError(t, err)

	deviceResp, ok := resp.(getDeviceHostResponse)
	require.True(t, ok)
	require.NoError(t, deviceResp.Err)
	require.NotNil(t, deviceResp.Host)

	// Verify scrubbed fields in Host
	assert.Empty(t, deviceResp.Host.HardwareSerial)
	assert.Empty(t, deviceResp.Host.UUID)
	assert.Empty(t, deviceResp.Host.PrimaryMac)
	assert.Nil(t, deviceResp.Host.TeamName)
	assert.Nil(t, deviceResp.Host.MDM.Profiles)
	assert.Nil(t, deviceResp.Host.Labels)

	// Verify scrubbed fields in License
	assert.Empty(t, deviceResp.License.Organization)
	assert.Zero(t, deviceResp.License.DeviceCount)
	assert.True(t, deviceResp.License.Expiration.IsZero())
	assert.Equal(t, "", deviceResp.Host.Hostname)
	assert.Equal(t, "", deviceResp.Host.ComputerName)
	assert.Equal(t, "", deviceResp.Host.DisplayText)
	assert.Equal(t, "", deviceResp.Host.DisplayName)

	// Verify other fields are present
	assert.Equal(t, "http://example.com/logo.png", deviceResp.OrgLogoURL)
}

func TestGetDeviceHostEndpointNoScrubbingForMacOS(t *testing.T) {
	ds := new(mock.Store)
	testLicense := &fleet.LicenseInfo{
		Tier:         fleet.TierPremium,
		Organization: "Test Org",
		DeviceCount:  100,
		Expiration:   time.Now().Add(24 * time.Hour),
	}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		SkipCreateTestUsers: true,
		License:             testLicense,
	})

	h := &fleet.Host{
		ID:             1,
		Hostname:       "test-host-mac",
		UUID:           "visible-uuid",
		HardwareSerial: "visible-serial",
		PrimaryMac:     "visible-mac",
		TeamName:       ptr.String("visible-team"),
		Platform:       "darwin",
		MDM: fleet.MDMHostData{
			Profiles: &[]fleet.HostMDMProfile{
				{Identifier: "visible-profile"},
			},
		},
	}

	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return h, nil
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return h, nil
	}

	ds.GetHostIssuesLastUpdatedFunc = func(ctx context.Context, hostID uint) (time.Time, error) {
		return time.Now(), nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgLogoURL: "http://example.com/logo.png",
			},
		}, nil
	}

	ds.LoadHostSoftwareFunc = func(ctx context.Context, host *fleet.Host, includeVulnerabilities bool) error {
		return nil
	}
	ds.ListPoliciesForHostFunc = func(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
		return nil, nil
	}
	ds.ListHostUsersFunc = func(ctx context.Context, hostID uint) ([]fleet.HostUser, error) {
		return nil, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		return nil, nil
	}
	ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
		return nil, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hostID uint) ([]*fleet.Label, error) {
		return nil, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hostID uint) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.ListHostBatteriesFunc = func(ctx context.Context, id uint) ([]*fleet.HostBattery, error) {
		return nil, nil
	}
	ds.ListUpcomingHostMaintenanceWindowsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostMaintenanceWindow, error) {
		return nil, nil
	}
	ds.IsHostDiskEncryptionKeyArchivedFunc = func(ctx context.Context, hostID uint) (bool, error) {
		return false, nil
	}
	ds.GetHostLockWipeStatusFunc = func(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
		return &fleet.HostLockWipeStatus{}, nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}

	// Inject host into context
	ctx = host.NewContext(ctx, h)
	// Inject authz context
	authzCtx := &authz.AuthorizationContext{}
	authzCtx.SetAuthnMethod(authz.AuthnDeviceToken)
	ctx = authz.NewContext(ctx, authzCtx)

	req := &getDeviceHostRequest{
		Token: "test-token",
	}

	resp, err := getDeviceHostEndpoint(ctx, req, svc)
	require.NoError(t, err)

	deviceResp, ok := resp.(getDeviceHostResponse)
	require.True(t, ok)
	require.NoError(t, deviceResp.Err)
	require.NotNil(t, deviceResp.Host)

	// Verify fields are NOT scrubbed
	assert.Equal(t, "visible-serial", deviceResp.Host.HardwareSerial)
	assert.Equal(t, "visible-uuid", deviceResp.Host.UUID)
	assert.Equal(t, "visible-mac", deviceResp.Host.PrimaryMac)
	assert.NotNil(t, deviceResp.Host.TeamName)
	assert.Equal(t, "visible-team", *deviceResp.Host.TeamName)
	assert.NotNil(t, deviceResp.Host.MDM.Profiles)

	// Verify License is NOT scrubbed (values match what we set in testLicense)
	assert.Equal(t, "Test Org", deviceResp.License.Organization)
	assert.Equal(t, 100, deviceResp.License.DeviceCount)
	assert.False(t, deviceResp.License.Expiration.IsZero())
}
