package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	zendesk "github.com/nukosuke/go-zendesk/zendesk"
)

// zendeskName is the name of the job as registered in the worker.
const zendeskName = "zendesk"

var zendeskTemplates = struct {
	VulnSummary              *template.Template
	VulnDescription          *template.Template
	FailingPolicySummary     *template.Template
	FailingPolicyDescription *template.Template
}{
	VulnSummary: template.Must(template.New("").Parse(
		`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
	)),

	VulnDescription: template.Must(template.New("").Parse(
		`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: [{{ .CVE }}]({{ .NVDURL }}{{ .CVE }}).

Affected hosts:

{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .Hostname }}]({{ $.FleetURL }}/hosts/{{ .ID }})
{{ end }}

View the affected software and more affected hosts:

1. Go to the [Software]({{ .FleetURL }}/software/manage) page in Fleet.
2. Above the list of software, in the *Search software* box, enter "{{ .CVE }}".
3. Hover over the affected software and select *View all hosts*.

----

This ticket was created automatically by your Fleet Zendesk integration.
`)),

	FailingPolicySummary: template.Must(template.New("").Parse(
		`{{ .PolicyName }} policy failed on {{ len .Hosts }} host(s)`,
	)),

	FailingPolicyDescription: template.Must(template.New("").Parse(
		`Hosts:
{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .Hostname }}]({{ $.FleetURL }}/hosts/{{ .ID }})
{{ end }}

View hosts that failed {{ .PolicyName }} on the [**Hosts**]({{ .FleetURL }}/hosts/manage) page in Fleet.

----

This issue was created automatically by your Fleet Zendesk integration.
`)),
}

type zendeskVulnTemplateArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

// ZendeskClient defines the method required for the client that makes API calls
// to Zendesk.
type ZendeskClient interface {
	CreateZendeskTicket(ctx context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error)
}

// Zendesk is the job processor for zendesk integrations.
type Zendesk struct {
	FleetURL      string
	Datastore     fleet.Datastore
	Log           kitlog.Logger
	ZendeskClient ZendeskClient
}

// Name returns the name of the job.
func (z *Zendesk) Name() string {
	return zendeskName
}

// zendeskArgs are the arguments for the Zendesk integration job.
type zendeskArgs struct {
	CVE           string             `json:"cve,omitempty"`
	FailingPolicy *failingPolicyArgs `json:"failing_policy,omitempty"`
}

// Run executes the zendesk job.
func (z *Zendesk) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args zendeskArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	switch {
	case args.CVE != "":
		return z.runVuln(ctx, args)
	//case args.FailingPolicy != nil:
	default:
		return ctxerr.New(ctx, "empty ZendeskArgs, nothing to process")
	}
}

func (z *Zendesk) runVuln(ctx context.Context, args zendeskArgs) error {
	hosts, err := z.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tmplArgs := zendeskVulnTemplateArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: z.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	var buf bytes.Buffer
	if err := zendeskTemplates.VulnSummary.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := zendeskTemplates.VulnDescription.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	ticket := &zendesk.Ticket{
		Subject: summary,
		Comment: &zendesk.TicketComment{Body: description},
	}

	createdTicket, err := z.ZendeskClient.CreateZendeskTicket(ctx, ticket)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create ticket")
	}

	level.Debug(z.Log).Log(
		"msg", "created zendesk ticket for cve",
		"cve", args.CVE,
		"ticket_id", createdTicket.ID,
	)

	return nil
}

// QueueZendeskVulnJobs queues the Zendesk vulnerability jobs to process asynchronously
// via the worker.
func QueueZendeskVulnJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns map[string][]string) error {
	level.Info(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	// for troubleshooting, log in debug level the CVEs that we will process
	// (cannot be done in the loop below as we want to add the debug log
	// _before_ we start processing them).
	cves := make([]string, 0, len(recentVulns))
	for cve := range recentVulns {
		cves = append(cves, cve)
	}
	sort.Strings(cves)
	level.Debug(logger).Log("recent_cves", fmt.Sprintf("%v", cves))

	for cve := range recentVulns {
		job, err := QueueJob(ctx, ds, zendeskName, zendeskArgs{CVE: cve})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}
