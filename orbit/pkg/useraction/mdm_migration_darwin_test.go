package useraction

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type dummyHandler struct{}

func (d dummyHandler) NotifyRemote() error {
	return nil
}

func (d dummyHandler) ShowInstructions() error { return nil }

func TestWaitForUnenrollment(t *testing.T) {
	m := &swiftDialogMDMMigrator{
		handler:                   dummyHandler{},
		baseDialog:                newBaseDialog("foo/bar"),
		frequency:                 15 * time.Minute,
		unenrollmentRetryInterval: 300 * time.Millisecond,
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

			outErr := m.waitForUnenrollment()
			if c.wantErr {
				require.Error(t, outErr)
			} else {
				require.NoError(t, outErr)
				require.Equal(t, c.unenrollAfterNTries, tries)
			}
		})
	}

	t.Run("fallback to enrollment check file", func(t *testing.T) {
		m.testEnrollmentCheckFileFn = func() (bool, error) {
			return true, nil
		}

		m.testEnrollmentCheckStatusFn = func() (bool, string, error) {
			return false, "", nil
		}

		outErr := m.waitForUnenrollment()
		require.NoError(t, outErr)
	})
}
