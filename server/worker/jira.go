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

const (
	JiraName = "jira"

	nvdCVEURL = "https://nvd.nist.gov/vuln/detail/"
)

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
	CreateIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error)
}

type Jira struct {
	FleetURL   string
	Datastore  fleet.Datastore
	Log        kitlog.Logger
	JiraClient JiraClient
}

func (j *Jira) Name() string {
	return JiraName
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
		FleetURL: fmt.Sprintf("%s/hosts/manage?order_key=hostname&order_direction=asc&software_id=%d", j.FleetURL, 1),
	}

	var buf bytes.Buffer
	if err := jiraSummaryTmpl.Execute(&buf, &tmplArgs); err != nil { // TODO: separate type for template args?
		return fmt.Errorf("execute summary template: %w", err)
	}
	summary := buf.String()

	buf.Reset() // reuse buffer
	if err := jiraDescriptionTmpl.Execute(&buf, &args); err != nil {
		return fmt.Errorf("execute summary template: %w", err)
	}
	description := buf.String()

	issue := &jira.Issue{
		Fields: &jira.IssueFields{
			Type: jira.IssueType{
				// ID:
				Name: "Bug", // TODO: make this configurable
			},
			Summary:     summary,
			Description: description,
		},
	}

	createdIssue, err := j.JiraClient.CreateIssue(ctx, issue)
	if err != nil {
		return fmt.Errorf("create issue: %w", err)
	}

	level.Debug(j.Log).Log(
		"msg", "created jira issue for cve",
		"cve", args.CVE,
		"issue_id", createdIssue.ID,
	)

	return nil
}
