package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/schedule"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/log"
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

// TODO: fix races?
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

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	fmt.Println(webhooksInterval)
	webhooks, err := schedule.New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.SetConfigInterval(5 * time.Minute)
	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return cronWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

	<-calledOnce
	time.Sleep(1 * time.Second)
	assert.Equal(t, int32(1), atomic.LoadInt32(&endpointCalled))
	<-calledTwice
	time.Sleep(1 * time.Second)
	assert.GreaterOrEqual(t, int32(2), atomic.LoadInt32(&endpointCalled))
}

func TestCronVulnerabilitiesCreatesDatabasesPath(t *testing.T) {
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

	// because the path should be created before the bulk of vuln processing begins, we can use a call to any ds methods
	// below to signal that we are ready to make our test assertion without waiting for processing to finish
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	logger := log.NewNopLogger()
	vulnerabilities, err := schedule.New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return cronVulnerabilities(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			require.DirExists(t, vulnPath)
			break TEST

		case <-failCheck:
			require.DirExists(t, vulnPath)
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesAcceptsExistingDbPath(t *testing.T) {
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

	// because the path should be created before the bulk of vuln processing begins, we can use a call to any ds methods
	// below to signal that we are ready to make our test assertion without waiting for processing to finish
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())
	dbPath := t.TempDir()
	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         dbPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	vulnerabilities, err := schedule.New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return cronVulnerabilities(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			require.Contains(t, buf.String(), "checking for recent vulnerabilities")
			require.Contains(t, buf.String(), fmt.Sprintf(`"vuln-path":"%s"`, dbPath))
			break TEST

		case <-failCheck:
			require.Contains(t, buf.String(), "checking for recent vulnerabilities")
			require.Contains(t, buf.String(), fmt.Sprintf(`"vuln-path":"%s"`, dbPath))
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesQuitsIfErrorVulnPath(t *testing.T) {
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

	// because the logic we care about should be created before the bulk of vuln processing begins,
	// we can use a call to any ds methods below to signal a failed test
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	fileVulnPath := path.Join(t.TempDir(), "somefile")
	_, err := os.Create(fileVulnPath)
	require.NoError(t, err)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         fileVulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true, // TODO: do we need for this test?
		},
	}

	vulnerabilities, err := schedule.New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return cronVulnerabilities(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			t.FailNow() // TODO: review this test with Tomas
		case <-failCheck:
			require.Contains(t, buf.String(), `"databases-path":"creation failed, returning"`)
			break TEST
		}
	}
}

// TODO: fix races
func TestCronVulnerabilitiesSkipCreationIfStatic(t *testing.T) {
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

	// because the logic we care about should be created before the bulk of vuln processing begins,
	// we can use a call to any ds methods below to signal a failed test
	dsSoftwareFnCalled := make(chan bool)
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		dsSoftwareFnCalled <- true
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		dsSoftwareFnCalled <- true
		return fmt.Errorf("forced error for test purposes")
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	buf := new(bytes.Buffer)
	logger := kitlog.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           1 * time.Second,
			CurrentInstanceChecks: "1",
			DisableDataSync:       true, // TODO: do we need for this test?

		},
	}

	vulnerabilities, err := schedule.New(ctx, "vulnerabilities", "test_instance", fleetConfig.Vulnerabilities.Periodicity, ds, logger)
	require.NoError(t, err)

	vulnerabilities.SetPreflightCheck(func() bool { return fleetConfig.Vulnerabilities.CurrentInstanceChecks == "auto" })
	vulnerabilities.SetConfigCheck(func(time.Time, time.Duration) (*time.Duration, error) {
		return &fleetConfig.Vulnerabilities.Periodicity, nil
	})
	vulnerabilities.AddJob("cron_vulnerabilities", func(ctx context.Context) (interface{}, error) {
		return cronVulnerabilities(ctx, ds, logger, fleetConfig)
	}, func(interface{}, error) {})

	failCheck := time.After(5 * time.Second)

TEST:
	for {
		select {
		case <-dsSoftwareFnCalled:
			t.FailNow() // TODO: review this test with Tomas
		case <-failCheck:
			require.NoDirExists(t, vulnPath)
			break TEST
		}
	}
}

// TestCronWebhooksLockDuration tests that the Lock method is being called with a duration equal to the schedule interval
// TODO: should the lock duration be the schedule interval or always be set to one hour (see #3584)?
// TODO: fix races
func TestCronWebhooksLockDuration(t *testing.T) {
	ds := new(mock.Store)
	interval := 1 * time.Second

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				Interval: fleet.Duration{Duration: interval},
			},
		}, nil
	}
	hostStatus := make(chan struct{})
	hostStatusClosed := false
	failingPolicies := make(chan struct{})
	failingPoliciesClosed := false
	unknownName := false
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		if expiration != interval {
			return false, nil
		}
		switch name {
		case "webhooks":
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
		return true, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	fmt.Println(webhooksInterval)
	webhooks, err := schedule.New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return cronWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

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

// TODO: fix races
func TestCronWebhooksIntervalChange(t *testing.T) {
	ds := new(mock.Store)

	interval := struct {
		sync.Mutex
		value time.Duration
	}{
		value: 5 * time.Hour,
	}
	configLoaded := make(chan struct{}, 1)

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

	lockCalled := make(chan struct{}, 1)
	ds.LockFunc = func(ctx context.Context, name string, owner string, expiration time.Duration) (bool, error) {
		select {
		case lockCalled <- struct{}{}:
		default:
			// OK
		}
		return true, nil
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)

	webhooksLogger := log.NewNopLogger()
	webhooksInterval := appConfig.WebhookSettings.Interval.ValueOr(30 * time.Second)
	webhooks, err := schedule.New(ctx, "webhooks", "test_instance", webhooksInterval, ds, webhooksLogger)
	require.NoError(t, err)

	webhooks.SetConfigInterval(200 * time.Millisecond)
	webhooks.SetConfigCheck(SetWebhooksConfigCheck(ctx, ds, webhooksLogger))
	webhooks.AddJob("cron_webhooks", func(ctx context.Context) (interface{}, error) {
		return cronWebhooks(ctx, ds, webhooksLogger, service.NewMemFailingPolicySet())
	}, func(interface{}, error) {})

	select {
	case <-configLoaded:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: initial config load")
	}

	interval.Lock()
	interval.value = 1 * time.Second
	interval.Unlock()

	select {
	case <-lockCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: schedInterval change did not trigger lock call")
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
