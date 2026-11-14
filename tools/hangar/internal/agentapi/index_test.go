package agentapi

import (
	"net/http"
	"testing"
)

func TestIndexListsEveryAcceptedAction(t *testing.T) {
	e := newEnv(t)
	code, out := e.do("GET", "/v1", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /v1: code %d", code)
	}
	actions, _ := out["actions"].([]any)
	if len(actions) != len(apiIndex.Actions) {
		t.Fatalf("index has %d actions, want %d", len(actions), len(apiIndex.Actions))
	}
	// s1 is always refused, so each listed action reaches a guard without
	// starting anything; only an unlisted action gets "unknown action".
	for _, a := range apiIndex.Actions {
		body := map[string]any{"action": a.Name}
		for _, f := range a.Body {
			body[f] = map[string]any{"worktree": e.task, "target": e.task, "name": "index-test"}[f]
		}
		code, out := e.do("POST", "/v1/slots/s1/actions", body)
		if code == http.StatusBadRequest {
			t.Errorf("listed action %q is rejected as bad input: %v", a.Name, out)
		}
	}
	if code, _ := e.do("POST", "/v1/slots/s1/actions", map[string]any{"action": "nope", "worktree": e.task}); code != http.StatusBadRequest {
		t.Errorf("unknown action: code %d, want 400", code)
	}
}
