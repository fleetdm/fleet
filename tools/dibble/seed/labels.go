package seed

import (
	"github.com/fleetdm/fleet/v4/tools/dibble/themes"
)

// Labels seeds dynamic (query-based) labels. Manual labels require live host
// identifiers, which dibble can't synthesize meaningfully — those should be
// created against a Fleet populated by osquery-perf.
func Labels(c Client, log Logger, theme themes.Theme, count int) Result {
	res := Result{Entity: "labels"}

	labelQueries := []string{
		"SELECT 1 WHERE 1=0;",
		"SELECT 1 FROM osquery_info;",
		"SELECT 1 FROM os_version WHERE platform = 'darwin';",
		"SELECT 1 FROM os_version WHERE platform = 'windows';",
		"SELECT 1 FROM os_version WHERE platform IN ('ubuntu','rhel','centos','debian');",
	}

	for i := 0; i < count; i++ {
		n := themes.Pick(theme, "label", i)
		body := map[string]any{
			"name":        n.Name,
			"query":       labelQueries[i%len(labelQueries)],
			"platform":    "",
			"description": "Seeded by dibble — " + n.Name,
		}
		err := c.Post("/api/latest/fleet/labels", body, nil)
		switch {
		case err == nil:
			res.Created++
			log.Printf("label %q", n.Name)
		case IsAlreadyExists(err):
			res.Skipped++
		default:
			res.Errors = append(res.Errors, err)
		}
	}
	return res
}
