//go:build windows

package bitlocker

import (
	"errors"
	"runtime"
	"sync"

	"github.com/go-ole/go-ole"
)

// ErrWorkerClosed is returned when an operation is attempted on a closed COMWorker.
var ErrWorkerClosed = errors.New("COM worker is closed")

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
//
// Shutdown uses two channels to avoid a race between Close and exec:
//   - stop: closed by Close() to tell the loop goroutine to exit.
//   - done: closed by the loop goroutine after it has exited and cleaned up COM.
//
// Using a separate stop channel (instead of closing workCh) is necessary because
// exec() sends on workCh. Closing a channel that another goroutine may send on
// causes a panic. With this design, workCh is never closed -- it is garbage
// collected along with the COMWorker once all references are dropped. When
// exec() races with Close(), it sees <-w.done and returns ErrWorkerClosed.
type COMWorker struct {
	workCh    chan comWorkItem
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

// NewCOMWorker creates a new COMWorker that initializes COM on a dedicated OS
// thread and processes all BitLocker operations serially on that thread.
func NewCOMWorker() (*COMWorker, error) {
	w := &COMWorker{
		workCh: make(chan comWorkItem),
		stop:   make(chan struct{}),
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

	for {
		select {
		case item := <-w.workCh:
			val, err := item.fn()
			item.result <- comWorkResult{val, err}
		case <-w.stop:
			close(w.done)
			return
		}
	}
}

// Close shuts down the COM worker goroutine and waits for it to finish.
func (w *COMWorker) Close() {
	w.closeOnce.Do(func() {
		close(w.stop)
	})
	<-w.done
}

func (w *COMWorker) exec(fn func() (any, error)) comWorkResult {
	ch := make(chan comWorkResult, 1)
	select {
	case w.workCh <- comWorkItem{fn: fn, result: ch}:
		return <-ch
	case <-w.done:
		return comWorkResult{err: ErrWorkerClosed}
	}
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

// RotateRecoveryKey rotates the recovery key on an already-encrypted volume.
// It adds a new Fleet-managed recovery key, removes old recovery key protectors,
// and returns the new key for escrow. The disk is never decrypted.
func (w *COMWorker) RotateRecoveryKey(targetVolume string) (string, error) {
	r := w.exec(func() (any, error) { return rotateRecoveryKeyOnCOMThread(targetVolume) })
	key, _ := r.val.(string)
	return key, r.err
}
