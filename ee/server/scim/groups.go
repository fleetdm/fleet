package scim

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/scim2/filter-parser/v2"
)

const (
	// Group attributes: https://datatracker.ietf.org/doc/html/rfc7643#section-4.2
	displayNameAttr = "displayName"
	membersAttr     = "members"
)

type GroupHandler struct {
	ds     fleet.Datastore
	logger kitlog.Logger
}

// Compile-time check
var _ scim.ResourceHandler = &GroupHandler{}

func NewGroupHandler(ds fleet.Datastore, logger kitlog.Logger) scim.ResourceHandler {
	return &GroupHandler{ds: ds, logger: logger}
}

func (g *GroupHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	// Check for displayName uniqueness
	displayName, err := getRequiredResource[string](attributes, displayNameAttr)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to get displayName", "err", err)
		return scim.Resource{}, err
	}

	group, err := createGroupFromAttributes(attributes)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to create group from attributes", displayNameAttr, displayName, "err", err)
		return scim.Resource{}, err
	}
	group.ID, err = g.ds.CreateScimGroup(r.Context(), group)
	if err != nil {
		return scim.Resource{}, err
	}

	return createGroupResource(group), nil
}

func createGroupFromAttributes(attributes scim.ResourceAttributes) (*fleet.ScimGroup, error) {
	group := fleet.ScimGroup{}
	var err error
	group.DisplayName, err = getRequiredResource[string](attributes, displayNameAttr)
	if err != nil {
		return nil, err
	}
	group.ExternalID, err = getOptionalResource[string](attributes, externalIdAttr)
	if err != nil {
		return nil, err
	}

	// Process members
	members, err := getComplexResourceSlice(attributes, membersAttr)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(members))
	for _, member := range members {
		// Get the value attribute which contains the user ID
		valueIntf, ok := member["value"]
		if !ok || valueIntf == nil {
			continue
		}
		valueStr, ok := valueIntf.(string)
		if !ok {
			return nil, errors.ScimErrorBadParams([]string{"value"})
		}

		// Extract user ID from the value
		userID, err := extractUserIDFromValue(valueStr)
		if err != nil {
			return nil, errors.ScimErrorBadParams([]string{"value"})
		}
		userIDs = append(userIDs, userID)
	}
	group.ScimUsers = userIDs

	return &group, nil
}

// extractUserIDFromValue extracts the user ID from a value like "user-123"
func extractUserIDFromValue(value string) (uint, error) {
	// Check if the value starts with "user-"
	if !strings.HasPrefix(value, "user-") {
		return 0, fmt.Errorf("invalid user ID format: %s", value)
	}

	// Extract the ID part
	idStr := strings.TrimPrefix(value, "user-")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(id), nil
}

func (g *GroupHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := g.ds.ScimGroupByID(r.Context(), uint(idUint))
	switch {
	case fleet.IsNotFound(err):
		level.Info(g.logger).Log("msg", "failed to find group", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(g.logger).Log("msg", "failed to get group", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createGroupResource(group), nil
}

func createGroupResource(group *fleet.ScimGroup) scim.Resource {
	groupResource := scim.Resource{}
	groupResource.ID = fmt.Sprintf("%d", group.ID)
	if group.ExternalID != nil {
		groupResource.ExternalID = optional.NewString(*group.ExternalID)
	}
	groupResource.Attributes = scim.ResourceAttributes{}
	groupResource.Attributes[displayNameAttr] = group.DisplayName

	// Add members if any
	if len(group.ScimUsers) > 0 {
		members := make([]scim.ResourceAttributes, 0, len(group.ScimUsers))
		for _, userID := range group.ScimUsers {
			members = append(members, map[string]interface{}{
				"value": scimUserID(userID),
				"$ref":  "Users/" + scimUserID(userID),
			})
		}
		groupResource.Attributes[membersAttr] = members
	}

	return groupResource
}

func scimUserID(userID uint) string {
	return fmt.Sprintf("user-%d", userID)
}

func (g *GroupHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
	page := params.StartIndex
	if page < 1 {
		page = 1
	}
	count := params.Count
	if count > maxResults {
		return scim.Page{}, errors.ScimErrorTooMany
	}
	if count < 1 {
		count = maxResults
	}

	opts := fleet.ScimListOptions{
		Page:    uint(page),  // nolint:gosec // ignore G115
		PerPage: uint(count), // nolint:gosec // ignore G115
	}

	resourceFilter := r.URL.Query().Get("filter")
	if resourceFilter != "" {
		expr, err := filter.ParseAttrExp([]byte(resourceFilter))
		if err != nil {
			level.Error(g.logger).Log("msg", "failed to parse filter", "filter", resourceFilter, "err", err)
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		if !strings.EqualFold(expr.AttributePath.String(), displayNameAttr) || expr.Operator != "eq" {
			level.Info(g.logger).Log("msg", "unsupported filter", "filter", resourceFilter)
			return scim.Page{}, nil
		}
		displayName, ok := expr.CompareValue.(string)
		if !ok {
			level.Error(g.logger).Log("msg", "unsupported value", "value", expr.CompareValue)
			return scim.Page{}, nil
		}

		// Decode URL-encoded characters in displayName
		displayName, err = url.QueryUnescape(displayName)
		if err != nil {
			level.Error(g.logger).Log("msg", "failed to decode displayName", "displayName", displayName, "err", err)
			return scim.Page{}, nil
		}
		// Note: The current implementation of ListScimGroups doesn't support filtering by displayName
		// This would need to be added to the datastore implementation
	}

	groups, totalResults, err := g.ds.ListScimGroups(r.Context(), opts)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to list groups", "err", err)
		return scim.Page{}, err
	}

	result := scim.Page{
		TotalResults: int(totalResults), // nolint:gosec // ignore G115
		Resources:    make([]scim.Resource, 0, len(groups)),
	}
	for i := range groups {
		result.Resources = append(result.Resources, createGroupResource(&groups[i]))
	}

	return result, nil
}

func (g *GroupHandler) Replace(r *http.Request, id string, attributes scim.ResourceAttributes) (scim.Resource, error) {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := createGroupFromAttributes(attributes)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to create group from attributes", "id", id, "err", err)
		return scim.Resource{}, err
	}
	group.ID = uint(idUint)
	err = g.ds.ReplaceScimGroup(r.Context(), group)
	switch {
	case fleet.IsNotFound(err):
		level.Info(g.logger).Log("msg", "failed to find group to replace", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(g.logger).Log("msg", "failed to replace group", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createGroupResource(group), nil
}

func (g *GroupHandler) Delete(r *http.Request, id string) error {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return errors.ScimErrorResourceNotFound(id)
	}
	err = g.ds.DeleteScimGroup(r.Context(), uint(idUint))
	switch {
	case fleet.IsNotFound(err):
		level.Info(g.logger).Log("msg", "failed to find group to delete", "id", id)
		return errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(g.logger).Log("msg", "failed to delete group", "id", id, "err", err)
		return err
	}
	return nil
}

func (g *GroupHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	group, err := g.ds.ScimGroupByID(r.Context(), uint(idUint))
	switch {
	case fleet.IsNotFound(err):
		level.Info(g.logger).Log("msg", "failed to find group to patch", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(g.logger).Log("msg", "failed to get group to patch", "id", id, "err", err)
		return scim.Resource{}, err
	}

	for _, op := range operations {
		if op.Op != "replace" {
			level.Info(g.logger).Log("msg", "unsupported patch operation", "op", op.Op)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
		switch {
		case op.Path == nil:
			newValues, ok := op.Value.(map[string]interface{})
			if !ok {
				level.Info(g.logger).Log("msg", "unsupported patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			// Handle replacing the entire group
			newGroup, err := createGroupFromAttributes(newValues)
			if err != nil {
				level.Error(g.logger).Log("msg", "failed to create group from patch values", "err", err)
				return scim.Resource{}, err
			}
			newGroup.ID = group.ID
			group = newGroup
		case op.Path.String() == displayNameAttr:
			displayName, ok := op.Value.(string)
			if !ok {
				level.Error(g.logger).Log("msg", "unsupported 'displayName' patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			group.DisplayName = displayName
		case op.Path.String() == membersAttr:
			// Handle replacing members
			membersIntf, ok := op.Value.([]interface{})
			if !ok {
				level.Error(g.logger).Log("msg", "unsupported 'members' patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			userIDs := make([]uint, 0, len(membersIntf))
			for _, memberIntf := range membersIntf {
				memberMap, ok := memberIntf.(map[string]interface{})
				if !ok {
					return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
				}
				valueIntf, ok := memberMap["value"]
				if !ok || valueIntf == nil {
					continue
				}
				valueStr, ok := valueIntf.(string)
				if !ok {
					return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
				}
				userID, err := extractUserIDFromValue(valueStr)
				if err != nil {
					return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
				}
				userIDs = append(userIDs, userID)
			}
			group.ScimUsers = userIDs
		default:
			level.Info(g.logger).Log("msg", "unsupported patch path", "path", op.Path)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}

	err = g.ds.ReplaceScimGroup(r.Context(), group)
	switch {
	case fleet.IsNotFound(err):
		level.Info(g.logger).Log("msg", "failed to find group to patch", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		level.Error(g.logger).Log("msg", "failed to patch group", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createGroupResource(group), nil
}
