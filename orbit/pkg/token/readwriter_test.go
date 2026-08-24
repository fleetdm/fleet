package token

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestLoadOrGenerate(t *testing.T) {
	t.Run("creates file if it doesn't exist", func(t *testing.T) {
		dir := os.TempDir()
		file := filepath.Join(dir, "identifier")
		defer os.Remove(file)

		rw := NewReadWriter(file, nil)
		require.NoError(t, rw.LoadOrGenerate())
		token, err := rw.Read()
		require.NoError(t, err)
		require.NotEmpty(t, token)

		stat, err := os.Stat(file)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
	})

	t.Run("returns the file value if it exists", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		defer os.Remove(file.Name())
		stat, err := file.Stat()
		require.NoError(t, err)
		oldMtime := stat.ModTime()

		rw := NewReadWriter(file.Name(), nil)
		err = rw.LoadOrGenerate()
		require.NoError(t, err)
		token, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		stat, err = os.Stat(file.Name())
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
		require.Equal(t, oldMtime, stat.ModTime())
	})

	t.Run("sets the file mode to DefaultWorldReadableFileMode if exists", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		defer os.Remove(file.Name())

		err = file.Chmod(constant.DefaultFileMode)
		require.NoError(t, err)
		stat, err := file.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultFileMode), stat.Mode())

		rw := NewReadWriter(file.Name(), nil)
		err = rw.LoadOrGenerate()
		require.NoError(t, err)
		token, err := rw.Read()
		require.NoError(t, err)
		require.Equal(t, "test", token)

		stat, err = file.Stat()
		require.NoError(t, err)
		require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
	})

	t.Run("errors for other reasons", func(t *testing.T) {
		file, err := os.CreateTemp("", "identifier")
		require.NoError(t, err)
		_, err = file.WriteString("test")
		require.NoError(t, err)
		require.NoError(t, file.Chmod(0x600))
		defer os.Remove(file.Name())

		rw := NewReadWriter(file.Name(), nil)
		token, err := rw.Read()
		require.Error(t, err)
		require.Empty(t, token)
	})
}

func TestRotate(t *testing.T) {
	file, err := os.CreateTemp("", t.Name())
	require.NoError(t, err)
	defer os.Remove(file.Name())
	rw := NewReadWriter(file.Name(), nil)

	token, err := rw.Read()
	require.NoError(t, err)
	require.Empty(t, token)

	err = rw.Rotate()
	require.NoError(t, err)
	token, err = rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, token)
	stat, err := file.Stat()
	require.NoError(t, err)
	require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())

	err = rw.Rotate()
	require.NoError(t, err)
	newToken, err := rw.Read()
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	require.NotEqual(t, token, newToken)
	stat, err = file.Stat()
	require.NoError(t, err)
	require.Equal(t, os.FileMode(constant.DefaultWorldReadableFileMode), stat.Mode())
}

func TestRotator(t *testing.T) {
	var numRemoteChecks int32 // Use int32 for atomic
	var numUpdates int32      // Use int32 for atomic

	file, err := os.CreateTemp("", "identifier")
	require.NoError(t, err)
	_, err = file.WriteString("test")
	require.NoError(t, err)

	rw := NewReadWriter(file.Name(), func(token string) error {
		atomic.AddInt32(&numRemoteChecks, 1) // Atomic write
		return nil
	})
	rw.localCheckDuration = 100 * time.Millisecond
	rw.remoteCheckDuration = 200 * time.Millisecond

	err = rw.LoadOrGenerate()
	require.NoError(t, err)

	rw.SetRemoteUpdateFunc(func(token string) error {
		atomic.AddInt32(&numUpdates, 1) // Atomic write
		return nil
	})

	// Set the token's mtime to more than an hour ago so that it
	// will be considered expired and trigger a rotation.
	rw.mu.Lock()
	rw.mtime = time.Now().Add(-2 * time.Hour)
	rw.mu.Unlock()

	stop1 := rw.StartRotation()
	stop2 := rw.StartRotation()

	// Wait for the first rotation to complete.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&numUpdates) >= 1
	}, 5*time.Second, 10*time.Millisecond)

	// Wait for the rotation's Write() to fully complete — mtime will
	// be set to "now" by readFile() at the end of Write(). Without
	// this, setting mtime to -2h below can be clobbered by the
	// still-in-flight Write().
	require.Eventually(t, func() bool {
		return time.Since(rw.GetMtime()) < 30*time.Second
	}, 5*time.Second, 10*time.Millisecond)

	// Close the first stop channel, this should not stop the rotation.
	stop1()
	// Do it again to prove that closing multiple times is safe.
	stop1()

	// Set the token's mtime to more than an hour ago again.
	rw.mu.Lock()
	rw.mtime = time.Now().Add(-2 * time.Hour)
	rw.mu.Unlock()

	// Wait for the second rotation.
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&numUpdates) >= 2
	}, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, int32(1), atomic.LoadInt32(&numRemoteChecks)) // Atomic read

	// Reset the mtime one more time.
	rw.mu.Lock()
	rw.mtime = time.Now().Add(-2 * time.Hour)
	rw.mu.Unlock()

	// Now close the second stop channel, this should stop the rotation.
	stop2()

	// Wait enough time to ensure that if the rotation was still running
	// we would have done another remote check.
	time.Sleep(250 * time.Millisecond)
	require.Equal(t, int32(2), atomic.LoadInt32(&numUpdates))      // Atomic read
	require.Equal(t, int32(1), atomic.LoadInt32(&numRemoteChecks)) // Atomic read
}
