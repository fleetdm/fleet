package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

// TestHasUncheckedChecklistLine verifies unchecked checklist detection only
// passes for unchecked variants and not checked/missing lines.
func TestHasUncheckedChecklistLine(t *testing.T) {
	t.Parallel()

	text := "Engineer: Added comment to user story confirming successful completion of test plan."

	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "unchecked markdown list",
			body: "- [ ] " + text,
			want: true,
		},
		{
			name: "unchecked plain checklist",
			body: "[ ] " + text,
			want: true,
		},
		{
			name: "checked markdown list",
			body: "- [x] " + text,
			want: false,
		},
		{
			name: "checked plain checklist uppercase",
			body: "[X] " + text,
			want: false,
		},
		{
			name: "missing line",
			body: "- [ ] something else",
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasUncheckedChecklistLine(tc.body, text)
			if got != tc.want {
				t.Fatalf("hasUncheckedChecklistLine() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestUncheckedChecklistItems verifies extraction of unchecked items and
// confirms ignore-prefix checklist entries are excluded.
func TestUncheckedChecklistItems(t *testing.T) {
	t.Parallel()

	body := `
- [ ] keep this one
* [ ] keep this one too
[ ] keep this one three
- [x] checked should not appear
- [ ] Once shipped, requester has been notified to customer
- [ ] Review of windows_mdm_profiles.go and compare changes
`

	got := uncheckedChecklistItems(body)
	want := []string{
		"keep this one",
		"keep this one too",
		"keep this one three",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uncheckedChecklistItems() = %#v, want %#v", got, want)
	}
}

func TestUncheckedChecklistItems_ReproChecklistOrGate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "both unchecked remains violation",
			body: `
- [ ] Have been confirmed to consistently lead to reproduction in multiple Fleet instances.
- [ ] Describe the workflow that led to the error, but have not yet been reproduced in multiple Fleet instances.
`,
			want: []string{
				"Have been confirmed to consistently lead to reproduction in multiple Fleet instances.",
				"Describe the workflow that led to the error, but have not yet been reproduced in multiple Fleet instances.",
			},
		},
		{
			name: "confirmed checked suppresses workflow unchecked",
			body: `
- [x] Have been confirmed to consistently lead to reproduction in multiple Fleet instances.
- [ ] Describe the workflow that led to the error, but have not yet been reproduced in multiple Fleet instances.
`,
			want: []string{},
		},
		{
			name: "workflow checked suppresses confirmed unchecked",
			body: `
- [ ] Have been confirmed to consistently lead to reproduction in multiple Fleet instances.
- [X] Describe the workflow that led to the error, but have not yet been reproduced in multiple Fleet instances.
`,
			want: []string{},
		},
		{
			name: "either checked still keeps other unchecked items",
			body: `
- [x] Have been confirmed to consistently lead to reproduction in multiple Fleet instances.
- [ ] Describe the workflow that led to the error, but have not yet been reproduced in multiple Fleet instances.
- [ ] another actionable task
`,
			want: []string{"another actionable task"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := uncheckedChecklistItems(tc.body)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("uncheckedChecklistItems() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestNormalizeStatusName verifies status normalization removes emoji/symbol
// prefixes and normalizes casing/whitespace.
func TestNormalizeStatusName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "emoji prefix",
			in:   "✔️ Awaiting QA",
			want: "awaiting qa",
		},
		{
			name: "spaces and casing",
			in:   "   EsTiMaTeD   ",
			want: "estimated",
		},
		{
			name: "symbol prefix",
			in:   "🧩 Ready to estimate",
			want: "ready to estimate",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeStatusName(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeStatusName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestUniqueInts verifies duplicate project numbers are removed while keeping
// first occurrence order.
func TestUniqueInts(t *testing.T) {
	t.Parallel()

	in := []int{71, 97, 71, 105, 97}
	want := []int{71, 97, 105}
	got := uniqueInts(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueInts() = %#v, want %#v", got, want)
	}
}

// TestAwaitingAndDoneAndMatchedStatusAndStale validates helper behavior for
// Awaiting QA detection, Done detection, status matching, and stale logic.
func TestAwaitingAndDoneAndMatchedStatusAndStale(t *testing.T) {
	t.Parallel()

	awaiting := testIssueWithStatus(1, "A", "https://github.com/fleetdm/fleet/issues/1", "✔️Awaiting QA")
	if !inAwaitingQA(awaiting) {
		t.Fatal("expected awaiting QA")
	}
	if inDoneColumn(awaiting) {
		t.Fatal("did not expect done")
	}

	done := testIssueWithStatus(2, "B", "https://github.com/fleetdm/fleet/issues/2", "✅ Done")
	if !inDoneColumn(done) {
		t.Fatal("expected done")
	}

	inProgress := testIssueWithStatus(3, "C", "https://github.com/fleetdm/fleet/issues/3", "In progress")
	if got, ok := matchedStatus(inProgress, []string{"ready", "progress"}); !ok || got != "progress" {
		t.Fatalf("matchedStatus got=(%q,%v), want (progress,true)", got, ok)
	}

	var stale Item
	stale.UpdatedAt.Time = time.Now().UTC().Add(-48 * time.Hour)
	if !isStaleAwaitingQA(stale, time.Now().UTC(), 24*time.Hour) {
		t.Fatal("expected stale item")
	}
}

// TestCompileAndMatchLabelFilter verifies label filter normalization and item
// matching behavior, including nil-filter "match all" semantics.
func TestCompileAndMatchLabelFilter(t *testing.T) {
	t.Parallel()

	filter := compileLabelFilter([]string{"#g-orchestration", " g-security-compliance ", "", "#G-Orchestration"})
	if len(filter) != 2 {
		t.Fatalf("expected 2 unique normalized labels, got %d", len(filter))
	}

	it := testIssueWithStatus(10, "Labeled", "https://github.com/fleetdm/fleet/issues/10", "In review")
	it.Content.Issue.Labels.Nodes = []struct {
		Name githubv4.String
	}{
		{Name: githubv4.String("g-security-compliance")},
	}

	if !matchesLabelFilter(it, filter) {
		t.Fatal("expected issue to match label filter")
	}

	other := testIssueWithStatus(11, "Other", "https://github.com/fleetdm/fleet/issues/11", "In review")
	other.Content.Issue.Labels.Nodes = []struct {
		Name githubv4.String
	}{
		{Name: githubv4.String("bug")},
	}
	if matchesLabelFilter(other, filter) {
		t.Fatal("did not expect issue to match label filter")
	}

	if !matchesLabelFilter(other, nil) {
		t.Fatal("expected nil filter to match all issues")
	}
}
