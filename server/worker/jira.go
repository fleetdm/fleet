package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"text/template"
	"time"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

	// Jira uses wiki markup in the v2 api. See
	// https://jira.atlassian.com/secure/WikiRendererHelpAction.jspa?section=all
	// for some reference. The `\\` marks force a newline to have the desired spacing
	// around the scores, when present.
	VulnDescription: template.Must(template.New("").Funcs(template.FuncMap{
		// CISAKnownExploit is *bool, so any condition check on it in the template
		// will test if nil or not, and not its actual boolean value. Hence, "deref".
		"deref": func(b *bool) bool { return *b },
	}).Parse(
		`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: [{{ .CVE }}|{{ .NVDURL }}{{ .CVE }}].

{{ if .IsPremium }}{{ if .EPSSProbability }}\\Probability of exploit (reported by [FIRST.org/epss|https://www.first.org/epss/]): {{ .EPSSProbability }}
{{ end }}
{{ if .CVSSScore }}CVSS score (reported by [NVD|https://nvd.nist.gov/]): {{ .CVSSScore }}
{{ end }}
{{ if .CVEPublished }}Published (reported by [NVD|https://nvd.nist.gov/]): {{ .CVEPublished }}
{{ end }}
{{ if .CISAKnownExploit }}Known exploits (reported by [CISA|https://www.cisa.gov/known-exploited-vulnerabilities-catalog]): {{ if deref .CISAKnownExploit }}Yes{{ else }}No{{ end }}
\\
{{ end }}{{ end }}

Affected hosts:

{{ $end := len .Hosts }}{{ if gt $end 50 }}{{ $end = 50 }}{{ end }}
{{ range slice .Hosts 0 $end }}
* [{{ .DisplayName }}|{{ $.FleetURL }}/hosts/{{ .ID }}]
{{ range $path := .SoftwareInstalledPaths }}
** {{ $path }}
{{ end }}
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
		`{{ if .PolicyCritical }}This policy is marked as *Critical* in Fleet.

{{ end }}Hosts:
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
	Hosts    []fleet.HostVulnerabilitySummary

	IsPremium bool

	// the following fields are only included in the ticket for premium licenses.
	EPSSProbability  *float64
	CVSSScore        *float64
	CISAKnownExploit *bool
	CVEPublished     *time.Time
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
	Vulnerability *vulnArgs          `json:"vulnerability,omitempty"`
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
	vargs := args.Vulnerability
	if vargs == nil {
		return errors.New("invalid job args")
	}

	var hosts []fleet.HostVulnerabilitySummary
	var err error

	// Default to deprecated method in case we are processing an 'old' job payload
	// we are deprecating this because of performance reasons - querying by software_id should be
	// way more efficient than by CVE.
	if len(vargs.AffectedSoftwareIDs) == 0 {
		hosts, err = j.Datastore.HostsByCVE(ctx, vargs.CVE)
	} else {
		hosts, err = j.Datastore.HostVulnSummariesBySoftwareIDs(ctx, vargs.AffectedSoftwareIDs)
	}
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching hosts")
	}

	tplArgs := &jiraVulnTplArgs{
		NVDURL:           nvdCVEURL,
		FleetURL:         j.FleetURL,
		CVE:              vargs.CVE,
		Hosts:            hosts,
		IsPremium:        license.IsPremium(ctx),
		EPSSProbability:  vargs.EPSSProbability,
		CVSSScore:        vargs.CVSSScore,
		CISAKnownExploit: vargs.CISAKnownExploit,
		CVEPublished:     vargs.CVEPublished,
	}

	createdIssue, err := j.createTemplatedIssue(ctx, cli, jiraTemplates.VulnSummary, jiraTemplates.VulnDescription, tplArgs)
	if err != nil {
		return err
	}
	level.Debug(j.Log).Log(
		"msg", "created jira issue for cve",
		"cve", vargs.CVE,
		"issue_id", createdIssue.ID,
		"issue_key", createdIssue.Key,
	)
	return nil
}

func (j *Jira) runFailingPolicy(ctx context.Context, cli JiraClient, args jiraArgs) error {
	tplArgs := newFailingPoliciesTplArgs(j.FleetURL, args.FailingPolicy)

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
func QueueJiraVulnJobs(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	recentVulns []fleet.SoftwareVulnerability,
	cveMeta map[string]fleet.CVEMeta,
) error {
	level.Info(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	// for troubleshooting, log in debug level the CVEs that we will process
	// (cannot be done in the loop below as we want to add the debug log
	// _before_ we start processing them).
	cves := make([]string, 0, len(recentVulns))
	for _, vuln := range recentVulns {
		cves = append(cves, vuln.GetCVE())
	}
	sort.Strings(cves)
	level.Debug(logger).Log("recent_cves", fmt.Sprintf("%v", cves))

	cveGrouped := make(map[string][]uint)
	for _, v := range recentVulns {
		cveGrouped[v.GetCVE()] = append(cveGrouped[v.GetCVE()], v.Affected())
	}

	for cve, sIDs := range cveGrouped {
		args := vulnArgs{CVE: cve, AffectedSoftwareIDs: sIDs}
		if meta, ok := cveMeta[cve]; ok {
			args.EPSSProbability = meta.EPSSProbability
			args.CVSSScore = meta.CVSSScore
			args.CISAKnownExploit = meta.CISAKnownExploit
			args.CVEPublished = meta.Published
		}
		job, err := QueueJob(ctx, ds, jiraName, jiraArgs{Vulnerability: &args})
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
		PolicyID:       policy.ID,
		PolicyName:     policy.Name,
		PolicyCritical: policy.Critical,
		Hosts:          hosts,
		TeamID:         policy.TeamID,
	}
	job, err := QueueJob(ctx, ds, jiraName, jiraArgs{FailingPolicy: args})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queueing job")
	}
	level.Debug(logger).Log("job_id", job.ID)
	return nil
}
