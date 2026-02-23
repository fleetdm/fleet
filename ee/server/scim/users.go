package scim

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/modules/activities"
	"github.com/scim2/filter-parser/v2"
)

const (
	// Common attributes: https://datatracker.ietf.org/doc/html/rfc7643#section-3.1
	externalIdAttr = "externalId"

	// User attributes: https://datatracker.ietf.org/doc/html/rfc7643#section-4.1
	userNameAttr   = "userName"
	nameAttr       = "name"
	givenNameAttr  = "givenName"
	familyNameAttr = "familyName"
	activeAttr     = "active"
	emailsAttr     = "emails"
	groupsAttr     = "groups"
	valueAttr      = "value"
	typeAttr       = "type"
	primaryAttr    = "primary"

	extensionEnterpriseUserAttributes = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	departmentAttr                    = "department"
)

type UserHandler struct {
	ds             fleet.Datastore
	activityModule activities.ActivityModule
	logger         *slog.Logger
}

// Compile-time check
var _ scim.ResourceHandler = &UserHandler{}

func NewUserHandler(ds fleet.Datastore, activityModule activities.ActivityModule, logger *slog.Logger) scim.ResourceHandler {
	return &UserHandler{ds: ds, activityModule: activityModule, logger: logger}
}

func (u *UserHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	ctx := r.Context()

	// Check for userName uniqueness
	userName, err := getRequiredResource[string](attributes, userNameAttr)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get userName", "err", err)
		return scim.Resource{}, err
	}
	// In IETF documents, “non-empty” is generally used in the literal sense of “having at least one character.” That means if a value contains one or more spaces (and nothing else), it is still considered non-empty.
	if len(userName) == 0 {
		u.logger.InfoContext(ctx, "userName is empty")
		return scim.Resource{}, errors.ScimErrorBadParams([]string{userNameAttr})
	}
	existingUser, err := u.ds.ScimUserByUserName(ctx, userName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		u.logger.ErrorContext(ctx, "failed to check for userName uniqueness", userNameAttr, userName, "err", err)
		return scim.Resource{}, err
	case err == nil:
		// User exists - check if it's a deactivated user being reactivated
		// Reactivation ONLY happens when existing user has active=false AND incoming request has active=true
		incomingActive, _ := getOptionalResource[bool](attributes, activeAttr)
		if existingUser.Active != nil && !*existingUser.Active && incomingActive != nil && *incomingActive {
			// Reactivate the user by updating their record
			u.logger.InfoContext(ctx, "reactivating deactivated user", userNameAttr, userName)
			user, err := u.createUserFromAttributes(ctx, attributes)
			if err != nil {
				u.logger.ErrorContext(ctx, "failed to create user from attributes for reactivation",
					userNameAttr, userName, "err", err)
				return scim.Resource{}, err
			}
			user.ID = existingUser.ID
			err = u.ds.ReplaceScimUser(ctx, user)
			if err != nil {
				u.logger.ErrorContext(ctx, "failed to reactivate user", userNameAttr, userName, "err", err)
				return scim.Resource{}, err
			}
			return createUserResource(user), nil
		}
		u.logger.InfoContext(ctx, "user already exists", userNameAttr, userName)
		return scim.Resource{}, errors.ScimErrorUniqueness
	}

	user, err := u.createUserFromAttributes(ctx, attributes)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to create user from attributes", userNameAttr, userName, "err", err)
		return scim.Resource{}, err
	}
	user.ID, err = u.ds.CreateScimUser(ctx, user)
	if err != nil {
		return scim.Resource{}, err
	}

	return createUserResource(user), nil
}

func (u *UserHandler) createUserFromAttributes(
	ctx context.Context, attributes scim.ResourceAttributes,
) (*fleet.ScimUser, error) {
	user := fleet.ScimUser{}
	var err error
	user.UserName, err = getRequiredResource[string](attributes, userNameAttr)
	if err != nil {
		return nil, err
	}
	user.ExternalID, err = getOptionalResource[string](attributes, externalIdAttr)
	if err != nil {
		return nil, err
	}
	user.Active, err = getOptionalResource[bool](attributes, activeAttr)
	if err != nil {
		return nil, err
	}
	name, err := getComplexResource(attributes, nameAttr)
	if err != nil {
		return nil, err
	}
	user.FamilyName, err = getOptionalResource[string](name, familyNameAttr)
	if err != nil {
		return nil, err
	}
	if user.FamilyName == nil || len(*user.FamilyName) == 0 {
		return nil, errors.ScimErrorInvalidValue // Disallow non set field and empty value
	}

	user.GivenName, err = getOptionalResource[string](name, givenNameAttr)
	if err != nil {
		return nil, err
	}
	if user.GivenName == nil || len(*user.GivenName) == 0 {
		return nil, errors.ScimErrorInvalidValue // Disallow non set field and empty value
	}
	emails, err := getComplexResourceSlice(attributes, emailsAttr)
	if err != nil {
		return nil, err
	}
	userEmails := make([]fleet.ScimUserEmail, 0, len(emails))
	for _, email := range emails {
		userEmail := fleet.ScimUserEmail{}
		userEmail.Email, err = getRequiredResource[string](email, valueAttr)
		if err != nil {
			return nil, err
		}
		// Service providers SHOULD canonicalize the value according to [RFC5321]
		// https://datatracker.ietf.org/doc/html/rfc7643#section-4.1.2
		userEmail.Email, err = normalizeEmail(userEmail.Email)
		if err != nil {
			return nil, errors.ScimErrorBadParams([]string{valueAttr})
		}
		userEmail.Type, err = getOptionalResource[string](email, typeAttr)
		if err != nil {
			return nil, err
		}
		userEmail.Primary, err = getOptionalResource[bool](email, primaryAttr)
		if err != nil {
			return nil, err
		}
		userEmails = append(userEmails, userEmail)
	}
	user.Emails = userEmails

	// Attempt to get extension enterprise user attributes.
	extendedAttributes := u.getExtensionEnterpriseUserAttributes(ctx, user.UserName, attributes)
	user.Department = extendedAttributes.department

	return &user, nil
}

type extendedAttributes struct {
	department *string
}

func (u *UserHandler) getExtensionEnterpriseUserAttributes(
	ctx context.Context, userName string, attributes scim.ResourceAttributes,
) extendedAttributes {
	var attrs extendedAttributes
	m_, ok := attributes[extensionEnterpriseUserAttributes]
	if !ok {
		return attrs
	}
	m, ok := m_.(map[string]any)
	if !ok {
		u.logger.ErrorContext(ctx,
			fmt.Sprintf("unexpected type for %s: %T", extensionEnterpriseUserAttributes, m_),
			userNameAttr, userName,
		)
		return attrs
	}

	// Attempt to get department attribute.
	if department_, ok := m[departmentAttr]; ok {
		if department, ok := department_.(string); ok {
			attrs.department = &department
		} else {
			u.logger.ErrorContext(ctx,
				fmt.Sprintf("unexpected type for %s.department: %T", extensionEnterpriseUserAttributes, department_),
				userNameAttr, userName,
			)
		}
	}

	return attrs
}

func getRequiredResource[T string | bool](attributes scim.ResourceAttributes, key string) (T, error) {
	var val T
	valIntf, ok := attributes[key]
	if !ok || valIntf == nil {
		return val, errors.ScimErrorBadParams([]string{key})
	}
	val, ok = valIntf.(T)
	if !ok {
		return val, errors.ScimErrorBadParams([]string{key})
	}
	return val, nil
}

func getOptionalResource[T string | bool](attributes scim.ResourceAttributes, key string) (*T, error) {
	var valPtr *T
	valIntf, ok := attributes[key]
	if ok && valIntf != nil {
		val, ok := valIntf.(T)
		if !ok {
			return nil, errors.ScimErrorBadParams([]string{key})
		}
		valPtr = &val
	}
	return valPtr, nil
}

func getComplexResource(attributes scim.ResourceAttributes, key string) (map[string]interface{}, error) {
	valIntf, ok := attributes[key]
	if ok && valIntf != nil {
		val, ok := valIntf.(map[string]interface{})
		if !ok {
			return nil, errors.ScimErrorBadParams([]string{key})
		}
		return val, nil
	}
	return nil, nil
}

func getComplexResourceSlice(attributes scim.ResourceAttributes, key string) ([]map[string]interface{}, error) {
	valIntf, ok := attributes[key]
	if ok && valIntf != nil {
		valSliceIntf, ok := valIntf.([]interface{})
		if !ok {
			return nil, errors.ScimErrorBadParams([]string{key})
		}
		val := make([]map[string]interface{}, 0, len(valSliceIntf))
		for _, v := range valSliceIntf {
			valMap, ok := v.(map[string]interface{})
			if !ok {
				return nil, errors.ScimErrorBadParams([]string{key})
			}
			if len(valMap) > 0 {
				val = append(val, valMap)
			}
		}
		return val, nil
	}
	return nil, nil
}

func (u *UserHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	ctx := r.Context()

	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		u.logger.InfoContext(ctx, "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := u.ds.ScimUserByID(ctx, idUint)
	switch {
	case fleet.IsNotFound(err):
		u.logger.InfoContext(ctx, "failed to find user", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		u.logger.ErrorContext(ctx, "failed to get user", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createUserResource(user), nil
}

func createUserResource(user *fleet.ScimUser) scim.Resource {
	userResource := scim.Resource{}
	userResource.ID = scimUserID(user.ID)
	if user.ExternalID != nil {
		userResource.ExternalID = optional.NewString(*user.ExternalID)
	}
	userResource.Attributes = scim.ResourceAttributes{}
	userResource.Attributes[userNameAttr] = user.UserName
	if user.Active != nil {
		userResource.Attributes[activeAttr] = *user.Active
	}
	if user.FamilyName != nil || user.GivenName != nil {
		userResource.Attributes[nameAttr] = make(scim.ResourceAttributes)
		if user.FamilyName != nil {
			userResource.Attributes[nameAttr].(scim.ResourceAttributes)[familyNameAttr] = *user.FamilyName
		}
		if user.GivenName != nil {
			userResource.Attributes[nameAttr].(scim.ResourceAttributes)[givenNameAttr] = *user.GivenName
		}
	}
	if len(user.Emails) > 0 {
		emails := make([]scim.ResourceAttributes, 0, len(user.Emails))
		for _, email := range user.Emails {
			emailResource := make(scim.ResourceAttributes)
			emailResource[valueAttr] = email.Email
			if email.Type != nil {
				emailResource[typeAttr] = *email.Type
			}
			if email.Primary != nil {
				emailResource[primaryAttr] = *email.Primary
			}
			emails = append(emails, emailResource)
		}
		userResource.Attributes[emailsAttr] = emails
	}
	if len(user.Groups) > 0 {
		groups := make([]scim.ResourceAttributes, 0, len(user.Groups))
		for _, group := range user.Groups {
			groups = append(groups, map[string]interface{}{
				valueAttr: scimGroupID(group.ID),
				"$ref":    "Groups/" + scimGroupID(group.ID),
				"display": group.DisplayName,
			})
		}
		userResource.Attributes[groupsAttr] = groups
	}
	if user.Department != nil {
		extensionEnterpriseUserAttributesMap := make(scim.ResourceAttributes)
		extensionEnterpriseUserAttributesMap[departmentAttr] = *user.Department
		userResource.Attributes[extensionEnterpriseUserAttributes] = extensionEnterpriseUserAttributesMap
	}
	return userResource
}

// GetAll
// Pagination is 1-indexed on the startIndex. The startIndex is the index of the resource (not the index of the page), per RFC7644.
//
// Per RFC7644 3.4.2, SHOULD ignore any query parameters they do not recognize instead of rejecting the query for versioning compatibility reasons
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.4.2
//
// Providers MUST decline to filter results if the specified filter operation is not recognized and return an HTTP 400 error with a
// "scimType" error of "invalidFilter" and an appropriate human-readable response as per Section 3.12.  For example, if a client specified an
// unsupported operator named 'regex', the service provider should specify an error response description identifying the client error,
// e.g., 'The operator 'regex' is not supported.'
//
// If a SCIM service provider determines that too many results would be returned the server base URI, the server SHALL reject the request by
// returning an HTTP response with HTTP status code 400 (Bad Request) and JSON attribute "scimType" set to "tooMany" (see Table 9).
//
// totalResults: The total number of results returned by the list or query operation.  The value may be larger than the number of
// resources returned, such as when returning a single page (see Section 3.4.2.4) of results where multiple pages are available.
func (u *UserHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	ctx := r.Context()

	startIndex := params.StartIndex
	if startIndex < 1 {
		startIndex = 1
	}
	count := params.Count
	if count > maxResults {
		return scim.Page{}, errors.ScimErrorTooMany
	}
	if count < 1 {
		count = maxResults
	}

	opts := fleet.ScimUsersListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: uint(startIndex), // nolint:gosec // ignore G115
			PerPage:    uint(count),      // nolint:gosec // ignore G115
		},
	}
	resourceFilter := r.URL.Query().Get("filter")
	if resourceFilter != "" {
		expr, err := filter.ParseAttrExp([]byte(resourceFilter))
		if err != nil {
			u.logger.ErrorContext(ctx, "failed to parse filter", "filter", resourceFilter, "err", err)
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		if !strings.EqualFold(expr.AttributePath.String(), "userName") || expr.Operator != "eq" {
			u.logger.InfoContext(ctx, "unsupported filter", "filter", resourceFilter)
			return scim.Page{}, nil
		}
		userName, ok := expr.CompareValue.(string)
		if !ok {
			u.logger.ErrorContext(ctx, "unsupported value", "value", expr.CompareValue)
			return scim.Page{}, nil
		}

		// Decode URL-encoded characters in userName, which is required to pass Microsoft Entra ID SCIM Validator
		userName, err = url.QueryUnescape(userName)
		if err != nil {
			u.logger.ErrorContext(ctx, "failed to decode userName", "userName", userName, "err", err)
			return scim.Page{}, nil
		}
		opts.UserNameFilter = &userName
	}
	users, totalResults, err := u.ds.ListScimUsers(ctx, opts)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to list users", "err", err)
		return scim.Page{}, err
	}

	result := scim.Page{
		TotalResults: int(totalResults), // nolint:gosec // ignore G115
		Resources:    make([]scim.Resource, 0, len(users)),
	}
	for _, user := range users {
		result.Resources = append(result.Resources, createUserResource(&user))
	}

	return result, nil
}

func (u *UserHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	ctx := r.Context()

	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		u.logger.InfoContext(ctx, "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := u.createUserFromAttributes(ctx, attributes)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to create user from attributes", "id", id, "err", err)
		return scim.Resource{}, err
	}
	user.ID = idUint

	// Username is unique, so we must check if another user already exists with that username to return a clear error
	// We also use this to get the previous active state when the username isn't changing
	var previousActive *bool
	userWithSameUsername, err := u.ds.ScimUserByUserName(ctx, user.UserName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		u.logger.ErrorContext(ctx, "failed to check for userName uniqueness", userNameAttr, user.UserName, "err", err)
		return scim.Resource{}, err
	case err == nil && user.ID != userWithSameUsername.ID:
		u.logger.InfoContext(ctx, "user already exists with this username", userNameAttr, user.UserName)
		return scim.Resource{}, errors.ScimErrorUniqueness
	case err == nil && user.ID == userWithSameUsername.ID:
		// Same user, username not changing - use this for previous active state
		previousActive = userWithSameUsername.Active
	case fleet.IsNotFound(err):
		// Username is being changed - need to fetch existing user by ID for previous active state
		existingUser, err := u.ds.ScimUserByID(ctx, idUint)
		if fleet.IsNotFound(err) {
			u.logger.InfoContext(ctx, "failed to find scim user by id", "id", id)
			return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
		}
		if err != nil {
			u.logger.ErrorContext(ctx, "failed to get existing scim user by id", "id", id, "err", err)
			return scim.Resource{}, err
		}
		previousActive = existingUser.Active
	}

	err = u.ds.ReplaceScimUser(ctx, user)
	switch {
	case fleet.IsNotFound(err):
		u.logger.InfoContext(ctx, "failed to find user to replace", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		u.logger.ErrorContext(ctx, "failed to replace user", "id", id, "err", err)
		return scim.Resource{}, err
	}

	// Check if user was deactivated and delete matching Fleet user if so
	if wasDeactivated(previousActive, user.Active) {
		if err := u.deleteMatchingFleetUser(ctx, user); err != nil {
			u.logger.ErrorContext(ctx, "failed to delete fleet user on deactivation", "err", err)
		}
	}

	return createUserResource(user), nil
}

// Delete
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.6
// MUST return a 404 (Not Found) error code for all operations associated with the previously deleted resource
func (u *UserHandler) Delete(r *http.Request, id string) error {
	ctx := r.Context()

	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		u.logger.InfoContext(ctx, "failed to parse id", "id", id, "err", err)
		return errors.ScimErrorResourceNotFound(id)
	}

	scimUser, err := u.ds.ScimUserByID(ctx, idUint)
	if fleet.IsNotFound(err) {
		// proceed with DeleteScimUser call which calls triggerResendProfilesForIDPUserDeleted even before checking if the user exists
		u.logger.WarnContext(ctx, "scim user not found", "id", id)
	} else if err != nil {
		u.logger.ErrorContext(ctx, "failed to get scim user", "id", id, "err", err)
		return err
	}

	if scimUser != nil {
		if err := u.deleteMatchingFleetUser(ctx, scimUser); err != nil {
			// Log but don't fail - SCIM deletion should still proceed
			u.logger.ErrorContext(ctx, "failed to delete matching fleet user", "err", err)
		}
	}

	err = u.ds.DeleteScimUser(ctx, idUint)
	switch {
	case fleet.IsNotFound(err):
		u.logger.InfoContext(ctx, "failed to find user to delete", "id", id)
		return errors.ScimErrorResourceNotFound(id)
	case err != nil:
		u.logger.ErrorContext(ctx, "failed to delete user", "id", id, "err", err)
		return err
	}

	return nil
}

// wasDeactivated returns true if the user was deactivated (active changed from true/nil to false)
func wasDeactivated(previous, current *bool) bool {
	// Not deactivated if current is nil or true
	if current == nil || *current {
		return false
	}
	// current is false - deactivated if previous was nil or true
	return previous == nil || *previous
}

func (u *UserHandler) deleteMatchingFleetUser(ctx context.Context, scimUser *fleet.ScimUser) error {
	// Collect unique emails from SCIM user (userName is often the email in many IdP configurations, e.g. Okta).
	// userName is added first so it's checked first when looking up Fleet users.
	emails := make([]string, 0, len(scimUser.Emails)+1)

	if strings.Contains(scimUser.UserName, "@") {
		emails = append(emails, strings.ToLower(scimUser.UserName))
	}

	for _, e := range scimUser.Emails {
		emails = append(emails, strings.ToLower(e.Email))
	}

	emails = server.RemoveDuplicatesFromSlice(emails)

	if len(emails) == 0 {
		u.logger.DebugContext(ctx, "no emails found for scim user",
			"scim_user_id", scimUser.ID, "user_name", scimUser.UserName)
		return nil
	}

	var fleetUser *fleet.User
	for _, email := range emails {
		user, err := u.ds.UserByEmail(ctx, email)
		if err == nil {
			fleetUser = user
			break
		}
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "lookup fleet user by email")
		}
	}

	if fleetUser == nil {
		u.logger.DebugContext(ctx, "no matching fleet user found for scim user",
			"scim_user_id", scimUser.ID, "user_name", scimUser.UserName)
		return nil
	}

	// Skip API-only users or non-SSO users
	if fleetUser.APIOnly || !fleetUser.SSOEnabled {
		u.logger.InfoContext(ctx, "skipping deletion of API-only or non-SSO user",
			"user_id", fleetUser.ID, "email", fleetUser.Email)
		return nil
	}

	// Check if user is a global admin - if so, ensure we're not deleting the last one
	if fleetUser.GlobalRole != nil && *fleetUser.GlobalRole == fleet.RoleAdmin {
		count, err := u.ds.CountGlobalAdmins(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count global admins")
		}

		if count <= 1 {
			u.logger.WarnContext(ctx, "cannot delete last global admin via SCIM",
				"user_id", fleetUser.ID, "email", fleetUser.Email)
			return ctxerr.New(ctx, "cannot delete last global admin")
		}
	}

	u.logger.InfoContext(ctx, "deleting fleet user via SCIM deletion",
		"user_id", fleetUser.ID, "email", fleetUser.Email)

	// TODO: Ideally this should go through a Users service/module instead of directly accessing
	// the datastore. We're in the SCIM domain but accessing the Users datastore which belongs
	// to the Users domain. This would require a larger refactor to introduce a Users module.
	if err := u.ds.DeleteUser(ctx, fleetUser.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete fleet user")
	}

	if err := u.activityModule.NewActivity(
		ctx,
		nil,
		fleet.ActivityTypeDeletedUser{
			UserID:               fleetUser.ID,
			UserName:             fleetUser.Name,
			UserEmail:            fleetUser.Email,
			FromScimUserDeletion: true,
		},
	); err != nil {
		u.logger.ErrorContext(ctx, "failed to create activity for fleet user deletion", "err", err)
	}

	return nil
}

// Patch - https://datatracker.ietf.org/doc/html/rfc7644#section-3.5.2
func (u *UserHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	ctx := r.Context()

	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		u.logger.InfoContext(ctx, "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	user, err := u.ds.ScimUserByID(ctx, idUint)
	switch {
	case fleet.IsNotFound(err):
		u.logger.InfoContext(ctx, "failed to find user to patch", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		u.logger.ErrorContext(ctx, "failed to get user to patch", "id", id, "err", err)
		return scim.Resource{}, err
	}

	// Store previous active state before applying patches
	previousActive := user.Active

	for _, op := range operations {
		if op.Op != scim.PatchOperationAdd && op.Op != scim.PatchOperationReplace && op.Op != scim.PatchOperationRemove {
			u.logger.InfoContext(ctx, "unsupported patch operation", "op", op.Op)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
		switch {
		// If path is not specified, we look for the path in the value attribute.
		case op.Path == nil:
			if op.Op == scim.PatchOperationRemove {
				u.logger.InfoContext(ctx, "the 'path' attribute is REQUIRED for 'remove' operations", "op", op.Op)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			newValues, ok := op.Value.(map[string]interface{})
			if !ok {
				u.logger.InfoContext(ctx, "unsupported patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			for k, v := range newValues {
				switch k {
				case externalIdAttr:
					err = u.patchExternalId(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case userNameAttr:
					err = u.patchUserName(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case activeAttr:
					err = u.patchActive(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case nameAttr + "." + givenNameAttr:
					err = u.patchGivenName(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case nameAttr + "." + familyNameAttr:
					err = u.patchFamilyName(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case nameAttr:
					err = u.patchName(ctx, v, op, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case emailsAttr:
					err = u.patchEmails(ctx, v, op, user)
					if err != nil {
						return scim.Resource{}, err
					}
				case extensionEnterpriseUserAttributes + ":" + departmentAttr:
					err = u.patchDepartment(ctx, op.Op, v, user)
					if err != nil {
						return scim.Resource{}, err
					}
				default:
					u.logger.InfoContext(ctx, "unsupported patch value field", "field", k)
					return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
				}
			}
		case op.Path.String() == externalIdAttr:
			err = u.patchExternalId(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == userNameAttr:
			err = u.patchUserName(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == activeAttr:
			err = u.patchActive(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == nameAttr+"."+givenNameAttr:
			err = u.patchGivenName(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == nameAttr+"."+familyNameAttr:
			err = u.patchFamilyName(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == nameAttr:
			err = u.patchName(ctx, op.Value, op, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == emailsAttr:
			err = u.patchEmails(ctx, op.Value, op, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.AttributePath.String() == emailsAttr:
			err = u.patchEmailsWithPathFiltering(ctx, op, user)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.AttributePath.String() == extensionEnterpriseUserAttributes+":"+departmentAttr:
			err = u.patchDepartment(ctx, op.Op, op.Value, user)
			if err != nil {
				return scim.Resource{}, err
			}
		default:
			u.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}

	if len(operations) != 0 {
		err = u.ds.ReplaceScimUser(ctx, user)
		switch {
		case fleet.IsNotFound(err):
			u.logger.InfoContext(ctx, "failed to find user to patch", "id", id)
			return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
		case err != nil:
			u.logger.ErrorContext(ctx, "failed to patch user", "id", id, "err", err)
			return scim.Resource{}, err
		}

		// Check if user was deactivated and delete matching Fleet user if so
		if wasDeactivated(previousActive, user.Active) {
			if err := u.deleteMatchingFleetUser(ctx, user); err != nil {
				u.logger.ErrorContext(ctx, "failed to delete fleet user on deactivation", "err", err)
			}
		}
	}

	return createUserResource(user), nil
}

func (u *UserHandler) patchEmailsWithPathFiltering(
	ctx context.Context, op scim.PatchOperation, user *fleet.ScimUser,
) error {
	emailType, err := u.getEmailType(ctx, op)
	if err != nil {
		return err
	}
	emailFound := false
	var emailIndex int
	for i, email := range user.Emails {
		if email.Type != nil && *email.Type == emailType {
			emailIndex = i
			emailFound = true
			break
		}
	}
	if !emailFound && op.Op != scim.PatchOperationAdd {
		u.logger.InfoContext(ctx, "email not found", "email_type", emailType, "op", fmt.Sprintf("%v", op))
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	if op.Path.SubAttribute == nil {
		if op.Op == scim.PatchOperationRemove {
			user.Emails = slices.Delete(user.Emails, emailIndex, emailIndex+1)
			return nil
		}

		// For add and replace operations, we need to extract the emails
		var emailsList []interface{}
		// Handle different value formats
		switch val := op.Value.(type) {
		case []interface{}:
			// Direct array of members
			emailsList = val
		case map[string]interface{}:
			// Single member as a map
			emailsList = []interface{}{val}
		default:
			u.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' patch value", emailsAttr), "value", op.Value)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}

		switch op.Op {
		case scim.PatchOperationReplace:
			if len(emailsList) == 0 {
				user.Emails = slices.Delete(user.Emails, emailIndex, emailIndex+1)
				return nil
			}
			if len(emailsList) != 1 {
				u.logger.InfoContext(ctx, "only 1 email should be present for replacement", "emails", emailsList)
				return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			userEmail, err := u.extractEmail(ctx, emailsList[0], op)
			if err != nil {
				return err
			}
			// If setting primary to true, then unset true from other emails
			if userEmail.Primary != nil && *userEmail.Primary {
				clearPrimaryFlagFromEmails(user)
			}
			user.Emails[emailIndex] = userEmail
		case scim.PatchOperationAdd:
			if len(emailsList) == 0 {
				u.logger.InfoContext(ctx, "no emails provided to add", "emails", emailsList)
				return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			var newEmails []fleet.ScimUserEmail
			for e := range emailsList {
				userEmail, err := u.extractEmail(ctx, emailsList[e], op)
				if err != nil {
					return err
				}
				userEmail.Type = &emailType
				newEmails = append(newEmails, userEmail)
			}
			primaryExists, err := u.checkEmailPrimary(ctx, newEmails)
			if err != nil {
				return err
			}
			if primaryExists {
				clearPrimaryFlagFromEmails(user)
			}
			user.Emails = append(user.Emails, newEmails...)
		}
		return nil
	}
	if op.Op == scim.PatchOperationAdd && !emailFound {
		user.Emails = append(user.Emails, fleet.ScimUserEmail{
			Type: ptr.String(emailType),
		})
		emailIndex = len(user.Emails) - 1
	}
	switch *op.Path.SubAttribute {
	case primaryAttr:
		if op.Op == scim.PatchOperationRemove {
			user.Emails[emailIndex].Primary = nil
			return nil
		}
		if op.Value == nil {
			user.Emails[emailIndex].Primary = nil
			return nil
		}
		primary, err := getConcreteType[bool](ctx, u, op.Value, primaryAttr)
		if err != nil {
			return err
		}
		// If setting primary to true, then unset true from other emails
		if primary {
			clearPrimaryFlagFromEmails(user)
		}
		user.Emails[emailIndex].Primary = &primary
	case valueAttr:
		if op.Op == scim.PatchOperationRemove {
			// The operation of removing an email value doesn't make sense, but we allow it.
			user.Emails[emailIndex].Email = ""
			return nil
		}
		value, err := getConcreteType[string](ctx, u, op.Value, valueAttr)
		if err != nil {
			return err
		}
		user.Emails[emailIndex].Email = value
	case typeAttr:
		if op.Op == scim.PatchOperationRemove {
			user.Emails[emailIndex].Type = nil
			return nil
		}
		if op.Value == nil {
			user.Emails[emailIndex].Type = nil
			return nil
		}
		newEmailType, err := getConcreteType[string](ctx, u, op.Value, typeAttr)
		if err != nil {
			return err
		}
		user.Emails[emailIndex].Type = &newEmailType
	default:
		u.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	return nil
}

func (u *UserHandler) getEmailType(ctx context.Context, op scim.PatchOperation) (string, error) {
	attrExpression, ok := op.Path.ValueExpression.(*filter.AttributeExpression)
	if !ok {
		u.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path)
		return "", errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	// Only matching by email type (work, etc.) is supported.
	if attrExpression.AttributePath.String() != typeAttr || attrExpression.Operator != filter.EQ {
		u.logger.InfoContext(ctx, "unsupported patch path",
			"path", op.Path, "expression", attrExpression.AttributePath.String())
		return "", errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	emailType, ok := attrExpression.CompareValue.(string)
	if !ok {
		u.logger.InfoContext(ctx, "unsupported patch path",
			"path", op.Path, "compare_value", attrExpression.CompareValue)
		return "", errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	return emailType, nil
}

func getConcreteType[T string | bool](ctx context.Context, u *UserHandler, v any, name string) (T, error) {
	concreteType, ok := v.(T)
	if !ok {
		var zeroValue T
		u.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' value", name), "value", v)
		return zeroValue, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}
	return concreteType, nil
}

func (u *UserHandler) patchFamilyName(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove {
		u.logger.InfoContext(ctx, "cannot remove required attribute", "attribute", nameAttr+"."+familyNameAttr)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	familyName, err := getConcreteType[string](ctx, u, v, nameAttr+"."+familyNameAttr)
	if err != nil {
		return err
	}
	user.FamilyName = &familyName
	return nil
}

func (u *UserHandler) patchGivenName(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove {
		u.logger.InfoContext(ctx, "cannot remove required attribute", "attribute", nameAttr+"."+givenNameAttr)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	givenName, err := getConcreteType[string](ctx, u, v, nameAttr+"."+givenNameAttr)
	if err != nil {
		return err
	}
	user.GivenName = &givenName
	return nil
}

func (u *UserHandler) patchActive(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove || v == nil {
		user.Active = nil
		return nil
	}
	active, err := getConcreteType[bool](ctx, u, v, activeAttr)
	if err != nil {
		return err
	}
	user.Active = &active
	return nil
}

func (u *UserHandler) patchExternalId(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove || v == nil {
		user.ExternalID = nil
		return nil
	}
	externalId, err := getConcreteType[string](ctx, u, v, externalIdAttr)
	if err != nil {
		return err
	}
	user.ExternalID = ptr.String(externalId)
	return nil
}

func (u *UserHandler) patchUserName(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove {
		u.logger.InfoContext(ctx, "cannot remove required attribute", "attribute", userNameAttr)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	userName, err := getConcreteType[string](ctx, u, v, userNameAttr)
	if err != nil {
		return err
	}
	if userName == "" {
		u.logger.InfoContext(ctx, fmt.Sprintf("'%s' cannot be empty", userNameAttr), "value", v)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}
	user.UserName = userName
	return nil
}

func (u *UserHandler) patchDepartment(ctx context.Context, op string, v any, user *fleet.ScimUser) error {
	if op == scim.PatchOperationRemove || v == nil {
		user.Department = nil
		return nil
	}
	department, err := getConcreteType[string](ctx, u, v, departmentAttr)
	if err != nil {
		return err
	}
	user.Department = &department
	return nil
}

func clearPrimaryFlagFromEmails(user *fleet.ScimUser) {
	for i, email := range user.Emails {
		if email.Primary != nil && *email.Primary {
			user.Emails[i].Primary = ptr.Bool(false)
		}
	}
}

func (u *UserHandler) patchEmails(
	ctx context.Context, v any, op scim.PatchOperation, user *fleet.ScimUser,
) error {
	if op.Op == scim.PatchOperationRemove {
		user.Emails = nil
		return nil
	}

	// For add and replace operations, we need to extract the emails
	var emailsList []interface{}
	// Handle different value formats
	switch val := v.(type) {
	case []interface{}:
		// Direct array of members
		emailsList = val
	case map[string]interface{}:
		// Single member as a map
		emailsList = []interface{}{val}
	default:
		u.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' patch value", emailsAttr), "value", op.Value)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	if op.Op == scim.PatchOperationAdd && len(emailsList) == 0 {
		u.logger.InfoContext(ctx, "no emails provided to add", "emails", emailsList)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	// Convert the emails to the expected format
	userEmails := make([]fleet.ScimUserEmail, 0, len(emailsList))
	for _, emailIntf := range emailsList {
		userEmail, err := u.extractEmail(ctx, emailIntf, op)
		if err != nil {
			return err
		}
		userEmails = append(userEmails, userEmail)
	}
	primaryExists, err := u.checkEmailPrimary(ctx, userEmails)
	if err != nil {
		return err
	}

	if op.Op == scim.PatchOperationAdd {
		if primaryExists {
			// Clear the primary flag from current emails because we are merging the two email lists and a new email has that flag.
			clearPrimaryFlagFromEmails(user)
		}
		userEmails = append(user.Emails, userEmails...)
	}

	user.Emails = userEmails
	return nil
}

// checkEmailPrimary ensures at most one email is marked as primary
func (u *UserHandler) checkEmailPrimary(ctx context.Context, userEmails []fleet.ScimUserEmail) (bool, error) {
	primaryEmailCount := 0
	for _, email := range userEmails {
		if email.Primary != nil && *email.Primary {
			primaryEmailCount++
			if primaryEmailCount > 1 {
				u.logger.InfoContext(ctx, "multiple primary emails found")
				return false, errors.ScimErrorBadParams([]string{"Only one email can be marked as primary"})
			}
		}
	}
	return primaryEmailCount > 0, nil
}

func (u *UserHandler) extractEmail(
	ctx context.Context, emailIntf any, op scim.PatchOperation,
) (fleet.ScimUserEmail, error) {
	emailMap, ok := emailIntf.(map[string]interface{})
	if !ok {
		u.logger.InfoContext(ctx, "email is not a map", "email", emailIntf)
		return fleet.ScimUserEmail{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	// Extract the email value (required)
	emailValue, ok := emailMap[valueAttr].(string)
	if !ok || emailValue == "" {
		u.logger.InfoContext(ctx, "email value is missing or invalid", "email", emailMap)
		return fleet.ScimUserEmail{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	// Normalize the email
	normalizedEmail, err := normalizeEmail(emailValue)
	if err != nil {
		u.logger.InfoContext(ctx, "failed to normalize email", "email", emailValue, "err", err)
		return fleet.ScimUserEmail{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	// Create the email object
	userEmail := fleet.ScimUserEmail{
		Email:   normalizedEmail,
		Type:    nil,
		Primary: nil,
	}

	// Extract the type (optional)
	if typeValue, ok := emailMap[typeAttr].(string); ok {
		userEmail.Type = &typeValue
	}

	// Extract the primary flag (optional)
	if primaryValue, ok := emailMap[primaryAttr].(bool); ok {
		userEmail.Primary = &primaryValue
	}
	return userEmail, nil
}

func (u *UserHandler) patchName(ctx context.Context, v any, op scim.PatchOperation, user *fleet.ScimUser) error {
	if op.Op == scim.PatchOperationRemove {
		u.logger.InfoContext(ctx, "cannot remove required attribute", "attribute", nameAttr)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	name, ok := v.(map[string]interface{})
	if !ok {
		u.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' patch value", nameAttr), "value", op.Value)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	for nameKey, nameValue := range name {
		switch nameKey {
		case givenNameAttr:
			givenName, ok := nameValue.(string)
			if !ok {
				u.logger.InfoContext(ctx,
					fmt.Sprintf("unsupported '%s' patch value", nameAttr+"."+givenNameAttr), "value", op.Value)
				return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			user.GivenName = &givenName
		case familyNameAttr:
			familyName, ok := nameValue.(string)
			if !ok {
				u.logger.InfoContext(ctx,
					fmt.Sprintf("unsupported '%s' patch value", nameAttr+"."+familyNameAttr), "value", op.Value)
				return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			user.FamilyName = &familyName
		default:
			u.logger.InfoContext(ctx, "unsupported patch value field", "field", nameAttr+"."+nameKey)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}
	return nil
}

// normalizeEmail
// The local-part of a mailbox MUST BE treated as case sensitive.
// Mailbox domains follow normal DNS rules and are hence not case sensitive.
// https://datatracker.ietf.org/doc/html/rfc5321#section-2.4
func normalizeEmail(email string) (string, error) {
	email = removeWhitespace(email)
	emailParts := strings.SplitN(email, "@", 2)
	if len(emailParts) != 2 {
		return "", fmt.Errorf("invalid email %s", email)
	}
	emailParts[1] = strings.ToLower(emailParts[1])
	return strings.Join(emailParts, "@"), nil
}

func removeWhitespace(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, str)
}

func scimUserID(userID uint) string {
	return fmt.Sprintf("%d", userID)
}

// extractUserIDFromValue extracts the user ID from a value like "123"
func extractUserIDFromValue(value string) (uint, error) {
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}
