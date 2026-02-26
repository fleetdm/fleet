package main

import (
	"testing"
	"time"
)

func TestPickCurrentIteration(t *testing.T) {
	t.Parallel()

	iters := []projectIteration{
		{ID: "1", Title: "Sprint A", StartDate: "2026-02-01", Duration: 14},
		{ID: "2", Title: "Sprint B", StartDate: "2026-02-15", Duration: 14},
		{ID: "3", Title: "Sprint C", StartDate: "2026-03-01", Duration: 14},
	}

	t.Run("current span", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)
		got, ok := pickCurrentIteration(now, iters)
		if !ok {
			t.Fatal("expected iteration")
		}
		if got.ID != "2" {
			t.Fatalf("got %q, want %q", got.ID, "2")
		}
	})

	t.Run("after all spans picks latest past", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		got, ok := pickCurrentIteration(now, iters)
		if !ok {
			t.Fatal("expected iteration")
		}
		if got.ID != "3" {
			t.Fatalf("got %q, want %q", got.ID, "3")
		}
	})

	t.Run("before all spans picks earliest", func(t *testing.T) {
		t.Parallel()
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		got, ok := pickCurrentIteration(now, iters)
		if !ok {
			t.Fatal("expected iteration")
		}
		if got.ID != "1" {
			t.Fatalf("got %q, want %q", got.ID, "1")
		}
	})
}

func TestPickCurrentIterationNoValid(t *testing.T) {
	t.Parallel()
	_, ok := pickCurrentIteration(time.Now().UTC(), []projectIteration{
		{ID: "x", Title: "bad", StartDate: "not-a-date", Duration: 14},
		{ID: "y", Title: "bad2", StartDate: "", Duration: 0},
	})
	if ok {
		t.Fatal("expected no valid iteration")
	}
}

func TestSprintColumnGrouping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status string
		group  string
		label  string
	}{
		{status: "🧩 Ready to estimate", group: "ready", label: "Ready"},
		{status: "✅ Ready for release", group: "ready_for_release", label: "Ready for release"},
		{status: "⏳ Waiting", group: "waiting", label: "Waiting"},
		{status: "In progress", group: "in_progress", label: "In progress"},
		{status: "🦃 In review", group: "in_review", label: "In review"},
		{status: "✔️Awaiting QA", group: "awaiting_qa", label: "Awaiting QA"},
		{status: "Backlog", group: "other", label: "Other"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			gotGroup := sprintColumnGroup(tc.status)
			if gotGroup != tc.group {
				t.Fatalf("group=%q want=%q", gotGroup, tc.group)
			}
			gotLabel := sprintColumnLabel(gotGroup)
			if gotLabel != tc.label {
				t.Fatalf("label=%q want=%q", gotLabel, tc.label)
			}
		})
	}
}
