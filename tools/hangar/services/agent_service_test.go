package services

import "testing"

func TestAgentServiceSlotUsage(t *testing.T) {
	if got := (&AgentService{}).SlotUsage(); got == nil || len(got) != 0 {
		t.Fatalf("nil Usage: want empty non-nil map, got %v", got)
	}
	by := "/w/x"
	s := &AgentService{Usage: func() map[string]SlotUsage {
		return map[string]SlotUsage{"s2": {ReservedBy: &by, Job: &SlotJob{ID: "j1", Action: "build"}}}
	}}
	got := s.SlotUsage()
	if *got["s2"].ReservedBy != by || got["s2"].Job.Action != "build" {
		t.Fatalf("unexpected %v", got)
	}
	if got := (&AgentService{Usage: func() map[string]SlotUsage { return nil }}).SlotUsage(); got == nil {
		t.Fatal("nil result must become empty map")
	}
}
