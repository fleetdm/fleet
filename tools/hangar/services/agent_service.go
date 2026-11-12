package services

import "github.com/fleetdm/fleet/tools/hangar/internal/agentapi"

// AgentService exposes slot reservation and agent-job state to the frontend.
type AgentService struct {
	// Usage is wired by main from the agent API; nil means no data.
	Usage func() map[string]agentapi.SlotUsage
}

// SlotUsage returns usage per slot id, empty when nothing is wired.
func (s *AgentService) SlotUsage() map[string]agentapi.SlotUsage {
	if s.Usage == nil {
		return map[string]agentapi.SlotUsage{}
	}
	if m := s.Usage(); m != nil {
		return m
	}
	return map[string]agentapi.SlotUsage{}
}
