package service

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/fleetdm/fleet/server/logging"
	"github.com/stretchr/testify/require"
)

// TestRotateLoggerSIGHUP verifies that the osqueryd logfile is rotated by
// sending a SIGHUP signal.
func TestRotateLoggerSIGHUP(t *testing.T) {
	ctx := context.Background()
	filePrefix := "kolide-log-rotate-test"
	f, err := ioutil.TempFile("/tmp", filePrefix)
	require.Nil(t, err)
	defer os.Remove(f.Name())

	logFile, err := logging.NewFilesystemLogWriter(f.Name(), log.NewNopLogger(), true, false)
	require.Nil(t, err)

	// write a log line
	logFile.Write(ctx, []json.RawMessage{json.RawMessage("msg1")})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	defer signal.Reset(syscall.SIGHUP)

	// send SIGHUP to the process
	err = syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
	require.Nil(t, err)

	// wait for the SIGHUP signal, otherwise the test exits before the
	// log is rotated.
	<-sig
	time.Sleep(100 * time.Millisecond)

	// write a new log line and verify that the original file includes
	// the new log line but not any of the old ones.
	logFile.Write(ctx, []json.RawMessage{json.RawMessage("msg2")})
	logMsg, err := ioutil.ReadFile(f.Name())
	require.Nil(t, err)

	require.Equal(t, "msg2\n", string(logMsg))
}
