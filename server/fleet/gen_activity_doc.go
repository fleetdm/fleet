//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func main() {
	var b strings.Builder

	b.WriteString(`<!-- DO NOT EDIT. This document is automatically generated. -->
# Audit logs

Fleet logs the following information for administrative actions (in JSON):

- ` + "`" + `created_at` + "`" + `: Timestamp of the event.
- ` + "`" + `id` + "`" + `: Unique ID of the generated event in Fleet.
- ` + "`" + `actor_full_name` + "`" + `: Author user name (missing if the user was deleted).
- ` + "`" + `actor_id` + "`" + `: Unique ID of the author in Fleet (missing if the user was deleted).
- ` + "`" + `actor_gravatar` + "`" + `: Gravatar URL of the author (missing if the user was deleted).
- ` + "`" + `actor_email` + "`" + `: E-mail of the author (missing if the user was deleted).
- ` + "`" + `type` + "`" + `: Type of the activity (see all types below).
- ` + "`" + `details` + "`" + `: Specific details depending on the type of activity (see details for each activity type below).

Example:
` + "```" + `json
{
	"created_at": "2022-12-20T14:54:17Z",
	"id": 6,
	"actor_full_name": "Gandalf",
	"actor_id": 2,
	"actor_gravatar": "foo@example.com",
	"actor_email": "foo@example.com",
	"type": "edited_saved_query",
	"details":{
		"query_id": 42,
		"query_name": "Some query name"
	}
}
` + "```" + `
	
## List of activities and their specific details

`)

	activityMap := map[string]struct{}{}
	for _, activity := range fleet.ActivityDetailsList {
		if _, ok := activityMap[activity.ActivityName()]; ok {
			panic(fmt.Sprintf("type %s already used", activity.ActivityName()))
		}
		activityMap[activity.ActivityName()] = struct{}{}

		fmt.Fprintf(&b, "### Type `%s`\n\n", activity.ActivityName())
		activityTypeDoc, detailsDoc, detailsExampleDoc := activity.Documentation()
		fmt.Fprintf(&b, activityTypeDoc+"\n\n"+detailsDoc+"\n\n")
		if detailsExampleDoc != "" {
			fmt.Fprintf(&b, "#### Example\n\n```json\n%s\n```\n\n", detailsExampleDoc)
		}
	}
	b.WriteString(`
<meta name="pageOrderInSection" value="1400">
<meta name="description" value="Learn how Fleet logs administrative actions in JSON format.">
<meta name="navSection" value="Dig deeper">
`)

	if err := os.WriteFile(os.Args[1], []byte(b.String()), 0600); err != nil {
		panic(err)
	}
}
