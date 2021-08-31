package fleet

import (
	"time"

	"github.com/pkg/errors"
)

// ModifyLabelPayload is used to change editable fields for a Label
type ModifyLabelPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type LabelPayload struct {
	Name        *string `json:"name"`
	Query       *string `json:"query"`
	Platform    *string `json:"platform"`
	Description *string `json:"description"`
}

// LabelType is used to catagorize the kind of label
type LabelType uint

const (
	// LabelTypeRegular is for user created labels that can be modified.
	LabelTypeRegular LabelType = iota
	// LabelTypeBuiltIn is for labels built into Fleet that cannot be
	// modified by users.
	LabelTypeBuiltIn
)

func (t LabelType) MarshalJSON() ([]byte, error) {
	switch t {
	case LabelTypeRegular:
		return []byte(`"regular"`), nil
	case LabelTypeBuiltIn:
		return []byte(`"builtin"`), nil
	default:
		return nil, errors.Errorf("invalid LabelType: %d", t)
	}
}

func (t *LabelType) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"regular"`, "0":
		*t = LabelTypeRegular
	case `"builtin"`, "1":
		*t = LabelTypeBuiltIn
	default:
		return errors.Errorf("invalid LabelType: %s", string(b))
	}
	return nil
}

// LabelMembershipType sets how the membership of the label is determined.
type LabelMembershipType uint

const (
	// LabelTypeDynamic indicates that the label is populated dynamically (by
	// the execution of a label query).
	LabelMembershipTypeDynamic LabelMembershipType = iota
	// LabelTypeManual indicates that the label is populated manually.
	LabelMembershipTypeManual
)

func (t LabelMembershipType) MarshalJSON() ([]byte, error) {
	switch t {
	case LabelMembershipTypeDynamic:
		return []byte(`"dynamic"`), nil
	case LabelMembershipTypeManual:
		return []byte(`"manual"`), nil
	default:
		return nil, errors.Errorf("invalid LabelMembershipType: %d", t)
	}
}

func (t *LabelMembershipType) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"dynamic"`:
		*t = LabelMembershipTypeDynamic
	case `"manual"`:
		*t = LabelMembershipTypeManual
	default:
		return errors.Errorf("invalid LabelMembershipType: %s", string(b))
	}
	return nil
}

type Label struct {
	UpdateCreateTimestamps
	ID                  uint                `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Query               string              `json:"query"`
	Platform            string              `json:"platform"`
	LabelType           LabelType           `json:"label_type" db:"label_type"`
	LabelMembershipType LabelMembershipType `json:"label_membership_type" db:"label_membership_type"`
	HostCount           int                 `json:"host_count,omitempty" db:"host_count"`
}

func (l Label) AuthzType() string {
	return "label"
}

const (
	LabelKind = "label"
)

type LabelQueryExecution struct {
	ID        uint
	UpdatedAt time.Time
	Matches   bool
	LabelID   uint
	HostID    uint
}

type LabelSpec struct {
	ID                  uint                `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Query               string              `json:"query"`
	Platform            string              `json:"platform,omitempty"`
	LabelType           LabelType           `json:"label_type,omitempty" db:"label_type"`
	LabelMembershipType LabelMembershipType `json:"label_membership_type" db:"label_membership_type"`
	Hosts               []string            `json:"hosts,omitempty"`
}
