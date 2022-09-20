package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"text/template"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// jiraName is the name of the job as registered in the worker.
const jiraName = "jira"

var jiraTemplates = struct {
	VulnSummary              *template.Template
	VulnDescription          *template.Template
	FailingPolicySummary     *template.Template
	FailingPolicyDescription *template.Template
}{
	VulnSummary: template.Must(template.New("").Parse(
		`Vulnerability {{ .CVE }} detected on {{ len .Hosts }} host(s)`,
	)),

	// Jira uses wiki markup in the v2 api.
	VulnDescription: template.Must(template.New("").Parse(
		`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: [{{ .CVE }}|{{ .NVDURL }}{{ .CVE }}].

Affected hosts:

{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .DisplayName }}|{{ $.FleetURL }}/hosts/{{ .ID }}]
{{ end }}

View the affected software and more affected hosts:

# Go to the [Software|{{ .FleetURL }}/software/manage] page in Fleet.
# Above the list of software, in the *Search software* box, enter "{{ .CVE }}".
# Hover over the affected software and select *View all hosts*.

----

This issue was created automatically by your Fleet Jira integration.
`)),

	FailingPolicySummary: template.Must(template.New("").Parse(
		`{{ .PolicyName }} policy failed on {{ len .Hosts }} host(s)`,
	)),

	FailingPolicyDescription: template.Must(template.New("").Parse(
		`Hosts:
{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .DisplayName }}|{{ $.FleetURL }}/hosts/{{ .ID }}]
{{ end }}

View hosts that failed {{ .PolicyName }} on the [*Hosts*|{{ .FleetURL }}/hosts/manage/?order_key=hostname&order_direction=asc&{{ if .TeamID }}team_id={{ .TeamID }}&{{ end }}policy_id={{ .PolicyID }}&policy_response=failing] page in Fleet.

----

This issue was created automatically by your Fleet Jira integration.
`)),
}

type jiraVulnTplArgs struct {
	NVDURL   string
	FleetURL string
	CVE      string
	Hosts    []*fleet.HostShort
}

type jiraFailingPoliciesTplArgs struct {
	FleetURL   string
	PolicyID   uint
	PolicyName string
	TeamID     *uint
	Hosts      []fleet.PolicySetHost
}

// JiraClient defines the method required for the client that makes API calls
// to Jira.
type JiraClient interface {
	CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error)
	JiraConfigMatches(opts *externalsvc.JiraOptions) bool
}

// Jira is the job processor for jira integrations.
type Jira struct {
	FleetURL      string
	Datastore     fleet.Datastore
	Log           kitlog.Logger
	NewClientFunc func(*externalsvc.JiraOptions) (JiraClient, error)

	// mu protects concurrent access to clientsCache, so that the job processor
	// can potentially be run concurrently.
	mu sync.Mutex
	// map of integration type + team ID to Jira client (empty team ID for
	// global), e.g. "vuln:123", "failingPolicy:", etc.
	clientsCache map[string]JiraClient
}

// Name returns the name of the job.
func (j *Jira) Name() string {
	return jiraName
}

// returns nil, nil if there is no integration enabled for that message.
func (j *Jira) getClient(ctx context.Context, args jiraArgs) (JiraClient, error) {
	var teamID uint
	var useTeamCfg bool

	intgType := args.integrationType()
	key := intgType + ":"
	if intgType == intgTypeFailingPolicy && args.FailingPolicy.TeamID != nil {
		teamID = *args.FailingPolicy.TeamID
		useTeamCfg = true
		key += fmt.Sprint(teamID)
	}

	ac, err := j.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	// load the config that would be used to create the client first - it is
	// needed to check if an existing client is configured the same or if its
	// configuration has changed since it was created.
	var opts *externalsvc.JiraOptions
	if useTeamCfg {
		tm, err := j.Datastore.Team(ctx, teamID)
		if err != nil {
			return nil, err
		}

		intgs, err := tm.Config.Integrations.MatchWithIntegrations(ac.Integrations)
		if err != nil {
			return nil, err
		}
		for _, intg := range intgs.Jira {
			if intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies {
				opts = &externalsvc.JiraOptions{
					BaseURL:           intg.URL,
					BasicAuthUsername: intg.Username,
					BasicAuthPassword: intg.APIToken,
					ProjectKey:        intg.ProjectKey,
				}
				break
			}
		}
	} else {
		for _, intg := range ac.Integrations.Jira {
			if (intgType == intgTypeVuln && intg.EnableSoftwareVulnerabilities) ||
				(intgType == intgTypeFailingPolicy && intg.EnableFailingPolicies) {
				opts = &externalsvc.JiraOptions{
					BaseURL:           intg.URL,
					BasicAuthUsername: intg.Username,
					BasicAuthPassword: intg.APIToken,
					ProjectKey:        intg.ProjectKey,
				}
				break
			}
		}
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	if j.clientsCache == nil {
		j.clientsCache = make(map[string]JiraClient)
	}
	if opts == nil {
		// no integration configured, clear any existing one
		delete(j.clientsCache, key)
		return nil, nil
	}

	// check if the existing one can be reused
	if cli := j.clientsCache[key]; cli != nil && cli.JiraConfigMatches(opts) {
		return cli, nil
	}

	// otherwise create a new one
	cli, err := j.NewClientFunc(opts)
	if err != nil {
		return nil, err
	}
	j.clientsCache[key] = cli
	return cli, nil
}

// jiraArgs are the arguments for the Jira integration job.
type jiraArgs struct {
	CVE           string             `json:"cve,omitempty"`
	FailingPolicy *failingPolicyArgs `json:"failing_policy,omitempty"`
}

func (a *jiraArgs) integrationType() string {
	if a.FailingPolicy == nil {
		return intgTypeVuln
	}
	return intgTypeFailingPolicy
}

// Run executes the jira job.
func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args jiraArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal args")
	}

	cli, err := j.getClient(ctx, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get Jira client")
	}
	if cli == nil {
		// this message was queued when an integration was enabled, but since
		// then it has been disabled, so return success to mark the message
		// as processed.
		return nil
	}

	switch intgType := args.integrationType(); intgType {
	case intgTypeVuln:
		return j.runVuln(ctx, cli, args)
	case intgTypeFailingPolicy:
		return j.runFailingPolicy(ctx, cli, args)
	default:
		return ctxerr.Errorf(ctx, "unknown integration type: %v", intgType)
	}
}

func (j *Jira) runVuln(ctx context.Context, cli JiraClient, args jiraArgs) error {
	hosts, err := j.Datastore.HostsByCVE(ctx, args.CVE)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find hosts by cve")
	}

	tplArgs := &jiraVulnTplArgs{
		NVDURL:   nvdCVEURL,
		FleetURL: j.FleetURL,
		CVE:      args.CVE,
		Hosts:    hosts,
	}

	createdIssue, err := j.createTemplatedIssue(ctx, cli, jiraTemplates.VulnSummary, jiraTemplates.VulnDescription, tplArgs)
	if err != nil {
		return err
	}
	level.Debug(j.Log).Log(
		"msg", "created jira issue for cve",
		"cve", args.CVE,
		"issue_id", createdIssue.ID,
		"issue_key", createdIssue.Key,
	)
	return nil
}

func (j *Jira) runFailingPolicy(ctx context.Context, cli JiraClient, args jiraArgs) error {
	tplArgs := &jiraFailingPoliciesTplArgs{
		FleetURL:   j.FleetURL,
		PolicyName: args.FailingPolicy.PolicyName,
		PolicyID:   args.FailingPolicy.PolicyID,
		TeamID:     args.FailingPolicy.TeamID,
		Hosts:      args.FailingPolicy.Hosts,
	}

	createdIssue, err := j.createTemplatedIssue(ctx, cli, jiraTemplates.FailingPolicySummary, jiraTemplates.FailingPolicyDescription, tplArgs)
	if err != nil {
		return err
	}

	attrs := []interface{}{
		"msg", "created jira issue for failing policy",
		"policy_id", args.FailingPolicy.PolicyID,
		"policy_name", args.FailingPolicy.PolicyName,
		"issue_id", createdIssue.ID,
		"issue_key", createdIssue.Key,
	}
	if args.FailingPolicy.TeamID != nil {
		attrs = append(attrs, "team_id", *args.FailingPolicy.TeamID)
	}
	level.Debug(j.Log).Log(attrs...)
	return nil
}

func (j *Jira) createTemplatedIssue(ctx context.Context, cli JiraClient, summaryTpl, descTpl *template.Template, args interface{}) (*jira.Issue, error) {
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

	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				Name: "Task",
			},
			Summary:     summary,
			Description: description,
		},
	}

	createdIssue, err := cli.CreateJiraIssue(ctx, issue)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create issue")
	}
	return createdIssue, nil
}

// QueueJiraVulnJobs queues the Jira vulnerability jobs to process asynchronously
// via the worker.
func QueueJiraVulnJobs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, recentVulns []fleet.SoftwareVulnerability) error {
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
		job, err := QueueJob(ctx, ds, jiraName, jiraArgs{CVE: cve})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "queueing job")
		}
		level.Debug(logger).Log("job_id", job.ID)
	}
	return nil
}

// QueueJiraFailingPolicyJob queues a Jira job for a failing policy to process
// asynchronously via the worker.
func QueueJiraFailingPolicyJob(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
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
		Hosts:      hosts,
		TeamID:     policy.TeamID,
	}
	job, err := QueueJob(ctx, ds, jiraName, jiraArgs{FailingPolicy: args})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
