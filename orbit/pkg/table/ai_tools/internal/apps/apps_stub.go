//go:build !darwin && !linux && !windows

package apps

import (
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

// scanApps has no implementation on unsupported platforms; orbit builds a
// fallback binary for OSs other than macOS, Linux, and Windows, and this stub
// keeps the apps collector compiling there.
func scanApps(_ []homes.Home) []App {
	return nil
}
