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

// RotateRecoveryKey is a no-op on non-Windows platforms.
func (w *COMWorker) RotateRecoveryKey(string) (string, error) { return "", nil }

func (w *COMWorker) HasTPMFamilyProtector(string) (bool, error) { return false, nil }

func (w *COMWorker) AddTPMProtector(string) error { return nil }

func (w *COMWorker) EnableProtection(string) error { return nil }
