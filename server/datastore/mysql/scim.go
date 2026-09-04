package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-cmp/cmp"
	"github.com/jmoiron/sqlx"
)

const (
	SCIMMaxStatusLength         = 31
	SCIMDefaultResourcesPerPage = 100
)

// CreateScimUser creates a new SCIM user in the database
func (ds *Datastore) CreateScimUser(ctx context.Context, user *fleet.ScimUser) (uint, error) {
	if err := validateScimUserFields(user); err != nil {
		return 0, err
	}

	var userID uint
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		const insertUserQuery = `
		INSERT INTO scim_users (
			external_id, user_name, given_name, family_name, department, active
		) VALUES (?, ?, ?, ?, ?, ?)`
		result, err := tx.ExecContext(
			ctx,
			insertUserQuery,
			user.ExternalID,
			user.UserName,
			user.GivenName,
			user.FamilyName,
			user.Department,
			user.Active,
		)
		if err != nil {
			if IsDuplicate(err) {
				return ctxerr.Wrap(ctx, alreadyExists("ScimUser", user.UserName), "insert scim user")
			}
			return ctxerr.Wrap(ctx, err, "insert scim user")
		}

		id, err := result.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim user last insert id")
		}
		user.ID = uint(id) // nolint:gosec // dismiss G115
		userID = user.ID

		if err := insertEmails(ctx, tx, user); err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim user emails")
		}

		// FIXME: Consider ways we could lift ancillary actions like this to the service layer,
		// perhaps some `WithCallback` pattern to inject these into the SCIM handlers.
		if err := maybeAssociateScimUserWithHostMDMIdP(ctx, tx, ds.logger, user); err != nil {
			return ctxerr.Wrap(ctx, err, "associate scim user with host mdm idp")
		}
		return nil
	})
	return userID, err
}

// SetScimUserFleetUserID sets the durable link from a SCIM user to its
// matching Fleet user. Set-once at the DB level so a concurrent or future
// caller can never re-point an established link.
func (ds *Datastore) SetScimUserFleetUserID(ctx context.Context, scimUserID uint, fleetUserID uint) error {
	const query = `UPDATE scim_users SET user_id = ? WHERE id = ? AND user_id IS NULL`
	if _, err := ds.writer(ctx).ExecContext(ctx, query, fleetUserID, scimUserID); err != nil {
		return ctxerr.Wrap(ctx, err, "set scim user user_id")
	}
	return nil
}

// ScimUserByID retrieves a SCIM user by ID
func (ds *Datastore) ScimUserByID(ctx context.Context, id uint) (*fleet.ScimUser, error) {
	const query = `
		SELECT
			id, external_id, user_name, given_name, family_name, department, active, updated_at, user_id
		FROM scim_users
		WHERE id = ?
	`
	user := &fleet.ScimUser{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim user").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user")
	}

	// Get the user's emails
	emails, err := ds.getScimUserEmails(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Emails = emails

	// Get the user's groups
	groups, err := ds.getScimUserGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	user.Groups = groups

	return user, nil
}

// ScimUserByUserName retrieves a SCIM user by username
func (ds *Datastore) ScimUserByUserName(ctx context.Context, userName string) (*fleet.ScimUser, error) {
	return scimUserByUserName(ctx, ds.reader(ctx), userName)
}

func scimUserByUserName(ctx context.Context, q sqlx.QueryerContext, userName string) (*fleet.ScimUser, error) {
	const query = `
		SELECT
			id, external_id, user_name, given_name, family_name, department, active, updated_at, user_id
		FROM scim_users
		WHERE user_name = ?
	`
	user := &fleet.ScimUser{}
	err := sqlx.GetContext(ctx, q, user, query, userName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim user")
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user by userName")
	}

	// Get the user's emails
	emails, err := getScimUserEmails(ctx, q, user.ID)
	if err != nil {
		return nil, err
	}
	user.Emails = emails

	// Get the user's groups
	groups, err := getScimUserGroups(ctx, q, user.ID)
	if err != nil {
		return nil, err
	}
	user.Groups = groups

	return user, nil
}

// ScimUserByUserNameOrEmail finds a SCIM user by username. If it cannot find one, then it tries email, if set.
// If multiple users are found with the same email, we log an error and return nil.
// Emails and groups are NOT populated in this method.
func (ds *Datastore) ScimUserByUserNameOrEmail(ctx context.Context, userName string, email string) (*fleet.ScimUser, error) {
	return scimUserByUserNameOrEmail(ctx, ds.reader(ctx), ds.logger, userName, email)
}

func scimUserByUserNameOrEmail(ctx context.Context, q sqlx.QueryerContext, logger *slog.Logger, userName string, email string) (*fleet.ScimUser, error) {
	// First, try to find the user by userName
	if userName != "" {
		user, err := scimUserByUserName(ctx, q, userName)
		switch {
		case err == nil:
			return user, nil
		case !fleet.IsNotFound(err):
			return nil, ctxerr.Wrap(ctx, err, "select scim user by userName")
		}
	}
	if email == "" {
		return nil, notFound("scim user")
	}

	// Now, try to find the user by using the email as the userName
	user, err := scimUserByUserName(ctx, q, email)
	switch {
	case err == nil:
		return user, nil
	case !fleet.IsNotFound(err):
		return nil, ctxerr.Wrap(ctx, err, "select scim user by userName")
	}

	// Next, to find the user by email
	const query = `
		SELECT
			scim_users.id, external_id, user_name, given_name, family_name, department, active, scim_users.updated_at
		FROM scim_users
		JOIN scim_user_emails ON scim_users.id = scim_user_emails.scim_user_id
		WHERE scim_user_emails.email = ?
	`

	var users []fleet.ScimUser
	err = sqlx.SelectContext(ctx, q, &users, query, email)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select scim user by email")
	}

	if len(users) == 0 {
		return nil, notFound("scim user")
	}

	// If multiple users found, log a message and return nil
	if len(users) > 1 {
		logger.ErrorContext(ctx, "Multiple SCIM users found with the same email", "email", email)
		return nil, nil
	}

	return &users[0], nil
}

// ScimUserByHostID retrieves a SCIM user associated with a host ID
func (ds *Datastore) ScimUserByHostID(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
	user, err := getScimUserLiteByHostID(ctx, ds.reader(ctx), hostID)
	if err != nil {
		return nil, err
	}

	// Get the user's emails
	emails, err := ds.getScimUserEmails(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Emails = emails

	// Get the user's groups
	groups, err := ds.getScimUserGroups(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Groups = groups

	return user, nil
}

// returns the ScimUser for the host, without emails and groups filled (only
// the scim_users table attributes).
func getScimUserLiteByHostID(ctx context.Context, q sqlx.QueryerContext, hostID uint) (*fleet.ScimUser, error) {
	const query = `
		SELECT
			su.id, su.external_id, su.user_name, su.given_name, su.family_name, su.department, su.active, su.updated_at
		FROM scim_users su
		JOIN host_scim_user ON su.id = host_scim_user.scim_user_id
		WHERE host_scim_user.host_id = ?
		ORDER BY su.id LIMIT 1
	`
	var user fleet.ScimUser
	err := sqlx.GetContext(ctx, q, &user, query, hostID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim user for host").WithID(hostID)
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user by host ID")
	}
	return &user, nil
}

// ReplaceScimUser replaces an existing SCIM user in the database
func (ds *Datastore) ReplaceScimUser(ctx context.Context, user *fleet.ScimUser) ([]fleet.ActivityTypeResentCertificate, error) {
	if err := validateScimUserFields(user); err != nil {
		return nil, err
	}

	// Validate that at most one email is marked as primary
	primaryCount := 0
	for _, email := range user.Emails {
		if email.Primary != nil && *email.Primary {
			primaryCount++
		}
	}
	if primaryCount > 1 {
		return nil, ctxerr.New(ctx, "only one email can be marked as primary")
	}

	// Get current emails and check if they need to be updated
	currentEmails, err := ds.getScimUserEmails(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	emailsNeedUpdate := emailsRequireUpdate(currentEmails, user.Emails)

	var resentCerts []fleet.ActivityTypeResentCertificate
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		resentCerts = nil
		// load the username and department before updating the user, to check if it changed
		old := struct {
			UserName   string  `db:"user_name"`
			Department *string `db:"department"`
			GivenName  *string `db:"given_name"`
			FamilyName *string `db:"family_name"`
		}{}
		err := sqlx.GetContext(ctx, tx, &old, `SELECT user_name, department, given_name, family_name FROM scim_users WHERE id = ?`, user.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return notFound("scim user").WithID(user.ID)
			}
			return ctxerr.Wrap(ctx, err, "load existing scim username and department before update")
		}

		// Update the SCIM user
		const updateUserQuery = `
		UPDATE scim_users SET
			external_id = ?,
			user_name = ?,
			given_name = ?,
			family_name = ?,
			department = ?,
			active = ?
		WHERE id = ?`
		result, err := tx.ExecContext(
			ctx,
			updateUserQuery,
			user.ExternalID,
			user.UserName,
			user.GivenName,
			user.FamilyName,
			user.Department,
			user.Active,
			user.ID,
		)
		if err != nil {
			if IsDuplicate(err) {
				return ctxerr.Wrap(ctx, alreadyExists("ScimUser", user.UserName), "update scim user")
			}
			return ctxerr.Wrap(ctx, err, "update scim user")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for update scim user")
		}
		if rowsAffected == 0 {
			return notFound("scim user").WithID(user.ID)
		}

		usernameChanged := old.UserName != user.UserName
		departmentChanged := !cmp.Equal(old.Department, user.Department)
		nameChanged := !cmp.Equal(old.GivenName, user.GivenName) || !cmp.Equal(old.FamilyName, user.FamilyName)

		// Only update emails if they've changed
		if emailsNeedUpdate {
			// We assume that email is not blank/null.
			// However, we do not assume that email/type are unique for a user. To keep the code simple, we:
			// 1. Delete all existing emails
			// 2. Insert all new emails
			// This is less efficient and can be optimized if we notice a load on these tables in production.

			const deleteEmailsQuery = `DELETE FROM scim_user_emails WHERE scim_user_id = ?`
			_, err = tx.ExecContext(ctx, deleteEmailsQuery, user.ID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "delete scim user emails")
			}
			err = insertEmails(ctx, tx, user)
			if err != nil {
				return err
			}
		}

		// Get the user's groups
		groups, err := ds.getScimUserGroups(ctx, user.ID)
		if err != nil {
			return err
		}
		user.Groups = groups

		// resend profiles that depend on this username if it changed
		if usernameChanged || departmentChanged || nameChanged {
			certs, err := triggerResendProfilesForIDPUserChange(ctx, tx, user.ID)
			if err != nil {
				return err
			}
			resentCerts = append(resentCerts, certs...)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return resentCerts, nil
}

func insertEmails(ctx context.Context, tx sqlx.ExtContext, user *fleet.ScimUser) error {
	// Insert the user's emails in a single batch if any
	if len(user.Emails) > 0 {
		// Build the batch insert query
		valueStrings := make([]string, 0, len(user.Emails))
		valueArgs := make([]interface{}, 0, len(user.Emails)*4)

		for i := range user.Emails {
			user.Emails[i].ScimUserID = user.ID
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			valueArgs = append(valueArgs,
				user.Emails[i].ScimUserID,
				user.Emails[i].Email,
				user.Emails[i].Primary,
				user.Emails[i].Type,
			)
		}

		// Construct the batch insert query
		insertEmailQuery := `
			INSERT INTO scim_user_emails (
				scim_user_id, email, ` + "`primary`" + `, type
			) VALUES ` + strings.Join(valueStrings, ",")

		// Execute the batch insert
		_, err := tx.ExecContext(ctx, insertEmailQuery, valueArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "batch insert scim user emails")
		}
	}
	return nil
}

// DeleteScimUser deletes a SCIM user from the database
func (ds *Datastore) DeleteScimUser(ctx context.Context, id uint) ([]fleet.ActivityTypeResentCertificate, error) {
	var resentCerts []fleet.ActivityTypeResentCertificate
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		resentCerts = nil

		// trigger resend of profiles that depend on this SCIM user (must be done
		// _before_ deleting the scim user so that we can find the affected hosts)
		certs, err := triggerResendProfilesForIDPUserDeleted(ctx, tx, id)
		if err != nil {
			return err
		}
		resentCerts = append(resentCerts, certs...)

		// Delete the user
		const deleteUserQuery = `DELETE FROM scim_users WHERE id = ?`
		result, err := tx.ExecContext(ctx, deleteUserQuery, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete scim user")
		}

		// Check if the user existed
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for delete scim user")
		}
		if rowsAffected == 0 {
			return notFound("scim user").WithID(id)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return resentCerts, nil
}

// ListScimUsers retrieves a list of SCIM users with optional filtering
func (ds *Datastore) ListScimUsers(ctx context.Context, opts fleet.ScimUsersListOptions) (users []fleet.ScimUser, totalResults uint, err error) {
	// Default pagination values if not provided
	if opts.StartIndex == 0 {
		opts.StartIndex = 1
	}
	if opts.PerPage == 0 {
		opts.PerPage = SCIMDefaultResourcesPerPage
	}

	// Build the base query
	baseQuery := `
		SELECT DISTINCT
			scim_users.id, external_id, user_name, given_name, family_name, department, active, scim_users.updated_at
		FROM scim_users
	`

	// Add joins and where clauses based on filters
	var whereClause string
	var params []interface{}

	if opts.UserNameFilter != nil {
		// Filter by username
		whereClause = " WHERE scim_users.user_name = ?"
		params = append(params, *opts.UserNameFilter)
	} else if opts.EmailTypeFilter != nil && opts.EmailValueFilter != nil {
		// Filter by email type and value
		baseQuery += " LEFT JOIN scim_user_emails ON scim_users.id = scim_user_emails.scim_user_id"
		whereClause = " WHERE scim_user_emails.type = ? AND scim_user_emails.email = ?"
		params = append(params, *opts.EmailTypeFilter, *opts.EmailValueFilter)
	}

	// First, get the total count without pagination
	countQuery := "SELECT COUNT(DISTINCT id) FROM (" + baseQuery + whereClause + ") AS filtered_users"
	err = sqlx.GetContext(ctx, ds.reader(ctx), &totalResults, countQuery, params...)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "count total scim users")
	}

	// Add pagination to the main query
	query := baseQuery + whereClause + " ORDER BY scim_users.id LIMIT ? OFFSET ?"
	params = append(params, opts.PerPage, opts.StartIndex-1)

	// Execute the query
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &users, query, params...)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "list scim users")
	}

	// Process the results
	userIDs := make([]uint, 0, len(users))
	userMap := make(map[uint]*fleet.ScimUser, len(users))

	for i, user := range users {
		userIDs = append(userIDs, user.ID)
		userMap[user.ID] = &users[i]
	}

	// If no users found, return empty slice
	if len(users) == 0 {
		return users, totalResults, nil
	}

	// Fetch emails for all users in a single query
	emailQuery, args, err := sqlx.In(`
		SELECT
			scim_user_id, email, `+"`primary`"+`, type
		FROM scim_user_emails
		WHERE scim_user_id IN (?)
		ORDER BY email ASC
	`, userIDs)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "prepare emails query")
	}

	// Convert query for the specific DB dialect
	emailQuery = ds.reader(ctx).Rebind(emailQuery)

	// Execute the email query
	var allEmails []fleet.ScimUserEmail
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allEmails, emailQuery, args...); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ctxerr.Wrap(ctx, err, "select scim user emails")
		}
	}

	// Associate emails with their users
	for i := range allEmails {
		email := allEmails[i]
		if user, ok := userMap[email.ScimUserID]; ok {
			user.Emails = append(user.Emails, email)
		}
	}

	// Fetch groups for all users in a single query
	groupQuery, groupArgs, err := sqlx.In(`
		SELECT
			sug.scim_user_id, sg.id, sg.display_name
		FROM scim_user_group sug
		JOIN scim_groups sg ON sug.group_id = sg.id
		WHERE sug.scim_user_id IN (?)
		ORDER BY sg.id ASC
	`, userIDs)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "prepare groups query")
	}

	// Execute the group query
	type userGroup struct {
		UserID      uint   `db:"scim_user_id"`
		ID          uint   `db:"id"`
		DisplayName string `db:"display_name"`
	}
	var allUserGroups []userGroup
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allUserGroups, groupQuery, groupArgs...); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ctxerr.Wrap(ctx, err, "select scim user groups")
		}
	}

	// Associate groups with their users
	for _, ug := range allUserGroups {
		if user, ok := userMap[ug.UserID]; ok {
			user.Groups = append(user.Groups, fleet.ScimUserGroup{
				ID:          ug.ID,
				DisplayName: ug.DisplayName,
			})
		}
	}

	return users, totalResults, nil
}

// getScimUserEmails retrieves all emails for a SCIM user
func (ds *Datastore) getScimUserEmails(ctx context.Context, userID uint) ([]fleet.ScimUserEmail, error) {
	return getScimUserEmails(ctx, ds.reader(ctx), userID)
}

func getScimUserEmails(ctx context.Context, q sqlx.QueryerContext, userID uint) ([]fleet.ScimUserEmail, error) {
	const query = `
		SELECT
			scim_user_id, email, ` + "`primary`" + `, type
		FROM scim_user_emails
		WHERE scim_user_id = ? ORDER BY email ASC
	`
	var emails []fleet.ScimUserEmail
	err := sqlx.SelectContext(ctx, q, &emails, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user emails")
	}
	return emails, nil
}

// getScimUserGroups retrieves all groups for a SCIM user
func (ds *Datastore) getScimUserGroups(ctx context.Context, userID uint) ([]fleet.ScimUserGroup, error) {
	return getScimUserGroups(ctx, ds.reader(ctx), userID)
}

func getScimUserGroups(ctx context.Context, q sqlx.QueryerContext, userID uint) ([]fleet.ScimUserGroup, error) {
	// A user's effective group membership is the set of groups they are a direct
	// member of, plus every ancestor group reachable by walking parent -> child
	// edges upward (nested groups, as provisioned by Entra ID). The recursive CTE
	// seeds from the user's direct groups and walks up to each parent group. UNION
	// (not UNION ALL) dedupes and guarantees termination even if a cycle exists.
	const query = `
		WITH RECURSIVE user_groups AS (
			SELECT group_id FROM scim_user_group WHERE scim_user_id = ?
			UNION
			SELECT gg.parent_group_id
			FROM user_groups ug
			JOIN scim_group_group gg ON gg.child_group_id = ug.group_id
		)
		SELECT sg.id, sg.display_name
		FROM scim_groups sg
		JOIN user_groups ug ON sg.id = ug.group_id
		ORDER BY sg.id ASC
	`
	var groups []fleet.ScimUserGroup
	err := sqlx.SelectContext(ctx, q, &groups, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user groups")
	}
	return groups, nil
}

// validateScimUserFields checks if the user fields exceed the maximum allowed length
func validateScimUserFields(user *fleet.ScimUser) error {
	if user.ExternalID != nil && len(*user.ExternalID) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "external_id", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	if len(user.UserName) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "user_name", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	if user.GivenName != nil && len(*user.GivenName) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "given_name", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	if user.FamilyName != nil && len(*user.FamilyName) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "family_name", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	if user.Department != nil && len(*user.Department) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "department", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	return nil
}

// validateScimGroupFields checks if the group fields exceed the maximum allowed length
func validateScimGroupFields(group *fleet.ScimGroup) error {
	if group.ExternalID != nil && len(*group.ExternalID) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "external_id", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	if len(group.DisplayName) > fleet.SCIMMaxFieldLength {
		return &fleet.SCIMValidationError{Field: "display_name", Message: fmt.Sprintf("exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)}
	}
	return nil
}

// CreateScimGroup creates a new SCIM group in the database
func (ds *Datastore) CreateScimGroup(ctx context.Context, group *fleet.ScimGroup) (uint, error) {
	if err := validateScimGroupFields(group); err != nil {
		return 0, err
	}

	var groupID uint
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		const insertGroupQuery = `
		INSERT INTO scim_groups (
			external_id, display_name
		) VALUES (?, ?)`
		result, err := tx.ExecContext(
			ctx,
			insertGroupQuery,
			group.ExternalID,
			group.DisplayName,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim group")
		}

		id, err := result.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim group last insert id")
		}
		group.ID = uint(id) // nolint:gosec // dismiss G115
		groupID = group.ID

		// Insert nested child group edges if any
		if len(group.ChildGroups) > 0 {
			if err := insertScimGroupChildren(ctx, tx, group.ID, group.ChildGroups); err != nil {
				return err
			}
		}

		// Insert user-group relationships if any
		if len(group.ScimUsers) > 0 {
			if err := insertScimGroupUsers(ctx, tx, group.ID, group.ScimUsers); err != nil {
				return err
			}
		}

		// this is a new group, but it may already be associated with existing
		// users (directly, or transitively through nested child groups) - trigger
		// a resend of profiles that use the IdP groups variable for the affected
		// hosts.
		if len(group.ScimUsers) > 0 || len(group.ChildGroups) > 0 {
			affectedUsers, err := getTransitiveScimGroupUserIDs(ctx, tx, group.ID)
			if err != nil {
				return err
			}
			return triggerResendProfilesForIDPGroupChangeByUsers(ctx, tx, affectedUsers)
		}

		return nil
	})
	return groupID, err
}

// insertScimGroupUsers inserts the relationships between a SCIM group and its users
func insertScimGroupUsers(ctx context.Context, tx sqlx.ExtContext, groupID uint, userIDs []uint) error {
	if len(userIDs) == 0 {
		return nil
	}

	// TODO: We could consider using string interpolation without placeholders for better performance
	// to the extent these queries are dependent only on the group ID and user IDs, which are integers.
	// See https://github.com/fleetdm/fleet/pull/30264

	batchSize := 10000
	return common_mysql.BatchProcessSimple(userIDs, batchSize, func(userIDsInBatch []uint) error {
		// Build the batch insert query
		valueStrings := make([]string, 0, len(userIDsInBatch))
		valueArgs := make([]interface{}, 0, len(userIDsInBatch)*2)
		for _, userID := range userIDsInBatch {
			valueStrings = append(valueStrings, "(?, ?)")
			valueArgs = append(valueArgs, userID, groupID)
		}

		// Construct the batch insert query
		insertQuery := `
		INSERT INTO scim_user_group (
			scim_user_id, group_id
		) VALUES ` + strings.Join(valueStrings, ",") + `
		ON DUPLICATE KEY UPDATE created_at = scim_user_group.created_at` // no-op update to avoid duplicate key errors

		// Execute the batch insert
		_, err := tx.ExecContext(ctx, insertQuery, valueArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "batch insert scim group users")
		}
		return nil
	})
}

// ScimGroupByID retrieves a SCIM group by ID
// If excludeUsers is true, the group's users (and nested child groups) will not be fetched
func (ds *Datastore) ScimGroupByID(ctx context.Context, id uint, excludeUsers bool) (*fleet.ScimGroup, error) {
	const query = `
		SELECT
			id, external_id, display_name
		FROM scim_groups
		WHERE id = ?
	`
	group := &fleet.ScimGroup{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), group, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim group").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim group")
	}

	// Get the group's members (users and nested child groups) if not excluded
	if !excludeUsers {
		users, err := getScimGroupUsers(ctx, ds.reader(ctx), id)
		if err != nil {
			return nil, err
		}
		group.ScimUsers = users

		children, err := getScimGroupChildren(ctx, ds.reader(ctx), id)
		if err != nil {
			return nil, err
		}
		group.ChildGroups = children
	}

	return group, nil
}

// ScimGroupsExist checks if all the provided SCIM group IDs exist in the datastore.
// If the slice is empty, it returns true. This mirrors ScimUsersExist.
func (ds *Datastore) ScimGroupsExist(ctx context.Context, ids []uint) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}

	// Create a set to track which IDs we've found
	foundIDs := make(map[uint]struct{}, len(ids))

	batchSize := 10000
	err := common_mysql.BatchProcessSimple(ids, batchSize, func(batchIDs []uint) error {
		query, args, err := sqlx.In(`
			SELECT id
			FROM scim_groups
			WHERE id IN (?)
		`, batchIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare scim groups exist batch query")
		}

		var foundBatchIDs []uint
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &foundBatchIDs, query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "check if scim groups exist in batch")
		}

		for _, id := range foundBatchIDs {
			foundIDs[id] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	// Verify that all requested IDs were found
	for _, id := range ids {
		if _, ok := foundIDs[id]; !ok {
			return false, nil
		}
	}
	return true, nil
}

// insertScimGroupChildren inserts direct parent -> child SCIM group edges
func insertScimGroupChildren(ctx context.Context, tx sqlx.ExtContext, parentGroupID uint, childGroupIDs []uint) error {
	if len(childGroupIDs) == 0 {
		return nil
	}

	batchSize := 10000
	return common_mysql.BatchProcessSimple(childGroupIDs, batchSize, func(childIDsInBatch []uint) error {
		valueStrings := make([]string, 0, len(childIDsInBatch))
		valueArgs := make([]any, 0, len(childIDsInBatch)*2)
		for _, childID := range childIDsInBatch {
			valueStrings = append(valueStrings, "(?, ?)")
			valueArgs = append(valueArgs, parentGroupID, childID)
		}

		insertQuery := `
		INSERT INTO scim_group_group (
			parent_group_id, child_group_id
		) VALUES ` + strings.Join(valueStrings, ",") + `
		ON DUPLICATE KEY UPDATE created_at = scim_group_group.created_at` // no-op update to avoid duplicate key errors

		if _, err := tx.ExecContext(ctx, insertQuery, valueArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "batch insert scim group children")
		}
		return nil
	})
}

// getScimGroupChildren retrieves the IDs of the direct (nested) child groups of a SCIM group
func getScimGroupChildren(ctx context.Context, q sqlx.QueryerContext, groupID uint) ([]uint, error) {
	const query = `
		SELECT
			child_group_id
		FROM scim_group_group
		WHERE parent_group_id = ? ORDER BY child_group_id ASC
	`
	var childIDs []uint
	err := sqlx.SelectContext(ctx, q, &childIDs, query, groupID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select scim group children")
	}
	return childIDs, nil
}

// getTransitiveScimGroupUserIDs returns the IDs of all SCIM users who are
// effective members of the given group -- that is, direct members of the group
// or of any of its (recursively) nested child groups.
func getTransitiveScimGroupUserIDs(ctx context.Context, q sqlx.QueryerContext, groupID uint) ([]uint, error) {
	const query = `
		WITH RECURSIVE descendants AS (
			SELECT ? AS group_id
			UNION
			SELECT gg.child_group_id
			FROM descendants d
			JOIN scim_group_group gg ON gg.parent_group_id = d.group_id
		)
		SELECT DISTINCT sug.scim_user_id
		FROM descendants d
		JOIN scim_user_group sug ON sug.group_id = d.group_id
	`
	var userIDs []uint
	err := sqlx.SelectContext(ctx, q, &userIDs, query, groupID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select transitive scim group users")
	}
	return userIDs, nil
}

// ScimGroupByDisplayName retrieves a SCIM group by display name
// This method always fetches the group's users
func (ds *Datastore) ScimGroupByDisplayName(ctx context.Context, displayName string) (*fleet.ScimGroup, error) {
	const query = `
		SELECT
			id, external_id, display_name
		FROM scim_groups
		WHERE display_name = ?
	`
	group := &fleet.ScimGroup{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), group, query, displayName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim group")
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim group by displayName")
	}

	// Get the group's members (users and nested child groups)
	users, err := getScimGroupUsers(ctx, ds.reader(ctx), group.ID)
	if err != nil {
		return nil, err
	}
	group.ScimUsers = users

	children, err := getScimGroupChildren(ctx, ds.reader(ctx), group.ID)
	if err != nil {
		return nil, err
	}
	group.ChildGroups = children

	return group, nil
}

// getScimGroupUsers retrieves all user IDs for a SCIM group
func getScimGroupUsers(ctx context.Context, q sqlx.QueryerContext, groupID uint) ([]uint, error) {
	const query = `
		SELECT
			scim_user_id
		FROM scim_user_group
		WHERE group_id = ? ORDER BY scim_user_id ASC
	`
	var userIDs []uint
	err := sqlx.SelectContext(ctx, q, &userIDs, query, groupID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select scim group users")
	}
	return userIDs, nil
}

// scimGroupAttributes holds a SCIM group's stored scalar attributes.
type scimGroupAttributes struct {
	ExternalID  *string `db:"external_id"`
	DisplayName string  `db:"display_name"`
}

// loadScimGroupAttributes reads a SCIM group's scalar attributes, so a caller can
// tell what the update would change.
func loadScimGroupAttributes(ctx context.Context, tx sqlx.ExtContext, groupID uint) (scimGroupAttributes, error) {
	var existing scimGroupAttributes
	err := sqlx.GetContext(ctx, tx, &existing,
		`SELECT external_id, display_name FROM scim_groups WHERE id = ?`, groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return existing, notFound("scim group").WithID(groupID)
		}
		return existing, ctxerr.Wrap(ctx, err, "load existing scim group before update")
	}
	return existing, nil
}

// updateScimGroupAttributes writes a SCIM group's scalar attributes.
func updateScimGroupAttributes(ctx context.Context, tx sqlx.ExtContext, group *fleet.ScimGroup) error {
	const updateGroupQuery = `
		UPDATE scim_groups SET
			external_id = ?,
			display_name = ?
		WHERE id = ?`
	result, err := tx.ExecContext(
		ctx,
		updateGroupQuery,
		group.ExternalID,
		group.DisplayName,
		group.ID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update scim group")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get rows affected for update scim group")
	}
	// The row was there a moment ago, so this only fires if it was deleted since.
	if rowsAffected == 0 {
		return notFound("scim group").WithID(group.ID)
	}

	return nil
}

// ApplyScimGroupPatch updates an existing SCIM group's attributes and applies
// only the membership changes described by deltas, leaving every other member of
// the group untouched.
func (ds *Datastore) ApplyScimGroupPatch(ctx context.Context, group *fleet.ScimGroup, deltas fleet.ScimGroupMemberDeltas) error {
	if err := validateScimGroupFields(group); err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		existing, err := loadScimGroupAttributes(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		groupNameChanged := existing.DisplayName != group.DisplayName

		// Skip the write when the attributes are untouched, which is the common case
		// for a members-only patch. The UPDATE would be a no-op but still takes an
		// exclusive lock on the group row until commit, serializing concurrent
		// patches that the targeted member writes below do not need serialized.
		if groupNameChanged || !ptr.Equal(existing.ExternalID, group.ExternalID) {
			if err := updateScimGroupAttributes(ctx, tx, group); err != nil {
				return err
			}
		}

		return applyScimGroupMemberDeltas(ctx, tx, group.ID, deltas, groupNameChanged)
	})
}

// applyScimGroupMemberDeltas writes the membership changes described by deltas
// and triggers the profile resends they require.
func applyScimGroupMemberDeltas(
	ctx context.Context, tx sqlx.ExtContext, groupID uint, deltas fleet.ScimGroupMemberDeltas, groupNameChanged bool,
) error {
	// Read which named members and child edges exist before the writes: the resend
	// below then covers only members whose membership actually changes, so a no-op
	// delta (an IdP retrying an add, removing a non-member) resends nothing. The
	// read feeds only the resend set; the writes stay driven by the full deltas.
	existingUsers, err := selectExistingScimGroupMembers(ctx, tx, "scim_user_group", "group_id", "scim_user_id",
		groupID, append(slices.Clone(deltas.AddUsers), deltas.RemoveUsers...))
	if err != nil {
		return err
	}
	existingChildren, err := selectExistingScimGroupMembers(ctx, tx, "scim_group_group", "parent_group_id", "child_group_id",
		groupID, append(slices.Clone(deltas.AddChildGroups), deltas.RemoveChildGroups...))
	if err != nil {
		return err
	}

	if err := insertScimGroupUsers(ctx, tx, groupID, deltas.AddUsers); err != nil {
		return err
	}
	if err := deleteScimGroupUsers(ctx, tx, groupID, deltas.RemoveUsers); err != nil {
		return err
	}
	if err := insertScimGroupChildren(ctx, tx, groupID, deltas.AddChildGroups); err != nil {
		return err
	}
	if err := deleteScimGroupChildren(ctx, tx, groupID, deltas.RemoveChildGroups); err != nil {
		return err
	}

	// The transitive lookups below run after the deletes, so removed users and
	// removed-child subtrees are no longer reachable from the group and are
	// collected explicitly.
	affectedUsers := membershipChanges(deltas.AddUsers, deltas.RemoveUsers, existingUsers)
	// A child group edge change affects every user in that child's subtree,
	// since their effective membership in this group (and its ancestors)
	// changed.
	subtreeRoots := membershipChanges(deltas.AddChildGroups, deltas.RemoveChildGroups, existingChildren)
	if groupNameChanged {
		// A rename also affects every user still in the group, directly or
		// through nested child groups.
		subtreeRoots = append(subtreeRoots, groupID)
	}
	for _, subtreeID := range subtreeRoots {
		subtreeUsers, err := getTransitiveScimGroupUserIDs(ctx, tx, subtreeID)
		if err != nil {
			return err
		}
		affectedUsers = append(affectedUsers, subtreeUsers...)
	}
	return triggerResendProfilesForIDPGroupChangeByUsers(ctx, tx, affectedUsers)
}

// ReplaceScimGroup replaces an existing SCIM group in the database
func (ds *Datastore) ReplaceScimGroup(ctx context.Context, group *fleet.ScimGroup) error {
	if err := validateScimGroupFields(group); err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		existing, err := loadScimGroupAttributes(ctx, tx, group.ID)
		if err != nil {
			return err
		}
		if err := updateScimGroupAttributes(ctx, tx, group); err != nil {
			return err
		}
		groupNameChanged := existing.DisplayName != group.DisplayName

		// Diff the desired membership against the stored one, then write only the
		// difference, reusing the same targeted writes a patch uses.
		existingUsers, err := getScimGroupUsers(ctx, tx, group.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get existing scim group users")
		}
		existingChildren, err := getScimGroupChildren(ctx, tx, group.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get existing scim group children")
		}

		var deltas fleet.ScimGroupMemberDeltas
		deltas.AddUsers, deltas.RemoveUsers = diffUintSlices(existingUsers, group.ScimUsers)
		deltas.AddChildGroups, deltas.RemoveChildGroups = diffUintSlices(existingChildren, group.ChildGroups)
		return applyScimGroupMemberDeltas(ctx, tx, group.ID, deltas, groupNameChanged)
	})
}

// selectExistingScimGroupMembers returns which of memberIDs are currently linked
// to the group in table.
func selectExistingScimGroupMembers(
	ctx context.Context, tx sqlx.ExtContext, table, groupCol, memberCol string, groupID uint, memberIDs []uint,
) (map[uint]struct{}, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}
	stmt, args, err := sqlx.In(
		fmt.Sprintf("SELECT %s FROM %s WHERE %s = ? AND %s IN (?)", memberCol, table, groupCol, memberCol),
		groupID, memberIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build select existing scim group members")
	}
	var ids []uint
	if err := sqlx.SelectContext(ctx, tx, &ids, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select existing scim group members from "+table)
	}
	existing := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		existing[id] = struct{}{}
	}
	return existing, nil
}

// membershipChanges returns the members whose membership the deltas actually
// change: adds not already present and removes that are present.
func membershipChanges(adds, removes []uint, existing map[uint]struct{}) []uint {
	changed := make([]uint, 0, len(adds)+len(removes))
	for _, id := range adds {
		if _, ok := existing[id]; !ok {
			changed = append(changed, id)
		}
	}
	for _, id := range removes {
		if _, ok := existing[id]; ok {
			changed = append(changed, id)
		}
	}
	return changed
}

// deleteScimGroupMembers removes rows linking a SCIM group to the given member
// IDs, leaving the group's other members in place. Both membership tables have
// the same shape: a column pointing at the group, and one at the member.
func deleteScimGroupMembers(
	ctx context.Context, tx sqlx.ExtContext, table, groupCol, memberCol string, groupID uint, memberIDs []uint,
) error {
	if len(memberIDs) == 0 {
		return nil
	}

	batchSize := 10000
	return common_mysql.BatchProcessSimple(memberIDs, batchSize, func(batch []uint) error {
		params := make([]any, 0, len(batch)+1)
		params = append(params, groupID)
		for _, memberID := range batch {
			params = append(params, memberID)
		}

		deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE %s = ? AND %s IN (%s?)",
			table, groupCol, memberCol, strings.Repeat("?, ", len(batch)-1))

		if _, err := tx.ExecContext(ctx, deleteQuery, params...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete scim group members from "+table)
		}
		return nil
	})
}

func deleteScimGroupUsers(ctx context.Context, tx sqlx.ExtContext, groupID uint, userIDs []uint) error {
	return deleteScimGroupMembers(ctx, tx, "scim_user_group", "group_id", "scim_user_id", groupID, userIDs)
}

func deleteScimGroupChildren(ctx context.Context, tx sqlx.ExtContext, parentGroupID uint, childGroupIDs []uint) error {
	return deleteScimGroupMembers(ctx, tx, "scim_group_group", "parent_group_id", "child_group_id", parentGroupID, childGroupIDs)
}

// diffUintSlices returns the elements to add (in want but not in have) and to
// remove (in have but not in want). toAdd is deduplicated, preserving order:
// want may come straight from a SCIM payload, which can repeat members.
func diffUintSlices(have, want []uint) (toAdd, toRemove []uint) {
	haveSet := make(map[uint]struct{}, len(have))
	for _, id := range have {
		haveSet[id] = struct{}{}
	}
	wantSet := make(map[uint]struct{}, len(want))
	for _, id := range want {
		wantSet[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := haveSet[id]; !ok {
			toAdd = append(toAdd, id)
			haveSet[id] = struct{}{}
		}
	}
	for _, id := range have {
		if _, ok := wantSet[id]; !ok {
			toRemove = append(toRemove, id)
		}
	}
	return toAdd, toRemove
}

// DeleteScimGroup deletes a SCIM group from the database
func (ds *Datastore) DeleteScimGroup(ctx context.Context, id uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// trigger resend of profiles that depend on this SCIM group (must be done
		// _before_ deleting the scim group so that we can find the affected hosts)
		err := triggerResendProfilesForIDPGroupChange(ctx, tx, id)
		if err != nil {
			return err
		}

		// Delete the group
		const deleteGroupQuery = `DELETE FROM scim_groups WHERE id = ?`
		result, err := tx.ExecContext(ctx, deleteGroupQuery, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete scim group")
		}

		// Check if the group existed
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for delete scim group")
		}
		if rowsAffected == 0 {
			return notFound("scim group").WithID(id)
		}

		return nil
	})
}

// ListScimGroups retrieves a list of SCIM groups with pagination
// If opts.ExcludeUsers is true, the groups' users will not be fetched
func (ds *Datastore) ListScimGroups(ctx context.Context, opts fleet.ScimGroupsListOptions) (groups []fleet.ScimGroup, totalResults uint, err error) {
	// Default pagination values if not provided
	if opts.StartIndex == 0 {
		opts.StartIndex = 1
	}
	if opts.PerPage == 0 {
		opts.PerPage = SCIMDefaultResourcesPerPage
	}

	// Build the query
	baseQuery := `
		SELECT DISTINCT
			scim_groups.id, external_id, display_name
		FROM scim_groups
	`

	// Add where clause based on filters
	var whereClause string
	var params []interface{}

	if opts.DisplayNameFilter != nil {
		whereClause = " WHERE scim_groups.display_name = ?"
		params = append(params, *opts.DisplayNameFilter)
	}

	// First, get the total count without pagination
	countQuery := "SELECT COUNT(DISTINCT id) FROM (" + baseQuery + whereClause + ") AS filtered_groups"
	err = sqlx.GetContext(ctx, ds.reader(ctx), &totalResults, countQuery, params...)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "count total scim groups")
	}

	// Add pagination to the main query
	query := baseQuery + whereClause + " ORDER BY scim_groups.id LIMIT ? OFFSET ?"
	params = append(params, opts.PerPage, opts.StartIndex-1)

	// Execute the query
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &groups, query, params...)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "list scim groups")
	}

	// Process the results
	groupIDs := make([]uint, 0, len(groups))
	groupMap := make(map[uint]*fleet.ScimGroup, len(groups))

	for i, group := range groups {
		groupIDs = append(groupIDs, group.ID)
		groupMap[group.ID] = &groups[i]
		groups[i].ScimUsers = []uint{} // Initialize empty user list for each group
	}

	// If no groups found, return empty slice
	if len(groups) == 0 {
		return groups, totalResults, nil
	}

	// Skip fetching users if ExcludeUsers is true
	if !opts.ExcludeUsers {
		// Fetch users for all groups in a single query
		userQuery, args, err := sqlx.In(`
			SELECT
				group_id, scim_user_id
			FROM scim_user_group
			WHERE group_id IN (?)
			ORDER BY scim_user_id ASC
		`, groupIDs)
		if err != nil {
			return nil, 0, ctxerr.Wrap(ctx, err, "prepare users query")
		}

		// Execute the user query
		type groupUser struct {
			GroupID uint `db:"group_id"`
			UserID  uint `db:"scim_user_id"`
		}
		var allGroupUsers []groupUser
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allGroupUsers, userQuery, args...); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, 0, ctxerr.Wrap(ctx, err, "select scim group users")
			}
		}

		// Associate users with their groups
		for _, gu := range allGroupUsers {
			if group, ok := groupMap[gu.GroupID]; ok {
				group.ScimUsers = append(group.ScimUsers, gu.UserID)
			}
		}
	}

	return groups, totalResults, nil
}

// ScimLastRequest retrieves the last SCIM request info
func (ds *Datastore) ScimLastRequest(ctx context.Context) (*fleet.ScimLastRequest, error) {
	const query = `
				SELECT
					status, details, updated_at
				FROM scim_last_request
				ORDER BY id LIMIT 1
			`
	var lastRequest fleet.ScimLastRequest
	err := sqlx.GetContext(ctx, ds.reader(ctx), &lastRequest, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim last request")
	}
	return &lastRequest, nil
}

// UpdateScimLastRequest updates the last SCIM request information
// If no row exists, it creates a new one
func (ds *Datastore) UpdateScimLastRequest(ctx context.Context, lastRequest *fleet.ScimLastRequest) error {
	if lastRequest == nil {
		return nil
	}
	if len(lastRequest.Status) > SCIMMaxStatusLength {
		return fmt.Errorf("status exceeds maximum length of %d characters", SCIMMaxStatusLength)
	}
	if len(lastRequest.Details) > fleet.SCIMMaxFieldLength {
		return fmt.Errorf("details exceeds maximum length of %d characters", fleet.SCIMMaxFieldLength)
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Try to update first. We always update the timestamp since success requests all look the same.
		const updateQuery = `
				UPDATE scim_last_request
				SET status = ?, details = ?, updated_at = NOW(6)
				`
		result, err := tx.ExecContext(
			ctx,
			updateQuery,
			lastRequest.Status,
			lastRequest.Details,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update scim last request")
		}

		// Check if any rows were affected by the update
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for update scim last request")
		}

		// If no rows were affected, insert a new row
		if rowsAffected == 0 {
			const insertQuery = `
					INSERT INTO scim_last_request (
						status, details
					) VALUES (?, ?)
					`
			_, err = tx.ExecContext(
				ctx,
				insertQuery,
				lastRequest.Status,
				lastRequest.Details,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert scim last request")
			}
		}

		return nil
	})
}

func getHostIDsHavingScimIDPUser(ctx context.Context, tx sqlx.ExtContext, scimUserID uint) ([]uint, error) {
	// get all hosts that have this user as IdP user - this means that we only
	// consider hosts where this user id is the smallest user id associated with
	// the host (which is the one we consider as the IdP user of the host, see
	// the query in ScimUserByHostID)
	const getAssociatedHostIDsQuery = `
	SELECT DISTINCT
		hsu.host_id
	FROM
		host_scim_user hsu
		LEFT JOIN host_scim_user extra_hsu ON
			hsu.host_id = extra_hsu.host_id AND
			hsu.scim_user_id != extra_hsu.scim_user_id AND
			extra_hsu.scim_user_id < hsu.scim_user_id
	WHERE
		hsu.scim_user_id = ? AND
		extra_hsu.host_id IS NULL
`
	var hostIDs []uint
	err := sqlx.SelectContext(ctx, tx, &hostIDs, getAssociatedHostIDsQuery, scimUserID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get scim user host IDs")
	}
	return hostIDs, nil
}

func getHostIDsHavingScimIDPUsers(ctx context.Context, tx sqlx.ExtContext, scimUserIDs []uint) ([]uint, error) {
	// get all hosts that have any of those users as IdP user - this means that
	// we only consider hosts where the user id is the smallest user id
	// associated with the host (which is the one we consider as the IdP user of
	// the host, see the query in ScimUserByHostID)
	const getAssociatedHostIDsQuery = `
	SELECT DISTINCT
		hsu.host_id
	FROM
		host_scim_user hsu
		LEFT JOIN host_scim_user extra_hsu ON
			hsu.host_id = extra_hsu.host_id AND
			hsu.scim_user_id != extra_hsu.scim_user_id AND
			extra_hsu.scim_user_id < hsu.scim_user_id
	WHERE
		hsu.scim_user_id IN (?) AND
		extra_hsu.host_id IS NULL
`
	stmt, args, err := sqlx.In(getAssociatedHostIDsQuery, scimUserIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "prepare get scim users host IDs")
	}

	var hostIDs []uint
	err = sqlx.SelectContext(ctx, tx, &hostIDs, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get scim users host IDs")
	}
	return hostIDs, nil
}

func triggerResendProfilesForIDPUserChange(ctx context.Context, tx sqlx.ExtContext, updatedScimUserID uint) ([]fleet.ActivityTypeResentCertificate, error) {
	hostIDs, err := getHostIDsHavingScimIDPUser(ctx, tx, updatedScimUserID)
	if err != nil {
		return nil, err
	}
	vars := []fleet.FleetVarName{
		fleet.FleetVarHostEndUserIDPUsername,
		fleet.FleetVarHostEndUserIDPUsernameLocalPart,
		fleet.FleetVarHostEndUserIDPDepartment,
		fleet.FleetVarHostEndUserIDPFullname,
	}
	resentCerts, err := selectCertTemplatesToResend(ctx, tx, hostIDs, fleetVarNamesToDBVars(vars))
	if err != nil {
		return nil, err
	}
	if err := triggerResendProfilesUsingVariables(ctx, tx, hostIDs, vars); err != nil {
		return nil, err
	}
	return resentCerts, nil
}

func triggerResendProfilesForIDPUserDeleted(ctx context.Context, tx sqlx.ExtContext, deletedScimUserID uint) ([]fleet.ActivityTypeResentCertificate, error) {
	hostIDs, err := getHostIDsHavingScimIDPUser(ctx, tx, deletedScimUserID)
	if err != nil {
		return nil, err
	}
	vars := []fleet.FleetVarName{
		fleet.FleetVarHostEndUserIDPUsername,
		fleet.FleetVarHostEndUserIDPUsernameLocalPart,
		fleet.FleetVarHostEndUserIDPGroups,
		fleet.FleetVarHostEndUserIDPDepartment,
		fleet.FleetVarHostEndUserIDPFullname,
	}
	resentCerts, err := selectCertTemplatesToResend(ctx, tx, hostIDs, fleetVarNamesToDBVars(vars))
	if err != nil {
		return nil, err
	}
	if err := triggerResendProfilesUsingVariables(ctx, tx, hostIDs, vars); err != nil {
		return nil, err
	}
	return resentCerts, nil
}

func triggerResendProfilesForIDPGroupChange(ctx context.Context, tx sqlx.ExtContext, updatedScimGroupID uint) error {
	// get the updated list of effective users for that group (direct members plus
	// members of any nested child groups)
	userIDs, err := getTransitiveScimGroupUserIDs(ctx, tx, updatedScimGroupID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}

	// get hosts that have any of those users as IdP user
	hostIDs, err := getHostIDsHavingScimIDPUsers(ctx, tx, userIDs)
	if err != nil {
		return err
	}
	return triggerResendProfilesUsingVariables(ctx, tx, hostIDs,
		[]fleet.FleetVarName{fleet.FleetVarHostEndUserIDPGroups})
}

func triggerResendProfilesForIDPGroupChangeByUsers(ctx context.Context, tx sqlx.ExtContext, scimUserIDs []uint) error {
	if len(scimUserIDs) == 0 {
		return nil
	}

	hostIDs, err := getHostIDsHavingScimIDPUsers(ctx, tx, scimUserIDs)
	if err != nil {
		return err
	}
	return triggerResendProfilesUsingVariables(ctx, tx, hostIDs,
		[]fleet.FleetVarName{fleet.FleetVarHostEndUserIDPGroups})
}

func triggerResendProfilesForIDPUserAddedToHost(ctx context.Context, tx sqlx.ExtContext, hostID, updatedScimUserID uint) ([]fleet.ActivityTypeResentCertificate, error) {
	// check that this user is indeed the scim IdP user for this host (and not an
	// extra, unused one)
	user, err := getScimUserLiteByHostID(ctx, tx, hostID)
	if err != nil {
		return nil, err
	}
	if updatedScimUserID != user.ID {
		// host is not impacted, updated user is not its IdP user
		return nil, nil
	}
	vars := []fleet.FleetVarName{
		fleet.FleetVarHostEndUserIDPUsername,
		fleet.FleetVarHostEndUserIDPUsernameLocalPart,
		fleet.FleetVarHostEndUserIDPDepartment,
		fleet.FleetVarHostEndUserIDPGroups,
		fleet.FleetVarHostEndUserIDPFullname,
	}
	resentCerts, err := selectCertTemplatesToResend(ctx, tx, []uint{hostID}, fleetVarNamesToDBVars(vars))
	if err != nil {
		return nil, err
	}
	if err := triggerResendProfilesUsingVariables(ctx, tx, []uint{hostID}, vars); err != nil {
		return nil, err
	}
	return resentCerts, nil
}

func selectCertTemplatesToResend(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, vars []any) ([]fleet.ActivityTypeResentCertificate, error) {
	if len(hostIDs) == 0 || len(vars) == 0 {
		return nil, nil
	}

	const query = `
	SELECT DISTINCT
		h.id AS host_id,
		COALESCE(h.computer_name, '') AS computer_name,
		COALESCE(h.hostname, '') AS hostname,
		COALESCE(h.hardware_model, '') AS hardware_model,
		COALESCE(h.hardware_serial, '') AS hardware_serial,
		ct.id AS certificate_template_id,
		ct.name AS certificate_name
	FROM
		host_certificate_templates hct
		JOIN hosts h
			ON h.uuid = hct.host_uuid
		JOIN certificate_templates ct
			ON ct.id = hct.certificate_template_id AND
			   ct.team_id = COALESCE(h.team_id, 0)
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.certificate_template_id = ct.id
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	WHERE
		h.id IN (:host_ids) AND
		hct.operation_type = :operation_type_install AND
		hct.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	namedParams := map[string]any{
		"host_ids":               hostIDs,
		"operation_type_install": fleet.MDMOperationTypeInstall,
		"affected_vars":          vars,
	}

	stmt, args, err := sqlx.Named(query, namedParams)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "prepare select cert templates to resend names")
	}
	stmt, args, err = sqlx.In(stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "prepare select cert templates to resend arguments")
	}

	type row struct {
		HostID                uint   `db:"host_id"`
		ComputerName          string `db:"computer_name"`
		Hostname              string `db:"hostname"`
		HardwareModel         string `db:"hardware_model"`
		HardwareSerial        string `db:"hardware_serial"`
		CertificateTemplateID uint   `db:"certificate_template_id"`
		CertificateName       string `db:"certificate_name"`
	}
	var rows []row
	if err := sqlx.SelectContext(ctx, tx, &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select cert templates to resend")
	}

	activities := make([]fleet.ActivityTypeResentCertificate, 0, len(rows))
	for _, r := range rows {
		activities = append(activities, fleet.ActivityTypeResentCertificate{
			HostID:                r.HostID,
			HostDisplayName:       fleet.HostDisplayName(r.ComputerName, r.Hostname, r.HardwareModel, r.HardwareSerial),
			CertificateTemplateID: r.CertificateTemplateID,
			CertificateName:       r.CertificateName,
			Automated:             true,
		})
	}
	return activities, nil
}

func fleetVarNamesToDBVars(vars []fleet.FleetVarName) []any {
	result := make([]any, len(vars))
	for i, v := range vars {
		result[i] = "FLEET_VAR_" + string(v)
	}
	return result
}

func triggerResendProfilesUsingVariables(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, affectedVars []fleet.FleetVarName) error {
	if len(hostIDs) == 0 || len(affectedVars) == 0 {
		return nil
	}

	// NOTE: this cannot reuse bulkSetPendingMDMAppleHostProfilesDB, as this
	// (complex) function is based on changes it can detect itself, such as a
	// profile content change, label membership changes, etc. It does not receive
	// a list of host/profile to update, but relies on its own diff.
	//
	// In the case here where variable values change, we want a simple "resend"
	// with the new values, so we don't need the complex diff logic, we only set
	// to "pending" the profiles that depend on the variables that were already
	// installed on the affected hosts. ReconcileAppleProfilesBatched will take care of
	// resending as appropriate based on label membershup and all at the time it
	// runs.
	const appleUpdateStatusQuery = `
	UPDATE
		host_mdm_apple_profiles hmap
		JOIN hosts h
			ON h.uuid = hmap.host_uuid
		JOIN mdm_apple_configuration_profiles macp
			ON (macp.team_id = h.team_id OR (COALESCE(macp.team_id, 0) = 0 AND h.team_id IS NULL)) AND
				 macp.profile_uuid = hmap.profile_uuid
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.apple_profile_uuid = macp.profile_uuid
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	SET
		hmap.status = NULL,
		hmap.detail = NULL,
		hmap.command_uuid = ''
	WHERE
		h.id IN (:host_ids) AND
		hmap.operation_type = :operation_type_install AND
		hmap.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	const windowsUpdateStatusQuery = `
	UPDATE
		host_mdm_windows_profiles hmwp
		JOIN hosts h
			ON h.uuid = hmwp.host_uuid
		JOIN mdm_windows_configuration_profiles mwcp
			ON (mwcp.team_id = h.team_id OR (COALESCE(mwcp.team_id, 0) = 0 AND h.team_id IS NULL)) AND
				 mwcp.profile_uuid = hmwp.profile_uuid
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.windows_profile_uuid = mwcp.profile_uuid
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	SET
		hmwp.status = NULL,
		hmwp.command_uuid = '',
		hmwp.detail = NULL
	WHERE
		h.id IN (:host_ids) AND
		hmwp.operation_type = :operation_type_install AND
		hmwp.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	const declarationUpdateStatusQuery = `
	UPDATE
		host_mdm_apple_declarations hmad
		JOIN hosts h
			ON h.uuid = hmad.host_uuid
		JOIN mdm_apple_declarations mad
			ON (mad.team_id = h.team_id OR (COALESCE(mad.team_id, 0) = 0 AND h.team_id IS NULL)) AND
				 mad.declaration_uuid = hmad.declaration_uuid
		LEFT JOIN mdm_apple_ddm_activations act
			ON act.declaration_uuid = mad.declaration_uuid
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.apple_declaration_uuid = mad.declaration_uuid
				OR mcpv.apple_ddm_activation_uuid = act.activation_uuid
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	SET
		hmad.status = NULL,
		hmad.detail = NULL
	WHERE
		h.id IN (:host_ids) AND
		hmad.operation_type = :operation_type_install AND
		hmad.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	const certTemplateUpdateStatusQuery = `
	UPDATE
		host_certificate_templates hct
		JOIN hosts h
			ON h.uuid = hct.host_uuid
		JOIN certificate_templates ct
			ON ct.id = hct.certificate_template_id AND
			   ct.team_id = COALESCE(h.team_id, 0)
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.certificate_template_id = ct.id
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	SET
		hct.status = :cert_pending_status,
		hct.uuid = UUID_TO_BIN(UUID(), true),
		hct.fleet_challenge = NULL,
		hct.not_valid_before = NULL,
		hct.not_valid_after = NULL,
		hct.serial = NULL,
		hct.detail = NULL,
		hct.retry_count = 0
	WHERE
		h.id IN (:host_ids) AND
		hct.operation_type = :operation_type_install AND
		hct.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	vars := make([]any, len(affectedVars))
	for i, v := range affectedVars {
		vars[i] = "FLEET_VAR_" + string(v)
	}

	namedParams := map[string]any{
		"host_ids":               hostIDs,
		"operation_type_install": fleet.MDMOperationTypeInstall,
		"affected_vars":          vars,
	}

	const androidUpdateStatusQuery = `
	UPDATE
		host_mdm_android_profiles hmap
		JOIN hosts h
			ON h.uuid = hmap.host_uuid
		JOIN mdm_android_configuration_profiles macp
			ON (macp.team_id = COALESCE(h.team_id, 0)) AND
				 macp.profile_uuid = hmap.profile_uuid
		JOIN mdm_configuration_profile_variables mcpv
			ON mcpv.android_profile_uuid = macp.profile_uuid
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
	SET
		hmap.status = NULL,
		hmap.detail = NULL
	WHERE
		h.id IN (:host_ids) AND
		hmap.operation_type = :operation_type_install AND
		hmap.status IS NOT NULL AND
		fv.name IN (:affected_vars)
`

	for _, query := range []string{appleUpdateStatusQuery, windowsUpdateStatusQuery, declarationUpdateStatusQuery, androidUpdateStatusQuery} {
		updateStmt, args, err := sqlx.Named(query, namedParams)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare resend profiles replace names")
		}

		updateStmt, args, err = sqlx.In(updateStmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare resend profiles arguments")
		}

		_, err = tx.ExecContext(ctx, updateStmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "execute resend profiles")
		}
	}

	// Resend certificate templates that use affected variables.
	certParams := map[string]any{
		"host_ids":               hostIDs,
		"operation_type_install": fleet.MDMOperationTypeInstall,
		"affected_vars":          vars,
		"cert_pending_status":    fleet.CertificateTemplatePending,
	}
	certStmt, certArgs, err := sqlx.Named(certTemplateUpdateStatusQuery, certParams)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare resend certificate templates replace names")
	}
	certStmt, certArgs, err = sqlx.In(certStmt, certArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare resend certificate templates arguments")
	}
	if _, err = tx.ExecContext(ctx, certStmt, certArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "execute resend certificate templates")
	}

	// Queue make_android_app_available jobs for managed app configs that use affected variables,
	// scoped to the teams of the affected hosts.
	if err := queueManagedConfigResendJobs(ctx, tx, hostIDs, vars); err != nil {
		return ctxerr.Wrap(ctx, err, "queue managed config resend jobs")
	}

	// Re-enqueue host name templates that use an affected IdP variable.
	if err := triggerResendDeviceNamesForIDPChange(ctx, tx, hostIDs, affectedVars); err != nil {
		return ctxerr.Wrap(ctx, err, "resend host name templates for idp change")
	}

	return nil
}

// triggerResendDeviceNamesForIDPChange re-queues host-name enforcement rows so the
// device-name cron re-resolves with the updated IdP value and enqueues a fresh
// DeviceName command.
func triggerResendDeviceNamesForIDPChange(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, affectedVars []fleet.FleetVarName) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Restrict to the affected variables that are actually supported in host name
	// templates.
	varNames := make([]string, 0, len(affectedVars))
	for _, v := range affectedVars {
		if fleet.IsHostNameTemplateIDPVar(string(v)) {
			varNames = append(varNames, string(v))
		}
	}
	if len(varNames) == 0 {
		return nil
	}

	// A host's governing template is its team template, or the global "No team"
	// template when it has no team. Match it against the changed variables with a
	// single alternation (the names are [A-Z_], safe to embed in the pattern); the
	// pattern value is bound as a parameter.
	const selectStmt = `
		SELECT h.id
		FROM hosts h
		LEFT JOIN teams t ON t.id = h.team_id
		WHERE h.id IN (?)
			AND COALESCE(CASE WHEN h.team_id IS NULL
				THEN ` + deviceNameNoTeamTemplateExpr + `
				ELSE t.config->>'$.mdm.name_template' END, '') REGEXP ?`

	// The trailing word boundary keeps a changed HOST_END_USER_IDP_USERNAME from
	// matching a template that only uses HOST_END_USER_IDP_USERNAME_LOCAL_PART, the
	// same guard the secret-change path uses.
	stmt, args, err := sqlx.In(selectStmt, hostIDs, "FLEET_VAR_("+strings.Join(varNames, "|")+`)\b`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build select device name hosts for idp change")
	}
	var affectedHostIDs []uint
	if err := sqlx.SelectContext(ctx, tx, &affectedHostIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "select device name hosts for idp change")
	}
	if len(affectedHostIDs) == 0 {
		return nil
	}
	return reconcileHostDeviceNamesForHostsDB(ctx, tx, affectedHostIDs)
}

// queueManagedConfigResendJobs finds android app configs that reference any of
// the affected fleet variables and inserts worker jobs to re-push the managed
// configuration with the updated values.
func queueManagedConfigResendJobs(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, affectedVars []any) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Find app configs that use any of the affected variables.
	const findAffectedApps = `
	SELECT DISTINCT
		aac.application_id,
		vat.id AS app_team_id
	FROM
		mdm_configuration_profile_variables mcpv
		JOIN android_app_configurations aac
			ON mcpv.android_app_configuration_id = aac.id
		JOIN fleet_variables fv
			ON mcpv.fleet_variable_id = fv.id
		JOIN vpp_apps_teams vat
			ON vat.adam_id = aac.application_id AND vat.global_or_team_id = aac.global_or_team_id AND vat.platform = 'android'
		JOIN hosts h
			ON aac.global_or_team_id = COALESCE(h.team_id, 0)
	WHERE
		fv.name IN (?) AND
		h.id IN (?)
`

	findStmt, findArgs, err := sqlx.In(findAffectedApps, affectedVars, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare find affected app configs")
	}

	type affectedApp struct {
		ApplicationID string `db:"application_id"`
		AppTeamID     uint   `db:"app_team_id"`
	}
	var apps []affectedApp
	if err := sqlx.SelectContext(ctx, tx, &apps, findStmt, findArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "find affected app configs")
	}

	if len(apps) == 0 {
		return nil
	}

	// Get the enterprise name from the DB.
	var enterpriseID string
	if err := sqlx.GetContext(ctx, tx, &enterpriseID, `SELECT enterprise_id FROM android_enterprises WHERE enterprise_id != '' LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No enterprise configured — nothing to do.
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get android enterprise id")
	}
	enterpriseName := "enterprises/" + enterpriseID

	// Insert a job for each affected app config.
	const insertJob = `
	INSERT INTO jobs (name, args, state, error)
	VALUES (?, ?, 'queued', '')
`
	for _, app := range apps {
		args, err := json.Marshal(map[string]any{
			"task":               "make_android_app_available",
			"application_id":     app.ApplicationID,
			"app_team_id":        app.AppTeamID,
			"enterprise_name":    enterpriseName,
			"app_config_changed": true,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal job args for managed config resend")
		}
		if _, err := tx.ExecContext(ctx, insertJob, "software_worker", json.RawMessage(args)); err != nil {
			return ctxerr.Wrap(ctx, err, "insert managed config resend job")
		}
	}

	return nil
}

// emailsRequireUpdate compares two slices of emails and returns true if they are different
// and require an update in the database.
func emailsRequireUpdate(currentEmails, newEmails []fleet.ScimUserEmail) bool {
	if len(currentEmails) != len(newEmails) {
		return true
	}

	// Create maps for efficient comparison
	currentEmailMap := make(map[string]fleet.ScimUserEmail)
	for i := range currentEmails {
		key := currentEmails[i].GenerateComparisonKey()
		currentEmailMap[key] = currentEmails[i]
	}

	// Check if all new emails exist in current emails with the same attributes
	for i := range newEmails {
		key := newEmails[i].GenerateComparisonKey()
		if _, exists := currentEmailMap[key]; !exists {
			return true
		}
	}

	return false
}

// ScimUsersExist checks if all the provided SCIM user IDs exist in the datastore
// If the slice is empty, it returns true
// This method processes IDs in batches to handle large numbers of IDs efficiently
func (ds *Datastore) ScimUsersExist(ctx context.Context, ids []uint) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}

	// Create a map to track which IDs we've found
	foundIDs := make(map[uint]bool, len(ids))

	batchSize := 10000
	err := common_mysql.BatchProcessSimple(ids, batchSize, func(batchIDs []uint) error {
		query, args, err := sqlx.In(`
			SELECT id
			FROM scim_users
			WHERE id IN (?)
		`, batchIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare scim users exist batch query")
		}

		var foundBatchIDs []uint
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &foundBatchIDs, query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "check if scim users exist in batch")
		}

		// Mark found IDs
		for _, id := range foundBatchIDs {
			foundIDs[id] = true
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	// Check if all IDs were found
	for _, id := range ids {
		if !foundIDs[id] {
			return false, nil
		}
	}

	return true, nil
}
