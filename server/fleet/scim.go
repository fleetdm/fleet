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
}

// ScimUserEmail represents an email address associated with a SCIM user
type ScimUserEmail struct {
	ScimUserID uint    `db:"scim_user_id"`
	Email      string  `db:"email"`
	Primary    *bool   `db:"primary"`
	Type       *string `db:"type"`
}
