package seed

import (
	"github.com/fleetdm/fleet/v4/tools/dibble/themes"
)

// Software currently registers Fleet-maintained app references. The full
// custom-package upload path (multipart with a real installer) is intentionally
// out of scope for v1: producing a valid signed .pkg / .msi is itself a chunk
// of work, and the legacy `tools/loadtest/unified_queue` tool was effectively
// already doing only the enqueue side. We can extend this later.
//
// For v1 we record the intent in the result log so it's discoverable.
func Software(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "software"}
	// Print themed names so they show up in the dry-run output even when
	// the upload path is a TODO.
	for i := 0; i < count; i++ {
		n := themes.Pick(theme, "software", i)
		log.Printf("software (planned) %q — custom-package upload not yet implemented", n.Name)
		res.Skipped++
	}
	_ = teams
	_ = c
	return res
}
