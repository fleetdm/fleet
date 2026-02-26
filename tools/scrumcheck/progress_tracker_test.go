package main

import (
	"testing"
	"time"
)

// TestPhaseTrackerStateTransitions validates tracker state updates across phase
// transitions and bridge status events.
func TestPhaseTrackerStateTransitions(t *testing.T) {
	t.Parallel()

	p := &phaseTracker{
		phases: []phaseEntry{
			{name: "one", status: phasePending},
			{name: "two", status: phasePending},
		},
		globalRow: 1,
		phaseRow:  2,
		footerRow: 5,
		logRow:    7,
		logLines:  []string{},
	}

	p.phaseStart(0)
	if p.phases[0].status != phaseRunning {
		t.Fatalf("phaseStart status=%v", p.phases[0].status)
	}

	p.phaseDone(0, "ok")
	if p.phases[0].status != phaseDone || p.phases[0].summary != "ok" {
		t.Fatalf("phaseDone state=%#v", p.phases[0])
	}

	p.phaseWarn(1, "warn")
	if p.phases[1].status != phaseWarn {
		t.Fatalf("phaseWarn status=%v", p.phases[1].status)
	}
	p.phaseFail(1, "fail")
	if p.phases[1].status != phaseFail {
		t.Fatalf("phaseFail status=%v", p.phases[1].status)
	}

	if got := p.completedCount(); got != 2 {
		t.Fatalf("completedCount=%d want=2", got)
	}

	p.waitingForBrowser("/tmp/r")
	if p.statusText == "" {
		t.Fatal("expected waiting status text")
	}
	p.bridgeListening("http://127.0.0.1:9999", 10*time.Minute)
	if p.statusText == "" {
		t.Fatal("expected listening status text")
	}
	p.bridgeSignal("BRIDGE_OP caller=127.0.0.1 op=test stage=done status=ok repo=fleetdm/fleet issue=1 elapsed=10ms")
	if p.bridgeOpsDone == 0 {
		t.Fatal("expected bridge op accounted")
	}
	p.bridgeStopped("done")
}

// TestSmallProgressHelpers verifies small helper functions used by the tracker.
func TestSmallProgressHelpers(t *testing.T) {
	t.Parallel()

	if got := shortDuration(1500 * time.Millisecond); got == "" {
		t.Fatal("shortDuration should not be empty")
	}
	if got := phaseSummaryKV("a", "b"); got != "a | b" {
		t.Fatalf("phaseSummaryKV=%q", got)
	}

	awaiting := map[int][]Item{
		71: {testIssueWithStatus(1, "A", "https://github.com/fleetdm/fleet/issues/1", "✔️Awaiting QA")},
		97: {},
	}
	if got := countAwaitingViolations(awaiting); got != 1 {
		t.Fatalf("countAwaitingViolations=%d", got)
	}

	stale := map[int][]StaleAwaitingViolation{
		71: {{StaleDays: 3}},
		97: {},
	}
	if got := countStaleViolations(stale); got != 1 {
		t.Fatalf("countStaleViolations=%d", got)
	}
}

// TestNewPhaseTrackerSmoke does a minimal constructor/render smoke test.
func TestNewPhaseTrackerSmoke(t *testing.T) {
	p := newPhaseTracker([]string{"phase-a"})
	if p == nil {
		t.Fatal("expected tracker")
	}
	if len(p.phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(p.phases))
	}
	p.showReportLink("file:///tmp/report.html")
}
