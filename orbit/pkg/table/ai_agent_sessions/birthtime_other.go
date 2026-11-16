//go:build !darwin && !linux && !windows

package ai_agent_sessions

import (
	"os"
	"time"
)

// birthTime is unknown on other platforms.
func birthTime(string, os.FileInfo) time.Time { return time.Time{} }
