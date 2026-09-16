package ai_tools

import (
	"context"
	"os"
	"testing"
)

// TestSmokeLiveHost runs the unified table against the real host: first
// unconstrained (all types), then with a type='ide_plugins' constraint to prove
// pushdown returns only that type. Opt-in (AED_SMOKE=1) — reads live state.
//
//	AED_SMOKE=1 go test -run TestSmokeLiveHost -v ./orbit/pkg/table/ai_tools/
func TestSmokeLiveHost(t *testing.T) {
	if os.Getenv("AED_SMOKE") != "1" {
		t.Skip("set AED_SMOKE=1 to run the live-host smoke test")
	}
	p := All()[0]
	ctx := context.Background()

	// 1. Unconstrained: every type.
	resp := p.Call(ctx, map[string]string{"action": "generate", "context": "{}"})
	if resp.Status != nil && resp.Status.Code != 0 {
		t.Fatalf("generate failed: %s", resp.Status.Message)
	}
	counts := map[string]int{}
	for _, r := range resp.Response {
		counts[r["type"]]++
	}
	t.Logf("ai_tools: %d rows total %v", len(resp.Response), counts)
	for i, r := range resp.Response {
		if i >= 4 {
			break
		}
		t.Logf("  [%d] type=%s name=%q category=%q location=%s", i, r["type"], r["name"], r["category"], r["location"])
	}

	// 2. Constraint pushdown: type = 'ide_plugins' (op 2 = EQUALS).
	pruned := p.Call(ctx, map[string]string{
		"action":  "generate",
		"context": `{"constraints":[{"name":"type","affinity":"TEXT","list":[{"op":2,"expr":"ide_plugins"}]}]}`,
	})
	if pruned.Status != nil && pruned.Status.Code != 0 {
		t.Fatalf("constrained generate failed: %s", pruned.Status.Message)
	}
	for _, r := range pruned.Response {
		if r["type"] != "ide_plugins" {
			t.Fatalf("pushdown leaked a non-ide_plugins row: type=%s", r["type"])
		}
	}
	t.Logf("type='ide_plugins' pushdown: %d rows (all ide_plugins)", len(pruned.Response))
}
