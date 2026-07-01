// Package jarvis implements `gm jarvis` — a personal, read-only work dashboard
// that aggregates the authenticated user's GitHub issues and pull requests into
// leverage-ordered buckets (what's blocking others, what's a quick win, what
// needs your hands) rather than source-ordered lists.
package jarvis

import (
	"time"

	"fleetdm/gm/pkg/ghapi"
)

// Kind distinguishes the underlying work item type.
type Kind int

const (
	KindPR Kind = iota
	KindIssue
	KindSession
)

// Bucket groups items by leverage — who is blocked and how cheaply you can clear it.
type Bucket int

const (
	BucketWaitingOnYou   Bucket = iota // others blocked until you act / feedback bounced back to you
	BucketQuickWins                    // your PRs that can merge right now
	BucketNeedsYourHands               // your move on your own work
	BucketReviewQueue                  // PRs awaiting a first review from you
	BucketCold                         // waiting on others, or stale
	BucketSessions                     // local Claude sessions waiting on your reply
)

// BucketOrder is the display and priority order, highest leverage first.
var BucketOrder = []Bucket{
	BucketWaitingOnYou,
	BucketQuickWins,
	BucketNeedsYourHands,
	BucketSessions,
	BucketReviewQueue,
	BucketCold,
}

type bucketMeta struct {
	Title    string
	Subtitle string
}

var bucketMetas = map[Bucket]bucketMeta{
	BucketWaitingOnYou:   {"WAITING ON YOU", "blocking others"},
	BucketQuickWins:      {"QUICK WINS", "mergeable now"},
	BucketNeedsYourHands: {"NEEDS YOUR HANDS", "your move"},
	BucketReviewQueue:    {"REVIEW QUEUE", "awaiting your review"},
	BucketCold:           {"COLD", "waiting on others / stale"},
	BucketSessions:       {"CLAUDE SESSIONS", "waiting on your reply"},
}

// Title returns the human-readable header for a bucket.
func (b Bucket) Title() string { return bucketMetas[b].Title }

// Subtitle returns the short descriptor shown next to the header.
func (b Bucket) Subtitle() string { return bucketMetas[b].Subtitle }

// Item is one piece of work placed into a bucket, with the reason it landed there.
type Item struct {
	Kind    Kind
	Bucket  Bucket
	Number  int
	Title   string
	URL     string
	Updated time.Time
	Reason  string // why it's in this bucket, shown to the user

	// Underlying source data, kept for detail views and future actions.
	PR    *ghapi.PullRequest
	Issue *ghapi.Issue

	// Claude session linking. For a KindSession item these identify the session;
	// for a PR/issue item, HasSession marks that a waiting session is linked to it.
	SessionID  string
	Cwd        string
	Branch     string
	HasSession bool
}

// Board holds the classified items grouped by bucket.
type Board struct {
	Buckets map[Bucket][]Item
}

// Total returns the number of items across all buckets.
func (b Board) Total() int {
	n := 0
	for _, items := range b.Buckets {
		n += len(items)
	}
	return n
}

// AddItems appends items to their buckets and re-sorts the affected buckets.
// Used for items computed outside BuildBoard (e.g. cherry-pick detection).
func (b *Board) AddItems(items []Item) {
	touched := map[Bucket]bool{}
	for _, it := range items {
		b.Buckets[it.Bucket] = append(b.Buckets[it.Bucket], it)
		touched[it.Bucket] = true
	}
	for bk := range touched {
		sortBucket(bk, b.Buckets[bk])
	}
}

// Flat returns all items in BucketOrder, then each bucket's internal sort order.
// This is the order the TUI navigates and renders.
func (b Board) Flat() []Item {
	var out []Item
	for _, bk := range BucketOrder {
		out = append(out, b.Buckets[bk]...)
	}
	return out
}
