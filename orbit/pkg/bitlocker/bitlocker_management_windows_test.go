//go:build windows

package bitlocker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		{"full", 10000, "1.00%"},
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
	// Calls kernel32.dll GetLogicalDrives -- should return at least C:
	drives, err := getLogicalVolumes()
	require.NoError(t, err)
	require.NotEmpty(t, drives, "expected at least one logical volume")
	assert.Contains(t, drives, "C:", "expected C: drive to be present")
}

// TestCOMWorkerLifecycle verifies that COM can be initialized on a dedicated
// OS thread and cleanly torn down. This exercises the ole.CoInitializeEx /
// CoUninitialize path that every real BitLocker operation depends on.
func TestCOMWorkerLifecycle(t *testing.T) {
	w, err := NewCOMWorker()
	require.NoError(t, err, "NewCOMWorker should initialize COM successfully")
	t.Log("COMWorker initialized successfully, closing")
	w.Close()
	t.Log("COMWorker closed successfully")

	// After Close, operations must return ErrWorkerClosed (not panic).
	_, err = w.GetEncryptionStatus()
	assert.ErrorIs(t, err, ErrWorkerClosed)
}

// TestGetEncryptionStatus queries the BitLocker WMI provider for every logical
// volume. This is a read-only operation that exercises the full COM -> WMI ->
// Win32_EncryptableVolume pipeline. On a machine without BitLocker-capable
// volumes (e.g. a CI runner) the result may be empty, but should never error.
func TestGetEncryptionStatus(t *testing.T) {
	w, err := NewCOMWorker()
	require.NoError(t, err)
	defer w.Close()

	statuses, err := w.GetEncryptionStatus()
	require.NoError(t, err)
	t.Logf("GetEncryptionStatus returned %d volume(s)", len(statuses))

	for _, vs := range statuses {
		assert.NotEmpty(t, vs.DriveVolume, "drive volume should not be empty")
		require.NotNil(t, vs.Status, "status should not be nil for %s", vs.DriveVolume)
		assert.NotEmpty(t, vs.Status.EncryptionPercentage, "encryption percentage should be set for %s", vs.DriveVolume)
		t.Logf("  %s: protection=%d conversion=%d encryption=%s",
			vs.DriveVolume, vs.Status.ProtectionStatus, vs.Status.ConversionStatus, vs.Status.EncryptionPercentage)
	}
}

// TestBitlockerConnectAndStatus connects to C: via WMI and reads its
// encryption status. This isolates the bitlockerConnect + getBitlockerStatus
// path from the COMWorker layer.
func TestBitlockerConnectAndStatus(t *testing.T) {
	// COM must be initialized on the calling thread for WMI calls.
	w, err := NewCOMWorker()
	require.NoError(t, err)
	defer w.Close()

	// Run the connect+status inside the COM worker thread.
	type result struct {
		status *EncryptionStatus
		err    error
	}
	r := w.exec(func() (any, error) {
		vol, err := bitlockerConnect("C:")
		if err != nil {
			return nil, err
		}
		defer vol.bitlockerClose()
		return vol.getBitlockerStatus()
	})

	status, _ := r.val.(*EncryptionStatus)
	// C: may not be an encryptable volume on all machines (e.g. VMs without
	// BitLocker support). If the connect fails we accept that.
	if r.err != nil {
		t.Logf("bitlockerConnect(C:) returned error (acceptable on non-BitLocker VMs): %v", r.err)
		return
	}

	require.NotNil(t, status)
	assert.NotEmpty(t, status.EncryptionPercentage)
	t.Logf("C: protection=%d conversion=%d encryption=%s",
		status.ProtectionStatus, status.ConversionStatus, status.EncryptionPercentage)
}

// TestEncryptionFlagFromRegistry reads the OSEncryptionType GPO registry
// value. On most machines (and CI runners) the key won't exist, so the
// function should return the default EncryptDataOnly.
func TestEncryptionFlagFromRegistry(t *testing.T) {
	flag := encryptionFlagFromRegistry()
	t.Logf("encryptionFlagFromRegistry returned %d (EncryptDataOnly=%d, EncryptDemandWipe=%d)",
		flag, EncryptDataOnly, EncryptDemandWipe)
	// Without a GPO or another MDM setting this key, the default is EncryptDataOnly.
	// We can't guarantee the value on every machine, but we can verify it returns
	// one of the two valid values.
	assert.True(t, flag == EncryptDataOnly || flag == EncryptDemandWipe,
		"expected EncryptDataOnly or EncryptDemandWipe, got %d", flag)
}
