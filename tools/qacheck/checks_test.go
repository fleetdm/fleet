package main

import (
	"reflect"
	"testing"
	"time"
)

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

func TestUniqueInts(t *testing.T) {
	t.Parallel()

	in := []int{71, 97, 71, 105, 97}
	want := []int{71, 97, 105}
	got := uniqueInts(in)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueInts() = %#v, want %#v", got, want)
	}
}

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
