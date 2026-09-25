package jarvis

import (
	"strings"
	"testing"

	"fleetdm/gm/pkg/ghapi"
)

func TestTruncateTitle(t *testing.T) {
	short := "a short title"
	if got := truncateTitle(short); got != short {
		t.Errorf("short title changed: %q", got)
	}
	exactly35 := "12345678901234567890123456789012345" // 35 chars
	if got := truncateTitle(exactly35); got != exactly35 {
		t.Errorf("35-char title should be unchanged, got %q (len %d)", got, len([]rune(got)))
	}
	long := "this title is definitely longer than thirty-five characters"
	got := truncateTitle(long)
	if len([]rune(got)) != 35 {
		t.Errorf("expected 35 runes, got %d (%q)", len([]rune(got)), got)
	}
	if got[len(got)-3:] != "..." {
		t.Errorf("expected trailing ..., got %q", got)
	}
}

func TestRenderBar(t *testing.T) {
	bars := func(s string) (fill, empty int) {
		for _, r := range s {
			switch r {
			case '█':
				fill++
			case '░':
				empty++
			}
		}
		return fill, empty
	}
	for _, c := range []struct {
		done, total, wantFill int
	}{
		{0, 10, 0},
		{5, 10, 12},
		{10, 10, 24},
		{3, 6, 12},
		{0, 0, 0},  // no total: empty, not a divide-by-zero
		{7, 5, 24}, // clamped to full
		{-1, 5, 0}, // clamped to empty
	} {
		fill, empty := bars(renderBar(c.done, c.total, 24))
		if fill != c.wantFill || fill+empty != 24 {
			t.Errorf("renderBar(%d, %d, 24): fill=%d empty=%d, want fill=%d width=24",
				c.done, c.total, fill, empty, c.wantFill)
		}
	}
}

func TestRenderLoadingShowsBothBars(t *testing.T) {
	m := Model{state: stateLoading, loadProgress: FetchProgress{
		Phase: 5, Phases: 8, PhaseName: "issue board statuses", Done: 3, Total: 10,
	}}
	out := m.renderLoading()
	if !strings.Contains(out, "step 5/8") || !strings.Contains(out, "issue board statuses") || !strings.Contains(out, "3/10") {
		t.Errorf("loading view missing progress details:\n%s", out)
	}
}

func TestPRStatusLabel(t *testing.T) {
	if prStatusLabel(nil) != "" {
		t.Error("nil PR should be empty")
	}
	cases := map[*ghapi.PullRequest]string{
		{State: "MERGED"}:                           "merged",
		{State: "CLOSED"}:                           "closed",
		{State: "OPEN", IsDraft: true}:              "draft",
		{State: "OPEN", ReviewDecision: "APPROVED"}: "approved",
		{State: "OPEN"}:                             "open",
		// A draft that also happens to be approved still reads as draft (not mergeable yet).
		{State: "OPEN", IsDraft: true, ReviewDecision: "APPROVED"}: "draft",
	}
	for pr, want := range cases {
		if got := prStatusLabel(pr); got != want {
			t.Errorf("prStatusLabel(%+v) = %q, want %q", pr, got, want)
		}
	}

	// Colors: draft/closed are dim; open/approved/merged are green.
	for _, label := range []string{"draft", "closed"} {
		if prStatusStyle(label).GetForeground() != dimStyle.GetForeground() {
			t.Errorf("expected %q to use dim style", label)
		}
	}
	for _, label := range []string{"open", "approved", "merged"} {
		if prStatusStyle(label).GetForeground() != reasonStyle.GetForeground() {
			t.Errorf("expected %q to use green style", label)
		}
	}
}
