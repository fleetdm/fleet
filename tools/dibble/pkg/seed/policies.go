package seed

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
)

// Policies creates `count` global policies plus a handful per team.
// Each policy rotates a platform (darwin/windows/linux/all) so the seeded set
// exercises every cross-platform code path in the UI.
var seededPlatforms = []string{"", "darwin", "windows", "linux", "darwin,windows,linux"}

// dummyPolicyQuery is a query that always returns "compliant" — enough for the
// UI to show the row, but never alarming on a fresh dev Fleet.
const dummyPolicyQuery = "SELECT 1 WHERE 1=1;"

func Policies(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "policies"}

	// Global policies.
	for i := 0; i < count; i++ {
		n := themes.Pick(theme, "policy", i)
		body := map[string]any{
			"name":        n.Name,
			"query":       dummyPolicyQuery,
			"description": n.Desc,
			"resolution":  "Wave the dibble at the host until it complies.",
			"platform":    seededPlatforms[i%len(seededPlatforms)],
		}
		err := c.Post("/api/latest/fleet/policies", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("policy (global) %q", n.Name)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}

	// Per-team policies — 2 per team.
	for _, t := range teams {
		for i := 0; i < 2; i++ {
			n := themes.Pick(theme, "policy", count+i)
			body := map[string]any{
				"name":        fmt.Sprintf("%s — %s", n.Name, t.Name),
				"query":       dummyPolicyQuery,
				"description": n.Desc,
				"platform":    seededPlatforms[i%len(seededPlatforms)],
			}
			err := c.Post(fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", t.ID), body, nil)
			switch {
			case err == nil:
				res.Created++
				log.Printf("policy (team=%s) %q", t.Name, body["name"])
			case IsAlreadyExists(err):
				res.Skipped++
			default:
				res.Errors = append(res.Errors, err)
			}
		}
	}
	return res
}
