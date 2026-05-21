//go:build windows

package bitlocker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWorker is a process-wide COMWorker shared by all tests. This mirrors
// production, where orbit creates exactly one COMWorker for its lifetime.
// Creating multiple sequential COMWorkers in the same process is unsafe because
// COM has process-level state and re-initialization races with teardown.
var testWorker *COMWorker

func TestMain(m *testing.M) {
	w, err := NewCOMWorker()
	if err != nil {
		os.Exit(1)
	}
	testWorker = w
	code := m.Run()
	w.Close()
	os.Exit(code)
}

func TestFveErrorCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  int32
		want string
	}{
		{"zero", 0, "0 (0x00000000)"},
		{"positive", 42, "42 (0x0000002a)"},
		{"negative (FVE error)", ErrorCodeNotDecrypted, "2150694969 (0x80310039)"},
		{"E_INVALIDARG", ErrorCodeInvalidArg, "2147942487 (0x80070057)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fveErrorCode(tt.val))
		})
	}
}

func TestEncryptErrHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     int32
		wantCode int32
		wantMsg  string
	}{
		{"InvalidArg", ErrorCodeInvalidArg, ErrorCodeInvalidArg, "encryption flags conflict"},
		{"IODevice", ErrorCodeIODevice, ErrorCodeIODevice, "I/O error"},
		{"NotDecrypted", ErrorCodeNotDecrypted, ErrorCodeNotDecrypted, "fully decrypted"},
		{"ProtectorExists", ErrorCodeProtectorExists, ErrorCodeProtectorExists, "only one key protector"},
		{"unknown code", 99, 99, "error code returned during encryption"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := encryptErrHandler(tt.code)
			require.Error(t, err)

			var encErr *EncryptionError
			require.ErrorAs(t, err, &encErr)
			assert.Equal(t, tt.wantCode, encErr.Code())
			assert.Contains(t, encErr.Error(), tt.wantMsg)
		})
	}
}

func TestIntToPercentage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		num  int32
		want string
	}{
		{"zero", 0, "0.00%"},
		{"one percent", 10000, "1.00%"},
		{"hundred percent", 1000000, "100.00%"},
		{"half", 500000, "50.00%"},
		{"fractional", 123456, "12.35%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, intToPercentage(tt.num))
		})
	}
}

func TestBitsToDrives(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bitMap  uint32
		want    []string
		wantLen int
	}{
		{"no drives", 0x0, nil, 0},
		{"only A:", 0x1, []string{"A:"}, 1},
		{"only C:", 0x4, []string{"C:"}, 1},
		{"A: and C:", 0x5, []string{"A:", "C:"}, 2},
		{"C: and D:", 0xC, []string{"C:", "D:"}, 2},
		{"all 26 drives", 0x03FFFFFF, nil, 26},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bitsToDrives(tt.bitMap)
			assert.Len(t, got, tt.wantLen)
			if tt.want != nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetLogicalVolumes(t *testing.T) {
	drives, err := getLogicalVolumes()
	require.NoError(t, err)
	require.NotEmpty(t, drives, "expected at least one logical volume")
	assert.Contains(t, drives, "C:", "expected C: drive to be present")
}

func TestCOMWorkerClosedReturnsError(t *testing.T) {
	w, err := NewCOMWorker()
	require.NoError(t, err)
	w.Close()

	// After Close, operations return ErrWorkerClosed without panicking.
	_, err = w.GetEncryptionStatus()
	assert.ErrorIs(t, err, ErrWorkerClosed)
}

// TestGetEncryptionStatus exercises the full COM -> WMI -> Win32_EncryptableVolume
// pipeline. The CI runner (windows-latest) has encryptable volumes (C:, D:).
func TestGetEncryptionStatus(t *testing.T) {
	statuses, err := testWorker.GetEncryptionStatus()
	require.NoError(t, err)
	require.NotEmpty(t, statuses, "expected at least one encryptable volume")

	for _, vs := range statuses {
		assert.NotEmpty(t, vs.DriveVolume)
		require.NotNilf(t, vs.Status, "status should not be nil for %s", vs.DriveVolume)
		assert.NotEmpty(t, vs.Status.EncryptionPercentage)
	}
}

// TestBitlockerConnectAndStatus isolates the bitlockerConnect + getBitlockerStatus
// path by calling them directly on the shared COM thread.
func TestBitlockerConnectAndStatus(t *testing.T) {
	r := testWorker.exec(func() (any, error) {
		vol, err := bitlockerConnect("C:")
		if err != nil {
			return nil, err
		}
		defer vol.bitlockerClose()
		return vol.getBitlockerStatus()
	})

	require.NoError(t, r.err)
	status, ok := r.val.(*EncryptionStatus)
	require.True(t, ok)
	require.NotNil(t, status)
	assert.NotEmpty(t, status.EncryptionPercentage)
}

// TestEncryptionFlagFromRegistry reads the OSEncryptionType GPO registry value.
// CI runners have no GPO configured, so the default EncryptDataOnly is returned.
func TestEncryptionFlagFromRegistry(t *testing.T) {
	flag := encryptionFlagFromRegistry()
	assert.Equal(t, EncryptDataOnly, flag)
}
