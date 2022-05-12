package main

import (
	"bytes"
	"context"
	"errors"
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

	startWebhooksSchedule(ctx, "test_instance", ds, appConfig, service.NewMemFailingPolicySet(), log.NewNopLogger())

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

	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		return nil, errors.New("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		return errors.New("forced error for test purposes")
	}

	vulnPath := path.Join(t.TempDir(), "something")
	require.NoDirExists(t, vulnPath)

	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         vulnPath,
			Periodicity:           100 * time.Millisecond,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Use schedule to test that the schedule does indeed call cronVulnerabilities.
	s := startVulnerabilitiesSchedule(ctx, "test_instance", ds, fleetConfig, log.NewNopLogger())

	time.Sleep(1 * time.Second)
	cancel()

	select {
	case <-s.Done():
		require.True(t, ds.AllSoftwareWithoutCPEIteratorFuncInvoked)
		require.True(t, ds.CalculateHostsPerSoftwareFuncInvoked)
		require.DirExists(t, vulnPath)
	case <-time.After(5 * time.Second):
		t.Error("timeout")
	}
}

// mbuffer is a mutex protected bytes.Buffer.
type mbuffer struct {
	m sync.Mutex
	b bytes.Buffer
}

func (b *mbuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Read(p)
}

func (b *mbuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.Write(p)
}

func (b *mbuffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()

	return b.b.String()
}

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

	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		return nil, fmt.Errorf("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		return fmt.Errorf("forced error for test purposes")
	}

	buf := mbuffer{}
	logger := level.NewFilter(kitlog.NewJSONLogger(&buf), level.AllowDebug())
	dbPath := t.TempDir()
	fleetConfig := config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DatabasesPath:         dbPath,
			Periodicity:           100 * time.Millisecond,
			CurrentInstanceChecks: "auto",
			DisableDataSync:       true,
		},
	}

	cronVulnerabilities(context.Background(), ds, logger, fleetConfig)

	require.True(t, ds.AllSoftwareWithoutCPEIteratorFuncInvoked)
	require.True(t, ds.CalculateHostsPerSoftwareFuncInvoked)
	require.Contains(t, buf.String(), "checking for recent vulnerabilities")
	require.Contains(t, buf.String(), fmt.Sprintf(`"vuln-path":"%s"`, dbPath))
}

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
	ds.AllSoftwareWithoutCPEIteratorFunc = func(ctx context.Context) (fleet.SoftwareIterator, error) {
		return nil, errors.New("forced error for test purposes")
	}
	ds.CalculateHostsPerSoftwareFunc = func(ctx context.Context, time time.Time) error {
		return errors.New("forced error for test purposes")
	}

	buf := mbuffer{}
	logger := level.NewFilter(kitlog.NewJSONLogger(&buf), level.AllowDebug())
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

	cronVulnerabilities(context.Background(), ds, logger, fleetConfig)

	require.Contains(t, buf.String(), `"databases-path":"creation failed, returning"`)
	require.False(t, ds.AllSoftwareWithoutCPEIteratorFuncInvoked)
	require.False(t, ds.CalculateHostsPerSoftwareFuncInvoked)
}

// TestCronWebhooksLockDuration tests that the Lock method is being called with a duration equal to the schedule interval
// TODO: should the lock duration be the schedule interval or always be set to one hour (see #3584)?
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

	startWebhooksSchedule(ctx, "test_instance", ds, appConfig, service.NewMemFailingPolicySet(), log.NewNopLogger())

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

	startWebhooksSchedule(ctx, "test_instance", ds, appConfig, service.NewMemFailingPolicySet(), log.NewNopLogger())

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
