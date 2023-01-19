//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
)

func main() {
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{
		Vulnerabilities: config.VulnerabilitiesConfig{
			DisableWinOSVulnerabilities: false,
		},
		App: config.AppConfig{
			EnableScheduledQueryStats: true,
		},
	}, &fleet.Features{
		EnableSoftwareInventory: true,
		EnableHostUsers:         true,
	})
	var b strings.Builder

	b.WriteString(`<!-- DO NOT EDIT. This document is automatically generated. -->
# Detail Queries Summary

Following is a summary of the detail queries hardcoded in Fleet used to populate the device details:

`)
	for queryName, sqlQuery := range detailQueries {
		fmt.Fprintf(&b, "## %s\n\n", queryName)
		platforms := strings.Join(sqlQuery.Platforms, ", ")
		if len(sqlQuery.Platforms) == 0 {
			platforms = "all"
		}
		fmt.Fprintf(&b, "- Platforms: %s\n\n", platforms)
		if sqlQuery.Discovery != "" {
			fmt.Fprintf(&b, "- Discovery query:\n```sql\n%s\n```\n\n", sqlQuery.Discovery)
		}
		fmt.Fprintf(&b, "- Query:\n```sql\n%s\n```\n\n", sqlQuery.Query)
	}

	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0600); err != nil {
		panic(err)
	}
}
