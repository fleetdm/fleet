package watcher

import (
	"fmt"
	"io"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/shirou/gopsutil/v3/process"
)

// Start starts a goroutine that samples a process CPU and memory usage every interval
// of a process with the given pid.
//
// It writes a sample line on every interval, and the format of the sample is:
//
//	$TIME $PROCESS_CPU_PERCENT_UTILIZATION $PROCESS_RSS_IN_MB
//
// E.g.:
//
//	15:04:05 67.12 4355.12
//
// Returns a channel you can close to stop the goroutine.
func Start(pid int32, w io.Writer, interval time.Duration) chan struct{} {
	process_, err := process.NewProcess(pid)
	if err != nil {
		panic(err)
	}
	done := watchProcess(w, interval, func() *process.Process {
		return process_
	})
	return done
}

// StartWithName starts a goroutine that samples a process CPU and memory usage every interval,
// of a process with the given name.
// This method is useful if you want to track a process even if there are process restarts.
//
// It writes a sample line on every interval, and the format of the sample is:
//
//	$TIME $PROCESS_CPU_PERCENT_UTILIZATION $PROCESS_RSS_IN_MB
//
// E.g.:
//
//	15:04:05 67.12 4355.12
//
// Returns a channel you can close to stop the goroutine.
func StartWithName(name string, w io.Writer, interval time.Duration) chan struct{} {
	done := watchProcess(w, interval, func() *process.Process {
		process_, err := platform.GetProcessByName(name)
		if err != nil {
			panic(err)
		}
		return process_
	})
	return done
}

func watchProcess(w io.Writer, interval time.Duration, getProcess func() *process.Process) chan struct{} {
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(interval):
				process_ := getProcess()
				if process_ == nil {
					continue
				}
				cpuPercent, err := process_.CPUPercent()
				if err != nil {
					panic(err)
				}
				memInfo, err := process_.MemoryInfo()
				if err != nil {
					panic(err)
				}
				now := time.Now().UTC().Format("15:04:05")
				fmt.Fprintf(w, "%s %.2f %.2f\n", now, cpuPercent, float64(memInfo.RSS)/1024.0/1024.0)
			}
		}
	}()
	return done
}
