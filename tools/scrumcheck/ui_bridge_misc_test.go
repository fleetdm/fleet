package main

import (
	"context"
	"strings"
	"testing"
)

// TestGithubJSONAndSignalBridgeOp verifies GitHub request execution and bridge
// signal accounting behavior.
func TestGithubJSONAndSignalBridgeOp(t *testing.T) {
	t.Parallel()

	var events []string
	b := &uiBridge{
		token:   "tok",
		onEvent: func(s string) { events = append(events, s) },
	}

	if err := b.githubJSON(context.Background(), "GET", "://bad-endpoint", nil, nil); err == nil {
		t.Fatal("expected request build error")
	}
	if err := b.githubJSON(context.Background(), "GET", "http://127.0.0.1:1/unreachable", nil, nil); err == nil {
		t.Fatal("expected request failure for unreachable endpoint")
	}

	b.signalBridgeOp("127.0.0.1:1 (loopback)", "op-x", "done", "ok", "fleetdm/fleet", 9, "100ms")
	if len(events) == 0 || !strings.Contains(events[0], "BRIDGE_OP") || !strings.Contains(events[0], "caller=127.0.0.1:1_(loopback)") {
		t.Fatalf("unexpected signal event: %#v", events)
	}
}

// TestIntListFlagString verifies int list flags render as comma-joined strings.
func TestIntListFlagString(t *testing.T) {
	t.Parallel()

	var f intListFlag = []int{71, 97}
	if got := f.String(); got != "71,97" {
		t.Fatalf("String()=%q want 71,97", got)
	}
}
