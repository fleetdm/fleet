//go:build windows

package bitlocker

import (
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
)

type comWorkItem struct {
	fn     func() (any, error)
	result chan comWorkResult
}

type comWorkResult struct {
	val any
	err error
}

// COMWorker runs all BitLocker COM/WMI operations on a single dedicated OS
// thread. This prevents deadlocks with other COM callers (MDM Bridge, Windows
// Update) that share the global comshim singleton.
type COMWorker struct {
	workCh    chan comWorkItem
	done      chan struct{}
	closeOnce sync.Once
}

// NewCOMWorker creates a new COMWorker that initializes COM on a dedicated OS
// thread and processes all BitLocker operations serially on that thread.
func NewCOMWorker() (*COMWorker, error) {
	w := &COMWorker{
		workCh: make(chan comWorkItem),
		done:   make(chan struct{}),
	}
	initErr := make(chan error, 1)
	go w.loop(initErr)
	if err := <-initErr; err != nil {
		return nil, err
	}
	return w, nil
}

func (w *COMWorker) loop(initErr chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		initErr <- err
		close(w.done)
		return
	}
	defer ole.CoUninitialize()
	initErr <- nil

	for item := range w.workCh {
		val, err := item.fn()
		item.result <- comWorkResult{val, err}
	}
	close(w.done)
}

// Close shuts down the COM worker goroutine and waits for it to finish.
func (w *COMWorker) Close() {
	w.closeOnce.Do(func() {
		close(w.workCh)
	})
	<-w.done
}

func (w *COMWorker) exec(fn func() (any, error)) comWorkResult {
	ch := make(chan comWorkResult, 1)
	w.workCh <- comWorkItem{fn: fn, result: ch}
	return <-ch
}

// GetEncryptionStatus returns the BitLocker encryption status for all logical volumes.
func (w *COMWorker) GetEncryptionStatus() ([]VolumeStatus, error) {
	r := w.exec(func() (any, error) { return getEncryptionStatusOnCOMThread() })
	status, _ := r.val.([]VolumeStatus)
	return status, r.err
}

// EncryptVolume encrypts the specified volume and returns the recovery key.
func (w *COMWorker) EncryptVolume(targetVolume string) (string, error) {
	r := w.exec(func() (any, error) { return encryptVolumeOnCOMThread(targetVolume) })
	key, _ := r.val.(string)
	return key, r.err
}

// DecryptVolume decrypts the specified volume.
func (w *COMWorker) DecryptVolume(targetVolume string) error {
	r := w.exec(func() (any, error) { return nil, decryptVolumeOnCOMThread(targetVolume) })
	return r.err
}
