//go:build linux

package update

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestApplyAgentDisplayNames(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	// Each goroutine holds its own OS thread, so the runtime has to start more of them.
	lockThreads := func(n int) {
		var started sync.WaitGroup
		for range n {
			started.Add(1)
			go func() {
				runtime.LockOSThread()
				started.Done()
				<-release
			}()
		}
		started.Wait()
	}

	lockThreads(4)
	require.NoError(t, applyAgentDisplayNames())
	lockThreads(4)

	tasks, err := os.ReadDir("/proc/self/task")
	require.NoError(t, err)
	for _, task := range tasks {
		comm, err := os.ReadFile(filepath.Join("/proc/self/task", task.Name(), "comm"))
		// A thread that exits while it's read is gone (ENOENT) or dead (ESRCH).
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH) {
			continue
		}
		require.NoError(t, err)
		require.Equal(t, constant.AgentDisplayName, strings.TrimSpace(string(comm)), "thread %s", task.Name())
	}
}
