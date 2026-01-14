package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockservice "github.com/fleetdm/fleet/v4/server/mock/service"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteMatchingFleetUser(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("no emails in SCIM user", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "johndoe",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)
		assert.False(t, ds.UserByEmailFuncInvoked)
	})

	t.Run("userName is email, matches Fleet user", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "John Doe",
			Email:      "john@example.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			if email == "john@example.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}

		var activityCreated bool
		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityCreated = true
			deleted, ok := activity.(fleet.ActivityTypeDeletedUser)
			require.True(t, ok)
			assert.Equal(t, uint(100), deleted.UserID)
			assert.Equal(t, "John Doe", deleted.UserName)
			assert.Equal(t, "john@example.com", deleted.UserEmail)
			assert.True(t, deleted.FromScimUserDeletion)
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "john@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, ds.UserByEmailFuncInvoked)
		assert.True(t, ds.DeleteUserFuncInvoked)
		assert.True(t, activityCreated)
	})

	t.Run("skips deletion of API-only user", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "API User",
			Email:      "api@example.com",
			GlobalRole: ptr.String(fleet.RoleAdmin),
			APIOnly:    true,
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "api@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, ds.UserByEmailFuncInvoked)
		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("prevents deleting last global admin", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "Admin User",
			Email:      "admin@example.com",
			GlobalRole: ptr.String(fleet.RoleAdmin),
			APIOnly:    false,
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.CountGlobalAdminsFunc = func(ctx context.Context) (int, error) {
			return 1, nil // Only 1 admin
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "admin@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete last global admin")

		assert.True(t, ds.UserByEmailFuncInvoked)
		assert.True(t, ds.CountGlobalAdminsFuncInvoked)
		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("allows deleting admin when multiple admins exist", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "Admin User",
			Email:      "admin@example.com",
			GlobalRole: ptr.String(fleet.RoleAdmin),
			APIOnly:    false,
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.CountGlobalAdminsFunc = func(ctx context.Context) (int, error) {
			return 3, nil // Multiple admins
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "admin@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("matches on scim_user_emails when userName is not email", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "Jane Doe",
			Email:      "jane@work.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			if email == "jane@work.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "janedoe", // Not an email
			Emails: []fleet.ScimUserEmail{
				{Email: "jane@personal.com"},
				{Email: "jane@work.com"},
			},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("no matching Fleet user found", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return nil, platform_mysql.NotFound("User")
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "nobody@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, ds.UserByEmailFuncInvoked)
		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("email case insensitive matching", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "User",
			Email:      "user@example.com", // lowercase
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		var emailQueried string
		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			emailQueried = email
			if email == "user@example.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "USER@EXAMPLE.COM",
			Emails:   []fleet.ScimUserEmail{},
		}

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.Equal(t, "user@example.com", emailQueried)
		assert.True(t, ds.DeleteUserFuncInvoked)
	})
}

func TestUserHandlerDelete(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("deletes SCIM user and matching Fleet user", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "user@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "User",
			Email:      "user@example.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return scimUser, nil
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		ds.DeleteScimUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/1", nil)
		err := handler.Delete(req, "1")
		require.NoError(t, err)

		assert.True(t, ds.ScimUserByIDFuncInvoked)
		assert.True(t, ds.DeleteUserFuncInvoked)
		assert.True(t, ds.DeleteScimUserFuncInvoked)
	})

	t.Run("SCIM deletion proceeds even if Fleet user deletion fails", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		scimUser := &fleet.ScimUser{
			ID:       1,
			UserName: "admin@example.com",
			Emails:   []fleet.ScimUserEmail{},
		}

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "Admin",
			Email:      "admin@example.com",
			GlobalRole: ptr.String(fleet.RoleAdmin),
			APIOnly:    false,
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return scimUser, nil
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		// Last admin - Fleet user deletion will fail
		ds.CountGlobalAdminsFunc = func(ctx context.Context) (int, error) {
			return 1, nil
		}

		// SCIM user deletion should still succeed
		ds.DeleteScimUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/1", nil)
		err := handler.Delete(req, "1")
		require.NoError(t, err)

		assert.True(t, ds.DeleteScimUserFuncInvoked)
		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("returns error when SCIM user not found", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return nil, platform_mysql.NotFound("ScimUser")
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/999", nil)
		err := handler.Delete(req, "999")
		require.Error(t, err)
	})
}
