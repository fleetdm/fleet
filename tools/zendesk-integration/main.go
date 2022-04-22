// Command zendesk-integration tests creating a ticket to a Zendesk instance via
// the Fleet worker processor. It creates it exactly as if a Zendesk integration
// was configured and a new CVE and related CPEs was found.
//
// Note that the Zendesk API token must be provided via an environment
// variable, ZENDESK_TOKEN.
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
		zendeskURL     = flag.String("zendesk-url", "", "The Zendesk instance URL")
		zendeskEmail   = flag.String("zendesk-email", "", "The Zendesk email")
		zendeskGroupID = flag.String("zendesk-group-id", "", "The Zendesk group id")
		fleetURL       = flag.String("fleet-url", "https://localhost:8080", "The Fleet server URL")
		cve            = flag.String("cve", "", "The CVE to create a Zendesk issue for")
		hostsCount     = flag.Int("hosts-count", 1, "The number of hosts to match the CVE")
	)

	flag.Parse()

	if *zendeskURL == "" {
		fmt.Fprintf(os.Stderr, "-zendesk-url is required")
		os.Exit(1)
	}
	if *zendeskEmail == "" {
		fmt.Fprintf(os.Stderr, "-zendesk-username is required")
		os.Exit(1)
	}
	if *zendeskGroupID == "" {
		fmt.Fprintf(os.Stderr, "-zendesk-project-key is required")
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

	zendeskToken := os.Getenv("ZENDESK_TOKEN")
	if zendeskToken == "" {
		fmt.Fprintf(os.Stderr, "ZENDESK_TOKEN is required")
		os.Exit(1)
	}

	logger := kitlog.NewLogfmtLogger(os.Stdout)

	client, err := externalsvc.NewZendeskClient(&externalsvc.ZendeskOptions{
		URL:      *zendeskURL,
		Email:    *zendeskEmail,
		APIToken: zendeskToken,
		GroupID:  *zendeskGroupID,
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

	zendesk := &worker.Zendesk{
		FleetURL:      *fleetURL,
		Datastore:     ds,
		Log:           logger,
		ZendeskClient: client,
	}

	argsJSON := json.RawMessage(fmt.Sprintf(`{"cve":%q}`, *cve))

	err = zendesk.Run(context.Background(), argsJSON)
	if err != nil {
		log.Fatal(err)
	}
}
