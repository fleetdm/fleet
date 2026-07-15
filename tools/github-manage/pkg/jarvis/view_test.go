package jarvis

import (
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

func TestPRStatusLabel(t *testing.T) {
	if prStatusLabel(nil) != "" {
		t.Error("nil PR should be empty")
	}
	cases := map[*ghapi.PullRequest]string{
		{State: "MERGED"}: "merged",
		{State: "CLOSED"}: "closed",
		{State: "OPEN", ReviewDecision: "APPROVED"}: "approved",
		{State: "OPEN"}: "open",
	}
	for pr, want := range cases {
		if got := prStatusLabel(pr); got != want {
			t.Errorf("prStatusLabel(%+v) = %q, want %q", pr, got, want)
		}
	}
}
