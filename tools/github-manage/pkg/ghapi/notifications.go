package ghapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Notification is a GitHub notification. The Reason field encodes why it's in
// your inbox (mention, review_requested, assign, author, comment, team_mention,
// ci_activity, ...) — the canonical "waiting on me" signal.
type Notification struct {
	Reason    string `json:"reason"`
	Unread    bool   `json:"unread"`
	UpdatedAt string `json:"updated_at"`
	Subject   struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Type  string `json:"type"` // Issue, PullRequest, ...
	} `json:"subject"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}

// IsPR reports whether the notification subject is a pull request.
func (n Notification) IsPR() bool { return n.Subject.Type == "PullRequest" }

// Number extracts the issue/PR number from the subject API URL, or 0 if absent
// (e.g. release or discussion notifications).
func (n Notification) Number() int {
	parts := strings.Split(strings.TrimRight(n.Subject.URL, "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	num, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return num
}

// HTMLURL builds the browser URL for the subject.
func (n Notification) HTMLURL() string {
	num := n.Number()
	if num == 0 || n.Repository.HTMLURL == "" {
		return ""
	}
	seg := "issues"
	if n.IsPR() {
		seg = "pull"
	}
	return fmt.Sprintf("%s/%s/%d", n.Repository.HTMLURL, seg, num)
}

// GetNotifications returns the authenticated user's unread notifications for the
// repo. Requires the gh token to have the notifications scope; callers should
// treat a failure as non-fatal.
func GetNotifications(repo string) ([]Notification, error) {
	if repo == "" {
		repo = DefaultRepo
	}
	cmd := fmt.Sprintf("gh api /repos/%s/notifications --paginate", repo)
	out, err := RunCommandWithRetry(cmd, 3)
	if err != nil {
		return nil, err
	}
	var ns []Notification
	if err := json.Unmarshal(out, &ns); err != nil {
		return nil, err
	}
	return ns, nil
}
