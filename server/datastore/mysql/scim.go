package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	// SCIMMaxFieldLength is the maximum length for SCIM user fields
	SCIMMaxFieldLength = 255

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
			external_id, user_name, given_name, family_name, active
		) VALUES (?, ?, ?, ?, ?)`
		result, err := tx.ExecContext(
			ctx,
			insertUserQuery,
			user.ExternalID,
			user.UserName,
			user.GivenName,
			user.FamilyName,
			user.Active,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim user")
		}

		id, err := result.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert scim user last insert id")
		}
		user.ID = uint(id) // nolint:gosec // dismiss G115
		userID = user.ID

		return insertEmails(ctx, tx, user)
	})
	return userID, err
}

// ScimUserByID retrieves a SCIM user by ID
func (ds *Datastore) ScimUserByID(ctx context.Context, id uint) (*fleet.ScimUser, error) {
	const query = `
		SELECT
			id, external_id, user_name, given_name, family_name, active
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
	const query = `
		SELECT
			id, external_id, user_name, given_name, family_name, active
		FROM scim_users
		WHERE user_name = ?
	`
	user := &fleet.ScimUser{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), user, query, userName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("scim user")
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user by userName")
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

// ReplaceScimUser replaces an existing SCIM user in the database
func (ds *Datastore) ReplaceScimUser(ctx context.Context, user *fleet.ScimUser) error {
	if err := validateScimUserFields(user); err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Update the SCIM user
		const updateUserQuery = `
		UPDATE scim_users SET
			external_id = ?,
			user_name = ?,
			given_name = ?,
			family_name = ?,
			active = ?
		WHERE id = ?`
		result, err := tx.ExecContext(
			ctx,
			updateUserQuery,
			user.ExternalID,
			user.UserName,
			user.GivenName,
			user.FamilyName,
			user.Active,
			user.ID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update scim user")
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get rows affected for update scim user")
		}
		if rowsAffected == 0 {
			return notFound("scim user").WithID(user.ID)
		}

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

		// Get the user's groups
		groups, err := ds.getScimUserGroups(ctx, user.ID)
		if err != nil {
			return err
		}
		user.Groups = groups

		return nil
	})
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
func (ds *Datastore) DeleteScimUser(ctx context.Context, id uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Delete all email entries for the user
		const deleteEmailsQuery = `DELETE FROM scim_user_emails WHERE scim_user_id = ?`
		_, err := tx.ExecContext(ctx, deleteEmailsQuery, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete scim user emails")
		}

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
}

// ListScimUsers retrieves a list of SCIM users with optional filtering
func (ds *Datastore) ListScimUsers(ctx context.Context, opts fleet.ScimUsersListOptions) (users []fleet.ScimUser, totalResults uint, err error) {
	// Default pagination values if not provided
	if opts.Page == 0 {
		opts.Page = 1
	}
	if opts.PerPage == 0 {
		opts.PerPage = SCIMDefaultResourcesPerPage
	}

	// Calculate offset for pagination
	offset := (opts.Page - 1) * opts.PerPage

	// Build the base query
	baseQuery := `
		SELECT DISTINCT
			scim_users.id, external_id, user_name, given_name, family_name, active
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
	params = append(params, opts.PerPage, offset)

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
			scim_user_id, group_id
		FROM scim_user_group
		WHERE scim_user_id IN (?)
		ORDER BY group_id ASC
	`, userIDs)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "prepare groups query")
	}

	// Execute the group query
	type userGroup struct {
		UserID  uint `db:"scim_user_id"`
		GroupID uint `db:"group_id"`
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
			user.Groups = append(user.Groups, ug.GroupID)
		}
	}

	return users, totalResults, nil
}

// getScimUserEmails retrieves all emails for a SCIM user
func (ds *Datastore) getScimUserEmails(ctx context.Context, userID uint) ([]fleet.ScimUserEmail, error) {
	const query = `
		SELECT
			scim_user_id, email, ` + "`primary`" + `, type
		FROM scim_user_emails
		WHERE scim_user_id = ? ORDER BY email ASC
	`
	var emails []fleet.ScimUserEmail
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &emails, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user emails")
	}
	return emails, nil
}

// getScimUserGroups retrieves all group IDs for a SCIM user
func (ds *Datastore) getScimUserGroups(ctx context.Context, userID uint) ([]uint, error) {
	const query = `
		SELECT
			group_id
		FROM scim_user_group
		WHERE scim_user_id = ? ORDER BY group_id ASC
	`
	var groupIDs []uint
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &groupIDs, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim user groups")
	}
	return groupIDs, nil
}

// validateScimUserFields checks if the user fields exceed the maximum allowed length
func validateScimUserFields(user *fleet.ScimUser) error {
	if user.ExternalID != nil && len(*user.ExternalID) > SCIMMaxFieldLength {
		return fmt.Errorf("external_id exceeds maximum length of %d characters", SCIMMaxFieldLength)
	}
	if len(user.UserName) > SCIMMaxFieldLength {
		return fmt.Errorf("user_name exceeds maximum length of %d characters", SCIMMaxFieldLength)
	}
	if user.GivenName != nil && len(*user.GivenName) > SCIMMaxFieldLength {
		return fmt.Errorf("given_name exceeds maximum length of %d characters", SCIMMaxFieldLength)
	}
	if user.FamilyName != nil && len(*user.FamilyName) > SCIMMaxFieldLength {
		return fmt.Errorf("family_name exceeds maximum length of %d characters", SCIMMaxFieldLength)
	}
	return nil
}

// validateScimGroupFields checks if the group fields exceed the maximum allowed length
func validateScimGroupFields(group *fleet.ScimGroup) error {
	if group.ExternalID != nil && len(*group.ExternalID) > SCIMMaxFieldLength {
		return fmt.Errorf("external_id exceeds maximum length of %d characters", SCIMMaxFieldLength)
	}
	if len(group.DisplayName) > SCIMMaxFieldLength {
		return fmt.Errorf("display_name exceeds maximum length of %d characters", SCIMMaxFieldLength)
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

		// Insert user-group relationships if any
		if len(group.ScimUsers) > 0 {
			return insertScimGroupUsers(ctx, tx, group.ID, group.ScimUsers)
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
		) VALUES ` + strings.Join(valueStrings, ",")

		// Execute the batch insert
		_, err := tx.ExecContext(ctx, insertQuery, valueArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "batch insert scim group users")
		}
		return nil
	})
}

// ScimGroupByID retrieves a SCIM group by ID
func (ds *Datastore) ScimGroupByID(ctx context.Context, id uint) (*fleet.ScimGroup, error) {
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

	// Get the group's users
	users, err := ds.getScimGroupUsers(ctx, ds.reader(ctx), id)
	if err != nil {
		return nil, err
	}
	group.ScimUsers = users

	return group, nil
}

// ScimGroupByDisplayName retrieves a SCIM group by display name
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

	// Get the group's users
	users, err := ds.getScimGroupUsers(ctx, ds.reader(ctx), group.ID)
	if err != nil {
		return nil, err
	}
	group.ScimUsers = users

	return group, nil
}

// getScimGroupUsers retrieves all user IDs for a SCIM group
func (ds *Datastore) getScimGroupUsers(ctx context.Context, q sqlx.QueryerContext, groupID uint) ([]uint, error) {
	const query = `
		SELECT
			scim_user_id
		FROM scim_user_group
		WHERE group_id = ? ORDER BY scim_user_id ASC
	`
	var userIDs []uint
	err := sqlx.SelectContext(ctx, q, &userIDs, query, groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "select scim group users")
	}
	return userIDs, nil
}

// ReplaceScimGroup replaces an existing SCIM group in the database
func (ds *Datastore) ReplaceScimGroup(ctx context.Context, group *fleet.ScimGroup) error {
	if err := validateScimGroupFields(group); err != nil {
		return err
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Update the SCIM group
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
		if rowsAffected == 0 {
			return notFound("scim group").WithID(group.ID)
		}

		// Get existing user-group relationships
		existingUsers, err := ds.getScimGroupUsers(ctx, tx, group.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get existing scim group users")
		}

		// Create maps for efficient lookup
		existingUserMap := make(map[uint]bool)
		for _, userID := range existingUsers {
			existingUserMap[userID] = true
		}

		newUserMap := make(map[uint]bool)
		for _, userID := range group.ScimUsers {
			newUserMap[userID] = true
		}

		// Find users to add (in new but not in existing)
		var usersToAdd []uint
		for _, userID := range group.ScimUsers {
			if !existingUserMap[userID] {
				usersToAdd = append(usersToAdd, userID)
			}
		}

		// Find users to remove (in existing but not in new)
		var usersToRemove []uint
		for _, userID := range existingUsers {
			if !newUserMap[userID] {
				usersToRemove = append(usersToRemove, userID)
			}
		}

		// Add new user-group relationships
		if len(usersToAdd) > 0 {
			err = insertScimGroupUsers(ctx, tx, group.ID, usersToAdd)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert new scim group users")
			}
		}

		// Remove old user-group relationships
		if len(usersToRemove) > 0 {
			batchSize := 10000
			return common_mysql.BatchProcessSimple(usersToRemove, batchSize, func(usersToRemoveInBatch []uint) error {
				params := make([]interface{}, len(usersToRemoveInBatch)+1)
				params[0] = group.ID
				for i, userID := range usersToRemoveInBatch {
					params[i+1] = userID
				}

				deleteQuery := "DELETE FROM scim_user_group WHERE group_id = ? AND scim_user_id IN (" +
					strings.Repeat("?, ", len(usersToRemoveInBatch)-1) + "?)"

				_, err = tx.ExecContext(ctx, deleteQuery, params...)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "delete removed scim group users")
				}
				return nil
			})
		}

		return nil
	})
}

// DeleteScimGroup deletes a SCIM group from the database
func (ds *Datastore) DeleteScimGroup(ctx context.Context, id uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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
func (ds *Datastore) ListScimGroups(ctx context.Context, opts fleet.ScimListOptions) (groups []fleet.ScimGroup, totalResults uint, err error) {
	// Default pagination values if not provided
	if opts.Page == 0 {
		opts.Page = 1
	}
	if opts.PerPage == 0 {
		opts.PerPage = SCIMDefaultResourcesPerPage
	}

	// Calculate offset for pagination
	offset := (opts.Page - 1) * opts.PerPage

	// Build the query
	baseQuery := `
		SELECT DISTINCT
			scim_groups.id, external_id, display_name
		FROM scim_groups
	`

	// First, get the total count without pagination
	countQuery := "SELECT COUNT(DISTINCT id) FROM (" + baseQuery + ") AS filtered_groups"
	err = sqlx.GetContext(ctx, ds.reader(ctx), &totalResults, countQuery)
	if err != nil {
		return nil, 0, ctxerr.Wrap(ctx, err, "count total scim groups")
	}

	// Add pagination to the main query
	query := baseQuery + " ORDER BY scim_groups.id LIMIT ? OFFSET ?"
	params := []interface{}{opts.PerPage, offset}

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

	return groups, totalResults, nil
}
