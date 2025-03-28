package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// CreateScimUser creates a new SCIM user in the database
func (ds *Datastore) CreateScimUser(ctx context.Context, user *fleet.ScimUser) (uint, error) {
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

	return user, nil
}

// ReplaceScimUser replaces an existing SCIM user in the database
func (ds *Datastore) ReplaceScimUser(ctx context.Context, user *fleet.ScimUser) error {
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

		return insertEmails(ctx, tx, user)
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
		opts.PerPage = 100
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
