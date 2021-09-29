package logging

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemLogger(t *testing.T) {
	ctx := context.Background()
	tempPath := t.TempDir()
	require.NoError(t, os.Chmod(tempPath, 0755))
	fileName := path.Join(tempPath, "filesystemLogWriter")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), false, false)
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
		rand.Read(randInput)
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

func BenchmarkFilesystemLogger(b *testing.B) {
	ctx := context.Background()
	tempPath, err := ioutil.TempDir("", "test")
	if err != nil {
		b.Fatal("temp dir failed", err)
	}
	fileName := path.Join(tempPath, "filesystemLogWriter")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), false, false)
	if err != nil {
		b.Fatal("new failed ", err)
	}
	defer os.Remove(fileName)

	var logs []json.RawMessage
	for i := 0; i < 50; i++ {
		randInput := make([]byte, 512)
		rand.Read(randInput)
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
	tempPath, err := ioutil.TempDir("", "test")
	if err != nil {
		b.Fatal("temp dir failed", err)
	}
	fileName := path.Join(tempPath, "lumberjack")
	lgr, err := NewFilesystemLogWriter(fileName, log.NewNopLogger(), true, compression)
	if err != nil {
		b.Fatal("new failed ", err)
	}
	defer os.Remove(fileName)

	var logs []json.RawMessage
	for i := 0; i < 50; i++ {
		randInput := make([]byte, 512)
		rand.Read(randInput)
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
