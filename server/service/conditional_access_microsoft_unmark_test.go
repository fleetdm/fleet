package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/stretchr/testify/require"
)

func TestUnmarkConditionalAccessMicrosoftDevice(t *testing.T) {
	ds := new(mock.Store)
	ds.ConditionalAccessMicrosoftGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
		return &fleet.ConditionalAccessMicrosoftIntegration{TenantID: "tenant", ProxyServerSecret: "secret"}, nil
	}
	type report struct {
		deviceID           string
		managed, compliant bool
	}
	var reports []report
	proxy := &mockedConditionalAccessMicrosoftProxy{
		setComplianceStatusFunc: func(
			ctx context.Context, tenantID, secret, deviceID, userPrincipalName string, mdmEnrolled bool,
			deviceName, osName, osVersion string, compliant bool, lastCheckInTime time.Time,
		) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
			reports = append(reports, report{deviceID, mdmEnrolled, compliant})
			return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{}, nil
		},
	}
	svc := &Service{ds: ds, logger: testutils.TestLogger(t), conditionalAccessMicrosoftProxy: proxy}

	dropped := &fleet.HostConditionalAccessStatus{HostID: 7, DeviceID: "dropped-device-id", Managed: new(true), Compliant: new(true)}
	require.NoError(t, svc.unmarkConditionalAccessMicrosoftDevice(t.Context(), dropped))
	require.Equal(t, []report{{"dropped-device-id", false, false}}, reports)
	// The host's stored status belongs to its current device ID, if any, so it must not change.
	require.False(t, ds.SetHostConditionalAccessStatusFuncInvoked)
}
