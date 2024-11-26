package update

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/bitlocker"
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

			testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{RenewEnrollmentProfile: c.renewFlag}}

			var cmdGotCalled bool
			var depAssignedCheckGotCalled bool
			renewReceiver := &renewEnrollmentProfileConfigReceiver{
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

			err := renewReceiver.Run(testConfig)
			require.NoError(t, err) // the dummy receiver never returns an error

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

	testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{RenewEnrollmentProfile: true}}

	var cmdCallCount int
	isEnrolled := false
	isAssigned := true
	chProceed := make(chan struct{})
	renewReceiver := &renewEnrollmentProfileConfigReceiver{
		Frequency: 2 * time.Second, // just to be safe with slow environments (CI)
		runCmdFn: func() error {
			cmdCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the receiver's mutex
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

	started := make(chan struct{})
	frequencyMu := sync.Mutex{}
	go func() {
		frequencyMu.Lock()
		defer frequencyMu.Unlock()
		close(started)

		// the first call will block in runCmdFn
		err := renewReceiver.Run(testConfig)
		require.NoError(t, err)
	}()

	<-started
	t.Logf("%v started", time.Now())
	// this call will happen while the first call is blocked in checkEnrollmentFn, so it
	// won't call the command (won't be able to lock the mutex). However, it will
	// still complete successfully without being blocked by the other call in
	// progress.
	err := renewReceiver.Run(testConfig)
	require.NoError(t, err)

	// unblock the first call
	close(chProceed)
	t.Logf("%v unblock the first call", time.Now())

	// this next call won't execute the command because of the frequency
	// restriction (it got called less than N seconds ago)
	err = renewReceiver.Run(testConfig)
	require.NoError(t, err)
	t.Logf("%v frequency restriction check done", time.Now())

	frequencyMu.Lock()
	renewReceiver.Frequency = 200 * time.Millisecond
	frequencyMu.Unlock()
	// wait for the receiver's frequency to pass
	time.Sleep(renewReceiver.Frequency)

	// this call executes the command
	err = renewReceiver.Run(testConfig)
	require.NoError(t, err)

	// wait for the receiver's frequency to pass
	time.Sleep(renewReceiver.Frequency)

	// this call doesn't execute the command since the host is already
	// enrolled
	isEnrolled = true
	err = renewReceiver.Run(testConfig)
	require.NoError(t, err)

	require.Equal(t, 2, cmdCallCount) // the initial call and the one after sleep

	// wait for the receiver's frequency to pass
	time.Sleep(renewReceiver.Frequency)

	// this call doesn't execute the command since the assigned profile check fails
	isAssigned = false
	isEnrolled = false
	err = renewReceiver.Run(testConfig)
	require.NoError(t, err)

	require.Equal(t, 2, cmdCallCount) // the initial call and the one after sleep

	// wait for the receiver's frequency to pass
	time.Sleep(renewReceiver.Frequency)

	// this next call won't execute the command because the backoff
	// for a failed assigned check is always 2 minutes
	err = renewReceiver.Run(testConfig)
	require.NoError(t, err)
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
		{"unenroll=true,server", nil, ptr.Bool(true), "", errIsWindowsServer, true, "device is a Windows Server, skipping unenroll"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			logBuf.Reset()

			var (
				enroll     = c.enrollFlag != nil && *c.enrollFlag
				unenroll   = c.unenrollFlag != nil && *c.unenrollFlag
				isUnenroll = c.unenrollFlag != nil
			)

			testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
				NeedsProgrammaticWindowsMDMEnrollment:   enroll,
				NeedsProgrammaticWindowsMDMUnenrollment: unenroll,
				WindowsMDMDiscoveryEndpoint:             c.discoveryURL,
			}}

			var enrollGotCalled, unenrollGotCalled bool
			enrollReceiver := &windowsMDMEnrollmentConfigReceiver{
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

			err := enrollReceiver.Run(testConfig)
			require.NoError(t, err) // the dummy receiver never returns an error

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
			testConfig := &fleet.OrbitConfig{Notifications: cfg}

			var (
				apiCallCount int
				apiErr       error
			)
			chProceed := make(chan struct{})
			receiver := &windowsMDMEnrollmentConfigReceiver{
				Frequency:     2 * time.Second, // just to be safe with slow environments (CI)
				nodeKeyGetter: mockNodeKeyGetter{},
			}
			if cfg.NeedsProgrammaticWindowsMDMEnrollment {
				receiver.execEnrollFn = func(args WindowsMDMEnrollmentArgs) error {
					<-chProceed    // will be unblocked only when allowed
					apiCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the receiver's mutex
					return apiErr
				}
				receiver.execUnenrollFn = func(args WindowsMDMEnrollmentArgs) error {
					panic("should not be called")
				}
			} else {
				receiver.execUnenrollFn = func(args WindowsMDMEnrollmentArgs) error {
					<-chProceed    // will be unblocked only when allowed
					apiCallCount++ // no need for sync, single-threaded call of this func is guaranteed by the receiver's mutex
					return apiErr
				}
				receiver.execEnrollFn = func(args WindowsMDMEnrollmentArgs) error {
					panic("should not be called")
				}
			}

			go func() {
				// the first call will block in enroll/unenroll func
				err := receiver.Run(testConfig)
				require.NoError(t, err)
			}()

			// wait a little bit to ensure the first `receiver.Run` call runs first.
			time.Sleep(100 * time.Millisecond)

			// this call will happen while the first call is blocked in
			// enroll/unenrollfn, so it won't call the API (won't be able to lock the
			// mutex). However it will still complete successfully without being
			// blocked by the other call in progress.
			err := receiver.Run(testConfig)
			require.NoError(t, err)

			// unblock the first call and wait for it to complete
			close(chProceed)
			time.Sleep(100 * time.Millisecond)

			// this next call won't execute the command because of the frequency
			// restriction (it got called less than N seconds ago)
			err = receiver.Run(testConfig)
			require.NoError(t, err)

			// wait for the receiver's frequency to pass
			time.Sleep(receiver.Frequency)

			// this call executes the command, and it returns the Is Windows Server error
			apiErr = errIsWindowsServer
			err = receiver.Run(testConfig)
			require.NoError(t, err)

			// this next call won't execute the command (both due to frequency and the
			// detection of windows server)
			err = receiver.Run(testConfig)
			require.NoError(t, err)

			// wait for the receiver's frequency to pass
			time.Sleep(receiver.Frequency)

			// this next call still won't execute the command (due to the detection of
			// windows server)
			err = receiver.Run(testConfig)
			require.NoError(t, err)

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

	waitForRun := func(t *testing.T, r *runScriptsConfigReceiver) {
		var ok bool
		for start := time.Now(); !ok && time.Since(start) < time.Second; {
			ok = r.mu.TryLock()
		}
		require.True(t, ok, "timed out waiting for the lock to become available")
		r.mu.Unlock()
	}

	t.Run("no pending scripts", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset() })

		testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			PendingScriptExecutionIDs: nil,
		}}

		runner := &runScriptsConfigReceiver{
			runScriptsFn: mockRun,
		}
		err := runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error

		// the lock should be available because no goroutine was started
		require.True(t, runner.mu.TryLock())
		require.Zero(t, callsCount.Load()) // no calls to execute scripts
		require.Empty(t, logBuf.String())  // no logs written
	})

	t.Run("pending scripts succeed", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset() })

		testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			PendingScriptExecutionIDs: []string{"a", "b", "c"},
		}}

		runner := &runScriptsConfigReceiver{
			runScriptsFn: mockRun,
		}
		err := runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // all scripts executed in a single run
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts [a b c] succeeded")
	})

	t.Run("pending scripts failed", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset(); runFailure = nil })

		testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			PendingScriptExecutionIDs: []string{"a", "b", "c"},
		}}

		runFailure = io.ErrUnexpectedEOF
		runner := &runScriptsConfigReceiver{
			runScriptsFn: mockRun,
		}

		err := runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // all scripts executed in a single run
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts failed")
		require.Contains(t, logBuf.String(), io.ErrUnexpectedEOF.Error())
	})

	t.Run("concurrent run prevented", func(t *testing.T) {
		t.Cleanup(func() { callsCount.Store(0); logBuf.Reset(); blockRun = nil })

		testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			PendingScriptExecutionIDs: []string{"a", "b", "c"},
		}}

		blockRun = make(chan struct{})
		runner := &runScriptsConfigReceiver{
			runScriptsFn: mockRun,
		}

		err := runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error

		// call it again, while the previous run is still running
		err = runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error

		// unblock the initial run
		close(blockRun)

		waitForRun(t, runner)
		require.Equal(t, int64(1), callsCount.Load()) // only called once because of mutex
		require.Contains(t, logBuf.String(), "received request to run scripts [a b c]")
		require.Contains(t, logBuf.String(), "running scripts [a b c] succeeded")
	})

	t.Run("dynamic enabling of scripts", func(t *testing.T) {
		t.Cleanup(logBuf.Reset)

		testConfig := &fleet.OrbitConfig{Notifications: fleet.OrbitConfigNotifications{
			PendingScriptExecutionIDs: []string{"a"},
		}}

		var (
			scriptsEnabledCalls []bool
			dynamicEnabled      atomic.Bool

			dynamicInterval = 300 * time.Millisecond
		)

		runner := &runScriptsConfigReceiver{
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
		err := runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
		waitForRun(t, runner)

		// swap scripts execution to true and wait to ensure the dynamic check
		// did run.
		dynamicEnabled.Store(true)
		time.Sleep(dynamicInterval + 100*time.Millisecond)

		// second call, scripts are enabled (change exec ID to "b")
		testConfig.Notifications.PendingScriptExecutionIDs[0] = "b"
		err = runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
		waitForRun(t, runner)

		// swap scripts execution back to false and wait to ensure the dynamic
		// check did run.
		dynamicEnabled.Store(false)
		time.Sleep(dynamicInterval + 100*time.Millisecond)

		// third call, scripts are disabled (change exec ID to "c")
		testConfig.Notifications.PendingScriptExecutionIDs[0] = "c"
		err = runner.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
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

type mockDiskEncryptionKeySetter struct {
	SetOrUpdateDiskEncryptionKeyImpl    func(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error
	SetOrUpdateDiskEncryptionKeyInvoked bool
}

func (m *mockDiskEncryptionKeySetter) SetOrUpdateDiskEncryptionKey(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error {
	m.SetOrUpdateDiskEncryptionKeyInvoked = true
	return m.SetOrUpdateDiskEncryptionKeyImpl(diskEncryptionStatus)
}

func TestBitlockerOperations(t *testing.T) {
	var logBuf bytes.Buffer

	oldLog := log.Logger
	log.Logger = log.Output(&logBuf)
	t.Cleanup(func() { log.Logger = oldLog })

	var (
		shouldEncrypt          = true
		shouldFailEncryption   = false
		shouldFailDecryption   = false
		shouldFailServerUpdate = false
		encryptFnCalled        = false
		decryptFnCalled        = false
	)

	testConfig := &fleet.OrbitConfig{
		Notifications: fleet.OrbitConfigNotifications{
			EnforceBitLockerEncryption: shouldEncrypt,
		},
	}

	clientMock := &mockDiskEncryptionKeySetter{}
	clientMock.SetOrUpdateDiskEncryptionKeyImpl = func(diskEncryptionStatus fleet.OrbitHostDiskEncryptionKeyPayload) error {
		if shouldFailServerUpdate {
			return errors.New("server error")
		}
		return nil
	}

	var enrollReceiver *windowsMDMBitlockerConfigReceiver
	setupTest := func() {
		enrollReceiver = &windowsMDMBitlockerConfigReceiver{
			Frequency:        time.Hour, // doesn't matter for this test
			lastRun:          time.Now().Add(-2 * time.Hour),
			EncryptionResult: clientMock,
			execGetEncryptionStatusFn: func() ([]bitlocker.VolumeStatus, error) {
				return []bitlocker.VolumeStatus{}, nil
			},
			execEncryptVolumeFn: func(string) (string, error) {
				encryptFnCalled = true
				if shouldFailEncryption {
					return "", errors.New("error encrypting")
				}

				return "123456", nil
			},
			execDecryptVolumeFn: func(string) error {
				decryptFnCalled = true
				if shouldFailDecryption {
					return errors.New("error decrypting")
				}

				return nil
			},
		}
		shouldEncrypt = true
		shouldFailEncryption = false
		shouldFailDecryption = false
		shouldFailServerUpdate = false
		encryptFnCalled = false
		decryptFnCalled = false
		clientMock.SetOrUpdateDiskEncryptionKeyInvoked = false
		logBuf.Reset()
	}

	t.Run("bitlocker encryption is performed", func(t *testing.T) {
		setupTest()
		shouldEncrypt = true
		shouldFailEncryption = false
		shouldFailDecryption = false
		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
	})

	t.Run("bitlocker encryption is not performed", func(t *testing.T) {
		setupTest()
		shouldEncrypt = false
		shouldFailEncryption = false
		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
		require.True(t, encryptFnCalled, "encryption function should have been called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})

	t.Run("bitlocker encryption returns an error", func(t *testing.T) {
		setupTest()
		shouldEncrypt = true
		shouldFailEncryption = true
		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err) // the dummy receiver never returns an error
		require.True(t, encryptFnCalled, "encryption function should have been called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})

	t.Run("encryption skipped based on various current statuses", func(t *testing.T) {
		setupTest()
		statusesToTest := []int32{
			bitlocker.ConversionStatusDecryptionInProgress,
			bitlocker.ConversionStatusDecryptionPaused,
			bitlocker.ConversionStatusEncryptionInProgress,
			bitlocker.ConversionStatusEncryptionPaused,
		}

		for _, status := range statusesToTest {
			t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
				mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: status}
				enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
					return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
				}

				err := enrollReceiver.Run(testConfig)
				require.NoError(t, err)
				require.Contains(t, logBuf.String(), "skipping encryption as the disk is not available")
				require.False(t, encryptFnCalled, "encryption function should not be called")
				require.False(t, decryptFnCalled, "decryption function should not be called")
				logBuf.Reset() // Reset the log buffer for the next iteration
			})
		}
	})

	t.Run("handle misreported decryption error", func(t *testing.T) {
		setupTest()
		mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: bitlocker.ConversionStatusFullyDecrypted}
		enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
			return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
		}
		enrollReceiver.execEncryptVolumeFn = func(string) (string, error) {
			return "", bitlocker.NewEncryptionError("", bitlocker.ErrorCodeNotDecrypted)
		}

		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.Contains(t, logBuf.String(), "disk encryption failed due to previous unsuccessful attempt, user action required")
		require.False(t, encryptFnCalled, "encryption function should not be called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})

	t.Run("decrypts the disk if previously encrypted", func(t *testing.T) {
		setupTest()
		mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: bitlocker.ConversionStatusFullyEncrypted}
		enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
			return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
		}
		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.Contains(t, logBuf.String(), "disk was previously encrypted. Attempting to decrypt it")
		require.False(t, clientMock.SetOrUpdateDiskEncryptionKeyInvoked)
		require.False(t, encryptFnCalled, "encryption function should not have been called")
		require.True(t, decryptFnCalled, "decryption function should have been called")
	})

	t.Run("reports to the server if decryption fails", func(t *testing.T) {
		setupTest()
		shouldFailDecryption = true
		mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: bitlocker.ConversionStatusFullyEncrypted}
		enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
			return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
		}

		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.Contains(t, logBuf.String(), "disk was previously encrypted. Attempting to decrypt it")
		require.Contains(t, logBuf.String(), "decryption failed")
		require.True(t, clientMock.SetOrUpdateDiskEncryptionKeyInvoked)
		require.False(t, encryptFnCalled, "encryption function should not be called")
		require.True(t, decryptFnCalled, "decryption function should have been called")
	})

	t.Run("encryption skipped if last run too recent", func(t *testing.T) {
		setupTest()
		enrollReceiver.lastRun = time.Now().Add(-30 * time.Minute)
		enrollReceiver.Frequency = 1 * time.Hour

		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.Contains(t, logBuf.String(), "skipped encryption process, last run was too recent")
		require.False(t, encryptFnCalled, "encryption function should not be called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})

	t.Run("successful fleet server update", func(t *testing.T) {
		setupTest()
		shouldFailEncryption = false
		mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: bitlocker.ConversionStatusFullyDecrypted}
		enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
			return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
		}

		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.True(t, clientMock.SetOrUpdateDiskEncryptionKeyInvoked)
		require.True(t, encryptFnCalled, "encryption function should have been called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})

	t.Run("failed fleet server update", func(t *testing.T) {
		setupTest()
		shouldFailEncryption = false
		shouldFailServerUpdate = true
		mockStatus := &bitlocker.EncryptionStatus{ConversionStatus: bitlocker.ConversionStatusFullyDecrypted}
		enrollReceiver.execGetEncryptionStatusFn = func() ([]bitlocker.VolumeStatus, error) {
			return []bitlocker.VolumeStatus{{DriveVolume: "C:", Status: mockStatus}}, nil
		}

		err := enrollReceiver.Run(testConfig)
		require.NoError(t, err)
		require.Contains(t, logBuf.String(), "failed to send encryption result to Fleet Server")
		require.True(t, clientMock.SetOrUpdateDiskEncryptionKeyInvoked)
		require.True(t, encryptFnCalled, "encryption function should have been called")
		require.False(t, decryptFnCalled, "decryption function should not be called")
	})
}
