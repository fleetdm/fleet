//go:build linux

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
)

// applyAgentDisplayNames sets the process name that ps -e, top, systemctl status and System Monitor show, and the thread
// names that top -H, ps -L and audit logs show. The service starts bin/orbit/orbit, which names the process "orbit"; the
// executable path, and ps's full command line, stay the same.
func applyAgentDisplayNames() error {
	name := []byte(constant.AgentDisplayName)
	// /proc/self is the thread group leader, whose name is the process name.
	if err := os.WriteFile("/proc/self/comm", name, 0); err != nil {
		return fmt.Errorf("set process name: %w", err)
	}
	// A new thread copies the name of the thread that creates it, so list again until no new thread shows up.
	renamed := map[string]struct{}{}
	for range 3 {
		tasks, err := os.ReadDir("/proc/self/task")
		if err != nil {
			return fmt.Errorf("list threads: %w", err)
		}
		done := true
		for _, task := range tasks {
			if _, ok := renamed[task.Name()]; ok {
				continue
			}
			renamed[task.Name()] = struct{}{}
			done = false
			// Best effort: the thread may have exited meanwhile.
			_ = os.WriteFile(filepath.Join("/proc/self/task", task.Name(), "comm"), name, 0)
		}
		if done {
			break
		}
	}
	return nil
}
