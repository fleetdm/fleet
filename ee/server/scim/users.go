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
	"github.com/scim2/filter-parser/v2"
)

type UserHandler struct {
}

var mu sync.RWMutex
var users = make(map[uint]scim.Resource)
var nextId uint = 1

func (u UserHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := attributes["userName"]; !ok {
		return scim.Resource{}, errors.ScimErrorInvalidValue
	}

	for _, user := range users {
		if user.Attributes["userName"] == attributes["userName"] {
			return scim.Resource{}, errors.ScimErrorUniqueness
		}
	}

	users[nextId] = scim.Resource{
		ID:         fmt.Sprintf("%d", nextId),
		Attributes: attributes,
	}
	nextId++
	return users[nextId-1], nil
}

func (u UserHandler) Get(r *http.Request, id string) (scim.Resource, error) {

	mu.RLock()
	defer mu.RUnlock()
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	if user, ok := users[uint(idUint)]; ok {
		return user, nil
	}
	return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
}

func (u UserHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {

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

func (u UserHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
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
	user.Attributes = attributes
	users[uint(idUint)] = user
	return user, nil
}

func (u UserHandler) Delete(r *http.Request, id string) error {
	mu.Lock()
	defer mu.Unlock()
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user ID %s: %w", id, err)
	}
	delete(users, uint(idUint))
	return nil
}

// Patch
// Per https://developer.okta.com/docs/api/openapi/okta-scim/guides/scim-20/#update-a-specific-user-patch
// we only support patching the "active" attribute
func (u UserHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
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
		if op.Path == nil {
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
		} else if op.Path.String() == "active" {
			user.Attributes["active"] = op.Value
		} else {
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}
	users[uint(idUint)] = user
	return user, nil
}

var _ scim.ResourceHandler = &UserHandler{}
