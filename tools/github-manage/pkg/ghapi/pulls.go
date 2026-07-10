package ghapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DefaultRepo is the repository jarvis targets when none is specified.
const DefaultRepo = "fleetdm/fleet"

// prListFields is the full set of --json fields for PRs you authored, where CI
// state and mergeability drive bucketing. statusCheckRollup is expensive — only
// request it where it's actually used.
const prListFields = "number,title,url,state,isDraft,mergeable,reviewDecision,author,assignees,labels,createdAt,updatedAt,headRefName,reviewRequests,latestReviews,statusCheckRollup,body"

// prReviewFields is a lighter field set for PRs awaiting your review. Their bucket
// depends only on whether you've already reviewed (latestReviews), not on their CI
// or mergeability — so we omit statusCheckRollup to avoid overloading the GraphQL
// endpoint (it triggers 502s when you're requested on many PRs).
const prReviewFields = "number,title,url,state,author,createdAt,updatedAt,latestReviews"

// ReviewRequest is a requested reviewer on a PR. GitHub returns either a User
// (with Login) or a Team (with Name/Slug), discriminated by Typename.
type ReviewRequest struct {
	Typename string `json:"__typename"`
	Login    string `json:"login"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
}

// Review is a submitted review on a PR (from the latestReviews field).
type Review struct {
	Author      Author `json:"author"`
	State       string `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, DISMISSED
	SubmittedAt string `json:"submittedAt"`
}

// StatusCheck is one entry in a PR's statusCheckRollup. GitHub returns a union of
// CheckRun (status/conclusion) and StatusContext (state); we keep all fields and
// normalize in CIStatus.
type StatusCheck struct {
	Typename   string `json:"__typename"`
	Name       string `json:"name"`
	Status     string `json:"status"`     // CheckRun: QUEUED, IN_PROGRESS, COMPLETED
	Conclusion string `json:"conclusion"` // CheckRun: SUCCESS, FAILURE, CANCELLED, ...
	Context    string `json:"context"`    // StatusContext name
	State      string `json:"state"`      // StatusContext: SUCCESS, PENDING, FAILURE, ERROR
}

// PullRequest mirrors the gh pr list --json output for the fields in prListFields.
type PullRequest struct {
	Number            int             `json:"number"`
	Title             string          `json:"title"`
	URL               string          `json:"url"`
	State             string          `json:"state"`
	IsDraft           bool            `json:"isDraft"`
	Mergeable         string          `json:"mergeable"`      // MERGEABLE, CONFLICTING, UNKNOWN
	ReviewDecision    string          `json:"reviewDecision"` // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, ""
	Author            Author          `json:"author"`
	Assignees         []Author        `json:"assignees"`
	Labels            []Label         `json:"labels"`
	CreatedAt         string          `json:"createdAt"`
	UpdatedAt         string          `json:"updatedAt"`
	HeadRefName       string          `json:"headRefName"`
	Body              string          `json:"body"`
	ReviewRequests    []ReviewRequest `json:"reviewRequests"`
	LatestReviews     []Review        `json:"latestReviews"`
	StatusCheckRollup []StatusCheck   `json:"statusCheckRollup"`

	// UnresolvedThreads is populated separately (not from gh pr list) via
	// GetUnresolvedReviewThreadCount; it counts open review threads on the PR.
	UnresolvedThreads int `json:"-"`
}

// CIStatus normalizes the statusCheckRollup into one of: "passing", "failing",
// "pending", or "none" (no checks configured/reported).
func (pr PullRequest) CIStatus() string {
	if len(pr.StatusCheckRollup) == 0 {
		return "none"
	}
	failing := false
	pending := false
	for _, c := range pr.StatusCheckRollup {
		switch c.Typename {
		case "CheckRun":
			switch c.Conclusion {
			case "FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE":
				failing = true
			case "SUCCESS", "NEUTRAL", "SKIPPED":
				// ok
			default:
				// no conclusion yet -> still running
				if c.Status != "COMPLETED" {
					pending = true
				}
			}
		default: // StatusContext
			switch c.State {
			case "FAILURE", "ERROR":
				failing = true
			case "SUCCESS":
				// ok
			default: // PENDING, EXPECTED
				pending = true
			}
		}
	}
	switch {
	case failing:
		return "failing"
	case pending:
		return "pending"
	default:
		return "passing"
	}
}

// IsApproved reports whether the PR has an approving review decision.
func (pr PullRequest) IsApproved() bool { return pr.ReviewDecision == "APPROVED" }

// ChangesRequested reports whether a reviewer requested changes.
func (pr PullRequest) ChangesRequested() bool { return pr.ReviewDecision == "CHANGES_REQUESTED" }

// HasConflicts reports whether the PR has merge conflicts.
func (pr PullRequest) HasConflicts() bool { return pr.Mergeable == "CONFLICTING" }

// Mergeable-now: approved, no conflicts, CI green, not a draft.
func (pr PullRequest) CanMergeNow() bool {
	return !pr.IsDraft && pr.IsApproved() && pr.Mergeable == "MERGEABLE" && pr.CIStatus() == "passing"
}

// HasOpenReviewerComment reports whether someone other than the author left a
// COMMENTED review as their latest action — a question awaiting your response
// that reviewDecision (which only tracks APPROVED/CHANGES_REQUESTED) misses.
func (pr PullRequest) HasOpenReviewerComment(myLogin string) bool {
	for _, r := range pr.LatestReviews {
		if r.State == "COMMENTED" && !strings.EqualFold(r.Author.Login, myLogin) {
			return true
		}
	}
	return false
}

// ReviewedBy reports whether the given login has already submitted a review.
func (pr PullRequest) ReviewedBy(login string) bool {
	for _, r := range pr.LatestReviews {
		if strings.EqualFold(r.Author.Login, login) {
			return true
		}
	}
	return false
}

// closingKeywordRe matches GitHub's closing keywords ("closes #123", "fixes #45",
// "resolved fleetdm/fleet#678") that auto-link a PR to an issue in the Development
// panel. Case-insensitive; captures the issue number in group 1.
var closingKeywordRe = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\b[^\n#]*#(\d+)`)

// ClosesIssues returns the issue numbers this PR references with a closing keyword
// in its body. This mirrors GitHub's own auto-linking, so it recovers the issue↔PR
// link for PRs opened outside jarvis (jarvis-recorded links are authoritative).
func (pr PullRequest) ClosesIssues() []int {
	var out []int
	seen := map[int]struct{}{}
	for _, m := range closingKeywordRe.FindAllStringSubmatch(pr.Body, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// GetCurrentLogin returns the authenticated gh user's login.
func GetCurrentLogin() (string, error) {
	out, err := RunCommandAndReturnOutput("gh api user --jq .login")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// parsePullRequests unmarshals gh pr list JSON output.
func parsePullRequests(jsonData []byte) ([]PullRequest, error) {
	var prs []PullRequest
	if err := json.Unmarshal(jsonData, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// listPullRequests runs gh pr list with the given search qualifiers and field set.
func listPullRequests(repo, search, fields string, limit int) ([]PullRequest, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	if limit <= 0 {
		limit = 100
	}
	command := fmt.Sprintf(
		"gh pr list --repo %s --state open --search %q --json %s --limit %d",
		repo, search, fields, limit,
	)
	out, err := RunCommandWithRetry(command, 3)
	if err != nil {
		return nil, err
	}
	return parsePullRequests(out)
}

// GetMyPullRequests returns the authenticated user's open PRs in the repo.
func GetMyPullRequests(repo string, limit int) ([]PullRequest, error) {
	return listPullRequests(repo, "author:@me", prListFields, limit)
}

// GetPullRequest fetches a single PR's full field set (for per-item refresh).
func GetPullRequest(repo string, number int) (PullRequest, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	cmd := fmt.Sprintf("gh pr view %d --repo %s --json %s", number, repo, prListFields)
	out, err := RunCommandWithRetry(cmd, 3)
	if err != nil {
		return PullRequest{}, err
	}
	var pr PullRequest
	if err := json.Unmarshal(out, &pr); err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

// GetReviewRequestedPRs returns open PRs that request the authenticated user's review.
func GetReviewRequestedPRs(repo string, limit int) ([]PullRequest, error) {
	return listPullRequests(repo, "review-requested:@me", prReviewFields, limit)
}

// MergePR merges a pull request using the given method (default "squash").
// Returns the gh output. This is a GitHub write — callers should confirm first.
func MergePR(repo string, number int, method string) (string, error) {
	if method == "" {
		method = "squash"
	}
	if repo == "" {
		repo = DefaultRepo
	}
	out, err := RunGH("pr", "merge", fmt.Sprintf("%d", number), "--repo", repo, "--"+method)
	return strings.TrimSpace(string(out)), err
}

// CommentPR posts a comment on a pull request.
func CommentPR(repo string, number int, body string) (string, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	out, err := RunGH("pr", "comment", fmt.Sprintf("%d", number), "--repo", repo, "--body", body)
	return strings.TrimSpace(string(out)), err
}

// CommentIssue posts a comment on an issue.
func CommentIssue(repo string, number int, body string) (string, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	out, err := RunGH("issue", "comment", fmt.Sprintf("%d", number), "--repo", repo, "--body", body)
	return strings.TrimSpace(string(out)), err
}

// GetIssue fetches a single issue's state/metadata (for per-item refresh).
func GetIssue(repo string, number int) (Issue, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	cmd := fmt.Sprintf("gh issue view %d --repo %s --json number,title,url,state,updatedAt,assignees,labels,milestone", number, repo)
	out, err := RunCommandWithRetry(cmd, 3)
	if err != nil {
		return Issue{}, err
	}
	var iss Issue
	if err := json.Unmarshal(out, &iss); err != nil {
		return Issue{}, err
	}
	return iss, nil
}

// GetAssignedIssues returns open issues assigned to the authenticated user.
func GetAssignedIssues(repo string, limit int) ([]Issue, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	if limit <= 0 {
		limit = 100
	}
	command := fmt.Sprintf(
		"gh issue list --repo %s --state open --assignee @me --json number,title,url,author,assignees,createdAt,updatedAt,state,labels,milestone --limit %d",
		repo, limit,
	)
	out, err := RunCommandWithRetry(command, 3)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}
