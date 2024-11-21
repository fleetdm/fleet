package main

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeStore is a wrapper around mock.Store to allow for concurrent calling to
// AppConfig, Lock, and Unlock, in the past we have seen this test fail with a data race warning.
//
// TODO: if we see other tests failing for similar reasons, we should build a
// more robust pattern instead of doing this everywhere
type safeStore struct {
	mock.Store
	mu sync.Mutex
}

func (s *safeStore) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	s.mu.Lock()
	s.AppConfigFuncInvoked = true
	s.mu.Unlock()
	return s.AppConfigFunc(ctx)
}

func (s *safeStore) Lock(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
	s.mu.Lock()
	s.LockFuncInvoked = true
	s.mu.Unlock()
	return s.LockFunc(ctx, name, owner, expiration)
}

func (s *safeStore) Unlock(ctx context.Context, name string, owner string) error {
	s.mu.Lock()
	s.UnlockFuncInvoked = true
	s.mu.Unlock()
	return s.UnlockFunc(ctx, name, owner)
}

func TestMaybeSendStatistics(t *testing.T) {
	ds := new(mock.Store)

	fleetConfig := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	requestBody := ""

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(ctx context.Context, frequency time.Duration, config config.FleetConfig) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{
			AnonymousIdentifier:                  "ident",
			FleetVersion:                         "1.2.3",
			LicenseTier:                          "premium",
			NumHostsEnrolled:                     999,
			NumUsers:                             99,
			NumSoftwareVersions:                  100,
			NumHostSoftwares:                     101,
			NumSoftwareTitles:                    102,
			NumHostSoftwareInstalledPaths:        103,
			NumSoftwareCPEs:                      104,
			NumSoftwareCVEs:                      105,
			NumTeams:                             9,
			NumPolicies:                          0,
			NumQueries:                           200,
			NumLabels:                            3,
			SoftwareInventoryEnabled:             true,
			VulnDetectionEnabled:                 true,
			SystemUsersEnabled:                   true,
			HostsStatusWebHookEnabled:            true,
			NumWeeklyActiveUsers:                 111,
			NumWeeklyPolicyViolationDaysActual:   0,
			NumWeeklyPolicyViolationDaysPossible: 0,
			HostsEnrolledByOperatingSystem: map[string][]fleet.HostsCountByOSVersion{
				"linux": {
					fleet.HostsCountByOSVersion{Version: "1.2.3", NumEnrolled: 22},
				},
			},
			HostsEnrolledByOrbitVersion:   []fleet.HostsCountByOrbitVersion{},
			HostsEnrolledByOsqueryVersion: []fleet.HostsCountByOsqueryVersion{},
			StoredErrors:                  []byte(`[]`),
			Organization:                  "Fleet",
			AIFeaturesDisabled:            true,
			MaintenanceWindowsEnabled:     true,
			MaintenanceWindowsConfigured:  true,
			NumHostsFleetDesktopEnabled:   1984,
		}, true, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func(ctx context.Context) error {
		recorded = true
		return nil
	}
	cleanedup := false
	ds.CleanupStatisticsFunc = func(ctx context.Context) error {
		cleanedup = true
		return nil
	}

	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	err := trySendStatistics(ctx, ds, fleet.StatisticsFrequency, ts.URL, fleetConfig)
	require.NoError(t, err)
	assert.True(t, recorded)
	require.True(t, cleanedup)
	assert.Equal(t, `{"anonymousIdentifier":"ident","fleetVersion":"1.2.3","licenseTier":"premium","organization":"Fleet","numHostsEnrolled":999,"numUsers":99,"numSoftwareVersions":100,"numHostSoftwares":101,"numSoftwareTitles":102,"numHostSoftwareInstalledPaths":103,"numSoftwareCPEs":104,"numSoftwareCVEs":105,"numTeams":9,"numPolicies":0,"numQueries":200,"numLabels":3,"softwareInventoryEnabled":true,"vulnDetectionEnabled":true,"systemUsersEnabled":true,"hostsStatusWebHookEnabled":true,"mdmMacOsEnabled":false,"hostExpiryEnabled":false,"mdmWindowsEnabled":false,"liveQueryDisabled":false,"numWeeklyActiveUsers":111,"numWeeklyPolicyViolationDaysActual":0,"numWeeklyPolicyViolationDaysPossible":0,"hostsEnrolledByOperatingSystem":{"linux":[{"version":"1.2.3","numEnrolled":22}]},"hostsEnrolledByOrbitVersion":[],"hostsEnrolledByOsqueryVersion":[],"storedErrors":[],"numHostsNotResponding":0,"aiFeaturesDisabled":true,"maintenanceWindowsEnabled":true,"maintenanceWindowsConfigured":true,"numHostsFleetDesktopEnabled":1984}`, requestBody)
}

func TestMaybeSendStatisticsSkipsSendingIfNotNeeded(t *testing.T) {
	ds := new(mock.Store)

	fleetConfig := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{EnableAnalytics: true}}, nil
	}

	ds.ShouldSendStatisticsFunc = func(ctx context.Context, frequency time.Duration, cfg config.FleetConfig) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{}, false, nil
	}
	recorded := false
	ds.RecordStatisticsSentFunc = func(ctx context.Context) error {
		recorded = true
		return nil
	}
	cleanedup := false
	ds.CleanupStatisticsFunc = func(ctx context.Context) error {
		cleanedup = true
		return nil
	}

	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	err := trySendStatistics(ctx, ds, fleet.StatisticsFrequency, ts.URL, fleetConfig)
	require.NoError(t, err)
	assert.False(t, recorded)
	assert.False(t, cleanedup)
	assert.False(t, called)
}

func TestMaybeSendStatisticsSkipsIfNotConfigured(t *testing.T) {
	ds := new(mock.Store)

	fleetConfig := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	err := trySendStatistics(ctx, ds, fleet.StatisticsFrequency, ts.URL, fleetConfig)
	require.NoError(t, err)
	assert.False(t, called)
}

func TestMaybeSendStatisticsSendsIfNotConfiguredForPremium(t *testing.T) {
	ds := new(mock.Store)

	fleetConfig := config.FleetConfig{Osquery: config.OsqueryConfig{DetailUpdateInterval: 1 * time.Hour}}

	called := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer ts.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	ds.ShouldSendStatisticsFunc = func(ctx context.Context, frequency time.Duration, config config.FleetConfig) (fleet.StatisticsPayload, bool, error) {
		return fleet.StatisticsPayload{}, true, nil
	}

	ds.CleanupStatisticsFunc = func(ctx context.Context) error { return nil }

	ds.RecordStatisticsSentFunc = func(ctx context.Context) error { return nil }

	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	err := trySendStatistics(ctx, ds, fleet.StatisticsFrequency, ts.URL, fleetConfig)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAutomationsSchedule(t *testing.T) {
	ds := new(safeStore)

	endpointCalled := int32(0)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&endpointCalled, 1)
	}))
	defer ts.Close()

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		return []*fleet.TeamSummary{}, nil
	}

	ds.OutdatedAutomationBatchFunc = func(ctx context.Context) ([]fleet.PolicyFailure, error) {
		return []fleet.PolicyFailure{}, nil
	}

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

	mockLocker := schedule.SetupMockLocker("automations", "test_instance", time.Now().UTC())
	ds.LockFunc = mockLocker.Lock
	ds.UnlockFunc = mockLocker.Unlock

	mockStatsStore := schedule.SetUpMockStatsStore("automations")
	ds.GetLatestCronStatsFunc = mockStatsStore.GetLatestCronStats
	ds.InsertCronStatsFunc = mockStatsStore.InsertCronStats
	ds.UpdateCronStatsFunc = mockStatsStore.UpdateCronStats

	calledOnce := make(chan struct{})
	calledTwice := make(chan struct{})
	ds.TotalAndUnseenHostsSinceFunc = func(ctx context.Context, teamID *uint, daysCount int) (int, []uint, error) {
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
		return 10, []uint{1, 2, 3, 4, 5, 6}, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	failingPoliciesSet := service.NewMemFailingPolicySet()
	s, err := newAutomationsSchedule(ctx, "test_instance", ds, kitlog.NewNopLogger(), 5*time.Minute, failingPoliciesSet)
	require.NoError(t, err)
	s.Start()

	<-calledOnce
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&endpointCalled))
	<-calledTwice
	time.Sleep(1 * time.Second)
	assert.GreaterOrEqual(t, int32(2), atomic.LoadInt32(&endpointCalled))
}

func TestCronVulnerabilitiesCreatesDatabasesPath(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ds := new(safeStore)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Features: fleet.Features{EnableSoftwareInventory: true},
		}, nil
	}
	ds.InsertCVEMetaFunc = func(ctx context.Context, x []fleet.CVEMeta) error {
		return nil
	}
	ds.AllSoftwareIteratorFunc = func(ctx context.Context, query fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		// we should not get this far before we see the directory being created
		return nil, errors.New("shouldn't happen")
	}
	ds.OSVersionsFunc = func(
		ctx context.Context, teamFilter *fleet.TeamFilter, platform *string, name *string, version *string,
	) (*fleet.OSVersions, error) {
		return &fleet.OSVersions{}, nil
	}
	ds.SyncHostsSoftwareFunc = func(ctx context.Context, updatedAt time.Time) error {
		return nil
	}

	mockLocker := schedule.SetupMockLocker("vulnerabilities", "test_instance", time.Now().UTC())
	ds.LockFunc = mockLocker.Lock
	ds.UnlockFunc = mockLocker.Unlock

	mockStatsStore := schedule.SetUpMockStatsStore("vulnerabilities")
	ds.GetLatestCronStatsFunc = mockStatsStore.GetLatestCronStats
	ds.InsertCronStatsFunc = mockStatsStore.InsertCronStats
	ds.UpdateCronStatsFunc = mockStatsStore.UpdateCronStats

	vulnPath := filepath.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	config := config.VulnerabilitiesConfig{
		DatabasesPath:         vulnPath,
		Periodicity:           time.Second,
		CurrentInstanceChecks: "auto",
	}
	// Use schedule to test that the schedule does indeed call cronVulnerabilities.
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx, cancel := context.WithCancel(ctx)
	lg := kitlog.NewJSONLogger(os.Stdout)

	go func() {
		defer func() {
			// this test may panic as we're ending it early, so we shouldn't fail our suite on this panic
			// but depending on where we are in the cron call it may not panic, hence not checking the recover
			// value either way
			// nolint:errcheck
			recover()
		}()
		_ = cronVulnerabilities(ctx, ds, lg, &config)
	}()

	assert.Eventually(t, func() bool {
		info, err := os.Lstat(vulnPath)
		return err == nil && info.IsDir()
	}, 10*time.Second, 1*time.Second)

	t.Cleanup(func() {
		cancel()
	})

	// at this point, the assertion has succeeded or failed, try to remove the
	// temp dir and all content instead of leaving it to the testing package to
	// handle - the assumption is that this test used to fail because it tried to
	// remove the temp dir while trying to write to it concurrently, and resulted
	// in e.g.:
	// === RUN   TestCronVulnerabilitiesCreatesDatabasesPath
	//  testing.go:1231: TempDir RemoveAll cleanup: unlinkat /tmp/TestCronVulnerabilitiesCreatesDatabasesPath2986870230/001/something: directory not empty
	// --- FAIL: TestCronVulnerabilitiesCreatesDatabasesPath (60.22s)
	for err := os.RemoveAll(vulnPath); err != nil; err = os.RemoveAll(vulnPath) {
		if strings.Contains(err.Error(), "directory not empty") {
			time.Sleep(time.Second)
			continue
		}
		// for any other unexpected error, fail the test
		require.NoError(t, err)
	}
}

type softwareIterator struct {
	index     int
	softwares []*fleet.Software
}

func (f *softwareIterator) Next() bool {
	return f.index < len(f.softwares)
}

func (f *softwareIterator) Value() (*fleet.Software, error) {
	s := f.softwares[f.index]
	f.index++
	return s, nil
}

func (f *softwareIterator) Err() error   { return nil }
func (f *softwareIterator) Close() error { return nil }

func TestScanVulnerabilities(t *testing.T) {
	nettest.Run(t)

	logger := kitlog.NewNopLogger()
	logger = level.NewFilter(logger, level.AllowDebug())

	ctx := context.Background()

	webhookCount := 0
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCount++

		var payload map[string]json.RawMessage
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		expected := `
{
  "cve": "CVE-2022-39348",
  "cve_published": "2022-10-26T14:15:00Z",
  "details_link": "https://nvd.nist.gov/vuln/detail/CVE-2022-39348",
  "epss_probability": 0.0089,
  "cvss_score": 5.4,
  "cisa_known_exploit": false,
  "hosts_affected": [
    {
      "id": 1,
      "hostname": "1",
      "display_name": "1",
      "url": "hosts/1"
    }
  ]
}
`
		require.JSONEq(t, expected, string(payload["vulnerability"]))
	}))

	appConfig := &fleet.AppConfig{
		Features: fleet.Features{
			EnableSoftwareInventory: true,
		},
		WebhookSettings: fleet.WebhookSettings{
			VulnerabilitiesWebhook: fleet.VulnerabilitiesWebhookSettings{
				Enable:         true,
				DestinationURL: svr.URL,
			},
		},
	}

	ds := new(mock.Store)
	ds.InsertCVEMetaFunc = func(ctx context.Context, x []fleet.CVEMeta) error {
		return nil
	}
	ds.AllSoftwareIteratorFunc = func(ctx context.Context, query fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		iterator := &softwareIterator{
			softwares: []*fleet.Software{
				{
					ID:               1,
					Name:             "Twisted",
					Version:          "22.2.0",
					BundleIdentifier: "",
					Source:           "python_packages",
				},
			},
		}
		return iterator, nil
	}
	ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
		return []fleet.SoftwareCPE{
			{
				ID:         1,
				SoftwareID: 1,
				CPE:        "cpe:2.3:a:twistedmatrix:twisted:22.2.0:*:*:*:*:python:*:*",
			},
		}, nil
	}
	ds.InsertSoftwareVulnerabilityFunc = func(ctx context.Context, vuln fleet.SoftwareVulnerability, src fleet.VulnerabilitySource) (bool, error) {
		return true, nil
	}
	ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		return int64(0), nil
	}
	ds.DeleteSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		return int64(0), nil
	}
	ds.DeleteOutOfDateVulnerabilitiesFunc = func(ctx context.Context, source fleet.VulnerabilitySource, duration time.Duration) error {
		return nil
	}
	ds.OSVersionsFunc = func(
		ctx context.Context, teamFilter *fleet.TeamFilter, platform *string, name *string, version *string,
	) (*fleet.OSVersions, error) {
		return &fleet.OSVersions{
			CountsUpdatedAt: time.Now(),
			OSVersions: []fleet.OSVersion{
				{HostsCount: 1, Name: "Ubuntu 22.04.1 LTS", Platform: "ubuntu"},
			},
		}, nil
	}
	ds.HostIDsByOSVersionFunc = func(ctx context.Context, osVersion fleet.OSVersion, offset int, limit int) ([]uint, error) {
		if offset == 0 {
			return []uint{1}, nil
		}
		return []uint{}, nil
	}
	ds.ListSoftwareForVulnDetectionFunc = func(ctx context.Context, filter fleet.VulnSoftwareFilter) ([]fleet.Software, error) {
		return []fleet.Software{
			{
				ID:               1,
				Name:             "Twisted",
				Version:          "22.2.0",
				BundleIdentifier: "",
				Source:           "python_packages",
			},
		}, nil
	}
	ds.ListSoftwareVulnerabilitiesByHostIDsSourceFunc = func(ctx context.Context, hostIDs []uint, source fleet.VulnerabilitySource) (map[uint][]fleet.SoftwareVulnerability, error) {
		require.Equal(t, []uint{1}, hostIDs)
		require.Equal(t, fleet.UbuntuOVALSource, source)
		return map[uint][]fleet.SoftwareVulnerability{}, nil
	}
	ds.ListOperatingSystemsFunc = func(ctx context.Context) ([]fleet.OperatingSystem, error) {
		return []fleet.OperatingSystem{
			{
				ID:            1,
				Name:          "Ubuntu",
				Version:       "22.04.1 LTS",
				Arch:          "x86_64",
				KernelVersion: "5.10.124-linuxkit",
			},
		}, nil
	}

	ds.ListOperatingSystemsForPlatformFunc = func(ctx context.Context, platform string) ([]fleet.OperatingSystem, error) {
		return []fleet.OperatingSystem{}, nil
	}

	ds.DeleteOutOfDateOSVulnerabilitiesFunc = func(ctx context.Context, src fleet.VulnerabilitySource, d time.Duration) error {
		return nil
	}

	ds.ListCVEsFunc = func(ctx context.Context, maxAge time.Duration) ([]fleet.CVEMeta, error) {
		published := time.Date(2022, time.October, 26, 14, 15, 0, 0, time.UTC)

		return []fleet.CVEMeta{
			{
				CVE:              "CVE-2022-39348",
				CVSSScore:        ptr.Float64(5.4),
				EPSSProbability:  ptr.Float64(0.0089),
				CISAKnownExploit: ptr.Bool(false),
				Published:        &published,
			},
		}, nil
	}
	ds.HostVulnSummariesBySoftwareIDsFunc = func(ctx context.Context, softwareIDs []uint) ([]fleet.HostVulnerabilitySummary, error) {
		return []fleet.HostVulnerabilitySummary{
			{
				ID:          1,
				Hostname:    "1",
				DisplayName: "1",
			},
		}, nil
	}
	ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
		return true, nil
	}

	vulnPath := filepath.Join("..", "..", "server", "vulnerabilities", "testdata")

	config := config.VulnerabilitiesConfig{
		DatabasesPath:         vulnPath,
		Periodicity:           10 * time.Second,
		CurrentInstanceChecks: "auto",
		DisableDataSync:       true,
	}

	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	err := scanVulnerabilities(ctx, ds, logger, &config, appConfig, vulnPath)
	require.NoError(t, err)

	// ensure that nvd vulnerabilities are not deleted
	require.False(t, ds.DeleteSoftwareVulnerabilitiesFuncInvoked)

	// ensure that webhook was called
	require.Equal(t, 1, webhookCount)
}

func TestScanVulnerabilitiesMkdirFailsIfVulnPathIsFile(t *testing.T) {
	logger := kitlog.NewNopLogger()
	logger = level.NewFilter(logger, level.AllowDebug())

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig := &fleet.AppConfig{
		Features: fleet.Features{EnableSoftwareInventory: true},
	}
	ds := new(safeStore)

	// creating a file with the same path should result in an error when creating the directory
	fileVulnPath := filepath.Join(t.TempDir(), "somefile")
	_, err := os.Create(fileVulnPath)
	require.NoError(t, err)

	config := config.VulnerabilitiesConfig{
		DatabasesPath:         fileVulnPath,
		Periodicity:           10 * time.Second,
		CurrentInstanceChecks: "auto",
	}

	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	err = scanVulnerabilities(ctx, ds, logger, &config, appConfig, fileVulnPath)
	require.ErrorContains(t, err, "create vulnerabilities databases directory: mkdir")
}

func TestCronVulnerabilitiesSkipMkdirIfDisabled(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	ds := new(safeStore)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		// features.enable_software_inventory is false
		return &fleet.AppConfig{}, nil
	}
	ds.SyncHostsSoftwareFunc = func(ctx context.Context, updatedAt time.Time) error {
		return nil
	}

	ds.ReconcileSoftwareTitlesFunc = func(ctx context.Context) error {
		return nil
	}

	ds.SyncHostsSoftwareTitlesFunc = func(ctx context.Context, updatedAt time.Time) error {
		return nil
	}

	ds.UpdateHostIssuesVulnerabilitiesFunc = func(ctx context.Context) error {
		return nil
	}

	mockLocker := schedule.SetupMockLocker("vulnerabilities", "test_instance", time.Now().UTC())
	ds.LockFunc = mockLocker.Lock
	ds.UnlockFunc = mockLocker.Unlock

	mockStatsStore := schedule.SetUpMockStatsStore("vulnerabilities")
	ds.GetLatestCronStatsFunc = mockStatsStore.GetLatestCronStats
	ds.InsertCronStatsFunc = mockStatsStore.InsertCronStats
	ds.UpdateCronStatsFunc = mockStatsStore.UpdateCronStats

	vulnPath := filepath.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	config := config.VulnerabilitiesConfig{
		DatabasesPath:         vulnPath,
		Periodicity:           10 * time.Second,
		CurrentInstanceChecks: "1",
	}

	// Use schedule to test that the schedule does indeed call cronVulnerabilities.
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ctx, cancel := context.WithCancel(ctx)
	s, err := newVulnerabilitiesSchedule(ctx, "test_instance", ds, kitlog.NewNopLogger(), &config)
	require.NoError(t, err)
	s.Start()
	t.Cleanup(func() {
		cancel()
		<-s.Done()
	})

	// Every cron tick is 10 seconds ... here we just wait for a loop interation and assert the vuln
	// dir. was not created.
	require.Eventually(t, func() bool {
		_, err := os.Stat(vulnPath)
		return os.IsNotExist(err)
	}, 24*time.Second, 12*time.Second)
}

// TestCronAutomationsLockDuration tests that the Lock method is being called
// for the current automation crons and that their duration is equal to the current
// schedule interval.
func TestAutomationsScheduleLockDuration(t *testing.T) {
	ds := new(safeStore)
	expectedInterval := 1 * time.Second

	intitalConfigLoaded := make(chan struct{}, 1)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{Duration: 1 * time.Hour},
			},
		}
		select {
		case <-intitalConfigLoaded:
			ac.WebhookSettings.Interval = fleet.Duration{Duration: expectedInterval}
		default:
			// initial config
			close(intitalConfigLoaded)
		}
		return &ac, nil
	}

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		return []*fleet.TeamSummary{}, nil
	}

	ds.OutdatedAutomationBatchFunc = func(ctx context.Context) ([]fleet.PolicyFailure, error) {
		return []fleet.PolicyFailure{}, nil
	}

	hostStatus := make(chan struct{})
	hostStatusClosed := false
	failingPolicies := make(chan struct{})
	failingPoliciesClosed := false
	unknownName := false

	mockLocker := schedule.SetupMockLocker("vulnerabilities", "test_instance", time.Now().UTC())
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		if expiration != expectedInterval {
			return false, nil
		}
		switch name {
		case "automations":
			if !hostStatusClosed {
				close(hostStatus)
				hostStatusClosed = true
			}
			if !failingPoliciesClosed {
				close(failingPolicies)
				failingPoliciesClosed = true
			}
		default:
			unknownName = true
		}
		return mockLocker.Lock(ctx, name, owner, expiration)
	}
	ds.UnlockFunc = mockLocker.Unlock

	mockStatsStore := schedule.SetUpMockStatsStore("vulnerabilities")
	ds.GetLatestCronStatsFunc = mockStatsStore.GetLatestCronStats
	ds.InsertCronStatsFunc = mockStatsStore.InsertCronStats
	ds.UpdateCronStatsFunc = mockStatsStore.UpdateCronStats

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	s, err := newAutomationsSchedule(ctx, "test_instance", ds, kitlog.NewNopLogger(), 1*time.Second, service.NewMemFailingPolicySet())
	require.NoError(t, err)
	s.Start()

	select {
	case <-failingPolicies:
	case <-time.After(5 * time.Second):
		t.Error("failing policies timeout")
	}
	select {
	case <-hostStatus:
	case <-time.After(5 * time.Second):
		t.Error("host status timeout")
	}
	require.False(t, unknownName)
}

func TestAutomationsScheduleIntervalChange(t *testing.T) {
	ds := new(safeStore)

	interval := struct {
		sync.Mutex
		value time.Duration
	}{
		value: 5 * time.Hour,
	}
	configLoaded := make(chan struct{}, 1)

	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		return []*fleet.TeamSummary{}, nil
	}

	ds.OutdatedAutomationBatchFunc = func(ctx context.Context) ([]fleet.PolicyFailure, error) {
		return []fleet.PolicyFailure{}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		select {
		case configLoaded <- struct{}{}:
		default:
			// OK
		}

		interval.Lock()
		defer interval.Unlock()

		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{Duration: interval.value},
			},
		}, nil
	}

	mockLocker := schedule.SetupMockLocker("automations", "test_instance", time.Now().UTC())
	require.NoError(t, mockLocker.AddChannels(t, "locked"))
	ds.LockFunc = mockLocker.Lock
	ds.UnlockFunc = mockLocker.Unlock

	mockStatsStore := schedule.SetUpMockStatsStore("automations")
	ds.GetLatestCronStatsFunc = mockStatsStore.GetLatestCronStats
	ds.InsertCronStatsFunc = mockStatsStore.InsertCronStats
	ds.UpdateCronStatsFunc = mockStatsStore.UpdateCronStats

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	s, err := newAutomationsSchedule(ctx, "test_instance", ds, kitlog.NewNopLogger(), 200*time.Millisecond, service.NewMemFailingPolicySet())
	require.NoError(t, err)
	s.Start()

	// wait for config to be called once by startAutomationsSchedule and again by configReloadFunc
	for c := 0; c < 2; c++ {
		select {
		case <-configLoaded:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout: initial config load")
		}
	}

	interval.Lock()
	interval.value = 1 * time.Second
	interval.Unlock()

	select {
	case <-mockLocker.Locked:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: interval change did not trigger lock call")
	}
}

func TestBasicAuthHandler(t *testing.T) {
	for _, tc := range []struct {
		name           string
		username       string
		password       string
		passes         bool
		noBasicAuthSet bool
	}{
		{
			name:     "good-credentials",
			username: "foo",
			password: "bar",
			passes:   true,
		},
		{
			name:     "empty-credentials",
			username: "",
			password: "",
			passes:   false,
		},
		{
			name:           "no-basic-auth-set",
			username:       "",
			password:       "",
			noBasicAuthSet: true,
			passes:         false,
		},
		{
			name:     "wrong-username",
			username: "foo1",
			password: "bar",
			passes:   false,
		},
		{
			name:     "wrong-password",
			username: "foo",
			password: "bar1",
			passes:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pass := false
			h := basicAuthHandler("foo", "bar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pass = true
				w.WriteHeader(http.StatusOK)
			}))

			r, err := http.NewRequest("GET", "", nil)
			require.NoError(t, err)

			if !tc.noBasicAuthSet {
				r.SetBasicAuth(tc.username, tc.password)
			}

			var w httptest.ResponseRecorder
			h.ServeHTTP(&w, r)

			if pass != tc.passes {
				t.Fatal("unexpected pass")
			}

			expStatusCode := http.StatusUnauthorized
			if pass {
				expStatusCode = http.StatusOK
			}
			require.Equal(t, w.Result().StatusCode, expStatusCode)
		})
	}
}

func TestDebugMux(t *testing.T) {
	h1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) })

	cases := []struct {
		desc string
		mux  debugMux
		tok  string
		want int
	}{
		{
			"only fleet auth handler, no token",
			debugMux{fleetAuthenticatedHandler: h1},
			"",
			200,
		},
		{
			"only fleet auth handler, with token",
			debugMux{fleetAuthenticatedHandler: h1},
			"token",
			200,
		},
		{
			"both handlers, no token",
			debugMux{fleetAuthenticatedHandler: h1, tokenAuthenticatedHandler: h2},
			"",
			200,
		},
		{
			"both handlers, with token",
			debugMux{fleetAuthenticatedHandler: h1, tokenAuthenticatedHandler: h2},
			"token",
			400,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			path := "/debug/pprof"
			if c.tok != "" {
				path += "?token=" + c.tok
			}
			req := httptest.NewRequest("GET", path, nil)
			res := httptest.NewRecorder()
			c.mux.ServeHTTP(res, req)
			require.Equal(t, c.want, res.Code)
		})
	}
}

func TestCronActivitiesStreaming(t *testing.T) {
	ds := new(mock.Store)

	newActivity := func(
		id uint,
		actorName string,
		actorID uint,
		actorGravatar, actorEmail, actType string,
		details string,
	) *fleet.Activity {
		jsonRawMessage := json.RawMessage(details)
		return &fleet.Activity{
			ID:            id,
			ActorFullName: &actorName,
			ActorID:       &actorID,
			ActorGravatar: &actorGravatar,
			ActorEmail:    &actorEmail,
			Type:          actType,
			Details:       &jsonRawMessage,
		}
	}

	a1 := newActivity(1, "foo1", 7, "foo1_gravatar", "foo1_email", "foobar1", `{"foo1":"bar1"}`)
	a2 := newActivity(2, "foo2", 8, "foo2_gravatar", "foo2_email", "foobar2", `{"foo2":"bar2"}`)
	a3 := newActivity(3, "foo3", 9, "foo3_gravatar", "foo3_email", "foobar3", `{"foo3":"bar3"}`)

	t.Run("basic", func(t *testing.T) {
		as := []*fleet.Activity{a1, a2, a3}

		ds.ListActivitiesFunc = func(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
			return as, nil, nil
		}

		ds.MarkActivitiesAsStreamedFunc = func(ctx context.Context, activityIDs []uint) error {
			require.Equal(t, []uint{1, 2, 3}, activityIDs)
			return nil
		}

		var auditLogger jsonLogger
		err := cronActivitiesStreaming(context.Background(), ds, log.NewNopLogger(), &auditLogger)
		require.NoError(t, err)
		require.Len(t, auditLogger.logs, 3)
		for i, m := range auditLogger.logs {
			var a *fleet.Activity
			err := json.Unmarshal([]byte(m), &a)
			require.NoError(t, err)
			require.Equal(t, as[i], a)
		}
	})

	t.Run("fail_to_stream_an_activity", func(t *testing.T) {
		as := []*fleet.Activity{a1, a2, a3}

		ds.ListActivitiesFunc = func(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
			return as, nil, nil
		}

		ds.MarkActivitiesAsStreamedFunc = func(ctx context.Context, activityIDs []uint) error {
			require.Equal(t, []uint{1}, activityIDs)
			return nil
		}

		auditLogger := jsonLogger{failAfter: 1}
		err := cronActivitiesStreaming(context.Background(), ds, log.NewNopLogger(), &auditLogger)
		require.Error(t, err)
		require.ErrorIs(t, err, errStreamFailed)
		require.Len(t, auditLogger.logs, 1)
		var a *fleet.Activity
		err = json.Unmarshal([]byte(auditLogger.logs[0]), &a)
		require.NoError(t, err)
		require.Equal(t, a1, a)
	})

	t.Run("bigger_than_batch", func(t *testing.T) {
		// Make slice that will require three iterations (3 pages,
		// two pages of ActivitiesToStreamBatchCount and one extra page of one item.
		as := make([]*fleet.Activity, ActivitiesToStreamBatchCount*2+1)
		for i := range as {
			as[i] = newActivity(uint(i), "foo", uint(i), //nolint:gosec // dismiss G115
				"foog", "fooe", "bar", `{"bar": "foo"}`)
		}

		ds.ListActivitiesFunc = func(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
			require.Equal(t, opt.PerPage, ActivitiesToStreamBatchCount)
			switch opt.Page {
			case 0:
				return as[:ActivitiesToStreamBatchCount], nil, nil
			case 1:
				return as[ActivitiesToStreamBatchCount : ActivitiesToStreamBatchCount*2], nil, nil
			case 2:
				return as[ActivitiesToStreamBatchCount*2:], nil, nil
			default:
				t.Fatalf("invalid page requested: %d", opt.Page)
				return nil, nil, nil
			}
		}

		call := 0
		firstBatch := make([]uint, ActivitiesToStreamBatchCount)
		secondBatch := make([]uint, ActivitiesToStreamBatchCount)
		for i := range as[:ActivitiesToStreamBatchCount] {
			firstBatch[i] = as[i].ID
		}
		for i := range as[ActivitiesToStreamBatchCount : ActivitiesToStreamBatchCount*2] {
			secondBatch[i] = as[int(ActivitiesToStreamBatchCount)+i].ID //nolint:gosec // dismiss G115
		}
		thirdBatch := []uint{as[len(as)-1].ID}
		ds.MarkActivitiesAsStreamedFunc = func(ctx context.Context, activityIDs []uint) error {
			switch call {
			case 0:
				require.Equal(t, firstBatch, activityIDs)
			case 1:
				require.Equal(t, secondBatch, activityIDs)
			case 2:
				require.Equal(t, thirdBatch, activityIDs)
			default:
				t.Fatalf("invalid number of calls: %d", call)
			}
			call += 1
			return nil
		}

		var auditLogger jsonLogger
		err := cronActivitiesStreaming(context.Background(), ds, log.NewNopLogger(), &auditLogger)
		require.NoError(t, err)
		require.Len(t, auditLogger.logs, int(ActivitiesToStreamBatchCount)*2+1) //nolint:gosec // dismiss G115
		require.Equal(t, 3, call)
	})
}

var errStreamFailed = errors.New("streaming failed")

type jsonLogger struct {
	logs      []string
	failAfter int
}

func (j *jsonLogger) Write(ctx context.Context, logs []json.RawMessage) error {
	for _, log := range logs {
		if j.failAfter > 0 && len(j.logs) == j.failAfter {
			return errStreamFailed
		}
		j.logs = append(j.logs, string(log))
	}
	return nil
}

func TestVerifyDiskEncryptionKeysJob(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := log.NewNopLogger()

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	recoveryKey := "AAA-BBB-CCC"
	encryptedKey, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{testCert})
	require.NoError(t, err)
	base64EncryptedKey := base64.StdEncoding.EncodeToString(encryptedKey)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetCACert: {Value: testCertPEM},
		fleet.MDMAssetCAKey:  {Value: testKeyPEM},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return assets, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appCfg := fleet.AppConfig{}
		appCfg.MDM.EnabledAndConfigured = true
		return &appCfg, nil
	}

	now := time.Now()

	t.Run("able to decrypt", func(t *testing.T) {
		ds.GetUnverifiedDiskEncryptionKeysFunc = func(ctx context.Context) ([]fleet.HostDiskEncryptionKey, error) {
			return []fleet.HostDiskEncryptionKey{
				{HostID: 1, Base64Encrypted: base64EncryptedKey, UpdatedAt: now},
				{HostID: 2, Base64Encrypted: base64EncryptedKey, UpdatedAt: now.Add(time.Hour)},
				{HostID: 3, Base64Encrypted: "BAD-KEY", UpdatedAt: now.Add(-time.Hour)},
			}, nil
		}

		calls := 0
		ds.SetHostsDiskEncryptionKeyStatusFunc = func(ctx context.Context, hostIDs []uint, decryptable bool, threshold time.Time) error {
			calls++
			require.Equal(t, now.Add(time.Hour), threshold)

			// first call, decryptable values
			if decryptable {
				require.EqualValues(t, []uint{1, 2}, hostIDs)
				return nil
			}

			// second call, non-decryptable values
			require.EqualValues(t, []uint{3}, hostIDs)
			return nil
		}

		err = verifyDiskEncryptionKeys(ctx, logger, ds)
		require.NoError(t, err)
		require.True(t, ds.GetUnverifiedDiskEncryptionKeysFuncInvoked)
		require.True(t, ds.SetHostsDiskEncryptionKeyStatusFuncInvoked)
		require.Equal(t, 2, calls)
	})

	t.Run("unable to decrypt", func(t *testing.T) {
		ds.GetUnverifiedDiskEncryptionKeysFunc = func(ctx context.Context) ([]fleet.HostDiskEncryptionKey, error) {
			return []fleet.HostDiskEncryptionKey{{HostID: 1, Base64Encrypted: "RANDOM"}}, nil
		}

		calls := 0
		ds.SetHostsDiskEncryptionKeyStatusFunc = func(ctx context.Context, hostIDs []uint, encryptable bool, threshold time.Time) error {
			calls++
			if !encryptable {
				require.EqualValues(t, []uint{1}, hostIDs)
				return nil
			}
			require.Empty(t, hostIDs)
			return nil
		}

		err = verifyDiskEncryptionKeys(ctx, logger, ds)
		require.NoError(t, err)
		require.True(t, ds.GetUnverifiedDiskEncryptionKeysFuncInvoked)
		require.True(t, ds.SetHostsDiskEncryptionKeyStatusFuncInvoked)
		require.Equal(t, 2, calls)
	})
}
