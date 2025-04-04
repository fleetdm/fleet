package scim

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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
)

type UserHandler struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

// Compile-time check
var _ scim.ResourceHandler = &UserHandler{}

func NewUserHandler(ds fleet.Datastore, logger kitlog.Logger) scim.ResourceHandler {
	return &UserHandler{ds: ds, logger: logger}
}

func (u *UserHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	// Check for userName uniqueness
	userName, err := getRequiredResource[string](attributes, userNameAttr)
	if err != nil {
		level.Error(u.logger).Log("msg", "failed to get userName", "err", err)
		return scim.Resource{}, err
	}
	// In IETF documents, “non-empty” is generally used in the literal sense of “having at least one character.” That means if a value contains one or more spaces (and nothing else), it is still considered non-empty.
	if len(userName) == 0 {
		level.Info(u.logger).Log("msg", "userName is empty")
		return scim.Resource{}, errors.ScimErrorBadParams([]string{userNameAttr})
	}
	_, err = u.ds.ScimUserByUserName(r.Context(), userName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		level.Error(u.logger).Log("msg", "failed to check for userName uniqueness", userNameAttr, userName, "err", err)
		return scim.Resource{}, err
	case err == nil:
		level.Info(u.logger).Log("msg", "user already exists", userNameAttr, userName)
		return scim.Resource{}, errors.ScimErrorUniqueness
	}

	user, err := createUserFromAttributes(attributes)
	if err != nil {
		level.Error(u.logger).Log("msg", "failed to create user from attributes", userNameAttr, userName, "err", err)
		return scim.Resource{}, err
	}
	user.ID, err = u.ds.CreateScimUser(r.Context(), user)
	if err != nil {
		return scim.Resource{}, err
	}

	return createUserResource(user), nil
}

func createUserFromAttributes(attributes scim.ResourceAttributes) (*fleet.ScimUser, error) {
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
	user.GivenName, err = getOptionalResource[string](name, givenNameAttr)
	if err != nil {
		return nil, err
	}
	emails, err := getComplexResourceSlice(attributes, emailsAttr)
	if err != nil {
		return nil, err
	}
	userEmails := make([]fleet.ScimUserEmail, 0, len(emails))
	for _, email := range emails {
		userEmail := fleet.ScimUserEmail{}
		userEmail.Email, err = getRequiredResource[string](email, "value")
		if err != nil {
			return nil, err
		}
		// Service providers SHOULD canonicalize the value according to [RFC5321]
		// https://datatracker.ietf.org/doc/html/rfc7643#section-4.1.2
		userEmail.Email, err = normalizeEmail(userEmail.Email)
		if err != nil {
			return nil, errors.ScimErrorBadParams([]string{"value"})
		}
		userEmail.Type, err = getOptionalResource[string](email, "type")
		if err != nil {
			return nil, err
		}
		userEmail.Primary, err = getOptionalResource[bool](email, "primary")
		if err != nil {
			return nil, err
		}
		userEmails = append(userEmails, userEmail)
	}
	user.Emails = userEmails
	return &user, nil
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
	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		level.Info(u.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := u.ds.ScimUserByID(r.Context(), idUint)
	switch {
	case fleet.IsNotFound(err):
		level.Info(u.logger).Log("msg", "failed to find user", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(u.logger).Log("msg", "failed to get user", "id", id, "err", err)
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
			emailResource["value"] = email.Email
			if email.Type != nil {
				emailResource["type"] = *email.Type
			}
			if email.Primary != nil {
				emailResource["primary"] = *email.Primary
			}
			emails = append(emails, emailResource)
		}
		userResource.Attributes[emailsAttr] = emails
	}
	if len(user.Groups) > 0 {
		groups := make([]scim.ResourceAttributes, 0, len(user.Groups))
		for _, groupID := range user.Groups {
			groups = append(groups, map[string]interface{}{
				"value": scimGroupID(groupID),
				"$ref":  "Groups/" + scimGroupID(groupID),
			})
		}
		userResource.Attributes[groupsAttr] = groups
	}
	return userResource
}

// GetAll
// Pagination is 1-indexed.
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
			level.Error(u.logger).Log("msg", "failed to parse filter", "filter", resourceFilter, "err", err)
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		if !strings.EqualFold(expr.AttributePath.String(), "userName") || expr.Operator != "eq" {
			level.Info(u.logger).Log("msg", "unsupported filter", "filter", resourceFilter)
			return scim.Page{}, nil
		}
		userName, ok := expr.CompareValue.(string)
		if !ok {
			level.Error(u.logger).Log("msg", "unsupported value", "value", expr.CompareValue)
			return scim.Page{}, nil
		}

		// Decode URL-encoded characters in userName, which is required to pass Microsoft Entra ID SCIM Validator
		userName, err = url.QueryUnescape(userName)
		if err != nil {
			level.Error(u.logger).Log("msg", "failed to decode userName", "userName", userName, "err", err)
			return scim.Page{}, nil
		}
		opts.UserNameFilter = &userName
	}
	users, totalResults, err := u.ds.ListScimUsers(r.Context(), opts)
	if err != nil {
		level.Error(u.logger).Log("msg", "failed to list users", "err", err)
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
	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		level.Info(u.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := createUserFromAttributes(attributes)
	if err != nil {
		level.Error(u.logger).Log("msg", "failed to create user from attributes", "id", id, "err", err)
		return scim.Resource{}, err
	}
	user.ID = idUint
	// Username is unique, so we must check if another user already exists with that username to return a clear error
	userWithSameUsername, err := u.ds.ScimUserByUserName(r.Context(), user.UserName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		level.Error(u.logger).Log("msg", "failed to check for userName uniqueness", userNameAttr, user.UserName, "err", err)
		return scim.Resource{}, err
	case err == nil && user.ID != userWithSameUsername.ID:
		level.Info(u.logger).Log("msg", "user already exists with this username", userNameAttr, user.UserName)
		return scim.Resource{}, errors.ScimErrorUniqueness
		// Otherwise, we assume that we are replacing the username with this operation.
	}

	err = u.ds.ReplaceScimUser(r.Context(), user)
	switch {
	case fleet.IsNotFound(err):
		level.Info(u.logger).Log("msg", "failed to find user to replace", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(u.logger).Log("msg", "failed to replace user", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createUserResource(user), nil
}

// Delete
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.6
// MUST return a 404 (Not Found) error code for all operations associated with the previously deleted resource
func (u *UserHandler) Delete(r *http.Request, id string) error {
	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		level.Info(u.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return errors.ScimErrorResourceNotFound(id)
	}
	err = u.ds.DeleteScimUser(r.Context(), idUint)
	switch {
	case fleet.IsNotFound(err):
		level.Info(u.logger).Log("msg", "failed to find user to delete", "id", id)
		return errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(u.logger).Log("msg", "failed to delete user", "id", id, "err", err)
		return err
	}
	return nil
}

// Patch
// Okta only requires patching the "active" attribute:
// https://developer.okta.com/docs/api/openapi/okta-scim/guides/scim-20/#update-a-specific-user-patch
func (u *UserHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	idUint, err := extractUserIDFromValue(id)
	if err != nil {
		level.Info(u.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	user, err := u.ds.ScimUserByID(r.Context(), idUint)
	switch {
	case fleet.IsNotFound(err):
		level.Info(u.logger).Log("msg", "failed to find user to patch", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(u.logger).Log("msg", "failed to get user to patch", "id", id, "err", err)
		return scim.Resource{}, err
	}

	if len(operations) > 1 {
		level.Info(u.logger).Log("msg", "too many patch operations")
		return scim.Resource{}, errors.ScimErrorBadParams([]string{"Operations"})
	}
	for _, op := range operations {
		if op.Op != "replace" {
			level.Info(u.logger).Log("msg", "unsupported patch operation", "op", op.Op)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
		switch {
		case op.Path == nil:
			newValues, ok := op.Value.(map[string]interface{})
			if !ok {
				level.Info(u.logger).Log("msg", "unsupported patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			if len(newValues) != 1 {
				level.Info(u.logger).Log("msg", "too many patch values", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			active, err := getRequiredResource[bool](newValues, activeAttr)
			if err != nil {
				level.Info(u.logger).Log("msg", "failed to get active value", "value", op.Value)
				return scim.Resource{}, err
			}
			user.Active = &active
		case op.Path.String() == activeAttr:
			active, ok := op.Value.(bool)
			if !ok {
				level.Error(u.logger).Log("msg", "unsupported 'active' patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			user.Active = &active
		default:
			level.Info(u.logger).Log("msg", "unsupported patch path", "path", op.Path)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}

	if len(operations) != 0 {
		err = u.ds.ReplaceScimUser(r.Context(), user)
		switch {
		case fleet.IsNotFound(err):
			level.Info(u.logger).Log("msg", "failed to find user to patch", "id", id)
			return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
		case err != nil:
			level.Error(u.logger).Log("msg", "failed to patch user", "id", id, "err", err)
			return scim.Resource{}, err
		}
	}

	return createUserResource(user), nil
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
