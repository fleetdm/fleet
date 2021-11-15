package fleet

import (
	"errors"
)

// PolicyPayload holds data for policy creation.
type PolicyPayload struct {
	// QueryID allows creating a policy from an existing query.
	//
	// Using QueryID is the old way of creating policies.
	// Use Query, Name and Description instead.
	QueryID uint
	// Name is the name of the policy (ignored if QueryID != 0).
	Name string
	// Query is the policy query (ignored if QueryID != 0).
	Query string
	// Description is the policy description text (ignored if QueryID != 0).
	Description string
	// Resolution indicate the steps needed to solve a failing policy.
	Resolution string
}

var (
	errPolicyEmptyName     = errors.New("policy name cannot be empty")
	errPolicyEmptyQuery    = errors.New("policy query cannot be empty")
	errPolicyIDAndQuerySet = errors.New("both fields \"queryID\" and \"query\" cannot be set")
	errPolicyInvalidQuery  = errors.New("invalid policy query")
)

// Verify verifies the policy payload is valid.
func (p PolicyPayload) Verify() error {
	if p.QueryID != 0 {
		if p.Query != "" {
			return errPolicyIDAndQuerySet
		}
	} else {
		if err := verifyPolicyName(p.Name); err != nil {
			return err
		}
		if err := verifyPolicyQuery(p.Query); err != nil {
			return err
		}
	}
	return nil
}

func verifyPolicyName(name string) error {
	if name == "" {
		return errPolicyEmptyName
	}
	return nil
}

func verifyPolicyQuery(query string) error {
	if query == "" {
		return errPolicyEmptyQuery
	}
	if validateSQLRegexp.MatchString(query) {
		return errPolicyInvalidQuery
	}
	return nil
}

// ModifyPolicyPayload holds data for policy modification.
type ModifyPolicyPayload struct {
	// Name is the name of the policy.
	Name *string `json:"name"`
	// Query is the policy query.
	Query *string `json:"query"`
	// Description is the policy description text.
	Description *string `json:"description"`
	// Resolution indicate the steps needed to solve a failing policy.
	Resolution *string `json:"resolution"`
}

// Verify verifies the policy payload is valid.
func (p ModifyPolicyPayload) Verify() error {
	if p.Name != nil {
		if err := verifyPolicyName(*p.Name); err != nil {
			return err
		}
	}
	if p.Query != nil {
		if err := verifyPolicyQuery(*p.Query); err != nil {
			return err
		}
	}
	return nil
}

// Policy is a fleet's policy query.
type Policy struct {
	ID uint `json:"id"`
	// Name is the name of the policy query.
	// TODO(lucas): To not break clients (UI), I'm not changing this to json:"name" yet (#2595).
	Name        string `json:"query_name" db:"name"`
	Query       string `json:"query" db:"query"`
	Description string `json:"description" db:"description"`
	AuthorID    *uint  `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorName string `json:"author_name" db:"author_name"`
	// AuthorEmail is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorEmail      string  `json:"author_email" db:"author_email"`
	PassingHostCount uint    `json:"passing_host_count" db:"passing_host_count"`
	FailingHostCount uint    `json:"failing_host_count" db:"failing_host_count"`
	TeamID           *uint   `json:"team_id" db:"team_id"`
	Resolution       *string `json:"resolution,omitempty" db:"resolution"`

	TeamIDX uint `json:"-" db:"team_id_x"`

	UpdateCreateTimestamps
}

func (p Policy) AuthzType() string {
	return "policy"
}

const (
	PolicyKind = "policy"
)

type HostPolicy struct {
	ID uint `json:"id" db:"id"`
	// Name is the name of the policy query.
	// TODO(lucas): To not break clients (UI), I'm not changing this to json:"name" yet (#2595).
	Name  string `json:"query_name" db:"name"`
	Query string `json:"query" db:"query"`
	// Description is the policy description.
	// TODO(lucas): To not break clients (UI), I'm not changing this to json:"description" (#2595).
	Description string `json:"query_description" db:"description"`
	AuthorID    *uint  `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorName string `json:"author_name" db:"author_name"`
	// AuthorEmail is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorEmail string `json:"author_email" db:"author_email"`
	Response    string `json:"response" db:"response"`
	Resolution  string `json:"resolution" db:"resolution"`

	TeamIDX uint `json:"-" db:"team_id_x"`
}

type PolicySpec struct {
	Name        string `json:"name"`
	Query       string `json:"query"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
	Team        string `json:"team,omitempty"`
}

func (p PolicySpec) Verify() error {
	if err := verifyPolicyName(p.Name); err != nil {
		return err
	}
	if err := verifyPolicyQuery(p.Query); err != nil {
		return err
	}
	return nil
}
