package main

import (
	"strings"
	"testing"
	"time"
)

// TestParseBridgeOpSignal verifies structured bridge operation messages parse.
func TestParseBridgeOpSignal(t *testing.T) {
	t.Parallel()

	msg := "BRIDGE_OP caller=127.0.0.1:1234_(loopback) op=apply-milestone stage=done status=ok repo=fleetdm/fleet issue=42 elapsed=120ms"
	evt, ok := parseBridgeOpSignal(msg)
	if !ok {
		t.Fatal("expected parse success")
	}
	if evt.Caller == "" || evt.Op != "apply-milestone" || evt.Stage != "done" || evt.Status != "ok" || evt.Repo != "fleetdm/fleet" || evt.Issue != "42" {
		t.Fatalf("unexpected parsed event: %#v", evt)
	}
	if evt.Elapsed != 120*time.Millisecond {
		t.Fatalf("elapsed=%v want=120ms", evt.Elapsed)
	}

	if _, ok := parseBridgeOpSignal("hello world"); ok {
		t.Fatal("expected non bridge-op message to fail parse")
	}
}

// TestBridgeOpSummary verifies summary text includes key operation fields.
func TestBridgeOpSummary(t *testing.T) {
	t.Parallel()

	evt := bridgeOpEvent{
		Caller:  "127.0.0.1",
		Op:      "add-assignee",
		Stage:   "done",
		Status:  "ok",
		Repo:    "fleetdm/fleet",
		Issue:   "100",
		Elapsed: 250 * time.Millisecond,
	}
	s := evt.summary()
	for _, want := range []string{"add-assignee", "done", "fleetdm/fleet#100", "ok", "127.0.0.1"} {
		if !strings.Contains(s, want) {
			t.Fatalf("summary %q missing %q", s, want)
		}
	}
}

// TestRenderBridgeOpsLine verifies active and idle bridge summary rendering.
func TestRenderBridgeOpsLine(t *testing.T) {
	t.Parallel()

	p := &phaseTracker{
		bridgeOpsStarted: 4,
		bridgeOpsDone:    3,
		bridgeOpsOK:      2,
		bridgeOpsErr:     1,
		bridgeOpsTotal:   900 * time.Millisecond,
	}
	line := p.renderBridgeOpsLine()
	for _, want := range []string{"bridge ops", "3/4", "ok=2", "err=1"} {
		if !strings.Contains(line, want) {
			t.Fatalf("line %q missing %q", line, want)
		}
	}

	p = &phaseTracker{}
	idle := p.renderBridgeOpsLine()
	if !strings.Contains(idle, "bridge ops idle") {
		t.Fatalf("idle line %q missing idle text", idle)
	}
}

// TestBarsAndStageColor checks stage color thresholds and bar formatting basics.
func TestBarsAndStageColor(t *testing.T) {
	t.Parallel()

	if got := stageColor(0.1); got != clrRed {
		t.Fatalf("stageColor(0.1)=%q want red", got)
	}
	if got := stageColor(0.5); got != clrYellow {
		t.Fatalf("stageColor(0.5)=%q want yellow", got)
	}
	if got := stageColor(0.9); got != clrGreen {
		t.Fatalf("stageColor(0.9)=%q want green", got)
	}

	for _, bar := range []string{
		pendingPhaseBar(10),
		donePhaseBar(10),
		runningPhaseBar(10, clrYellow),
		coloredBar(10, 3, 5, clrGreen),
	} {
		if !strings.Contains(bar, "[") || !strings.Contains(bar, "]") {
			t.Fatalf("invalid bar output: %q", bar)
		}
	}
}
