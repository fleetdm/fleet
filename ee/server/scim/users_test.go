package scim

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elimity-com/scim"
	scimerrors "github.com/elimity-com/scim/errors"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockservice "github.com/fleetdm/fleet/v4/server/mock/service"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/scim2/filter-parser/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alreadyExistsErr implements fleet.AlreadyExistsError for testing.
type alreadyExistsErr struct {
	msg string
}

func (e *alreadyExistsErr) Error() string  { return e.msg }
func (e *alreadyExistsErr) IsExists() bool { return true }

type testMocks struct {
	ds  *mock.Store
	svc *mockservice.Service
}

func newTestMocks() *testMocks {
	return &testMocks{
		ds:  new(mock.Store),
		svc: new(mockservice.Service),
	}
}

func (m *testMocks) newTestHandler() *UserHandler {
	return &UserHandler{
		ds:          m.ds,
		newActivity: m.svc.NewActivity,
		logger:      slog.New(slog.DiscardHandler),
	}
}

type fleetUserOpts struct {
	id         uint
	name       string
	email      string
	globalRole string
	apiOnly    bool
	ssoEnabled bool
}

func newTestFleetUser(opts *fleetUserOpts) *fleet.User {
	user := &fleet.User{
		ID:         100,
		Name:       "Test User",
		Email:      "user@example.com",
		GlobalRole: ptr.String(fleet.RoleMaintainer),
		APIOnly:    false,
		SSOEnabled: true,
	}
	if opts != nil {
		if opts.id != 0 {
			user.ID = opts.id
		}
		if opts.name != "" {
			user.Name = opts.name
		}
		if opts.email != "" {
			user.Email = opts.email
		}
		if opts.globalRole != "" {
			user.GlobalRole = ptr.String(opts.globalRole)
		}
		user.APIOnly = opts.apiOnly
		user.SSOEnabled = opts.ssoEnabled
	}
	return user
}

type scimUserOpts struct {
	id         uint
	userName   string
	active     *bool
	givenName  string
	familyName string
	emails     []fleet.ScimUserEmail
}

func newTestScimUser(opts *scimUserOpts) *fleet.ScimUser {
	user := &fleet.ScimUser{
		ID:       1,
		UserName: "user@example.com",
		Emails:   []fleet.ScimUserEmail{},
	}
	if opts != nil {
		if opts.id != 0 {
			user.ID = opts.id
		}
		if opts.userName != "" {
			user.UserName = opts.userName
		}
		if opts.active != nil {
			user.Active = opts.active
		}
		if opts.givenName != "" {
			user.GivenName = ptr.String(opts.givenName)
		}
		if opts.familyName != "" {
			user.FamilyName = ptr.String(opts.familyName)
		}
		if opts.emails != nil {
			user.Emails = opts.emails
		}
	}
	return user
}

func newTestAttrs(userName string, active *bool, givenName, familyName string) map[string]any {
	attrs := map[string]any{
		"userName": userName,
		"name": map[string]any{
			"givenName":  givenName,
			"familyName": familyName,
		},
	}
	if active != nil {
		attrs["active"] = *active
	}
	return attrs
}

func TestDeleteMatchingFleetUser(t *testing.T) {
	t.Run("no emails in SCIM user", func(t *testing.T) {
		mocks := newTestMocks()
		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "johndoe"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)
		assert.False(t, mocks.ds.UserByEmailFuncInvoked)
	})

	t.Run("userName is email, matches Fleet user", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{name: "John Doe", email: "john@example.com", ssoEnabled: true})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			if email == "john@example.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}

		var activityCreated bool
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityCreated = true
			deleted, ok := activity.(fleet.ActivityTypeDeletedUser)
			require.True(t, ok)
			assert.Equal(t, uint(100), deleted.UserID)
			assert.Equal(t, "John Doe", deleted.UserName)
			assert.Equal(t, "john@example.com", deleted.UserEmail)
			assert.True(t, deleted.FromScimUserDeletion)
			return nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "john@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.UserByEmailFuncInvoked)
		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
		assert.True(t, activityCreated)
	})

	t.Run("skips deletion of API-only user", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{
			email:      "api@example.com",
			globalRole: fleet.RoleAdmin,
			apiOnly:    true,
			ssoEnabled: true,
		})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "api@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.UserByEmailFuncInvoked)
		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("skips deletion of non-SSO user", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{
			email:      "nonsso@example.com",
			ssoEnabled: false,
		})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "nonsso@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.UserByEmailFuncInvoked)
		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("prevents deleting last global admin", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{
			email:      "admin@example.com",
			globalRole: fleet.RoleAdmin,
			ssoEnabled: true,
		})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserIfNotLastAdminFunc = func(ctx context.Context, id uint) error {
			return fleet.ErrLastGlobalAdmin
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "admin@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete last global admin")

		assert.True(t, mocks.ds.UserByEmailFuncInvoked)
		assert.True(t, mocks.ds.DeleteUserIfNotLastAdminFuncInvoked)
		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("allows deleting admin when multiple admins exist", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{
			email:      "admin@example.com",
			globalRole: fleet.RoleAdmin,
			ssoEnabled: true,
		})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserIfNotLastAdminFunc = func(ctx context.Context, id uint) error {
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "admin@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteUserIfNotLastAdminFuncInvoked)
	})

	t.Run("matches on scim_user_emails when userName is not email", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{
			name:       "Jane Doe",
			email:      "jane@work.com",
			ssoEnabled: true,
		})

		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			if email == "jane@work.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{
			userName: "janedoe", // Not an email
			emails: []fleet.ScimUserEmail{
				{Email: "jane@personal.com"},
				{Email: "jane@work.com"},
			},
		})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("no matching Fleet user found", func(t *testing.T) {
		mocks := newTestMocks()
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return nil, platform_mysql.NotFound("User")
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "nobody@example.com"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.True(t, mocks.ds.UserByEmailFuncInvoked)
		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("email case insensitive matching", func(t *testing.T) {
		mocks := newTestMocks()
		fleetUser := newTestFleetUser(&fleetUserOpts{ssoEnabled: true})

		var emailQueried string
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			emailQueried = email
			if email == "user@example.com" {
				return fleetUser, nil
			}
			return nil, platform_mysql.NotFound("User")
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()
		scimUser := newTestScimUser(&scimUserOpts{userName: "USER@EXAMPLE.COM"})

		err := handler.deleteMatchingFleetUser(t.Context(), scimUser)
		require.NoError(t, err)

		assert.Equal(t, "user@example.com", emailQueried)
		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
	})
}

func TestUserHandlerDelete(t *testing.T) {
	t.Run("deletes SCIM user and matching Fleet user", func(t *testing.T) {
		mocks := newTestMocks()
		scimUser := newTestScimUser(nil)
		fleetUser := newTestFleetUser(&fleetUserOpts{ssoEnabled: true})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return scimUser, nil
		}
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}
		mocks.ds.DeleteScimUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/1", nil)
		err := handler.Delete(req, "1")
		require.NoError(t, err)

		assert.True(t, mocks.ds.ScimUserByIDFuncInvoked)
		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
		assert.True(t, mocks.ds.DeleteScimUserFuncInvoked)
	})

	t.Run("SCIM deletion proceeds even if Fleet user deletion fails", func(t *testing.T) {
		mocks := newTestMocks()
		scimUser := newTestScimUser(&scimUserOpts{userName: "admin@example.com"})
		fleetUser := newTestFleetUser(&fleetUserOpts{
			email:      "admin@example.com",
			globalRole: fleet.RoleAdmin,
			ssoEnabled: true,
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return scimUser, nil
		}
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		// Last admin - Fleet user deletion will fail
		mocks.ds.DeleteUserIfNotLastAdminFunc = func(ctx context.Context, id uint) error {
			return fleet.ErrLastGlobalAdmin
		}
		// SCIM user deletion should still succeed
		mocks.ds.DeleteScimUserFunc = func(ctx context.Context, id uint) error {
			return nil
		}

		handler := mocks.newTestHandler()

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/1", nil)
		err := handler.Delete(req, "1")
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteScimUserFuncInvoked)
		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("returns error when SCIM user not found", func(t *testing.T) {
		mocks := newTestMocks()
		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return nil, platform_mysql.NotFound("ScimUser")
		}
		// DeleteScimUser is still called to ensure triggerResendProfilesForIDPUserDeleted runs
		mocks.ds.DeleteScimUserFunc = func(ctx context.Context, id uint) error {
			return platform_mysql.NotFound("ScimUser")
		}

		handler := mocks.newTestHandler()

		req := httptest.NewRequest(http.MethodDelete, "/scim/v2/Users/999", nil)
		err := handler.Delete(req, "999")
		require.Error(t, err)
		assert.True(t, mocks.ds.DeleteScimUserFuncInvoked, "DeleteScimUser should be called even when ScimUserByID returns not found")
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
	t.Run("deletes Fleet user when SCIM user is deactivated via Replace", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})
		fleetUser := newTestFleetUser(&fleetUserOpts{ssoEnabled: true})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			deleted, ok := activity.(fleet.ActivityTypeDeletedUser)
			require.True(t, ok)
			assert.True(t, deleted.FromScimUserDeletion)
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(false), "John", "Doe")

		_, err := handler.Replace(req, "1", attrs)
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when active state unchanged", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := mocks.newTestHandler()
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.NoError(t, err)

		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when already inactive", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(false), // Already inactive
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := mocks.newTestHandler()
		attrs := newTestAttrs("user@example.com", ptr.Bool(false), "John", "Doe")

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.NoError(t, err)

		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("returns uniqueness error when ReplaceScimUser returns wrapped AlreadyExistsError", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		// Pre-check passes (no other user with this username)
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return nil, platform_mysql.NotFound("ScimUser")
		}
		// ReplaceScimUser returns a wrapped AlreadyExistsError (race condition: concurrent update took the username)
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return fmt.Errorf("update scim user: %w", &alreadyExistsErr{msg: "user_name already exists"})
		}

		handler := mocks.newTestHandler()
		attrs := newTestAttrs("taken@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.Error(t, err)

		scimErr, ok := err.(scimerrors.ScimError)
		require.True(t, ok, "expected ScimError, got %T: %v", err, err)
		assert.Equal(t, http.StatusConflict, scimErr.Status)
		assert.Equal(t, scimerrors.ScimTypeUniqueness, scimErr.ScimType)
	})

	t.Run("returns bad params error when ReplaceScimUser returns SCIMValidationError", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		// ReplaceScimUser returns a validation error (field too long)
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return &fleet.SCIMValidationError{Field: "given_name", Message: "exceeds maximum length of 255 characters"}
		}

		handler := mocks.newTestHandler()
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Replace(httptest.NewRequest(http.MethodPut, "/scim/v2/Users/1", nil), "1", attrs)
		require.Error(t, err)

		scimErr, ok := err.(scimerrors.ScimError)
		require.True(t, ok, "expected ScimError, got %T: %v", err, err)
		assert.Equal(t, http.StatusBadRequest, scimErr.Status)
		assert.Contains(t, scimErr.Detail, "given_name")
	})
}

func TestUserHandlerPatchDeactivation(t *testing.T) {
	t.Run("deletes Fleet user when SCIM user is deactivated via Patch with path", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})
		fleetUser := newTestFleetUser(&fleetUserOpts{ssoEnabled: true})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		activePath, err := filter.ParsePath([]byte("active"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &activePath, Value: false},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("deletes Fleet user when SCIM user is deactivated via Patch without path", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})
		fleetUser := newTestFleetUser(&fleetUserOpts{ssoEnabled: true})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}
		mocks.ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
			return fleetUser, nil
		}
		mocks.ds.DeleteUserFunc = func(ctx context.Context, id uint) error {
			assert.Equal(t, uint(100), id)
			return nil
		}
		mocks.svc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		patchOps := []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: nil, Value: map[string]any{"active": false}},
		}

		_, err := handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.True(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when active unchanged via Patch", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		givenNamePath, err := filter.ParsePath([]byte("name.givenName"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &givenNamePath, Value: "Jane"},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("does not delete Fleet user when already inactive", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(false), // Already inactive
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		activePath, err := filter.ParsePath([]byte("active"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &activePath, Value: false},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.NoError(t, err)

		assert.False(t, mocks.ds.DeleteUserFuncInvoked)
	})

	t.Run("returns uniqueness error when ReplaceScimUser returns wrapped AlreadyExistsError via Patch", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true),
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}
		// ReplaceScimUser returns a wrapped AlreadyExistsError (race condition: concurrent update took the username)
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			return fmt.Errorf("update scim user: %w", &alreadyExistsErr{msg: "user_name already exists"})
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		userNamePath, err := filter.ParsePath([]byte("userName"))
		require.NoError(t, err)

		patchOps := []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &userNamePath, Value: "taken@example.com"},
		}

		_, err = handler.Patch(req, "1", patchOps)
		require.Error(t, err)

		scimErr, ok := err.(scimerrors.ScimError)
		require.True(t, ok, "expected ScimError, got %T: %v", err, err)
		assert.Equal(t, http.StatusConflict, scimErr.Status)
		assert.Equal(t, scimerrors.ScimTypeUniqueness, scimErr.ScimType)
	})
}

func TestUserHandlerCreateReactivation(t *testing.T) {
	t.Run("reactivates deactivated user via Create", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(false), // Deactivated
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		var replacedUser *fleet.ScimUser
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			replacedUser = user
			return nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		resource, err := handler.Create(req, attrs)
		require.NoError(t, err)

		// Verify the user was reactivated (Replace was called, not Create)
		assert.True(t, mocks.ds.ReplaceScimUserFuncInvoked)
		assert.False(t, mocks.ds.CreateScimUserFuncInvoked)

		// Verify the active status is set to true
		require.NotNil(t, replacedUser)
		require.NotNil(t, replacedUser.Active)
		assert.True(t, *replacedUser.Active)

		// Verify the returned resource has the correct ID
		assert.Equal(t, "1", resource.ID)
	})

	t.Run("returns uniqueness error when active not explicitly true", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(false), // Deactivated
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		// Attributes without explicit active field - should NOT reactivate
		attrs := newTestAttrs("user@example.com", nil, "John", "Doe")

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, mocks.ds.ReplaceScimUserFuncInvoked)
		assert.False(t, mocks.ds.CreateScimUserFuncInvoked)
	})

	t.Run("returns uniqueness error for active user", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			active:     ptr.Bool(true), // Already active
			givenName:  "John",
			familyName: "Doe",
		})

		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, mocks.ds.ReplaceScimUserFuncInvoked)
		assert.False(t, mocks.ds.CreateScimUserFuncInvoked)
	})

	t.Run("returns uniqueness error when CreateScimUser returns wrapped AlreadyExistsError", func(t *testing.T) {
		mocks := newTestMocks()

		// No existing user found during pre-check (simulating race condition)
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return nil, platform_mysql.NotFound("ScimUser")
		}

		// CreateScimUser returns a wrapped AlreadyExistsError (as ctxerr.Wrap produces in production)
		mocks.ds.CreateScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) (uint, error) {
			return 0, fmt.Errorf("insert scim user: %w", &alreadyExistsErr{msg: "user_name already exists"})
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should be a SCIM uniqueness error (409), not a 500
		scimErr, ok := err.(scimerrors.ScimError)
		require.True(t, ok, "expected ScimError, got %T: %v", err, err)
		assert.Equal(t, http.StatusConflict, scimErr.Status)
		assert.Equal(t, scimerrors.ScimTypeUniqueness, scimErr.ScimType)
	})

	t.Run("returns bad params error when CreateScimUser returns SCIMValidationError", func(t *testing.T) {
		mocks := newTestMocks()

		// No existing user found during pre-check
		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return nil, platform_mysql.NotFound("ScimUser")
		}

		// CreateScimUser returns a validation error (field too long)
		mocks.ds.CreateScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) (uint, error) {
			return 0, &fleet.SCIMValidationError{Field: "user_name", Message: "exceeds maximum length of 255 characters"}
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should be a SCIM bad params error (400), not a 500
		scimErr, ok := err.(scimerrors.ScimError)
		require.True(t, ok, "expected ScimError, got %T: %v", err, err)
		assert.Equal(t, http.StatusBadRequest, scimErr.Status)
		assert.Contains(t, scimErr.Detail, "user_name")
	})

	t.Run("returns uniqueness error for user with nil active", func(t *testing.T) {
		mocks := newTestMocks()
		existingScimUser := newTestScimUser(&scimUserOpts{
			givenName:  "John",
			familyName: "Doe",
		}) // Active is nil (not explicitly deactivated)

		mocks.ds.ScimUserByUserNameFunc = func(ctx context.Context, userName string) (*fleet.ScimUser, error) {
			return existingScimUser, nil
		}

		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPost, "/scim/v2/Users", nil)
		attrs := newTestAttrs("user@example.com", ptr.Bool(true), "John", "Doe")

		_, err := handler.Create(req, attrs)
		require.Error(t, err)

		// Should not have called Replace or Create
		assert.False(t, mocks.ds.ReplaceScimUserFuncInvoked)
		assert.False(t, mocks.ds.CreateScimUserFuncInvoked)
	})
}

func TestUserHandlerPatchUnknownAttributes(t *testing.T) {
	const enterpriseExtURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"

	setupMocks := func(t *testing.T) (*testMocks, *fleet.ScimUser, **fleet.ScimUser) {
		t.Helper()
		mocks := newTestMocks()
		existingUser := newTestScimUser(&scimUserOpts{active: new(true), givenName: "John", familyName: "Doe"})
		mocks.ds.ScimUserByIDFunc = func(ctx context.Context, id uint) (*fleet.ScimUser, error) {
			return existingUser, nil
		}
		var saved *fleet.ScimUser
		mocks.ds.ReplaceScimUserFunc = func(ctx context.Context, user *fleet.ScimUser) error {
			saved = user
			return nil
		}
		return mocks, existingUser, &saved
	}

	t.Run("explicit-path with unrecognized path is ignored, department in same batch is applied", func(t *testing.T) {
		mocks, _, saved := setupMocks(t)
		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		deptPath, err := filter.ParsePath([]byte(enterpriseExtURN + ":department"))
		require.NoError(t, err)
		unknownPath, err := filter.ParsePath([]byte(enterpriseExtURN + ":employeeNumber"))
		require.NoError(t, err)

		_, err = handler.Patch(req, "1", []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "EMP-1"},
			{Op: scim.PatchOperationReplace, Path: &deptPath, Value: "Engineering"},
		})
		require.NoError(t, err)
		require.True(t, mocks.ds.ReplaceScimUserFuncInvoked)
		require.NotNil(t, *saved)
		require.NotNil(t, (*saved).Department)
		assert.Equal(t, "Engineering", *(*saved).Department)
	})

	t.Run("no-path value with unrecognized field is ignored, department in same value is applied", func(t *testing.T) {
		mocks, _, saved := setupMocks(t)
		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		_, err := handler.Patch(req, "1", []scim.PatchOperation{
			{
				Op:   scim.PatchOperationReplace,
				Path: nil,
				Value: map[string]any{
					enterpriseExtURN + ":department":     "Sales",
					enterpriseExtURN + ":employeeNumber": "EMP-2",
				},
			},
		})
		require.NoError(t, err)
		require.True(t, mocks.ds.ReplaceScimUserFuncInvoked)
		require.NotNil(t, *saved)
		require.NotNil(t, (*saved).Department)
		assert.Equal(t, "Sales", *(*saved).Department)
	})

	t.Run("patchName ignores unrecognized name subattributes", func(t *testing.T) {
		// givenName/familyName get applied; honorificPrefix and middleName are silently
		// dropped instead of aborting the PATCH.
		mocks, _, saved := setupMocks(t)
		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		_, err := handler.Patch(req, "1", []scim.PatchOperation{
			{
				Op:   scim.PatchOperationReplace,
				Path: nil,
				Value: map[string]any{
					"name": map[string]any{
						"givenName":       "WithPrefix",
						"familyName":      "User",
						"honorificPrefix": "Mr",
						"middleName":      "Q",
					},
				},
			},
		})
		require.NoError(t, err)
		require.True(t, mocks.ds.ReplaceScimUserFuncInvoked)
		require.NotNil(t, *saved)
		require.NotNil(t, (*saved).GivenName)
		require.NotNil(t, (*saved).FamilyName)
		assert.Equal(t, "WithPrefix", *(*saved).GivenName)
		assert.Equal(t, "User", *(*saved).FamilyName)
	})

	t.Run("PATCH with only unrecognized explicit-path operations skips database write", func(t *testing.T) {
		mocks, _, _ := setupMocks(t)
		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		unknownPath, err := filter.ParsePath([]byte(enterpriseExtURN + ":employeeNumber"))
		require.NoError(t, err)

		_, err = handler.Patch(req, "1", []scim.PatchOperation{
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "EMP-1"},
			{Op: scim.PatchOperationReplace, Path: &unknownPath, Value: "EMP-2"},
		})
		require.NoError(t, err)
		assert.False(t, mocks.ds.ReplaceScimUserFuncInvoked)
	})

	t.Run("PATCH with only unrecognized no-path fields skips database write", func(t *testing.T) {
		mocks, _, _ := setupMocks(t)
		handler := mocks.newTestHandler()
		req := httptest.NewRequest(http.MethodPatch, "/scim/v2/Users/1", nil)

		_, err := handler.Patch(req, "1", []scim.PatchOperation{
			{
				Op:   scim.PatchOperationReplace,
				Path: nil,
				Value: map[string]any{
					enterpriseExtURN + ":employeeNumber": "EMP-1",
					enterpriseExtURN + ":costCenter":     "CC-1",
				},
			},
		})
		require.NoError(t, err)
		assert.False(t, mocks.ds.ReplaceScimUserFuncInvoked)
	})
}
