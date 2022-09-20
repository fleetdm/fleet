package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
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
* [{{ .DisplayName }}]({{ $.FleetURL }}/hosts/{{ .ID }})
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
* [{{ .DisplayName }}]({{ $.FleetURL }}/hosts/{{ .ID }})
{{ end }}

View hosts that failed {{ .PolicyName }} on the [**Hosts**]({{ .FleetURL }}/hosts/manage/?order_key=hostname&order_direction=asc&{{ if .TeamID }}team_id={{ .TeamID }}&{{ end }}policy_id={{ .PolicyID }}&policy_response=failing) page in Fleet.

----

This issue was created automatically by your Fleet Zendesk integration.
`)),
}

type zendeskVulnTplArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

type zendeskFailingPoliciesTplArgs struct {
	FleetURL   string
	PolicyID   uint
	PolicyName string
	TeamID     *uint
	Hosts      []fleet.PolicySetHost
}

// ZendeskClient defines the method required for the client that makes API calls
// to Zendesk.
type ZendeskClient interface {
	CreateZendeskTicket(ctx context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error)
	ZendeskConfigMatches(opts *externalsvc.ZendeskOptions) bool
}

// Zendesk is the job processor for zendesk integrations.
type Zendesk struct {
	FleetURL      string
	Datastore     fleet.Datastore
	Log           kitlog.Logger
	NewClientFunc func(*externalsvc.ZendeskOptions) (ZendeskClient, error)

	// mu protects concurrent access to clientsCache, so that the job processor
	// can potentially be run concurrently.
	mu sync.Mutex
	// map of integration type + team ID to Zendesk client (empty team ID for
	// global), e.g. "vuln:123", "failingPolicy:", etc.
	clientsCache map[string]ZendeskClient
}

// returns nil, nil if there is no integration enabled for that message.
func (z *Zendesk) getClient(ctx context.Context, args zendeskArgs) (ZendeskClient, error) {
	var teamID uint
	var useTeamCfg bool

	intgType := args.integrationType()
	key := intgType + ":"
	if intgType == intgTypeFailingPolicy && args.FailingPolicy.TeamID != nil {
		teamID = *args.FailingPolicy.TeamID
		useTeamCfg = true
		key += fmt.Sprint(teamID)
	}

	ac, err := z.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	// load the config that would be used to create the client first - it is
	// needed to check if an existing client is configured the same or if its
	// configuration has changed since it was created.
	var opts *externalsvc.ZendeskOptions
	if useTeamCfg {
		tm, err := z.Datastore.Team(ctx, teamID)
		if err != nil {
			return nil, err
		}

		intgs, err := tm.Config.Integrations.MatchWithIntegrations(ac.Integrations)
		if err != nil {
			return nil, err
		}

		for _, intg := range intgs.Zendesk {
			if intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies {
				opts = &externalsvc.ZendeskOptions{
					URL:      intg.URL,
					Email:    intg.Email,
					APIToken: intg.APIToken,
					GroupID:  intg.GroupID,
				}
				break
			}
		}
	} else {
		for _, intg := range ac.Integrations.Zendesk {
			if (intgType == intgTypeVuln && intg.EnableSoftwareVulnerabilities) ||
				(intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies) {
				opts = &externalsvc.ZendeskOptions{
					URL:      intg.URL,
					Email:    intg.Email,
					APIToken: intg.APIToken,
					GroupID:  intg.GroupID,
				}
				break
			}
		}
	}

	z.mu.Lock()
	defer z.mu.Unlock()

	if z.clientsCache == nil {
		z.clientsCache = make(map[string]ZendeskClient)
	}
	if opts == nil {
		// no integration configured, clear any existing one
		delete(z.clientsCache, key)
		return nil, nil
	}

	// check if the existing one can be reused
	if cli := z.clientsCache[key]; cli != nil && cli.ZendeskConfigMatches(opts) {
		return cli, nil
	}

	// otherwise create a new one
	cli, err := z.NewClientFunc(opts)
	if err != nil {
		return nil, err
	}
	z.clientsCache[key] = cli
	return cli, nil
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

func (a *zendeskArgs) integrationType() string {
	if a.FailingPolicy == nil {
		return intgTypeVuln
	}
	return intgTypeFailingPolicy
}

// Run executes the zendesk job.
func (z *Zendesk) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args zendeskArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	cli, err := z.getClient(ctx, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get Zendesk client")
	}
	if cli == nil {
		// this message was queued when an integration was enabled, but since
		// then it has been disabled, so return success to mark the message
		// as processed.
		return nil
	}

	switch intgType := args.integrationType(); intgType {
	case intgTypeVuln:
		return z.runVuln(ctx, cli, args)
	case intgTypeFailingPolicy:
		return z.runFailingPolicy(ctx, cli, args)
	default:
		return ctxerr.Errorf(ctx, "unknown integration type: %v", intgType)
	}
}

func (z *Zendesk) runVuln(ctx context.Context, cli ZendeskClient, args zendeskArgs) error {
	hosts, err := z.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tplArgs := &zendeskVulnTplArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: z.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	createdTicket, err := z.createTemplatedTicket(ctx, cli, zendeskTemplates.VulnSummary, zendeskTemplates.VulnDescription, tplArgs)
	if err != nil {
		return err
	}
	level.Debug(z.Log).Log(
		"msg", "created zendesk ticket for cve",
		"cve", args.CVE,
		"ticket_id", createdTicket.ID,
	)
	return nil
}

func (z *Zendesk) runFailingPolicy(ctx context.Context, cli ZendeskClient, args zendeskArgs) error {
	tplArgs := &zendeskFailingPoliciesTplArgs{
		FleetURL:   z.FleetURL,
		PolicyName: args.FailingPolicy.PolicyName,
		PolicyID:   args.FailingPolicy.PolicyID,
		TeamID:     args.FailingPolicy.TeamID,
		Hosts:      args.FailingPolicy.Hosts,
	}

	createdTicket, err := z.createTemplatedTicket(ctx, cli, zendeskTemplates.FailingPolicySummary, zendeskTemplates.FailingPolicyDescription, tplArgs)
	if err != nil {
		return err
	}

	attrs := []interface{}{
		"msg", "created zendesk ticket for failing policy",
		"policy_id", args.FailingPolicy.PolicyID,
		"policy_name", args.FailingPolicy.PolicyName,
		"ticket_id", createdTicket.ID,
	}
	if args.FailingPolicy.TeamID != nil {
		attrs = append(attrs, "team_id", *args.FailingPolicy.TeamID)
	}
	level.Debug(z.Log).Log(attrs...)
	return nil
}

func (z *Zendesk) createTemplatedTicket(ctx context.Context, cli ZendeskClient, summaryTpl, descTpl *template.Template, args interface{}) (*zendesk.Ticket, error) {
	var buf bytes.Buffer
	if err := summaryTpl.Execute(&buf, args); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute summary template")
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := descTpl.Execute(&buf, args); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute description template")
	}
	description := buf.String()

	ticket := &zendesk.Ticket{
		Subject: summary,
		Comment: &zendesk.TicketComment{Body: description},
	}

	createdTicket, err := cli.CreateZendeskTicket(ctx, ticket)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create ticket")
	}
	return createdTicket, nil
}

// QueueZendeskVulnJobs queues the Zendesk vulnerability jobs to process asynchronously
// via the worker.
func QueueZendeskVulnJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns []fleet.SoftwareVulnerability) error {
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

	uniqCVEs := make(map[string]bool)
	for _, v := range recentVulns {
		uniqCVEs[v.CVE] = true
	}

	for cve := range uniqCVEs {
		job, err := QueueJob(ctx, ds, zendeskName, zendeskArgs{CVE: cve})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}

// QueueZendeskFailingPolicyJob queues a Zendesk job for a failing policy to
// process asynchronously via the worker.
func QueueZendeskFailingPolicyJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	policy *fleet.Policy, hosts []fleet.PolicySetHost,
) error {
	attrs := []interface{}{
		"enabled", "true",
		"failing_policy", policy.ID,
		"hosts_count", len(hosts),
	}
	if policy.TeamID != nil {
		attrs = append(attrs, "team_id", *policy.TeamID)
	}
	if len(hosts) == 0 {
		attrs = append(attrs, "msg", "skipping, no host")
		level.Debug(logger).Log(attrs...)
		return nil
	}

	level.Info(logger).Log(attrs...)

	args := &failingPolicyArgs{
		PolicyID:   policy.ID,
		PolicyName: policy.Name,
		TeamID:     policy.TeamID,
		Hosts:      hosts,
	}
	job, err := QueueJob(ctx, ds, zendeskName, zendeskArgs{FailingPolicy: args})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
