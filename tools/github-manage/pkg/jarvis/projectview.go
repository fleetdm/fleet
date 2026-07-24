package jarvis

import (
	"fmt"
	"strconv"
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

// ProjectView is the per-primary-project summary rendered in the top section:
// the project itself (always shown, even empty) plus the issues assigned to you
// (excluding Done / Ready for release) and a count of unassigned Ready issues.
type ProjectView struct {
	Number          int
	Title           string
	URL             string
	Issues          []Item // KindIssue, assigned to you, not Done/Ready-for-release
	ReadyUnassigned int
	Resolved        bool // false if the configured name couldn't be resolved
}

// normalizeStatus reduces a project Status option to lowercase words, dropping
// emoji and punctuation, so "🥚 Ready" → "ready" and "✅ Ready for release" →
// "ready for release".
func normalizeStatus(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace { // collapse emoji/space/punct runs into a single break
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// statusExcluded reports whether an issue in this status should be hidden from
// the Project View (finished work).
func statusExcluded(status string) bool {
	n := normalizeStatus(status)
	return strings.Contains(n, "done") || strings.Contains(n, "ready for release")
}

// statusIsReady reports whether a status is exactly the "Ready" backlog column
// (not "Ready for review" / "Ready for release").
func statusIsReady(status string) bool { return normalizeStatus(status) == "ready" }

// buildProjectViews resolves each configured primary project and loads its items.
// It returns the views (one per configured entry, in order), the set of issue
// numbers surfaced (to exclude from the leverage buckets), and status/project
// maps for the overlay.
func buildProjectViews(login, owner string, primary []string) (views []ProjectView, shown, projects map[int]int, statuses map[int]string) {
	shown = map[int]int{}
	projects = map[int]int{}
	statuses = map[int]string{}
	if len(primary) == 0 {
		return nil, shown, projects, statuses
	}
	orgProjects, _ := ghapi.ListOrgProjects(owner)
	for _, entry := range primary {
		pv := loadProject(entry, owner, orgProjects, login, statuses, projects)
		views = append(views, pv)
		for _, it := range pv.Issues {
			shown[it.Number] = pv.Number
		}
	}
	return views, shown, projects, statuses
}

// RefreshProjectView reloads a single project's view live (its header, the issues
// assigned to you, and the Ready-unassigned count) plus the status/project maps
// for those issues. It's the per-project counterpart to buildProjectViews, backing
// a targeted refresh so newly-assigned issues appear without a full pull.
func RefreshProjectView(num int, owner, login string) (ProjectView, map[int]string, map[int]int) {
	statuses := map[int]string{}
	projects := map[int]int{}
	orgProjects, _ := ghapi.ListOrgProjects(owner)
	pv := loadProject(strconv.Itoa(num), owner, orgProjects, login, statuses, projects)
	return pv, statuses, projects
}

// loadProject resolves one configured entry to a project and loads its items.
// Always returns a view (unresolved entries yield an empty, non-Resolved view so
// the row is still shown).
func loadProject(entry, owner string, orgProjects []ghapi.OrgProject, login string, statuses map[int]string, projects map[int]int) ProjectView {
	num, title, url := resolveProject(entry, owner, orgProjects)
	if num == 0 {
		return ProjectView{Title: entry, Resolved: false}
	}
	pv := ProjectView{Number: num, Title: title, URL: url, Resolved: true}
	items, err := ghapi.GetProjectItems(num, 500)
	if err != nil {
		return pv // header only
	}
	for _, it := range items {
		isIssue := strings.EqualFold(it.Content.Type, "Issue")
		isDraft := strings.EqualFold(it.Content.Type, "DraftIssue")
		if isIssue && containsFold(it.Assignees, login) && !statusExcluded(it.Status) {
			n := it.Content.Number
			pv.Issues = append(pv.Issues, Item{
				Kind: KindIssue, Bucket: BucketPrimary,
				Number: n, Title: it.Content.Title, URL: it.Content.URL,
				Reason: "assigned",
			})
			statuses[n] = it.Status
			projects[n] = num
		}
		if (isIssue || isDraft) && len(it.Assignees) == 0 && statusIsReady(it.Status) {
			pv.ReadyUnassigned++
		}
	}
	return pv
}

// resolveProject maps a configured entry (number, gm alias, or project name) to a
// project number/title/URL. Names match case-insensitively as a substring of the
// project title (titles carry emoji/# prefixes, e.g. "🍎 #g-apple-at-work").
func resolveProject(entry, owner string, orgProjects []ghapi.OrgProject) (num int, title, url string) {
	if id, err := ghapi.ResolveProjectID(entry); err == nil {
		for _, p := range orgProjects {
			if p.Number == id {
				return id, p.Title, p.URL
			}
		}
		return id, entry, fmt.Sprintf("https://github.com/orgs/%s/projects/%d", owner, id)
	}
	want := normalizeProjectName(entry)
	for _, p := range orgProjects {
		if strings.Contains(strings.ToLower(p.Title), want) {
			return p.Number, p.Title, p.URL
		}
	}
	return 0, entry, ""
}

// containsFold reports whether s contains target (case-insensitive).
func containsFold(s []string, target string) bool {
	for _, v := range s {
		if strings.EqualFold(v, target) {
			return true
		}
	}
	return false
}

// projectHeaderItem builds the navigable KindProject row for a project view.
func projectHeaderItem(pv ProjectView) Item {
	reason := fmt.Sprintf("%d unassigned in Ready", pv.ReadyUnassigned)
	if !pv.Resolved {
		reason = "could not resolve — check primary_projects in config.json"
	}
	return Item{
		Kind: KindProject, Bucket: BucketPrimary,
		Number: pv.Number, Title: pv.Title, URL: pv.URL,
		Reason: reason, ReadyUnassigned: pv.ReadyUnassigned,
	}
}
