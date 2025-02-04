package open

import (
	"fmt"
	"os"
	"os/exec"
)

func browser(url string) error {
	// xdg-open is available on most Linux-y systems
	cmd := exec.Command("xdg-open", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Must be asynchronous (Start, not Run) because xdg-open will continue running
	// and block this goroutine if it was the process that opened the browser.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xdg-open failed to start: %w", err)
	}
	go func() {
		// We must call wait to avoid defunct processes.
		cmd.Wait() //nolint:errcheck
	}()
	return nil
}
