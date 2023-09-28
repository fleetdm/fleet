package update

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestRenewEnrollmentProfile(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	cases := []struct {
		desc          string
		renewFlag     bool
		cmdErr        error
		wantCmdCalled bool
		wantLog       string
	}{
		{"renew=false", false, nil, false, ""},
		{"renew=true; success", true, nil, true, "successfully called /usr/bin/profiles to renew enrollment profile"},
		{"renew=true; fail", true, io.ErrUnexpectedEOF, true, "calling /usr/bin/profiles to renew enrollment profile failed"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			logBuf.Reset()

			fetcher := &dummyConfigFetcher{
				cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{RenewEnrollmentProfile: c.renewFlag}},
			}

			var cmdGotCalled bool
			var depAssignedCheckGotCalled bool
			renewFetcher := &renewEnrollmentProfileConfigFetcher{
				Fetcher:   fetcher,
				Frequency: time.Hour, // doesn't matter for this test
				runCmdFn: func() error {
					cmdGotCalled = true
					return c.cmdErr
				},
				checkEnrollmentFn: func() (bool, string, error) {
					return false, "", nil
				},
				checkAssignedEnrollmentProfileFn: func(url string) error {
					depAssignedCheckGotCalled = true
					return nil
				},
			}

			cfg, err := renewFetcher.GetConfig()
			require.NoError(t, err)            // the dummy fetcher never returns an error
			require.Equal(t, fetcher.cfg, cfg) // the renew enrollment wrapper properly returns the expected config

			require.Equal(t, c.wantCmdCalled, cmdGotCalled)
			require.Equal(t, c.wantCmdCalled, depAssignedCheckGotCalled)
			require.Contains(t, logBuf.String(), c.wantLog)
		})
	}
}

func TestRenewEnrollmentProfilePrevented(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	fetcher := &dummyConfigFetcher{
		cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{RenewEnrollmentProfile: true}},
	}

	var cmdCallCount int
	isEnrolled := false
	isAssigned := true
	chProceed := make(chan struct{})
	renewFetcher := &renewEnrollmentProfileConfigFetcher{
		Fetcher:   fetcher,
		Frequency: 2 * time.Second, // just to be safe with slow environments (CI)
		runCmdFn: func() error {
			cmdCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the fetcher's mutex
			return nil
		},
		checkEnrollmentFn: func() (bool, string, error) {
			<-chProceed // will be unblocked only when allowed
			return isEnrolled, "", nil
		},
		checkAssignedEnrollmentProfileFn: func(url string) error {
			<-chProceed // will be unblocked only when allowed
			if !isAssigned {
				return errors.New("not assigned")
			}
			return nil
		},
	}

	assertResult := func(cfg *fleet.OrbitConfig, err error) {
		require.NoError(t, err)
		require.Equal(t, fetcher.cfg, cfg)
	}

	started := make(chan struct{})
	go func() {
		close(started)

		// the first call will block in runCmdFn
		cfg, err := renewFetcher.GetConfig()
		assertResult(cfg, err)
	}()

	<-started
	// this call will happen while the first call is blocked in runCmdFn, so it
	// won't call the command (won't be able to lock the mutex). However it will
	// still complete successfully without being blocked by the other call in
	// progress.
	cfg, err := renewFetcher.GetConfig()
	assertResult(cfg, err)

	// unblock the first call
	close(chProceed)

	// this next call won't execute the command because of the frequency
	// restriction (it got called less than N seconds ago)
	cfg, err = renewFetcher.GetConfig()
	assertResult(cfg, err)

	// wait for the fetcher's frequency to pass
	time.Sleep(renewFetcher.Frequency)

	// this call executes the command
	cfg, err = renewFetcher.GetConfig()
	assertResult(cfg, err)

	// wait for the fetcher's frequency to pass
	time.Sleep(renewFetcher.Frequency)

	// this call doesn't execute the command since the host is already
	// enrolled
	isEnrolled = true
	cfg, err = renewFetcher.GetConfig()
	assertResult(cfg, err)

	require.Equal(t, 2, cmdCallCount) // the initial call and the one after sleep

	// wait for the fetcher's frequency to pass
	time.Sleep(renewFetcher.Frequency)

	// this call doesn't execute the command since the assigned profile check fails
	isAssigned = false
	isEnrolled = false
	cfg, err = renewFetcher.GetConfig()
	assertResult(cfg, err)

	require.Equal(t, 2, cmdCallCount) // the initial call and the one after sleep

	// wait for the fetcher's frequency to pass
	time.Sleep(renewFetcher.Frequency)

	// this next call won't execute the command because the backoff
	// for a failed assigned check is always 2 minutes
	cfg, err = renewFetcher.GetConfig()
	assertResult(cfg, err)
}

type mockNodeKeyGetter struct{}

func (m mockNodeKeyGetter) GetNodeKey() (string, error) {
	return "nodekey-test", nil
}

func TestWindowsMDMEnrollment(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	cases := []struct {
		desc          string
		enrollFlag    *bool
		unenrollFlag  *bool
		discoveryURL  string
		apiErr        error
		wantAPICalled bool
		wantLog       string
	}{
		{"enroll=false", ptr.Bool(false), nil, "", nil, false, ""},
		{"enroll=true,discovery=''", ptr.Bool(true), nil, "", nil, false, "discovery endpoint is empty"},
		{"enroll=true,discovery!='',success", ptr.Bool(true), nil, "http://example.com", nil, true, "successfully called RegisterDeviceWithManagement"},
		{"enroll=true,discovery!='',fail", ptr.Bool(true), nil, "http://example.com", io.ErrUnexpectedEOF, true, "enroll Windows device failed"},
		{"enroll=true,discovery!='',server", ptr.Bool(true), nil, "http://example.com", errIsWindowsServer, true, "device is a Windows Server, skipping enrollment"},

		{"unenroll=false", nil, ptr.Bool(false), "", nil, false, ""},
		{"unenroll=true,success", nil, ptr.Bool(true), "", nil, true, "successfully called UnregisterDeviceWithManagement"},
		{"unenroll=true,fail", nil, ptr.Bool(true), "", io.ErrUnexpectedEOF, true, "unenroll Windows device failed"},
		{"unenroll=true,server", nil, ptr.Bool(true), "", errIsWindowsServer, true, "device is a Windows Server, skipping unenrollment"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			logBuf.Reset()

			var (
				enroll     = c.enrollFlag != nil && *c.enrollFlag
				unenroll   = c.unenrollFlag != nil && *c.unenrollFlag
				isUnenroll = c.unenrollFlag != nil
			)
			fetcher := &dummyConfigFetcher{
				cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
					NeedsProgrammaticWindowsMDMEnrollment:   enroll,
					NeedsProgrammaticWindowsMDMUnenrollment: unenroll,
					WindowsMDMDiscoveryEndpoint:             c.discoveryURL,
				}},
			}

			var enrollGotCalled, unenrollGotCalled bool
			enrollFetcher := &windowsMDMEnrollmentConfigFetcher{
				Fetcher:   fetcher,
				Frequency: time.Hour, // doesn't matter for this test
				execEnrollFn: func(args WindowsMDMEnrollmentArgs) error {
					enrollGotCalled = true
					return c.apiErr
				},
				execUnenrollFn: func(args WindowsMDMEnrollmentArgs) error {
					unenrollGotCalled = true
					return c.apiErr
				},
				nodeKeyGetter: mockNodeKeyGetter{},
			}

			cfg, err := enrollFetcher.GetConfig()
			require.NoError(t, err)            // the dummy fetcher never returns an error
			require.Equal(t, fetcher.cfg, cfg) // the enrollment wrapper properly returns the expected config

			if isUnenroll {
				require.Equal(t, c.wantAPICalled, unenrollGotCalled)
				require.False(t, enrollGotCalled)
			} else {
				require.Equal(t, c.wantAPICalled, enrollGotCalled)
				require.False(t, unenrollGotCalled)
			}
			require.Contains(t, logBuf.String(), c.wantLog)
		})
	}
}

func TestWindowsMDMEnrollmentPrevented(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	cfgs := []fleet.OrbitConfigNotifications{
		{
			NeedsProgrammaticWindowsMDMEnrollment: true,
			WindowsMDMDiscoveryEndpoint:           "http://example.com",
		},
		{
			NeedsProgrammaticWindowsMDMUnenrollment: true,
		},
	}
	for _, cfg := range cfgs {
		t.Run(fmt.Sprintf("%+v", cfg), func(t *testing.T) {
			baseFetcher := &dummyConfigFetcher{
				cfg: &fleet.OrbitConfig{Notifications: cfg},
			}

			var (
				apiCallCount int
				apiErr       error
			)
			chProceed := make(chan struct{})
			fetcher := &windowsMDMEnrollmentConfigFetcher{
				Fetcher:       baseFetcher,
				Frequency:     2 * time.Second, // just to be safe with slow environments (CI)
				nodeKeyGetter: mockNodeKeyGetter{},
			}
			if cfg.NeedsProgrammaticWindowsMDMEnrollment {
				fetcher.execEnrollFn = func(args WindowsMDMEnrollmentArgs) error {
					<-chProceed    // will be unblocked only when allowed
					apiCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the fetcher's mutex
					return apiErr
				}
				fetcher.execUnenrollFn = func(args WindowsMDMEnrollmentArgs) error {
					panic("should not be called")
				}
			} else {
				fetcher.execUnenrollFn = func(args WindowsMDMEnrollmentArgs) error {
					<-chProceed    // will be unblocked only when allowed
					apiCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the fetcher's mutex
					return apiErr
				}
				fetcher.execEnrollFn = func(args WindowsMDMEnrollmentArgs) error {
					panic("should not be called")
				}
			}

			assertResult := func(cfg *fleet.OrbitConfig, err error) {
				require.NoError(t, err)
				require.Equal(t, baseFetcher.cfg, cfg)
			}

			started := make(chan struct{})
			go func() {
				close(started)

				// the first call will block in enroll/unenroll func
				cfg, err := fetcher.GetConfig()
				assertResult(cfg, err)
			}()

			<-started
			// this call will happen while the first call is blocked in
			// enroll/unenrollfn, so it won't call the API (won't be able to lock the
			// mutex). However it will still complete successfully without being
			// blocked by the other call in progress.
			cfg, err := fetcher.GetConfig()
			assertResult(cfg, err)

			// unblock the first call and wait for it to complete
			close(chProceed)
			time.Sleep(100 * time.Millisecond)

			// this next call won't execute the command because of the frequency
			// restriction (it got called less than N seconds ago)
			cfg, err = fetcher.GetConfig()
			assertResult(cfg, err)

			// wait for the fetcher's frequency to pass
			time.Sleep(fetcher.Frequency)

			// this call executes the command, and it returns the Is Windows Server error
			apiErr = errIsWindowsServer
			cfg, err = fetcher.GetConfig()
			assertResult(cfg, err)

			// this next call won't execute the command (both due to frequency and the
			// detection of windows server)
			cfg, err = fetcher.GetConfig()
			assertResult(cfg, err)

			// wait for the fetcher's frequency to pass
			time.Sleep(fetcher.Frequency)

			// this next call still won't execute the command (due to the detection of
			// windows server)
			cfg, err = fetcher.GetConfig()
			assertResult(cfg, err)

			require.Equal(t, 2, apiCallCount) // the initial call and the one that returned errIsWindowsServer after first sleep
		})
	}
}

func TestRunScripts(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	var (
		callsCount atomic.Int64
		runFailure error
		blockRun   chan struct{}
	)

	mockRun := func(r *scripts.Runner, ids []string) error {
		callsCount.Add(1)
		if blockRun != nil {
			<-blockRun
		}
		return runFailure
	}

	waitForRun := func(t *testing.T, r *runScriptsConfigFetcher) {
		var ok bool
		for start := time.Now(); !ok && time.Since(start) < time.Second; {
			ok = r.mu.TryLock()
		}
		require.True(t, ok, "timed out waiting for the lock to become available")
		r.mu.Unlock()
	}

	t.Run("no pending scripts", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset() })

		fetcher := &dummyConfigFetcher{
			cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				PendingScriptExecutionIDs: nil,
			}},
		}
		runner := &runScriptsConfigFetcher{
			Fetcher:      fetcher,
			runScriptsFn: mockRun,
		}
		cfg, err := runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config

		// the lock should be available because no goroutine was started
		require.True(t, runner.mu.TryLock())
		require.Zero(t, callsCount.Load()) // no calls to execute scripts
		require.Empty(t, logBuf.String())  // no logs written
	})

	t.Run("pending scripts succeed", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset() })

		fetcher := &dummyConfigFetcher{
			cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				PendingScriptExecutionIDs: []string{"a", "b", "c"},
			}},
		}
		runner := &runScriptsConfigFetcher{
			Fetcher:      fetcher,
			runScriptsFn: mockRun,
		}
		cfg, err := runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // all scripts executed in a single run
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts [a b c] succeeded")
	})

	t.Run("pending scripts failed", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset(); runFailure = nil })

		fetcher := &dummyConfigFetcher{
			cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				PendingScriptExecutionIDs: []string{"a", "b", "c"},
			}},
		}

		runFailure = io.ErrUnexpectedEOF
		runner := &runScriptsConfigFetcher{
			Fetcher:      fetcher,
			runScriptsFn: mockRun,
		}

		cfg, err := runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // all scripts executed in a single run
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts failed")
		require.Contains(t, logBuf.String(), io.ErrUnexpectedEOF.Error())
	})

	t.Run("concurrent run prevented", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset(); blockRun = nil })

		fetcher := &dummyConfigFetcher{
			cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				PendingScriptExecutionIDs: []string{"a", "b", "c"},
			}},
		}

		blockRun = make(chan struct{})
		runner := &runScriptsConfigFetcher{
			Fetcher:      fetcher,
			runScriptsFn: mockRun,
		}

		cfg, err := runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config

		// call it again, while the previous run is still running
		cfg, err = runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config

		// unblock the initial run
		close(blockRun)

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // only called once because of mutex
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts [a b c] succeeded")
	})

	t.Run("dynamic enabling of scripts", func(t *testing.T) {
		t.Cleanup(logBuf.Reset)

		fetcher := &dummyConfigFetcher{
			cfg: &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				PendingScriptExecutionIDs: []string{"a"},
			}},
		}

		var (
			scriptsEnabledCalls []bool
			dynamicEnabled      atomic.Bool

			dynamicInterval = 300 * time.Millisecond
		)

		runner := &runScriptsConfigFetcher{
			Fetcher:                 fetcher,
			ScriptsExecutionEnabled: false,
			runScriptsFn: func(r *scripts.Runner, s []string) error {
				scriptsEnabledCalls = append(scriptsEnabledCalls, r.ScriptExecutionEnabled)
				return nil
			},
			testGetFleetdConfig: func() (*fleet.MDMAppleFleetdConfig, error) {
				return &fleet.MDMAppleFleetdConfig{
					EnableScripts: dynamicEnabled.Load(),
				}, nil
			},
			dynamicScriptsEnabledCheckInterval: dynamicInterval,
		}

		// the static Scripts Enabled flag is false, so it relies on the dynamic check
		runner.runDynamicScriptsEnabledCheck()

		// first call, scripts are disabled
		cfg, err := runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config
		waitForRun(t, runner)

		// swap scripts execution to true and wait to ensure the dynamic check
		// did run.
		dynamicEnabled.Store(true)
		time.Sleep(dynamicInterval + 100*time.Millisecond)

		// second call, scripts are enabled (change exec ID to "b")
		cfg.Notifications.PendingScriptExecutionIDs[0] = "b"
		cfg, err = runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config
		waitForRun(t, runner)

		// swap scripts execution back to false and wait to ensure the dynamic
		// check did run.
		dynamicEnabled.Store(false)
		time.Sleep(dynamicInterval + 100*time.Millisecond)

		// third call, scripts are disabled (change exec ID to "c")
		cfg.Notifications.PendingScriptExecutionIDs[0] = "c"
		cfg, err = runner.GetConfig()
		require.NoError(t, err)            // the dummy fetcher never returns an error
		require.Equal(t, fetcher.cfg, cfg) // the wrapper properly returns the expected config
		waitForRun(t, runner)

		// validate the Scripts Enabled flags that were passed to the runScriptsFn
		require.Equal(t, []bool{false, true, false}, scriptsEnabledCalls)
		require.Contains(t, logBuf.String(), "received request to run scripts [a]")
		require.Contains(t, logBuf.String(), "running scripts [a] succeeded")
		require.Contains(t, logBuf.String(), "received request to run scripts [b]")
		require.Contains(t, logBuf.String(), "running scripts [b] succeeded")
		require.Contains(t, logBuf.String(), "received request to run scripts [c]")
		require.Contains(t, logBuf.String(), "running scripts [c] succeeded")
	})
}
