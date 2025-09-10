package fleet

import (
	"encoding/json"
	"errors"
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
	Hosts   []string `json:"hosts"`
	HostIDs []uint   `json:"host_ids"`
}

type HostVitalOperator string

const (
	HostVitalOperatorEqual    HostVitalOperator = "="
	HostVitalOperatorNotEqual HostVitalOperator = "!="
	HostVitalOperatorGreater  HostVitalOperator = ">"
	HostVitalOperatorLess     HostVitalOperator = "<"
	HostVitalOperatorLike     HostVitalOperator = "LIKE"
)

type HostVitalCriteria struct {
	Vital    *string             `json:"vital,omitempty"`
	Value    *string             `json:"value,omitempty"`
	Operator *HostVitalOperator  `json:"operator,omitempty"`
	And      []HostVitalCriteria `json:"and,omitempty"`
	Or       []HostVitalCriteria `json:"or,omitempty"`
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
	Hosts   []string `json:"hosts"`
	HostIDs []uint   `json:"host_ids"`
	// Criteria is the set of criteria that defines a host vitals label.
	Criteria *HostVitalCriteria `json:"criteria,omitempty"`
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
	// LabelMembershipTypeHostVitals indicates that the label is populated
	// dynamically based on host vitals data.
	LabelMembershipTypeHostVitals
)

func (t LabelMembershipType) MarshalJSON() ([]byte, error) {
	switch t {
	case LabelMembershipTypeDynamic:
		return []byte(`"dynamic"`), nil
	case LabelMembershipTypeManual:
		return []byte(`"manual"`), nil
	case LabelMembershipTypeHostVitals:
		return []byte(`"host_vitals"`), nil
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
	case `"host_vitals"`:
		*t = LabelMembershipTypeHostVitals
	default:
		return fmt.Errorf("invalid LabelMembershipType: %s", string(b))
	}
	return nil
}

// Create a separate interface for host vitals labels to allow for
// different query generation logic in tests.
type HostVitalsLabel interface {
	CalculateHostVitalsQuery() (query string, values []any, err error)
	GetLabel() *Label
}

type Label struct {
	UpdateCreateTimestamps
	ID                  uint                `json:"id"`
	AuthorID            *uint               `json:"author_id" db:"author_id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	Query               string              `json:"query"`
	HostVitalsCriteria  *json.RawMessage    `json:"criteria,omitempty" db:"criteria"`
	Platform            string              `json:"platform"`
	LabelType           LabelType           `json:"label_type" db:"label_type"`
	LabelMembershipType LabelMembershipType `json:"label_membership_type" db:"label_membership_type"`
	HostCount           int                 `json:"host_count,omitempty" db:"host_count"`
}

// Implement the HostVitalsLabel interface.
func (l *Label) GetLabel() *Label {
	return l
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
	HostVitalsCriteria  *json.RawMessage    `json:"criteria,omitempty" db:"criteria"`
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
	BuiltinLabelNameAndroid     = "Android"
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
		BuiltinLabelNameAndroid:     {},
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
	LabelID   uint   `json:"id"`
	LabelName string `json:"name"`
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

type LabelIdentsWithScope struct {
	LabelScope LabelScope
	ByName     map[string]LabelIdent
}

// Equal returns whether or not 2 LabelIdentsWithScope pointers point to equivalent values.
func (l *LabelIdentsWithScope) Equal(other *LabelIdentsWithScope) bool {
	if l == nil || other == nil {
		return l == other
	}

	if l.LabelScope != other.LabelScope {
		return false
	}

	if l.ByName == nil && other.ByName == nil {
		return true
	}

	if len(l.ByName) != len(other.ByName) {
		return false
	}

	for k, v := range l.ByName {
		otherV, ok := other.ByName[k]
		if !ok {
			return false
		}

		if v != otherV {
			return false
		}
	}

	return true
}

// Translate label host vitals crteria into a query.
// TODO -- add caching support for this query?
func (l *Label) CalculateHostVitalsQuery() (query string, values []any, err error) {
	var criteria *HostVitalCriteria
	if l.HostVitalsCriteria == nil {
		return "", nil, errors.New("label has no host vitals criteria")
	}
	// Unmarshal the criteria from JSON.
	if err := json.Unmarshal(*l.HostVitalsCriteria, &criteria); err != nil {
		return "", nil, fmt.Errorf("unmarshalling host vitals criteria: %w", err)
	}

	// We'll use a set to gather the foreign vitals groups we need to join on,
	// so that we can avoid duplicates.
	foreignVitalsGroups := make(map[*HostForeignVitalGroup]struct{})
	// Hold values to be substituted in the paramerized query.
	values = make([]any, 0)
	// Recursively parse the criteria to build the WHERE clause.
	whereClause, err := parseHostVitalCriteria(criteria, foreignVitalsGroups, &values)
	if err != nil {
		return "", nil, fmt.Errorf("parsing host vitals criteria: %w", err)
	}
	// If there are foreign vitals groups, concatenate all their joins.
	joins := make([]string, 0, len(foreignVitalsGroups))
	if len(foreignVitalsGroups) > 0 {
		for group := range foreignVitalsGroups {
			joins = append(joins, group.Query)
		}
	}

	// Leave SELECT and FROM to be filled in later for flexibility.
	query = "SELECT %s FROM %s " + strings.Join(joins, " ") + " WHERE " + whereClause + " GROUP BY hosts.id"
	return
}

// Translates a HostVitalCriteria into part of a SQL WHERE clause
// TODO: add support for And/Or criteria
func parseHostVitalCriteria(criteria *HostVitalCriteria, foreignVitalsGroups map[*HostForeignVitalGroup]struct{}, values *[]any) (string, error) {
	// We don't support anything other than vital/value right now.
	if criteria.And != nil || criteria.Or != nil {
		return "", errors.New("And/Or criteria not supported in host vitals labels yet")
	}
	if criteria.Vital == nil {
		return "", errors.New("vital criteria must have a vital")
	}
	if criteria.Value == nil {
		return "", fmt.Errorf("vital %s must have a value", *criteria.Vital)
	}
	// Look up the vital in the map.
	vital, ok := hostVitals[*criteria.Vital]
	if !ok {
		return "", fmt.Errorf("unknown vital %s", *criteria.Vital)
	}
	// If the vital is a foreign vitals group, add it to the list of foreign vitals groups.
	if vital.VitalType == HostVitalTypeForeign {
		foreignVitalsGroup, ok := hostForeignVitalGroups[*vital.ForeignVitalGroup]
		if !ok {
			return "", fmt.Errorf("unknown foreign vital group %s", *vital.ForeignVitalGroup)
		}
		foreignVitalsGroups[&foreignVitalsGroup] = struct{}{}
	}
	*values = append(*values, *criteria.Value)

	operator := criteria.Operator
	if operator == nil {
		// Default to equality if no operator is specified.
		op := HostVitalOperatorEqual
		operator = &op
	}
	// TODO - handle different vital data types and operator types.
	// For now, we only support equality checks.
	if *operator != HostVitalOperatorEqual {
		return "", fmt.Errorf("operator %s not supported for vital %s", *operator, *criteria.Vital)
	}
	return fmt.Sprintf("%s = ?", vital.Path), nil
}
