// Package autopilot reads the Windows Autopilot identity a device cached when it downloaded its Autopilot profile.
package autopilot

import (
	"encoding/json"
	"fmt"
	"uuid"
)

// parseRegistrationID extracts the Autopilot device ID (ZTDID) from the policy cache JSON. The field is absent when the device
// has no assigned Autopilot profile, which is not an error.
func parseRegistrationID(policyJSON string) (string, error) {
	var policy struct {
		ZtdRegistrationID string `json:"ZtdRegistrationId"`
	}
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		return "", fmt.Errorf("parse autopilot policy cache: %w", err)
	}
	if policy.ZtdRegistrationID == "" {
		return "", nil
	}
	id, err := uuid.Parse(policy.ZtdRegistrationID)
	if err != nil {
		return "", fmt.Errorf("parse autopilot registration id: %w", err)
	}
	return id.String(), nil
}
