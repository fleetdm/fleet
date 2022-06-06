package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	zendesk "github.com/nukosuke/go-zendesk/zendesk"
)

// zendeskName is the name of the job as registered in the worker.
const zendeskName = "zendesk"

var zendeskSummaryTmpl = template.Must(template.New("").Parse(
	`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
))

var zendeskDescriptionTmpl = template.Must(template.New("").Parse(
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
`,
))

type zendeskTemplateArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

// ZendeskClient defines the method required for the client that makes API calls
// to Zendesk.
type ZendeskClient interface {
	CreateTicket(ctx context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error)
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

// ZendeskArgs are the arguments for the Zendesk integration job.
type ZendeskArgs struct {
	CVE string `json:"cve"`
}

// Run executes the zendesk job.
func (z *Zendesk) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args ZendeskArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	hosts, err := z.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tmplArgs := zendeskTemplateArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: z.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	var buf bytes.Buffer
	if err := zendeskSummaryTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := zendeskDescriptionTmpl.Execute(&buf, &tmplArgs); err != nil {
		return ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	ticket := &zendesk.Ticket{
		Subject: summary,
		Comment: &zendesk.TicketComment{Body: description},
	}

	createdTicket, err := z.ZendeskClient.CreateTicket(ctx, ticket)
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

// QueueZendeskJobs queues the Zendesk vulnerability jobs to process asynchronously
// via the worker.
func QueueZendeskJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns []fleet.SoftwareVulnerability) error {
	level.Info(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	// for troubleshooting, log in debug level the CVEs that we will process
	// (cannot be done in the loop below as we want to add the debug log
	// _before_ we start processing them).
	cves := make([]string, 0, len(recentVulns))
	for _, vuln := range recentVulns {
		cves = append(cves, vuln.CVE)
	}
	sort.Strings(cves)
	level.Debug(logger).Log("recent_cves", fmt.Sprintf("%v", cves))

	for _, vuln := range recentVulns {
		job, err := QueueJob(ctx, ds, zendeskName, ZendeskArgs{CVE: vuln.CVE})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}
