package jarvis

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

var rcVersionRe = regexp.MustCompile(`rc-(?:minor|patch)-fleet-v(\d+\.\d+\.\d+)`)

// PendingCherryPicks returns merged PRs you authored that are milestoned for the
// current RC's version but whose merge commit isn't yet in the RC branch.
//
// It relies on the local git checkout (the RC ref must be fetched). Everything is
// best-effort: if there's no RC branch, the ref is missing, or git/gh fails, it
// returns nil rather than surfacing false positives.
func PendingCherryPicks(repo string) []Item {
	rc := detectRCBranch()
	if rc == "" || !rcRefExists(rc) {
		return nil
	}
	m := rcVersionRe.FindStringSubmatch(rc)
	if m == nil {
		return nil
	}
	version := m[1]

	var items []Item
	for _, pr := range mergedPRsForMilestone(repo, version) {
		if pr.MergeCommit.Oid == "" {
			continue
		}
		if commitInRC(pr.MergeCommit.Oid, rc) {
			continue
		}
		items = append(items, Item{
			Kind:    KindPR,
			Bucket:  BucketNeedsYourHands,
			Number:  pr.Number,
			Title:   pr.Title,
			URL:     pr.URL,
			Updated: parseTime(pr.UpdatedAt),
			Reason:  "cherry-pick → " + rc,
		})
	}
	return items
}

// detectRCBranch returns the most recent rc-minor-fleet-v* branch on origin.
func detectRCBranch() string {
	out, err := ghapi.RunCommandAndReturnOutput(
		`git for-each-ref 'refs/remotes/origin/rc-minor-fleet-v*' --format='%(refname:strip=3)' | ` +
			`grep -E '^rc-minor-fleet-v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1`,
	)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func rcRefExists(rc string) bool {
	_, err := ghapi.RunCommandAndReturnOutput("git rev-parse --verify --quiet origin/" + rc)
	return err == nil
}

// commitInRC reports whether sha is an ancestor of origin/rc.
func commitInRC(sha, rc string) bool {
	out, _ := ghapi.RunCommandAndReturnOutput(
		fmt.Sprintf("git merge-base --is-ancestor %s origin/%s && echo IN || echo OUT", sha, rc),
	)
	return strings.Contains(string(out), "IN")
}

type mergedPR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	UpdatedAt   string `json:"updatedAt"`
	MergeCommit struct {
		Oid string `json:"oid"`
	} `json:"mergeCommit"`
}

func mergedPRsForMilestone(repo, version string) []mergedPR {
	cmd := fmt.Sprintf(
		`gh pr list --repo %s --state merged --author @me --search %q --json number,title,url,mergeCommit,updatedAt --limit 50`,
		repo, "milestone:"+version,
	)
	out, err := ghapi.RunCommandWithRetry(cmd, 3)
	if err != nil {
		return nil
	}
	var prs []mergedPR
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil
	}
	return prs
}
