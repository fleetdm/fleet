package services

// SlotJob is the agent job currently holding a slot.
type SlotJob struct {
	ID     string `json:"id"`
	Action string `json:"action"`
}

// SlotUsage is what the GUI shows for one slot: the worktree that reserved it
// and the agent job running on it. Both are nil when absent. No secrets.
type SlotUsage struct {
	ReservedBy *string  `json:"reserved_by"`
	Job        *SlotJob `json:"job"`
}

// AgentService exposes slot reservation and agent-job state to the frontend.
type AgentService struct {
	// Usage is wired by main from the agent API; nil means no data.
	Usage func() map[string]SlotUsage
}

// SlotUsage returns usage per slot id, empty when nothing is wired.
func (s *AgentService) SlotUsage() map[string]SlotUsage {
	if s.Usage == nil {
		return map[string]SlotUsage{}
	}
	if m := s.Usage(); m != nil {
		return m
	}
	return map[string]SlotUsage{}
}
