package fleet

// ScimUser represents a SCIM user in the database
type ScimUser struct {
	ID         uint    `db:"id"`
	ExternalID *string `db:"external_id"`
	UserName   string  `db:"user_name"`
	GivenName  *string `db:"given_name"`
	FamilyName *string `db:"family_name"`
	Active     *bool   `db:"active"`
	Emails     []ScimUserEmail
	Groups     []uint
}

func (su *ScimUser) AuthzType() string {
	return "scim_user"
}

// ScimUserEmail represents an email address associated with a SCIM user
type ScimUserEmail struct {
	ScimUserID uint    `db:"scim_user_id"`
	Email      string  `db:"email"`
	Primary    *bool   `db:"primary"`
	Type       *string `db:"type"`
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

type ScimGroup struct {
	ID          uint    `db:"id"`
	ExternalID  *string `db:"external_id"`
	DisplayName string  `db:"display_name"`
	ScimUsers   []uint
}
