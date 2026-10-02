package jarvis

import (
	"sort"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"
)

// FetchResult is the data backing one render of the dashboard.
type FetchResult struct {
	Login string
	Board Board
	// Statuses maps issue key (ghapi.IssueRefKey — boards and notifications mix
	// repos, so numbers alone collide) → project Status column; Projects maps
	// issue key → the board id that Status came from. Both are best-effort.
	Statuses map[string]string
	Projects map[string]int
	// IssueProjects maps issue key → the projects it belongs to (with each
	// project's last-updated time), used to jump to an issue's most recently
	// updated board.
	IssueProjects map[string][]ProjectRef
	// LocalBranches maps a branch name → the local clone folder that has it, so
	// the Project View can show where a branch lives.
	LocalBranches map[string]string
	// LinkedMergedPRs maps an issue number → its merged/closed PR, discovered by
	// branch for Project View issues whose PR has dropped off the open list. Lets
	// the issue keep showing its shipped PR (as merged) so it's clearly ready for QA.
	LinkedMergedPRs map[int]*ghapi.PullRequest
}

// ProjectRef is a lightweight reference to a project an issue belongs to.
type ProjectRef struct {
	Number    int    `json:"number"`
	UpdatedAt string `json:"updated_at"` // RFC3339; "" if unknown
	Title     string `json:"title,omitempty"`
	Status    string `json:"status,omitempty"` // the issue's Status column on THIS board
}

// FetchProgress is one loading-progress tick: which phase of the full fetch is
// running (1-based, of Phases) and, within it, how many of its items are done.
// Total is 0 for phases with no per-item breakdown (single calls).
type FetchProgress struct {
	Phase     int
	Phases    int
	PhaseName string
	Done      int
	Total     int
}

// ProgressFunc receives loading-progress ticks during Fetch. May be nil.
type ProgressFunc func(FetchProgress)

// fetchPhaseNames are the phases of a full fetch, in order. The last phase
// (local branches & linked PRs) runs in fetchCmd after Fetch returns.
var fetchPhaseNames = []string{
	"your pull requests",
	"review requests",
	"assigned issues",
	"notifications & sessions",
	"issue board statuses",
	"project boards",
	"board cross-checks",
	"local branches & linked PRs",
}

// Fetch gathers the user's PRs, review requests, and assigned issues from the
// repo and classifies them into the leverage board. It runs the gh calls
// sequentially; each one wraps the gh CLI via ghapi. primaryProjects are the
// user's configured primary boards (numbers, aliases, or names) whose assigned
// issues surface in the top section. progress (may be nil) receives loading
// ticks for the TUI's two-level progress display.
func Fetch(repo string, limit int, primaryProjects []string, role string, progress ProgressFunc) (FetchResult, error) {
	step := func(phase, done, total int) {
		if progress != nil {
			progress(FetchProgress{
				Phase: phase, Phases: len(fetchPhaseNames), PhaseName: fetchPhaseNames[phase-1],
				Done: done, Total: total,
			})
		}
	}

	step(1, 0, 0)
	login, err := ghapi.GetCurrentLogin()
	if err != nil {
		return FetchResult{}, err
	}
	myPRs, err := ghapi.GetMyPullRequests(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	// Enrich your PRs with unresolved review-thread counts (a "your move" signal
	// reviewDecision misses). Per-PR GraphQL; best-effort per PR.
	step(1, 0, len(myPRs))
	for i := range myPRs {
		if c, err := ghapi.GetUnresolvedReviewThreadCount(repo, myPRs[i].Number); err == nil {
			myPRs[i].UnresolvedThreads = c
		}
		step(1, i+1, len(myPRs))
	}
	step(2, 0, 0)
	reviewPRs, err := ghapi.GetReviewRequestedPRs(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	step(3, 0, 0)
	issues, err := ghapi.GetAssignedIssues(repo, limit)
	if err != nil {
		return FetchResult{}, err
	}
	// These sources are best-effort — a failure shouldn't blank the dashboard.
	// (Notifications need the gh `notifications` scope; cherry-picks need a local
	// RC ref; sessions need ~/.claude/projects.)
	step(4, 0, 2)
	notifications, _ := ghapi.GetNotifications(repo)
	step(4, 1, 2)
	sessions, _ := DiscoverSessions(30)
	step(4, 2, 2)

	targets := statusTargets(issues, notifications, repo)
	step(5, 0, len(targets))
	statuses, projects, issueProjects := fetchIssueStatuses(targets, func(done, total int) {
		step(5, done, total)
	})

	// Project View (top section): per configured primary project, the issues
	// assigned to you + a Ready-backlog count. Its issues are excluded from the
	// leverage buckets below so they aren't shown twice.
	var primaryItems []Item
	exclude := map[string]bool{}
	step(6, 0, len(primaryProjects))
	if len(primaryProjects) > 0 {
		views, shown, pProjects, pStatuses := buildProjectViews(login, repoOwner(repo), repo, primaryProjects, role, func(done, total int) {
			step(6, done, total)
		})
		for k := range shown {
			exclude[k] = true
		}
		for k, p := range pProjects {
			projects[k] = p
		}
		for k, s := range pStatuses {
			statuses[k] = s
		}
		for _, pv := range views {
			primaryItems = append(primaryItems, projectHeaderItem(pv))
			primaryItems = append(primaryItems, pv.Issues...)
		}
	}

	step(7, 0, 2)
	board := BuildBoard(repo, login, myPRs, reviewPRs, issues, sessions, notifications, exclude, time.Now())
	if len(primaryItems) > 0 {
		board.Buckets[BucketPrimary] = primaryItems // ordered header→issues; not re-sorted
	}
	// Notification gap-fillers carry no PR/issue state, so a merged PR or closed
	// issue can linger (e.g. an old review_requested notification). Verify their
	// state in one batched query and drop the finished ones. Done before cherry-
	// picks are added — those are intentionally merged PRs.
	dropFinishedNotificationItems(&board, repo)
	step(7, 1, 2)
	board.AddItems(PendingCherryPicks(repo))
	step(7, 2, 2)

	return FetchResult{
		Login: login, Board: board,
		Statuses: statuses, Projects: projects, IssueProjects: issueProjects,
	}, nil
}

// dropFinishedNotificationItems removes notification-sourced items whose
// underlying PR is merged/closed or whose issue is closed. Notifications come
// from any repo (e.g. fleetdm/confidential), so states are batched per repo
// from each item's own URL. Best-effort: on a lookup error nothing is dropped.
func dropFinishedNotificationItems(board *Board, fallbackRepo string) {
	prNums := map[string][]int{}    // repo → PR numbers
	issueNums := map[string][]int{} // repo → issue numbers
	for _, bk := range BucketOrder {
		for _, it := range board.Buckets[bk] {
			if !it.FromNotification {
				continue
			}
			r := repoOr(it.URL, fallbackRepo)
			switch it.Kind {
			case KindPR:
				prNums[r] = append(prNums[r], it.Number)
			case KindIssue:
				issueNums[r] = append(issueNums[r], it.Number)
			}
		}
	}
	if len(prNums) == 0 && len(issueNums) == 0 {
		return
	}

	prState := map[string]string{}    // issue key → state
	issueState := map[string]string{} // issue key → state
	for r, nums := range prNums {
		states, _ := ghapi.GetPRStates(r, nums)
		for n, s := range states {
			prState[ghapi.IssueRefKey(r, n)] = s
		}
	}
	for r, nums := range issueNums {
		states, _ := ghapi.GetIssueStates(r, nums)
		for n, s := range states {
			issueState[ghapi.IssueRefKey(r, n)] = s
		}
	}

	finished := func(it Item) bool {
		if !it.FromNotification {
			return false
		}
		key := ghapi.IssueRefKey(repoOr(it.URL, fallbackRepo), it.Number)
		switch it.Kind {
		case KindPR:
			s := prState[key]
			return s == "MERGED" || s == "CLOSED"
		case KindIssue:
			return issueState[key] == "CLOSED"
		}
		return false
	}

	for bk := range board.Buckets {
		items := board.Buckets[bk]
		kept := items[:0]
		for _, it := range items {
			if !finished(it) {
				kept = append(kept, it)
			}
		}
		board.Buckets[bk] = kept
	}
}

// linkedMergedPRs finds, for each Project View issue with a recorded branch but no
// open PR in the board, the merged/closed PR on that branch — so the issue keeps
// showing its shipped PR (as merged) and is clearly ready to advance to QA.
// Best-effort: branches already served by an open PR are skipped (that PR flows
// through the normal bucket path), and lookups that error or find nothing are
// simply omitted. Returns nil when nothing was found.
func linkedMergedPRs(repo string, board Board, branchByIssue map[int]string) map[int]*ghapi.PullRequest {
	openBranch := map[string]bool{}
	for _, bk := range BucketOrder {
		for _, it := range board.Buckets[bk] {
			if it.Kind == KindPR && it.PR != nil && it.PR.HeadRefName != "" {
				openBranch[it.PR.HeadRefName] = true
			}
		}
	}
	out := map[int]*ghapi.PullRequest{}
	for _, it := range board.Buckets[BucketPrimary] {
		if it.Kind != KindIssue {
			continue
		}
		branch := branchByIssue[it.Number]
		if branch == "" || openBranch[branch] {
			continue
		}
		pr, ok, err := ghapi.GetPRByBranch(repo, branch)
		if err != nil || !ok || !pr.IsDone() {
			continue
		}
		prCopy := pr
		out[it.Number] = &prCopy
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func repoOwner(repo string) string {
	if i := strings.IndexByte(repo, '/'); i > 0 {
		return repo[:i]
	}
	return "fleetdm"
}

// repoFromURL extracts "owner/name" from a github.com URL, or "" when it
// can't. Project boards span repos (e.g. fleetdm/confidential), so an item's
// URL — not the dashboard's home repo — says where its issue lives.
func repoFromURL(url string) string { return ghapi.RepoFromURL(url) }

// repoOr returns the repo an issue URL points at, falling back when unparseable.
func repoOr(url, fallback string) string {
	if r := repoFromURL(url); r != "" {
		return r
	}
	return fallback
}

// fetchIssueStatuses reads each assigned issue's project Status column, picking
// the board that owns its workflow status, and records every project the issue
// belongs to (with its updatedAt). Best-effort and per-issue: a failure on one
// issue leaves it absent rather than failing the whole fetch.
func fetchIssueStatuses(targets []issueStatusTarget, tick func(done, total int)) (statuses map[string]string, projects map[string]int, issueProjects map[string][]ProjectRef) {
	statuses = map[string]string{}
	projects = map[string]int{}
	issueProjects = map[string][]ProjectRef{}

	// Group by repo so each repo's issues go out in batched (aliased) GraphQL
	// requests instead of one round-trip per issue.
	byRepo := map[string][]int{}
	var repoOrder []string
	for _, t := range targets {
		if _, ok := byRepo[t.repo]; !ok {
			repoOrder = append(repoOrder, t.repo)
		}
		byRepo[t.repo] = append(byRepo[t.repo], t.number)
	}

	done := 0
	for _, repo := range repoOrder {
		nums := byRepo[repo]
		base := done
		batch := ghapi.GetIssueProjectStatusesBatch(repo, nums, func(d, _ int) {
			if tick != nil {
				tick(base+d, len(targets))
			}
		})
		done += len(nums)
		for num, found := range batch {
			if len(found) == 0 {
				continue
			}
			key := ghapi.IssueRefKey(repo, num)
			var refs []ProjectRef
			for pid, ps := range found {
				if ps.Present {
					refs = append(refs, ProjectRef{Number: pid, UpdatedAt: ps.UpdatedAt, Title: ps.Title, Status: ps.Status})
				}
			}
			if len(refs) > 0 {
				issueProjects[key] = refs
			}
			pid, status := pickWorkflowStatus(found)
			if status != "" || pid != 0 {
				statuses[key] = status
				projects[key] = pid
			}
		}
	}
	return statuses, projects, issueProjects
}

// issueStatusTarget identifies an issue whose project statuses should be read.
type issueStatusTarget struct {
	repo   string // "owner/name"; "" falls back to the cwd repo
	number int
}

// statusTargets lists the issues to read project statuses for: the user's
// assigned issues plus issue-kind notifications (gap-filler items like
// "mentioned you" aren't assigned to the user, but still need their board
// membership for the team emoji and group status shown on their rows).
func statusTargets(issues []ghapi.Issue, notifications []ghapi.Notification, fallbackRepo string) []issueStatusTarget {
	seen := map[string]bool{}
	var out []issueStatusTarget
	add := func(repo string, number int) {
		if number == 0 {
			return
		}
		key := ghapi.IssueRefKey(repo, number) // lowercased, so repo casing dedups
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, issueStatusTarget{repo: repo, number: number})
	}
	for _, iss := range issues {
		add(repoOr(iss.URL, fallbackRepo), iss.Number)
	}
	for _, n := range notifications {
		if n.IsPR() {
			continue
		}
		repo := n.Repository.FullName
		if repo == "" {
			repo = fallbackRepo
		}
		add(repo, n.Number())
	}
	return out
}

// workflowKeywords are the substrings that identify a board's workflow Status
// (as opposed to an unrelated single-select field's value).
var workflowKeywords = []string{"ready", "progress", "review", "await", "qa", "draft", "blocked"}

// pickWorkflowStatus chooses, among the projects an issue belongs to, the one
// whose Status looks like a workflow column. Falls back to the lowest-numbered
// project with any Status set, then to the lowest-numbered present project.
// Projects are scanned in ascending number order so the pick is deterministic
// (map iteration order would otherwise flap the status between an issue's boards).
func pickWorkflowStatus(found map[int]ghapi.ProjectStatus) (int, string) {
	pids := make([]int, 0, len(found))
	for pid := range found {
		pids = append(pids, pid)
	}
	sort.Ints(pids)

	var firstSet, firstPresent int
	var firstSetStatus string
	for _, pid := range pids {
		ps := found[pid]
		if !ps.Present {
			continue
		}
		if firstPresent == 0 {
			firstPresent = pid
		}
		if ps.Status == "" {
			continue
		}
		if firstSet == 0 {
			firstSet, firstSetStatus = pid, ps.Status
		}
		low := strings.ToLower(ps.Status)
		for _, kw := range workflowKeywords {
			if strings.Contains(low, kw) {
				return pid, ps.Status
			}
		}
	}
	if firstSet != 0 {
		return firstSet, firstSetStatus
	}
	return firstPresent, ""
}
