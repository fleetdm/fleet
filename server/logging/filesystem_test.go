package logging

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemLogger(t *testing.T) {
	ctx := context.Background()
	tempPath := t.TempDir()
	require.NoError(t, os.Chmod(tempPath, 0o755))
	fileName := filepath.Join(tempPath, "filesystemLogWriter")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), false, false, 500, 28, 3)
	require.Nil(t, err)
	defer os.Remove(fileName)

	var (
		batches  = 50
		logCount = 100
		logSize  = 512
	)

	var logs []json.RawMessage
	for i := 0; i < logCount; i++ {
		randInput := make([]byte, logSize)
		rand.Read(randInput) //nolint:errcheck
		logs = append(logs, randInput)
	}

	for i := 0; i < batches; i++ {
		err := lgr.Write(ctx, logs)
		require.Nil(t, err)
	}

	err = lgr.writer.Close()
	assert.Nil(t, err)

	// can't write to a closed logger
	err = lgr.Write(ctx, logs)
	assert.NotNil(t, err)

	// call close twice noop
	err = lgr.writer.Close()
	assert.Nil(t, err)

	info, err := os.Stat(fileName)
	require.Nil(t, err)
	// + 1 below is for newlines that should be appended to each log
	assert.Equal(t, int64(batches*logCount*(logSize+1)), info.Size())
}

// TestFilesystemLoggerPermission tests that NewFilesystemLogWriter fails
// if the process does not have permissions to write to the provided path.
func TestFilesystemLoggerPermission(t *testing.T) {
	tempPath := t.TempDir()
	require.NoError(t, os.Chmod(tempPath, 0o000))
	fileName := filepath.Join(tempPath, "filesystemLogWriter")
	for _, tc := range []struct {
		name     string
		rotation bool
	}{
		{name: "with-rotation", rotation: true},
		{name: "without-rotation", rotation: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), tc.rotation, false, 500, 28, 3)
			require.Error(t, err)
			require.True(t, errors.Is(err, fs.ErrPermission), err)
		})
	}
}

func BenchmarkFilesystemLogger(b *testing.B) {
	ctx := context.Background()
	fileName := filepath.Join(b.TempDir(), "filesystemLogWriter")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), false, false, 500, 28, 3)
	if err != nil {
		b.Fatal("new failed ", err)
	}

	var logs []json.RawMessage
	for i := 0; i < 50; i++ {
		randInput := make([]byte, 512)
		rand.Read(randInput) //nolint:errcheck
		logs = append(logs, randInput)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := lgr.Write(ctx, logs)
		if err != nil {
			b.Fatal("write failed ", err)
		}
	}

	b.StopTimer()

	lgr.writer.Close()
}

func BenchmarkLumberjack(b *testing.B) {
	benchLumberjack(b, false)
}

func BenchmarkLumberjackWithCompression(b *testing.B) {
	benchLumberjack(b, true)
}

func benchLumberjack(b *testing.B, compression bool) {
	ctx := context.Background()
	fileName := filepath.Join(b.TempDir(), "lumberjack")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), true, compression, 500, 28, 3)
	if err != nil {
		b.Fatal("new failed ", err)
	}

	var logs []json.RawMessage
	for i := 0; i < 50; i++ {
		randInput := make([]byte, 512)
		rand.Read(randInput) //nolint:errcheck
		logs = append(logs, randInput)
	}
	// first lumberjack write opens file so we count that as part of initialization
	// just to make sure we're comparing apples to apples with our logger
	err = lgr.Write(ctx, logs)
	if err != nil {
		b.Fatal("first write failed ", err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := lgr.Write(ctx, logs)
		if err != nil {
			b.Fatal("write failed ", err)
		}
	}

	b.StopTimer()

	lgr.writer.Close()
}
