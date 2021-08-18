package fleet

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type LabelStore interface {
	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore,
	// creating and updating labels as necessary.
	ApplyLabelSpecs(specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs() ([]*LabelSpec, error)
	// GetLabelSpec returns the spec for the named label.
	GetLabelSpec(name string) (*LabelSpec, error)

	// Label methods
	NewLabel(Label *Label, opts ...OptionalArg) (*Label, error)
	SaveLabel(label *Label) (*Label, error)
	DeleteLabel(name string) error
	Label(lid uint) (*Label, error)
	ListLabels(filter TeamFilter, opt ListOptions) ([]*Label, error)

	// LabelQueriesForHost returns the label queries that should be executed
	// for the given host. The cutoff is the minimum timestamp a query
	// execution should have to be considered "fresh". Executions that are
	// not fresh will be repeated. Results are returned in a map of label
	// id -> query
	LabelQueriesForHost(host *Host, cutoff time.Time) (map[string]string, error)

	// RecordLabelQueryExecutions saves the results of label queries. The
	// results map is a map of label id -> whether or not the label
	// matches. The time parameter is the timestamp to save with the query
	// execution.
	RecordLabelQueryExecutions(host *Host, results map[uint]bool, t time.Time) error

	// LabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(hid uint) ([]*Label, error)

	// ListHostsInLabel returns a slice of hosts in the label with the
	// given ID.
	ListHostsInLabel(filter TeamFilter, lid uint, opt HostListOptions) ([]*Host, error)

	// ListUniqueHostsInLabels returns a slice of all of the hosts in the
	// given label IDs. A host will only appear once in the results even if
	// it is in multiple of the provided labels.
	ListUniqueHostsInLabels(filter TeamFilter, labels []uint) ([]*Host, error)

	SearchLabels(filter TeamFilter, query string, omit ...uint) ([]*Label, error)

	// LabelIDsByName Retrieve the IDs associated with the given labels
	LabelIDsByName(labels []string) ([]uint, error)
}

type LabelService interface {
	// ApplyLabelSpecs applies a list of LabelSpecs to the datastore,
	// creating and updating labels as necessary.
	ApplyLabelSpecs(ctx context.Context, specs []*LabelSpec) error
	// GetLabelSpecs returns all of the stored LabelSpecs.
	GetLabelSpecs(ctx context.Context) ([]*LabelSpec, error)
	// GetLabelSpec gets the spec for the label with the given name.
	GetLabelSpec(ctx context.Context, name string) (*LabelSpec, error)

	NewLabel(ctx context.Context, p LabelPayload) (label *Label, err error)
	ModifyLabel(ctx context.Context, id uint, payload ModifyLabelPayload) (*Label, error)
	ListLabels(ctx context.Context, opt ListOptions) (labels []*Label, err error)
	GetLabel(ctx context.Context, id uint) (label *Label, err error)

	DeleteLabel(ctx context.Context, name string) (err error)
	// DeleteLabelByID is for backwards compatibility with the UI
	DeleteLabelByID(ctx context.Context, id uint) (err error)

	// ListHostsInLabel returns a slice of hosts in the label with the
	// given ID.
	ListHostsInLabel(ctx context.Context, lid uint, opt HostListOptions) ([]*Host, error)

	// LabelsForHost returns the labels that the given host is in.
	ListLabelsForHost(ctx context.Context, hid uint) ([]*Label, error)
}

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
