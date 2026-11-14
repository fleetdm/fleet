package services

import (
	"testing"

	"github.com/fleetdm/fleet/tools/hangar/internal/agentapi"
)

func TestAgentServiceSlotUsage(t *testing.T) {
	if got := (&AgentService{}).SlotUsage(); got == nil || len(got) != 0 {
		t.Fatalf("nil Usage: want empty non-nil map, got %v", got)
	}
	by := "/w/x"
	s := &AgentService{Usage: func() map[string]agentapi.SlotUsage {
		return map[string]agentapi.SlotUsage{"s2": {ReservedBy: &by, Job: &agentapi.SlotJob{ID: "j1", Action: "build"}}}
	}}
	got := s.SlotUsage()
	if *got["s2"].ReservedBy != by || got["s2"].Job.Action != "build" {
		t.Fatalf("unexpected %v", got)
	}
	if got := (&AgentService{Usage: func() map[string]agentapi.SlotUsage { return nil }}).SlotUsage(); got == nil {
		t.Fatal("nil result must become empty map")
	}
}
