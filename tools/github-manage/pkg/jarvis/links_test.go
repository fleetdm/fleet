package jarvis

import (
	"path/filepath"
	"testing"
)

// TestSetAndSaveMergesConcurrentInstances simulates two jarvis instances sharing
// one links.json: each must be able to record its own started branch without
// dropping the other's entry (the lost-update the whole-map Save would otherwise
// cause), and a reload must surface the sibling's link.
func TestSetAndSaveMergesConcurrentInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")

	// Instance A and B both load the (empty) store.
	a, err := LoadLinkStore(path)
	if err != nil {
		t.Fatalf("load A: %v", err)
	}
	b, err := LoadLinkStore(path)
	if err != nil {
		t.Fatalf("load B: %v", err)
	}

	// A starts work on #100; B starts work on #200 — interleaved.
	if err := a.SetAndSave(100, Link{Branch: "a-100", ClonePath: "/tmp/fleet-a"}); err != nil {
		t.Fatalf("A SetAndSave: %v", err)
	}
	if err := b.SetAndSave(200, Link{Branch: "b-200", ClonePath: "/tmp/fleet-b"}); err != nil {
		t.Fatalf("B SetAndSave: %v", err)
	}

	// Disk must hold BOTH links, not just the last writer's.
	disk, err := LoadLinkStore(path)
	if err != nil {
		t.Fatalf("reload disk: %v", err)
	}
	if l, ok := disk.Get(100); !ok || l.Branch != "a-100" {
		t.Errorf("expected #100 -> a-100 on disk, got %+v (ok=%v)", l, ok)
	}
	if l, ok := disk.Get(200); !ok || l.Branch != "b-200" {
		t.Errorf("expected #200 -> b-200 on disk, got %+v (ok=%v)", l, ok)
	}

	// A had never seen #200 in memory; a reload surfaces the sibling's started work.
	if _, ok := a.Get(200); ok {
		t.Fatalf("precondition: A should not know #200 before reload")
	}
	if err := a.Reload(); err != nil {
		t.Fatalf("A reload: %v", err)
	}
	if l, ok := a.Get(200); !ok || l.Branch != "b-200" {
		t.Errorf("after reload, A should see #200 -> b-200, got %+v (ok=%v)", l, ok)
	}
}
