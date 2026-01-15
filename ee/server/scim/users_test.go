package scim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockservice "github.com/fleetdm/fleet/v4/server/mock/service"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/scim2/filter-parser/v2"
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

func TestWasDeactivated(t *testing.T) {
	tests := []struct {
		name     string
		previous *bool
		current  *bool
		expected bool
	}{
		{
			name:     "nil to false - deactivated",
			previous: nil,
			current:  ptr.Bool(false),
			expected: true,
		},
		{
			name:     "true to false - deactivated",
			previous: ptr.Bool(true),
			current:  ptr.Bool(false),
			expected: true,
		},
		{
			name:     "false to false - not deactivated (already inactive)",
			previous: ptr.Bool(false),
			current:  ptr.Bool(false),
			expected: false,
		},
		{
			name:     "nil to nil - not deactivated",
			previous: nil,
			current:  nil,
			expected: false,
		},
		{
			name:     "nil to true - not deactivated",
			previous: nil,
			current:  ptr.Bool(true),
			expected: false,
		},
		{
			name:     "true to true - not deactivated",
			previous: ptr.Bool(true),
			current:  ptr.Bool(true),
			expected: false,
		},
		{
			name:     "false to true - not deactivated (reactivated)",
			previous: ptr.Bool(false),
			current:  ptr.Bool(true),
			expected: false,
		},
		{
			name:     "true to nil - not deactivated",
			previous: ptr.Bool(true),
			current:  nil,
			expected: false,
		},
		{
			name:     "false to nil - not deactivated",
			previous: ptr.Bool(false),
			current:  nil,
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := wasDeactivated(tc.previous, tc.current)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUserHandlerReplaceDeactivation(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("deletes Fleet user when SCIM user is deactivated via Replace", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Currently active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "User",
			Email:      "user@example.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			deleted, ok := activity.(fleet.ActivityTypeDeletedUser)
			require.True(t, ok)
			assert.True(t, deleted.FromScimUserDeletion)
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil)

		// Replace with active: false (deactivation)
		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   false, // Deactivating
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Replace(req, "1", attrs)
		require.NoError(t, err)

		assert.True(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when active state unchanged", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Currently active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		// Replace with active: true (no change)
		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   true,
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.NoError(t, err)

		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when already inactive", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(false), // Already inactive
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		// Replace with active: false (already false, not a deactivation)
		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   false,
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.NoError(t, err)

		assert.False(t, ds.DeleteUserFuncInvoked)
	})
}

func TestUserHandlerPatchDeactivation(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("deletes Fleet user when SCIM user is deactivated via Patch with path", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Currently active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "User",
			Email:      "user@example.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		// Use scim.PatchOperation with path
		activePath, err := filter.ParsePath([]byte("active"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{
				Op:    scim.PatchOperationReplace,
				Path:  &activePath,
				Value: false,
			},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.True(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("deletes Fleet user when SCIM user is deactivated via Patch without path", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Currently active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		fleetUser := &fleet.User{
			ID:         100,
			Name:       "User",
			Email:      "user@example.com",
			GlobalRole: ptr.String(fleet.RoleMaintainer),
			APIOnly:    false,
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}

		svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		// Use scim.PatchOperation without path (value contains attribute name)
		patchOps := []scim.PatchOperation{
			{
				Op:   scim.PatchOperationReplace,
				Path: nil,
				Value: map[string]any{
					"active": false,
				},
			},
		}

		_, err := handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.True(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when active unchanged via Patch", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Currently active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		// Patch that doesn't change active status
		givenNamePath, err := filter.ParsePath([]byte("name.givenName"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{
				Op:    scim.PatchOperationReplace,
				Path:  &givenNamePath,
				Value: "Jane",
			},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.False(t, ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when already inactive", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(false), // Already inactive
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		// Patch with active: false but already inactive
		activePath, err := filter.ParsePath([]byte("active"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{
				Op:    scim.PatchOperationReplace,
				Path:  &activePath,
				Value: false,
			},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.False(t, ds.DeleteUserFuncInvoked)
	})
}

func TestUserHandlerCreateReactivation(t *testing.T) {
	logger := kitlog.NewNopLogger()

	t.Run("reactivates deactivated user via Create", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(false), // Deactivated
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		var replacedUser *fleet.ScimUser
		ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			replacedUser = user
			return nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)

		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   true,
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		resource, err := handler.Create(req, attrs)
		require.NoError(t, err)

		// Verify the user was reactivated (Replace was called, not Create)
		assert.True(t, ds.ReplaceScimUserFuncInvoked)
		assert.False(t, ds.CreateScimUserFuncInvoked)

		// Verify the active status is set to true
		require.NotNil(t, replacedUser)
		require.NotNil(t, replacedUser.Active)
		assert.True(t, *replacedUser.Active)

		// Verify the returned resource has the correct ID
		assert.Equal(t, "1", resource.ID)
	})

	t.Run("returns uniqueness error when active not explicitly true", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(false), // Deactivated
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)

		// Attributes without explicit active field - should NOT reactivate
		attrs := map[string]any{
			"userName": "user@example.com",
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, ds.ReplaceScimUserFuncInvoked)
		assert.False(t, ds.CreateScimUserFuncInvoked)
	})

	t.Run("returns uniqueness error for active user", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     ptr.Bool(true), // Already active
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)

		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   true,
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, ds.ReplaceScimUserFuncInvoked)
		assert.False(t, ds.CreateScimUserFuncInvoked)
	})

	t.Run("returns uniqueness error for user with nil active", func(t *testing.T) {
		ds := new(mock.Store)
		svc := new(mockservice.Service)

		existingScimUser := &fleet.ScimUser{
			ID:         1,
			UserName:   "user@example.com",
			Active:     nil, // Active is nil (not explicitly deactivated)
			GivenName:  ptr.String("John"),
			FamilyName: ptr.String("Doe"),
			Emails:     []fleet.ScimUserEmail{},
		}

		ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := &UserHandler{ds: ds, svc: svc, logger: logger}

		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)

		attrs := map[string]any{
			"userName": "user@example.com",
			"active":   true,
			"name": map[string]any{
				"givenName":  "John",
				"familyName": "Doe",
			},
		}

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, ds.ReplaceScimUserFuncInvoked)
		assert.False(t, ds.CreateScimUserFuncInvoked)
	})
}
