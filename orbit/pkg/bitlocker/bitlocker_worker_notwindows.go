//go:build !windows

package bitlocker

// COMWorker is a no-op on non-Windows platforms.
type COMWorker struct{}

// NewCOMWorker returns a no-op COMWorker on non-Windows platforms.
func NewCOMWorker() (*COMWorker, error) { return &COMWorker{}, nil }

// Close is a no-op on non-Windows platforms.
func (w *COMWorker) Close() {}

// GetEncryptionStatus is a no-op on non-Windows platforms.
func (w *COMWorker) GetEncryptionStatus() ([]VolumeStatus, error) { return nil, nil }

// EncryptVolume is a no-op on non-Windows platforms.
func (w *COMWorker) EncryptVolume(string) (string, error) { return "", nil }

// DecryptVolume is a no-op on non-Windows platforms.
func (w *COMWorker) DecryptVolume(string) error { return nil }
