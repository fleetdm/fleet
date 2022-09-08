package fleet

import "encoding/json"

type MDMAppleEnrollmentPayload struct {
	Name      string           `json:"name"`
	DEPConfig *json.RawMessage `json:"dep_config"`
}

type MDMAppleEnrollment struct {
	// TODO(lucas): Add UpdateCreateTimestamps
	ID        uint             `json:"id" db:"id"`
	Name      string           `json:"name" db:"name"`
	DEPConfig *json.RawMessage `json:"dep_config" db:"dep_config"`
}

func (m MDMAppleEnrollment) AuthzType() string {
	return "mdm_apple_enrollment"
}
