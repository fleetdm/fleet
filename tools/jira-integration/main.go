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

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
)

func main() {
	var (
		jiraURL             = flag.String("jira-url", "", "The Jira instance URL")
		jiraUsername        = flag.String("jira-username", "", "The Jira username")
		jiraProjectKey      = flag.String("jira-project-key", "", "The Jira project key")
		fleetURL            = flag.String("fleet-url", "https://localhost:8080", "The Fleet server URL")
		cve                 = flag.String("cve", "", "The CVE to create a Jira issue for")
		epssProbability     = flag.Float64("epss-probability", 0, "The EPSS Probability score of the CVE")
		cvssScore           = flag.Float64("cvss-score", 0, "The CVSS score of the CVE")
		cisaKnownExploit    = flag.Bool("cisa-known-exploit", false, "Whether CISA reported it as a known exploit")
		hostsCount          = flag.Int("hosts-count", 1, "The number of hosts to match the CVE or failing policy")
		failingPolicyID     = flag.Int("failing-policy-id", 0, "The failing policy ID")
		failingPolicyTeamID = flag.Int("failing-policy-team-id", 0, "The Team ID of the failing policy")
		premiumLicense      = flag.Bool("premium", false, "Whether to simulate a premium user or not")
	)

	flag.Parse()

	// keep set of flags that were provided, to handle those that can be absent
	setFlags := make(map[string]bool)
	flag.CommandLine.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

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
	if *cve == "" && *failingPolicyID == 0 {
		fmt.Fprintf(os.Stderr, "one of -cve or -failing-policy-id is required")
		os.Exit(1)
	}
	if *cve != "" && *failingPolicyID != 0 {
		fmt.Fprintf(os.Stderr, "only one of -cve or -failing-policy-id is allowed")
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

	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]fleet.HostVulnerabilitySummary, error) {
		hosts := make([]fleet.HostVulnerabilitySummary, *hostsCount)
		for i := 0; i < *hostsCount; i++ {
			hosts[i] = fleet.HostVulnerabilitySummary{ID: uint(i + 1), //nolint:gosec // dismiss G115
				Hostname:    fmt.Sprintf("host-test-%d", i+1),
				DisplayName: fmt.Sprintf("host-test-%d", i+1)}
		}
		return hosts, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Integrations: fleet.Integrations{
				Jira: []*fleet.JiraIntegration{
					{
						EnableSoftwareVulnerabilities: *cve != "",
						URL:                           *jiraURL,
						Username:                      *jiraUsername,
						APIToken:                      jiraPassword,
						ProjectKey:                    *jiraProjectKey,
						EnableFailingPolicies:         *failingPolicyID > 0,
					},
				},
			},
		}, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{
			ID:   tid,
			Name: fmt.Sprintf("team-test-%d", tid),
			Config: fleet.TeamConfig{
				Integrations: fleet.TeamIntegrations{
					Jira: []*fleet.TeamJiraIntegration{
						{
							URL:                   *jiraURL,
							ProjectKey:            *jiraProjectKey,
							EnableFailingPolicies: *failingPolicyID > 0,
						},
					},
				},
			},
		}, nil
	}

	lic := &fleet.LicenseInfo{Tier: fleet.TierFree}
	if *premiumLicense {
		lic.Tier = fleet.TierPremium
	}
	ctx := license.NewContext(context.Background(), lic)

	jira := &worker.Jira{
		FleetURL:  *fleetURL,
		Datastore: ds,
		Log:       logger,
		NewClientFunc: func(opts *externalsvc.JiraOptions) (worker.JiraClient, error) {
			return externalsvc.NewJiraClient(opts)
		},
	}

	var argsJSON json.RawMessage
	if *cve != "" {
		vulnArgs := struct {
			CVE              string   `json:"cve,omitempty"`
			EPSSProbability  *float64 `json:"epss_probability,omitempty"`
			CVSSScore        *float64 `json:"cvss_score,omitempty"`
			CISAKnownExploit *bool    `json:"cisa_known_exploit,omitempty"`
		}{
			CVE: *cve,
		}
		if setFlags["epss-probability"] {
			vulnArgs.EPSSProbability = epssProbability
		}
		if setFlags["cvss-score"] {
			vulnArgs.CVSSScore = cvssScore
		}
		if setFlags["cisa-known-exploit"] {
			vulnArgs.CISAKnownExploit = cisaKnownExploit
		}

		b, err := json.Marshal(vulnArgs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal vulnerability args: %v", err)
			os.Exit(1)
		}
		argsJSON = json.RawMessage(fmt.Sprintf(`{"vulnerability":%s}`, string(b)))

	} else if *failingPolicyID > 0 {
		jsonStr := fmt.Sprintf(`{"failing_policy":{"policy_id": %d, "policy_name": "test-policy-%[1]d", `, *failingPolicyID)
		if *failingPolicyTeamID > 0 {
			jsonStr += fmt.Sprintf(`"team_id":%d, `, *failingPolicyTeamID)
		}
		jsonStr += `"hosts": `
		hosts := make([]fleet.PolicySetHost, 0, *hostsCount)
		for i := 1; i <= *hostsCount; i++ {
			hosts = append(hosts, fleet.PolicySetHost{ID: uint(i), Hostname: fmt.Sprintf("host-test-%d", i)}) //nolint:gosec // dismiss G115
		}
		b, _ := json.Marshal(hosts)
		jsonStr += string(b) + "}}"
		argsJSON = json.RawMessage(jsonStr)
	}

	if err := jira.Run(ctx, argsJSON); err != nil {
		log.Fatal(err)
	}
}
