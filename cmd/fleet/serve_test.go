package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
	assert.Equal(t, `{"anonymousIdentifier":"ident","fleetVersion":"1.2.3","licenseTier":"premium","numHostsEnrolled":999,"numUsers":99,"numTeams":9,"numPolicies":0,"softwareInventoryEnabled":true,"vulnDetectionEnabled":true,"systemUsersEnabled":true,"hostsStatusWebHookEnabled":true}`, requestBody)
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

func TestCronWebhooks(t *testing.T) {
	ds := new(mock.Store)

	endpointCalled := int32(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&endpointCalled, 1)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				HostStatusWebhook: fleet.HostStatusWebhookSettings{
					Enable:         true,
					DestinationURL: ts.URL,
					HostPercentage: 43,
					DaysCount:      2,
				},
				Interval: fleet.Duration{Duration: 2 * time.Second},
			},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	calledOnce := make(chan struct{})
	calledTwice := make(chan struct{})
	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, daysCount int) (int, int, error) {
		defer func() {
			select {
			case <-calledOnce:
				select {
				case <-calledTwice:
				default:
					close(calledTwice)
				}
			default:
				close(calledOnce)
			}
		}()
		return 10, 6, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	go cronWebhooks(ctx, ds, kitlog.With(kitlog.NewNopLogger(), "cron", "webhooks"), "1234")

	<-calledOnce
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&endpointCalled))
	<-calledTwice
	time.Sleep(1 * time.Second)
	assert.GreaterOrEqual(t, int32(2), atomic.LoadInt32(&endpointCalled))
}

func TestCronVulnerabilitiesCreatesDatabasesPath(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           10 * time.Second,
			CurrentInstanceChecks: "auto",
		},
	}

	// We cancel right away so cronsVulnerailities finishes. The logic we are testing happens before the loop starts
	cancelFunc()
	cronVulnerabilities(ctx, ds, kitlog.NewNopLogger(), "AAA", fleetConfig)

	require.DirExists(t, vulnPath)
}

func TestCronVulnerabilitiesAcceptsExistingDbPath(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         t.TempDir(),
			Periodicity:           10 * time.Second,
			CurrentInstanceChecks: "auto",
		},
	}

	// We cancel right away so cronsVulnerailities finishes. The logic we are testing happens before the loop starts
	cancelFunc()
	cronVulnerabilities(ctx, ds, logger, "AAA", fleetConfig)

	require.Contains(t, buf.String(), `{"level":"debug","waiting":"on ticker"}`)
}

func TestCronVulnerabilitiesQuitsIfErrorVulnPath(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			HostSettings: fleet.HostSettings{EnableSoftwareInventory: true},
		}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	fileVulnPath := path.Join(t.TempDir(), "somefile")
	_, err := os.Create(fileVulnPath)
	require.NoError(t, err)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         fileVulnPath,
			Periodicity:           10 * time.Second,
			CurrentInstanceChecks: "auto",
		},
	}

	// We cancel right away so cronsVulnerailities finishes. The logic we are testing happens before the loop starts
	cancelFunc()
	cronVulnerabilities(ctx, ds, logger, "AAA", fleetConfig)

	require.Contains(t, buf.String(), `"databases-path":"creation failed, returning"`)
}

func TestCronVulnerabilitiesSkipCreationIfStatic(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		return true, nil
	}
	ds.UnlockFunc = func(ctx context.Context, name string, owner string) error {
		return nil
	}

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           10 * time.Second,
			CurrentInstanceChecks: "1",
		},
	}

	// We cancel right away so cronsVulnerailities finishes. The logic we are testing happens before the loop starts
	cancelFunc()
	cronVulnerabilities(ctx, ds, logger, "AAA", fleetConfig)

	require.NoDirExists(t, vulnPath)
}
