package schedule

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestMaybeSendStatistics(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(ctx context.Context, frequency time.Duration, license *fleet.LicenseInfo) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{
			AnonymousIdentifier:       "ident",
			FleetVersion:              "1.2.3",
			LicenseTier:               "premium",
			NumHostsEnrolled:          999,
			NumUsers:                  99,
			NumTeams:                  9,
			NumPolicies:               0,
			NumLabels:                 3,
			SoftwareInventoryEnabled:  true,
			VulnDetectionEnabled:      true,
			SystemUsersEnabled:        true,
			HostsStatusWebHookEnabled: true,
		}, true, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func(ctx context.Context) error {
		recorded = true
		return nil
	}

	err := trySendStatistics(context.Background(), ds, fleet.StatisticsFrequency, ts.URL, &fleet.LicenseInfo{Tier: "premium"})
	require.NoError(t, err)
	assert.True(t, recorded)
	assert.Equal(t, `{"anonymousIdentifier":"ident","fleetVersion":"1.2.3","licenseTier":"premium","numHostsEnrolled":999,"numUsers":99,"numTeams":9,"numPolicies":0,"numLabels":3,"softwareInventoryEnabled":true,"vulnDetectionEnabled":true,"systemUsersEnabled":true,"hostsStatusWebHookEnabled":true}`, requestBody)
}

func TestMaybeSendStatisticsSkipsSendingIfNotNeeded(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(ctx context.Context, frequency time.Duration, license *fleet.LicenseInfo) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{}, false, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func(ctx context.Context) error {
		recorded = true
		return nil
	}

	err := trySendStatistics(context.Background(), ds, fleet.StatisticsFrequency, ts.URL, &fleet.LicenseInfo{Tier: "premium"})
	require.NoError(t, err)
	assert.False(t, recorded)
	assert.False(t, called)
}

func TestMaybeSendStatisticsSkipsIfNotConfigured(t *testing.T) {
	ds := new(mock.Store)

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	err := trySendStatistics(context.Background(), ds, fleet.StatisticsFrequency, ts.URL, &fleet.LicenseInfo{Tier: "premium"})
	require.NoError(t, err)
	assert.False(t, called)
}
