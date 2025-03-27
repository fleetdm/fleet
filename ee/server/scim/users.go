package scim

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
)

type UserHandler struct {
	ds fleet.Datastore
}

// Compile-time check
var _ scim.ResourceHandler = &UserHandler{}

func NewUserHandler(ds fleet.Datastore) scim.ResourceHandler {
	return &UserHandler{ds: ds}
}

var mu sync.RWMutex
var users = make(map[uint]scim.Resource)

func (u *UserHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	// Check for userName uniqueness
	userName, err := getRequiredResource(attributes, userNameAttr)
	if err != nil {
		return scim.Resource{}, err
	}
	_, err = u.ds.ScimUserByUserName(r.Context(), userName)
	if !fleet.IsNotFound(err) {
		return scim.Resource{}, errors.ScimErrorUniqueness
	}

	user, err := createUserFromAttributes(attributes)
	if err != nil {
		return scim.Resource{}, err
	}
	userID, err := u.ds.CreateScimUser(r.Context(), user)
	if err != nil {
		return scim.Resource{}, err
	}

	return scim.Resource{
		ID:         fmt.Sprintf("%d", userID),
		Attributes: attributes,
	}, nil
}

func createUserFromAttributes(attributes scim.ResourceAttributes) (*fleet.ScimUser, error) {
	user := fleet.ScimUser{}
	var err error
	user.UserName, err = getRequiredResource(attributes, userNameAttr)
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
		userEmail.Email, err = getRequiredResource(email, "value")
		if err != nil {
			return nil, err
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

func getRequiredResource[T string](attributes scim.ResourceAttributes, key string) (T, error) {
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
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := u.ds.ScimUserByID(r.Context(), uint(idUint))
	switch {
	case fleet.IsNotFound(err):
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		return scim.Resource{}, err
	}

	userResource := scim.Resource{}
	userResource.ID = id
	if user.ExternalID != nil {
		userResource.ExternalID = optional.NewString(*user.ExternalID)
	}
	userResource.Attributes = scim.ResourceAttributes{}
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

	return userResource, nil
}

// GetAll
// Per RFC7644 3.4.2, SHOULD ignore any query parameters they do not recognize instead of rejecting the query for versioning compatibility reasons
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.4.2
//
// If a SCIM service provider determines that too many results would be returned the server base URI, the server SHALL reject the request by
// returning an HTTP response with HTTP status code 400 (Bad Request) and JSON attribute "scimType" set to "tooMany" (see Table 9).
func (u *UserHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {

	resourceFilter := r.URL.Query().Get("filter")
	if resourceFilter != "" {
		expr, err := filter.ParseAttrExp([]byte(resourceFilter))
		if err != nil {
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		if !strings.EqualFold(expr.AttributePath.String(), "userName") || expr.Operator != "eq" {
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		userName, ok := expr.CompareValue.(string)
		if !ok {
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}

		// Decode URL-encoded characters in userName, which is required to pass Microsoft Entra ID SCIM Validator
		userName, err = url.QueryUnescape(userName)
		if err != nil {
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}

		for _, user := range users {
			if user.Attributes["userName"] == userName {
				return scim.Page{
					TotalResults: 1,
					Resources:    []scim.Resource{user},
				}, nil
			}
		}
		return scim.Page{}, nil
	}

	mu.RLock()
	defer mu.RUnlock()

	// Convert users keys into a slice of uint
	keys := make([]uint, 0, len(users))
	for k := range users {
		keys = append(keys, k)
	}
	// Sort the keys
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	startIndex := params.StartIndex - 1
	if startIndex < 0 {
		startIndex = 0
	}
	if startIndex >= len(keys) {
		return scim.Page{}, nil
	}
	endIndex := startIndex + params.Count
	if endIndex > len(keys) {
		endIndex = len(keys)
	}
	keysToReturn := keys[startIndex:endIndex]
	result := scim.Page{
		TotalResults: endIndex - startIndex,
		Resources:    make([]scim.Resource, 0, len(keysToReturn)),
	}
	for _, key := range keysToReturn {
		result.Resources = append(result.Resources, users[key])
	}
	return result, nil
}

func (u *UserHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	user, err := createUserFromAttributes(attributes)
	if err != nil {
		return scim.Resource{}, err
	}
	user.ID = uint(idUint)
	err = u.ds.ReplaceScimUser(r.Context(), user)
	switch {
	case fleet.IsNotFound(err):
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		return scim.Resource{}, err
	}

	return scim.Resource{
		ID:         id,
		Attributes: attributes,
	}, nil
}

// Delete
// https://datatracker.ietf.org/doc/html/rfc7644#section-3.6
// MUST return a 404 (Not Found) error code for all operations associated with the previously deleted resource
func (u *UserHandler) Delete(r *http.Request, id string) error {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return errors.ScimErrorResourceNotFound(id)
	}
	err = u.ds.DeleteScimUser(r.Context(), uint(idUint))
	switch {
	case fleet.IsNotFound(err):
		return errors.ScimErrorResourceNotFound(id)
	case err != nil:
		return err
	}
	return nil
}

// Patch
// Okta only requires patching the "active" attribute:
// https://developer.okta.com/docs/api/openapi/okta-scim/guides/scim-20/#update-a-specific-user-patch
func (u *UserHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	mu.Lock()
	defer mu.Unlock()
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return scim.Resource{}, fmt.Errorf("invalid user ID %s: %w", id, err)
	}
	user, ok := users[uint(idUint)]
	if !ok {
		return scim.Resource{}, fmt.Errorf("user with ID %s not found", id)
	}
	for _, op := range operations {
		if op.Op != "replace" {
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
		switch {
		case op.Path == nil:
			newValues, ok := op.Value.(map[string]interface{})
			if !ok {
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			if len(newValues) != 1 {
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			var val interface{}
			if val, ok = newValues["active"]; !ok {
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			var valBool bool
			if valBool, ok = val.(bool); !ok {
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			user.Attributes["active"] = valBool
		case op.Path.String() == "active":
			user.Attributes["active"] = op.Value
		default:
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}
	users[uint(idUint)] = user
	return user, nil
}
