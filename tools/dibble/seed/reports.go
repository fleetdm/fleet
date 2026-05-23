package seed

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/tools/dibble/themes"
)

// Reports (formerly "queries") are saved SQL queries with an interval. We seed
// a mix of global and team-scoped reports, alternating intervals (0 = never,
// 60s, 3600s) so the schedule grid shows variety in the UI.
func Reports(c Client, log Logger, theme themes.Theme, teams []Team, count int) Result {
	res := Result{Entity: "reports"}

	exampleSQL := []string{
		"SELECT version FROM osquery_info;",
		"SELECT name FROM apps WHERE name LIKE '%Slack%';",
		"SELECT pid, name FROM processes ORDER BY pid LIMIT 5;",
		"SELECT * FROM users WHERE shell != '/usr/bin/false' LIMIT 10;",
		"SELECT path, type FROM mounts WHERE path LIKE '/Volumes/%';",
	}
	intervals := []int{0, 60, 3600}

	for i := 0; i < count; i++ {
		n := themes.Pick(theme, "policy", i) // reuse policy names — they read fine as queries too
		body := map[string]any{
			"name":             fmt.Sprintf("%s report", n.Name),
			"description":      n.Desc,
			"query":            exampleSQL[i%len(exampleSQL)],
			"interval":         intervals[i%len(intervals)],
			"observer_can_run": i%2 == 0,
			"platform":         seededPlatforms[i%len(seededPlatforms)],
			"logging":          "snapshot",
		}
		err := c.Post("/api/latest/fleet/reports", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("report (global) %q", body["name"])
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}

	// One report per team.
	for _, t := range teams {
		n := themes.Pick(theme, "policy", count)
		teamID := t.ID
		body := map[string]any{
			"name":        fmt.Sprintf("%s — %s report", n.Name, t.Name),
			"description": n.Desc,
			"query":       exampleSQL[0],
			"interval":    0,
			"team_id":     teamID,
		}
		err := c.Post("/api/latest/fleet/reports", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("report (team=%s) %q", t.Name, body["name"])
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}
	return res
}
