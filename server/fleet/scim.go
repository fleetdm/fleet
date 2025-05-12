package fleet

import "time"

// SCIMMaxFieldLength is the default maximum length for SCIM fields
const SCIMMaxFieldLength = 255

// ScimUser represents a SCIM user in the database
type ScimUser struct {
	ID         uint      `db:"id"`
	ExternalID *string   `db:"external_id"`
	UserName   string    `db:"user_name"`
	GivenName  *string   `db:"given_name"`
	FamilyName *string   `db:"family_name"`
	Active     *bool     `db:"active"`
	UpdatedAt  time.Time `db:"updated_at"`
	Emails     []ScimUserEmail
	Groups     []ScimUserGroup
}

type ScimUserGroup struct {
	ID          uint   `db:"id"`
	DisplayName string `db:"display_name"`
}

func (su *ScimUser) AuthzType() string {
	return "scim_user"
}

func (su *ScimUser) DisplayName() string {
	switch {
	case su.GivenName != nil && len(*su.GivenName) > 0 && su.FamilyName != nil && len(*su.FamilyName) > 0:
		return *su.GivenName + " " + *su.FamilyName
	case su.GivenName != nil && len(*su.GivenName) > 0:
		return *su.GivenName
	case su.FamilyName != nil && len(*su.FamilyName) > 0:
		return *su.FamilyName
	default:
		return ""
	}
}

// ScimUserEmail represents an email address associated with a SCIM user
type ScimUserEmail struct {
	ScimUserID uint    `db:"scim_user_id"`
	Email      string  `db:"email"`
	Primary    *bool   `db:"primary"`
	Type       *string `db:"type"`
}

// GenerateComparisonKey generates a unique string representation of the email
// that can be used for comparison, properly handling nil values.
func (e ScimUserEmail) GenerateComparisonKey() string {
	// Handle Type field which can be nil
	typeValue := "nil"
	if e.Type != nil {
		typeValue = *e.Type
	}

	// Handle Primary field which can be nil
	primaryValue := "nil"
	if e.Primary != nil {
		if *e.Primary {
			primaryValue = "true"
		} else {
			primaryValue = "false"
		}
	}

	return e.Email + ":" + typeValue + ":" + primaryValue
}

type ScimListOptions struct {
	// 1-based index of the first result to return (must be positive integer)
	StartIndex uint
	// How many results per page (must be positive integer)
	PerPage uint
}

type ScimUsersListOptions struct {
	ScimListOptions

	// UserNameFilter filters by userName -- max of 1 response is expected
	// Cannot be used with other filters.
	UserNameFilter *string

	// EmailTypeFilter and EmailValueFilter are needed to support Entra ID filter: emails[type eq "work"].value eq "user@contoso.com"
	// https://learn.microsoft.com/en-us/entra/identity/app-provisioning/use-scim-to-provision-users-and-groups#users
	// Cannot be used with other filters.
	EmailTypeFilter  *string
	EmailValueFilter *string
}

type ScimGroupsListOptions struct {
	ScimListOptions

	// DisplayNameFilter filters by displayName
	DisplayNameFilter *string

	// ExcludeUsers if true, the group's users will not be fetched
	ExcludeUsers bool
}

type ScimGroup struct {
	ID          uint    `db:"id"`
	ExternalID  *string `db:"external_id"`
	DisplayName string  `db:"display_name"`
	ScimUsers   []uint
}

type ScimLastRequest struct {
	Status      string    `db:"status" json:"status"`
	Details     string    `db:"details" json:"details"`
	RequestedAt time.Time `db:"updated_at" json:"requested_at"`
}

type ScimDetails struct {
	LastRequest *ScimLastRequest `json:"last_request"`
}
