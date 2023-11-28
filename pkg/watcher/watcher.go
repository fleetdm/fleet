package watcher

import (
	"fmt"
	"io"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Start starts a goroutine that samples a process CPU and memory usage every interval.
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
	done := make(chan struct{})
	process, err := process.NewProcess(pid)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(interval):
				cpuPercent, err := process.CPUPercent()
				if err != nil {
					panic(err)
				}
				memInfo, err := process.MemoryInfo()
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
