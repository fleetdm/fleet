package useraction

import (
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

// mockDialog is a mock implementation of the dialog interface for testing.
type mockDialog struct {
	exitCh chan int // exit code
}

func (d *mockDialog) CanRun() bool {
	return true
}

func (d *mockDialog) Exit() {
	select {
	case d.exitCh <- unknownExitCode:
	default:
	}
}

func (d *mockDialog) exitWithCode(code int) {
	select {
	case d.exitCh <- code:
	default:
	}
}

func (d *mockDialog) render(flags ...string) (chan swiftDialogExitCode, chan error) {
	exitCodeCh := make(chan swiftDialogExitCode, 1)
	errCh := make(chan error, 1)
	go func() {
		select {
		case code := <-d.exitCh:
			exitCodeCh <- swiftDialogExitCode(code)
		case <-time.After(15 * time.Second):
			errCh <- errors.New("timeout waiting for mock dialog to exit")
		}
	}()
	return exitCodeCh, errCh
}

// mockReadWriter is a mock implementation of the readWriter interface for testing.
type mockReadWriter struct {
	migrationType string
}

func (rw *mockReadWriter) GetMigrationType() (string, error) {
	return rw.migrationType, nil
}

func (rw *mockReadWriter) SetMigrationFile(typ string) error {
	rw.migrationType = typ
	return nil
}

func (rw *mockReadWriter) RemoveFile() error {
	rw.migrationType = ""
	return nil
}

type dummyHandler struct {
	TimeCalled int
}

func (d *dummyHandler) NotifyRemote() error {
	d.TimeCalled++
	return nil
}

func (d dummyHandler) ShowInstructions() error { return nil }

func TestWaitForUnenrollment(t *testing.T) {
	getMigratorInstance := func() *swiftDialogMDMMigrator {
		return &swiftDialogMDMMigrator{
			handler:                   &dummyHandler{},
			baseDialog:                newBaseDialog("foo/bar"),
			frequency:                 15 * time.Minute,
			unenrollmentRetryInterval: 1 * time.Millisecond,
			maxUnenrollmentWaitTime:   1 * time.Second,
		}
	}

	cases := []struct {
		name                string
		enrollErr           error
		unenrollAfterNTries int
		wantErr             bool
	}{
		{"unenroll after 3 tries", nil, 3, false},
		{"unenroll after one try", nil, 1, false},
		{"error after max number of tries is exceeded", nil, 1000, true},
		{"always error calling profiles func", errors.New("test"), 1, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			m := getMigratorInstance()
			tries := 0
			m.testEnrollmentCheckFileFn = func() (bool, error) {
				if tries >= c.unenrollAfterNTries {
					return false, c.enrollErr
				}
				tries++
				return true, c.enrollErr
			}

			m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
				return true, "example.com", nil
			}

			outErr := m.waitForUnenrollment(true)
			if c.wantErr {
				require.Error(t, outErr)
			} else {
				require.NoError(t, outErr)
				require.Equal(t, c.unenrollAfterNTries, tries)
			}
		})
	}

	t.Run("fallback to enrollment check file", func(t *testing.T) {
		t.Parallel()
		m := getMigratorInstance()
		m.testEnrollmentCheckFileFn = func() (bool, error) {
			return true, nil
		}

		m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
			return false, "", nil
		}

		outErr := m.waitForUnenrollment(true)
		require.NoError(t, outErr)
	})

	t.Run("only check file during ADE enrollment", func(t *testing.T) {
		t.Parallel()
		m := getMigratorInstance()
		var fileWasChecked bool
		m.testEnrollmentCheckFileFn = func() (bool, error) {
			fileWasChecked = true
			return true, nil
		}

		m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
			return false, "", nil
		}

		err := m.waitForUnenrollment(false)
		require.NoError(t, err)
		require.False(t, fileWasChecked)

		err = m.waitForUnenrollment(true)
		require.NoError(t, err)
		require.True(t, fileWasChecked)
	})
}

func TestShouldSendWebhookUntilUnmanaged(t *testing.T) {
	for _, typ := range []string{constant.MDMMigrationTypeADE, constant.MDMMigrationTypeManual, constant.MDMMigrationTypePreSonoma} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			handler := &dummyHandler{}
			mockDialog := &mockDialog{exitCh: make(chan int, 10)}
			m := &swiftDialogMDMMigrator{
				handler:                   handler,
				mrw:                       &mockReadWriter{},
				baseDialog:                mockDialog,
				frequency:                 15 * time.Minute,
				unenrollmentRetryInterval: 50 * time.Millisecond,
				maxUnenrollmentWaitTime:   100 * time.Millisecond,
				props: MDMMigratorProps{
					IsUnmanaged: false,
				},
			}

			// Set up enrollment check functions - device stays enrolled throughout
			m.testEnrollmentCheckFileFn = func() (bool, error) {
				return true, nil // Always enrolled (file exists)
			}

			m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
				return true, "example.com", nil // Always enrolled
			}

			// First migration attempt - should call webhook and see device never unenrolls for unenrollment
			mockDialog.exitWithCode(0) // Start button clicked
			mockDialog.exitWithCode(0) // Error ok? clicked
			err := m.renderMigration()

			// Should get host is still enrolled error
			require.Error(t, err)
			require.Contains(t, err.Error(), "host didn't unenroll from MDM") // This is okay
			require.Equal(t, 1, handler.TimeCalled)

			// We fake the migration file being set even though an error returned to simulate this weird state
			err = m.mrw.SetMigrationFile(typ)
			require.NoError(t, err)

			// Second migration attempt - device is still managed, should call webhook again
			mockDialog.exitWithCode(0)
			mockDialog.exitWithCode(0)
			err = m.renderMigration()

			// Should still get not unenrolled error
			require.Error(t, err)
			require.Contains(t, err.Error(), "host didn't unenroll from MDM")
			require.Equal(t, 2, handler.TimeCalled) // webhook was still called

			// Now we let it unenroll the device, and then simulate the ping for IsUnmanaged
			fileTries := 0
			statusTries := 0
			m.testEnrollmentCheckFileFn = func() (bool, error) {
				fileTries++
				if fileTries > 1 { // Unenroll after 2nd try
					return false, nil
				}
				return true, nil
			}

			m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
				statusTries++
				if statusTries > 1 {
					return false, "", nil // Not enrolled
				}
				return true, "example.com", nil
			}

			go func() {
				// start button click
				time.Sleep(10 * time.Millisecond)
				mockDialog.exitWithCode(0)

				// There is a loading spinner that takes over the exit call, so we need to call it ourselves again.
				time.Sleep(100 * time.Millisecond)
				mockDialog.Exit()
			}()
			err = m.renderMigration()

			// Now it successfully unenrolls
			require.NoError(t, err)
			require.Equal(t, 3, handler.TimeCalled) // webhook was called again.

			// Device is now seen as unmanaged by Fleet server
			m.props.IsUnmanaged = true

			// This simulates our runner that periodically shows the window - should NOT call webhook since device is unmanaged, it will hit the early exit
			mockDialog.exitWithCode(0)
			err = m.renderMigration()

			// Should succeed without error
			require.NoError(t, err)
			require.Equal(t, 3, handler.TimeCalled) // webhook was not called
		})
	}
}
