//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
)

func main() {
	detailQueriesMap := osquery_utils.GetDetailQueries(context.Background(),
		config.FleetConfig{
			Vulnerabilities: config.VulnerabilitiesConfig{
				DisableWinOSVulnerabilities: false,
			},
			App: config.AppConfig{
				EnableScheduledQueryStats: true,
			},
		},
		&fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}},
		&fleet.Features{
			EnableSoftwareInventory: true,
			EnableHostUsers:         true,
		},
	)
	var b strings.Builder

	b.WriteString(`<!-- DO NOT EDIT. This document is automatically generated. -->
# Understanding host vitals

Following is a summary of the detail queries hardcoded in Fleet used to populate the device details:

`)

	type queryInfo struct {
		name        string
		detailQuery osquery_utils.DetailQuery
	}
	detailQueries := make([]queryInfo, 0, len(detailQueriesMap))
	for name, detailQuery := range detailQueriesMap {
		detailQueries = append(detailQueries, queryInfo{
			name:        name,
			detailQuery: detailQuery,
		})
	}
	sort.Slice(detailQueries, func(i, j int) bool {
		return detailQueries[i].name < detailQueries[j].name
	})

	for _, q := range detailQueries {
		fmt.Fprintf(&b, "## %s\n\n", q.name)

		if q.detailQuery.Description != "" {
			fmt.Fprintf(&b, "- Description: %s\n\n", q.detailQuery.Description)
		}

		platforms := strings.Join(q.detailQuery.Platforms, ", ")
		if len(q.detailQuery.Platforms) == 0 {
			platforms = "all"
		}
		fmt.Fprintf(&b, "- Platforms: %s\n\n", platforms)
		if q.detailQuery.Discovery != "" {
			fmt.Fprintf(&b, "- Discovery query:\n```sql\n%s\n```\n\n", strings.TrimSpace(q.detailQuery.Discovery))
		}
		fmt.Fprintf(&b, "- Query:\n```sql\n%s\n```\n\n", strings.TrimSpace(q.detailQuery.Query))
	}

	// Footnotes
	fmt.Fprint(&b, `<br /><br />`)
	fmt.Fprintf(&b, "[^1]: Software override queries write over the default queries. They are used to populate the software inventory.")

	b.WriteString(`
<meta name="navSection" value="Dig deeper">
<meta name="pageOrderInSection" value="1600">`)

	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0600); err != nil {
		panic(err)
	}
}
