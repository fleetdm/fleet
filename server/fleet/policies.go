package fleet

import (
	"errors"
	"strings"
)

// PolicyPayload holds data for policy creation.
//
// If QueryID is not nil, then Name, Query and Description are ignored
// (such fields are fetched from the queries table).
type PolicyPayload struct {
	// QueryID allows creating a policy from an existing query.
	//
	// Using QueryID is the old way of creating policies.
	// Use Query, Name and Description instead.
	QueryID *uint
	// Name is the name of the policy (ignored if QueryID != nil).
	Name string
	// Query is the policy query (ignored if QueryID != nil).
	Query string
	// Description is the policy description text (ignored if QueryID != nil).
	Description string
	// Resolution indicates the steps needed to solve a failing policy.
	Resolution string
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string
}

var (
	errPolicyEmptyName       = errors.New("policy name cannot be empty")
	errPolicyEmptyQuery      = errors.New("policy query cannot be empty")
	errPolicyIDAndQuerySet   = errors.New("both fields \"queryID\" and \"query\" cannot be set")
	errPolicyInvalidQuery    = errors.New("invalid policy query")
	errPolicyInvalidPlatform = errors.New("invalid policy platform")
)

// Verify verifies the policy payload is valid.
func (p PolicyPayload) Verify() error {
	if p.QueryID != nil {
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
	if err := verifyPolicyPlatforms(p.Platform); err != nil {
		return err
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

func verifyPolicyPlatforms(platforms string) error {
	if platforms == "" {
		return nil
	}
	for _, s := range strings.Split(platforms, ",") {
		switch strings.TrimSpace(s) {
		case "windows", "linux", "darwin":
			// OK
		default:
			return errPolicyInvalidPlatform
		}
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

// PolicyData holds data of a fleet policy.
type PolicyData struct {
	// ID is the unique ID of a policy.
	ID uint `json:"id"`
	// Name is the name of the policy query.
	Name string `json:"name" db:"name"`
	// Query is the actual query to run on the osquery agents.
	Query string `json:"query" db:"query"`
	// Description describes the policy.
	Description string `json:"description" db:"description"`
	// AuthorID is the ID of the author of the policy.
	//
	// AuthorID is nil if the author is deleted from the system
	AuthorID *uint `json:"author_id" db:"author_id"`
	// AuthorName is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorName string `json:"author_name" db:"author_name"`
	// AuthorEmail is retrieved with a join to the users table in the MySQL backend (using AuthorID).
	AuthorEmail string `json:"author_email" db:"author_email"`
	// TeamID is the ID of the team the policy belongs to.
	// If TeamID is nil, then this is a global policy.
	TeamID *uint `json:"team_id" db:"team_id"`
	// Resolution describes how to solve a failing policy.
	Resolution *string `json:"resolution,omitempty" db:"resolution"`
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string `json:"platform" db:"platforms"`

	UpdateCreateTimestamps
}

// Policy is a fleet's policy query.
type Policy struct {
	PolicyData

	// PassingHostCount is the number of hosts this policy passes on.
	PassingHostCount uint `json:"passing_host_count" db:"passing_host_count"`
	// FailingHostCount is the number of hosts this policy fails on.
	FailingHostCount uint `json:"failing_host_count" db:"failing_host_count"`
}

func (p Policy) AuthzType() string {
	return "policy"
}

const (
	PolicyKind = "policy"
)

// HostPolicy is a fleet's policy query in the context of a host.
type HostPolicy struct {
	PolicyData

	// Response can be one of the following values:
	//	- "pass": if the policy was executed and passed.
	//	- "fail": if the policy was executed and did not pass.
	//	- "": if the policy did not run yet.
	Response string `json:"response" db:"response"`
}

// PolicySpec is used to hold policy data to apply policy specs.
//
// Policies are currently identified by name (unique).
type PolicySpec struct {
	// Name is the name of the policy.
	Name string `json:"name"`
	// Query is the policy's SQL query.
	Query string `json:"query"`
	// Description describes the policy.
	Description string `json:"description"`
	// Resolution describes how to solve a failing policy.
	Resolution string `json:"resolution,omitempty"`
	// Team is the name of the team.
	Team string `json:"team,omitempty"`
	// Platform is a comma-separated string to indicate the target platforms.
	//
	// Empty string targets all platforms.
	Platform string `json:"platform,omitempty"`
}

// Verify verifies the policy data is valid.
func (p PolicySpec) Verify() error {
	if err := verifyPolicyName(p.Name); err != nil {
		return err
	}
	if err := verifyPolicyQuery(p.Query); err != nil {
		return err
	}
	if err := verifyPolicyPlatforms(p.Platform); err != nil {
		return err
	}
	return nil
}
