//go:build linux

package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	// A new thread copies the name of the thread that creates it, so repeat until a pass finds none left to rename.
	for range 3 {
		tasks, err := os.ReadDir("/proc/self/task")
		if err != nil {
			return fmt.Errorf("list threads: %w", err)
		}
		left := 0
		for _, task := range tasks {
			comm := filepath.Join("/proc/self/task", task.Name(), "comm")
			current, err := os.ReadFile(comm)
			if err != nil || strings.TrimSpace(string(current)) == constant.AgentDisplayName {
				continue // exited, or already renamed
			}
			left++
			// Best effort: the thread may have exited meanwhile; the next pass checks again.
			_ = os.WriteFile(comm, name, 0)
		}
		if left == 0 {
			return nil
		}
	}
	return errors.New("set thread names: some threads still have the old name")
}
