package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const JiraName = "jira"

const nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"

var jiraSummaryTmpl = template.Must(template.New("").Parse(
	`{{ .CVE }} detected on hosts`,
))

// TODO: check if jira api supports markdown in summary/description
var jiraDescriptionTmpl = template.Must(template.New("").Parse(
	`See vulnerability (CVE) details in National Vulnerability Database (NVD) here: {{ .NVDURL }}

See all hosts affected by this vulnerability (CVE) in Fleet: {{ .FleetURL }}
`,
))

type jiraTemplateArgs struct {
	CVE      string
	NVDURL   string
	FleetURL string
}

type JiraClient interface {
	CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, *jira.Response, error)
}

type Jira struct {
	fleetURL   string
	projectKey string // TODO: should this be the whole *fleet.AppConfig?
	ds         fleet.Datastore
	log        kitlog.Logger
	jiraClient JiraClient
}

func NewJira(fleetURL, projectKey string, ds fleet.Datastore, log kitlog.Logger, jiraClient JiraClient) *Jira {
	return &Jira{
		fleetURL:   fleetURL,
		projectKey: projectKey,
		ds:         ds,
		log:        log,
		jiraClient: jiraClient,
	}
}

func (j *Jira) Name() string {
	return "jira"
}

type JiraArgs struct {
	CVE  string
	CPEs []string
}

func (j *Jira) Run(ctx context.Context, argsJSON json.RawMessage) error {
	var args JiraArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return fmt.Errorf("unmarshal args: %w", err)
	}

	// TODO: need software_id, not cpes...
	tmplArgs := jiraTemplateArgs{
		CVE:      args.CVE,
		NVDURL:   nvdCVEURL + args.CVE,
		FleetURL: fmt.Sprintf("%s/hosts/manage?order_key=hostname&order_direction=asc&software_id=%d", j.fleetURL, 1),
	}

	var buf bytes.Buffer
	err := jiraSummaryTmpl.Execute(&buf, &tmplArgs) // TODO: separate type for template args?
	if err != nil {
		return fmt.Errorf("execute summary template: %w", err)
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	err = jiraDescriptionTmpl.Execute(&buf, &args)
	if err != nil {
		return fmt.Errorf("execute summary template: %w", err)
	}
	description := buf.String()

	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Project: jira.Project{
				// ID:
				Key: j.projectKey,
			},
			Type: jira.IssueType{
				// ID:
				Name: "Bug", // TODO: make this configurable
			},
			Summary:     summary,
			Description: description,
		},
	}

	issue, _, err = j.jiraClient.CreateIssue(ctx, issue)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}

	level.Debug(j.log).Log(
		"msg", "created jira issue for cve",
		"cve", args.CVE,
		"project_key", j.projectKey,
		"issue_id", issue.ID,
	)

	return nil
}
