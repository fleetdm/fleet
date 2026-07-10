package jarvis

import (
	"strings"

	"fleetdm/gm/pkg/ghapi"
)

// Action is the single most useful next step jarvis can offer on a work item,
// derived purely from (project status, PR presence, PR mergeability).
type Action int

const (
	ActNone           Action = iota
	ActStartWork             // Ready/unstarted → branch off main, launch session, set In progress
	ActOpenPR                // work in a session but no PR yet → open one
	ActAddressPR             // PR open but not mergeable → your move (feedback/CI/conflicts)
	ActMarkInReview          // PR mergeable while In progress → set In review
	ActMerge                 // PR mergeable while In review → merge (then Awaiting QA)
	ActMarkAwaitingQA        // PR merged/closed but status not yet advanced → Awaiting QA
)

type actionMeta struct {
	Label string // shown on the card
	Key   string // key hint for the footer
}

var actionMetas = map[Action]actionMeta{
	ActNone:           {"", ""},
	ActStartWork:      {"start work", "w"},
	ActOpenPR:         {"open PR", "P"},
	ActAddressPR:      {"your move", "J"},
	ActMarkInReview:   {"mark in review", "v"},
	ActMerge:          {"merge", "m"},
	ActMarkAwaitingQA: {"→ awaiting QA", "a"},
}

// Label returns the human label for an action.
func (a Action) Label() string { return actionMetas[a].Label }

// Key returns the footer key hint for an action.
func (a Action) Key() string { return actionMetas[a].Key }

// WorkItem aggregates everything tied to one issue you're driving: the issue, its
// project Status column, the PR implementing it, the local branch/clone, and the
// Claude session working on it. It's an issue-centric overlay on the leverage
// board — the PR keeps its own bucket classification for the flat view.
type WorkItem struct {
	Issue     *ghapi.Issue
	Number    int
	Title     string
	URL       string
	Status    string // project Status column value ("" if none/unstarted)
	Project   int    // board that owns Status (0 if unknown)
	PR        *Item  // linked PR item (already leverage-classified), nil if none
	Branch    string
	ClonePath string
	SessionID string
	Cwd       string
	Focused   bool
	Next      Action
}

// status keyword tests. Fleet's board uses "Ready", "In progress", "In review",
// "Awaiting QA"; we match on the distinguishing substring so casing/wording drift
// doesn't matter. "Ready" deliberately doesn't contain "review".
func statusHas(s, sub string) bool { return strings.Contains(strings.ToLower(s), sub) }

func (w WorkItem) inReview() bool   { return statusHas(w.Status, "review") }
func (w WorkItem) awaitingQA() bool { return statusHas(w.Status, "await") || statusHas(w.Status, "qa") }
func (w WorkItem) hasSession() bool { return w.SessionID != "" }

// nextAction computes the state-machine step. Order matters: the first matching
// condition wins, surfacing the most advanced actionable step.
func (w WorkItem) nextAction() Action {
	if w.PR != nil {
		pr := w.PR.PR
		if pr != nil && pr.CanMergeNow() {
			if w.inReview() {
				return ActMerge
			}
			return ActMarkInReview
		}
		// PR exists but isn't mergeable — the ball is in your court on the PR.
		return ActAddressPR
	}
	// No open PR linked.
	if w.inReview() {
		// The PR left the open list (merged/closed) but status hasn't advanced.
		return ActMarkAwaitingQA
	}
	if w.awaitingQA() {
		return ActNone
	}
	if w.hasSession() {
		return ActOpenPR
	}
	return ActStartWork
}

// BuildWorkItems assembles issue-centric work items from an already-classified
// board. Linkage precedence: (1) jarvis-recorded links.json branch/session,
// (2) a PR's HeadRefName matching a session branch (already attached by
// linkSessions), (3) a PR body closing-keyword reference to the issue.
//
// statuses maps issue number → project Status; projects maps issue number → the
// board id that Status came from. Both may be empty (best-effort enrichment).
func BuildWorkItems(b Board, links *LinkStore, focus *FocusStore, statuses map[int]string, projects map[int]int) []WorkItem {
	// Index PR items by number and by head branch.
	prByNum := map[int]*Item{}
	prByBranch := map[string]*Item{}
	for _, bk := range BucketOrder {
		for i := range b.Buckets[bk] {
			it := &b.Buckets[bk][i]
			if it.Kind != KindPR {
				continue
			}
			prByNum[it.Number] = it
			if it.PR != nil && it.PR.HeadRefName != "" {
				prByBranch[it.PR.HeadRefName] = it
			}
		}
	}

	// Map PR → issue via closing-keyword references (the GitHub fallback link).
	issueFromPR := map[int]int{} // issue number → PR number
	for num, it := range prByNum {
		if it.PR == nil {
			continue
		}
		for _, iss := range it.PR.ClosesIssues() {
			if _, taken := issueFromPR[iss]; !taken {
				issueFromPR[iss] = num
			}
		}
	}

	var out []WorkItem
	for _, bk := range BucketOrder {
		for i := range b.Buckets[bk] {
			it := b.Buckets[bk][i]
			if it.Kind != KindIssue {
				continue
			}
			w := WorkItem{
				Issue:   it.Issue,
				Number:  it.Number,
				Title:   it.Title,
				URL:     it.URL,
				Status:  statuses[it.Number],
				Project: projects[it.Number],
			}
			if focus != nil {
				w.Focused = focus.Has(it.Number)
			}

			// Recorded link is authoritative for branch/clone/session/project.
			if links != nil {
				if l, ok := links.Get(it.Number); ok {
					w.Branch, w.ClonePath, w.SessionID = l.Branch, l.ClonePath, l.SessionID
					if w.Project == 0 {
						w.Project = l.Project
					}
				}
			}

			// Attach the PR: recorded branch first, then closing-keyword fallback.
			var pr *Item
			if w.Branch != "" {
				pr = prByBranch[w.Branch]
			}
			if pr == nil {
				if prNum, ok := issueFromPR[it.Number]; ok {
					pr = prByNum[prNum]
				}
			}
			if pr != nil {
				w.PR = pr
				if w.Branch == "" && pr.PR != nil {
					w.Branch = pr.PR.HeadRefName
				}
				// A session linked onto the PR (by linkSessions) drives this issue.
				if pr.HasSession {
					w.SessionID, w.Cwd = pr.SessionID, pr.Cwd
				}
			}

			w.Next = w.nextAction()
			out = append(out, w)
		}
	}
	return out
}
