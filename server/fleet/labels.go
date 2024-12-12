package fleet

import (
	"fmt"
	"strings"
	"time"
)

// ModifyLabelPayload is used to change editable fields for a Label
type ModifyLabelPayload struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	// Hosts is the new list of host identifiers to apply for this label, only
	// valid for manual labels. If it is nil (not just len() == 0, but == nil),
	// then the list of hosts is not modified. If it is not nil and len == 0,
	// then all members are removed.
	Hosts []string `json:"hosts"`
}

type LabelPayload struct {
	Name string `json:"name"`
	// Query is the SQL query that defines the label. This defines a dynamic
	// label, where the hosts that are part of the label are determined based on
	// the query result. Must be empty for a manual label, must be provided for a
	// dynamic one.
	Query       string `json:"query"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
	// Hosts is the list of host identifier (serial, uuid, name, etc. as
	// supported by HostByIdentifier) that are part of the label. This defines a
	// manual label. Can be empty for a manual label that doesn't target any
	// host. Must be empty for a dynamic label.
	Hosts []string `json:"hosts"`
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
		return nil, fmt.Errorf("invalid LabelType: %d", t)
	}
}

func (t *LabelType) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"regular"`, "0":
		*t = LabelTypeRegular
	case `"builtin"`, "1":
		*t = LabelTypeBuiltIn
	default:
		return fmt.Errorf("invalid LabelType: %s", string(b))
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
		return nil, fmt.Errorf("invalid LabelMembershipType: %d", t)
	}
}

func (t *LabelMembershipType) UnmarshalJSON(b []byte) error {
	switch string(b) {
	case `"dynamic"`:
		*t = LabelMembershipTypeDynamic
	case `"manual"`:
		*t = LabelMembershipTypeManual
	default:
		return fmt.Errorf("invalid LabelMembershipType: %s", string(b))
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

type LabelSummary struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	LabelType   LabelType `json:"label_type" db:"label_type"`
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
	Hosts               []string            `json:"hosts"`
}

const (
	BuiltinLabelNameAllHosts    = "All Hosts"
	BuiltinLabelNameMacOS       = "macOS"
	BuiltinLabelNameUbuntuLinux = "Ubuntu Linux"
	BuiltinLabelNameCentOSLinux = "CentOS Linux"
	BuiltinLabelNameWindows     = "MS Windows"
	BuiltinLabelNameRedHatLinux = "Red Hat Linux"
	BuiltinLabelNameAllLinux    = "All Linux"
	BuiltinLabelNameChrome      = "chrome"
	BuiltinLabelMacOS14Plus     = "macOS 14+ (Sonoma+)"
	BuiltinLabelIOS             = "iOS"
	BuiltinLabelIPadOS          = "iPadOS"
	BuiltinLabelFedoraLinux     = "Fedora Linux"
)

// ReservedLabelNames returns a map of label name strings
// that are reserved by Fleet.
func ReservedLabelNames() map[string]struct{} {
	return map[string]struct{}{
		BuiltinLabelNameAllHosts:    {},
		BuiltinLabelNameMacOS:       {},
		BuiltinLabelNameUbuntuLinux: {},
		BuiltinLabelNameCentOSLinux: {},
		BuiltinLabelNameWindows:     {},
		BuiltinLabelNameRedHatLinux: {},
		BuiltinLabelNameAllLinux:    {},
		BuiltinLabelNameChrome:      {},
		BuiltinLabelMacOS14Plus:     {},
		BuiltinLabelIOS:             {},
		BuiltinLabelIPadOS:          {},
		BuiltinLabelFedoraLinux:     {},
	}
}

// DetectMissingLabels returns a list of labels present in the unvalidatedLabels list that could not be found in the validLabelMap.
func DetectMissingLabels(validLabelMap map[string]uint, unvalidatedLabels []string) []string {
	missingLabels := make([]string, 0, len(unvalidatedLabels))

	for _, rawLabel := range unvalidatedLabels {
		label := strings.TrimSpace(rawLabel)
		if _, ok := validLabelMap[label]; len(label) > 0 && !ok {
			missingLabels = append(missingLabels, label)
		}
	}

	return missingLabels
}

// LabelIdent is a simple struct to hold the ID and Name of a label
type LabelIdent struct {
	LabelID   uint
	LabelName string
}

// LabelScope identifies the manner by which labels may be used to scope entities, such as MDM
// profiles and software installers, to subsets of hosts.
type LabelScope string

const (
	// LabelScopeExcludeAny indicates that a label-scoped entity (e.g., MDM profiles, software
	// installers) should NOT be applied to a host if the host is a mamber of any of the associated labels.
	LabelScopeExcludeAny LabelScope = "exclude_any"
	// LabelScopeIncludeAny indicates that a label-scoped entity (e.g., MDM profiles, software
	// installers) should be applied to a host that if the host is a member of all of the associated labels.
	LabelScopeIncludeAny LabelScope = "include_any"
	// LabelScopeIncludeAll indicates that a label-scoped entity (e.g., MDM profiles, software
	// installers) should be applied to a host if the host is a member of all of the associated labels.
	LabelScopeIncludeAll LabelScope = "include_all"
)

type LabelIndentsWithScope struct {
	LabelScope LabelScope
	ByName     map[string]LabelIdent
}
