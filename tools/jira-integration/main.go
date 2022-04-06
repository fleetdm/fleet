// Command jira-integration tests creating a ticket to a Jira instance via
// the Fleet worker processor. It creates it exactly as if a Jira integration
// was configured and a new CVE and related CPEs was found.
//
// Note that the Jira user's password must be provided via an environment
// variable, JIRA_PASSWORD.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/kit/log"
)

func main() {
	var (
		jiraURL        = flag.String("jira-url", "https://fleetdm.atlassian.net/", "The Jira instance URL")
		jiraUsername   = flag.String("jira-username", "", "The Jira username")
		jiraProjectKey = flag.String("jira-project-key", "", "The Jira project key")
		fleetURL       = flag.String("fleet-url", "https://localhost:1307/", "The Fleet server URL")
		cve            = flag.String("cve", "CVE-2020-8284", "The CVE to create a ticket for")
		cpes           = flag.String("cpes", "", "Comma-separated list of CPEs associated with the CVE")
	)

	flag.Parse()

	logger := kitlog.NewLogfmtLogger(os.Stdout)
	pwd := os.Getenv("JIRA_PASSWORD")

	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           *jiraURL,
		BasicAuthUsername: *jiraUsername,
		BasicAuthPassword: pwd,
		ProjectKey:        *jiraProjectKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	jira := &worker.Jira{
		FleetURL:   *fleetURL,
		Log:        logger,
		JiraClient: client,
	}

	cpeVals := strings.Split(*cpes, ",")
	for i, val := range cpeVals {
		cpeVals[i] = strconv.Quote(val)
	}
	err = jira.Run(context.Background(), json.RawMessage(fmt.Sprintf(`{"cve":%q,"cpes":[%s]}`, *cve, strings.Join(cpeVals, ","))))
	if err != nil {
		log.Fatal(err)
	}
}
