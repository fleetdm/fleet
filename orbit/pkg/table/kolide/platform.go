package osquery

import (
	"fmt"
	"runtime"
)

// OsqueryPlatform is the specific type assigned to osquery platform strings
type OsqueryPlatform string

const (
	Unknown OsqueryPlatform = "unknown"
	Windows OsqueryPlatform = "windows"
	Darwin  OsqueryPlatform = "darwin"
	Linux   OsqueryPlatform = "linux"
)

// DetectPlatform returns the runtime platform, or an error if the runtime
// platform cannot be sufficiently detected.
func DetectPlatform() (OsqueryPlatform, error) {
	switch runtime.GOOS {
	case "windows":
		return Windows, nil
	case "darwin":
		return Darwin, nil
	case "linux":
		return Linux, nil
	default:
		return Unknown, fmt.Errorf("unrecognized runtime.GOOS: %s", runtime.GOOS)
	}
}
