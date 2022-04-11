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

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/kit/log"
)

func main() {
	var (
		jiraURL        = flag.String("jira-url", "", "The Jira instance URL")
		jiraUsername   = flag.String("jira-username", "", "The Jira username")
		jiraProjectKey = flag.String("jira-project-key", "", "The Jira project key")
		fleetURL       = flag.String("fleet-url", "https://localhost:8080", "The Fleet server URL")
		cve            = flag.String("cve", "", "The CVE to create a Jira issue for")
		hostsCount     = flag.Int("hosts-count", 1, "The number of hosts to match the CVE")
	)

	flag.Parse()

	if *jiraURL == "" {
		fmt.Fprintf(os.Stderr, "-jira-url is required")
		os.Exit(1)
	}
	if *jiraUsername == "" {
		fmt.Fprintf(os.Stderr, "-jira-username is required")
		os.Exit(1)
	}
	if *jiraProjectKey == "" {
		fmt.Fprintf(os.Stderr, "-jira-project-key is required")
		os.Exit(1)
	}
	if *cve == "" {
		fmt.Fprintf(os.Stderr, "-cve is required")
		os.Exit(1)
	}
	if *hostsCount <= 0 {
		fmt.Fprintf(os.Stderr, "-hosts-count must be at least 1")
		os.Exit(1)
	}

	jiraPassword := os.Getenv("JIRA_PASSWORD")
	if jiraPassword == "" {
		fmt.Fprintf(os.Stderr, "JIRA_PASSWORD is required")
		os.Exit(1)
	}

	logger := kitlog.NewLogfmtLogger(os.Stdout)

	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           *jiraURL,
		BasicAuthUsername: *jiraUsername,
		BasicAuthPassword: jiraPassword,
		ProjectKey:        *jiraProjectKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]*fleet.HostShort, error) {
		hosts := make([]*fleet.HostShort, *hostsCount)
		for i := 0; i < *hostsCount; i++ {
			hosts[i] = &fleet.HostShort{ID: uint(i + 1), Hostname: fmt.Sprintf("host-test-%d", i+1)}
		}
		return hosts, nil
	}

	jira := &worker.Jira{
		FleetURL:   *fleetURL,
		Datastore:  ds,
		Log:        logger,
		JiraClient: client,
	}

	argsJSON := json.RawMessage(fmt.Sprintf(`{"cve":%q}`, *cve))

	err = jira.Run(context.Background(), argsJSON)
	if err != nil {
		log.Fatal(err)
	}
}
