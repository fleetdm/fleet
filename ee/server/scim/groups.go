package scim

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

// Create creates a SCIM group
func (g *GroupHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	displayName, err := getRequiredResource[string](attributes, displayNameAttr)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to get displayName", "err", err)
		return scim.Resource{}, err
	}

	// Microsoft’s SCIM implementation (Entra ID) imposes additional constraints—like enforcing uniqueness on a group’s
	// displayName—that the SCIM spec itself does not mandate.
	// In effect, Microsoft’s implementation diverges from strict SCIM compliance by making displayName behave like a unique key.
	// SCIM only mandates that each group’s "id" is unique
	_, err = g.ds.ScimGroupByDisplayName(r.Context(), displayName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		level.Error(g.logger).Log("msg", "failed to check for displayName uniqueness", displayNameAttr, displayName, "err", err)
		return scim.Resource{}, err
	case err == nil:
		level.Info(g.logger).Log("msg", "group already exists", displayNameAttr, displayName)
		return scim.Resource{}, errors.ScimErrorUniqueness
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

// Get the Scim group by ID. The group id is of the format: group-123
// SCIM resource IDs must be unique across all resources.
func (g *GroupHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := g.ds.ScimGroupByID(r.Context(), idUint)
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
	groupResource.ID = scimGroupID(group.ID)
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
				"type":  "User",
			})
		}
		groupResource.Attributes[membersAttr] = members
	}

	return groupResource
}

// GetAll
// Pagination is 1-indexed on the startIndex. The startIndex is the index of the resource (not the index of the page), per RFC7644.
func (g *GroupHandler) GetAll(r *http.Request, params scim.ListRequestParams) (scim.Page, error) {
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

	opts := fleet.ScimListOptions{
		StartIndex: uint(startIndex), // nolint:gosec // ignore G115
		PerPage:    uint(count),      // nolint:gosec // ignore G115
	}

	resourceFilter := r.URL.Query().Get("filter")
	if resourceFilter != "" {
		level.Info(g.logger).Log("msg", "group filter not supported", "filter", resourceFilter)
		return scim.Page{}, nil
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
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := createGroupFromAttributes(attributes)
	if err != nil {
		level.Error(g.logger).Log("msg", "failed to create group from attributes", "id", id, "err", err)
		return scim.Resource{}, err
	}
	group.ID = idUint
	// Display name is unique to comply with Entra ID requirements,
	// so we must check if another group already exists with that display name to return a clear error
	groupWithSameDisplayName, err := g.ds.ScimGroupByDisplayName(r.Context(), group.DisplayName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		level.Error(g.logger).Log("msg", "failed to check for displayName uniqueness", displayNameAttr, group.DisplayName, "err", err)
		return scim.Resource{}, err
	case err == nil && group.ID != groupWithSameDisplayName.ID:
		level.Info(g.logger).Log("msg", "group already exists with this displayName", displayNameAttr, group.DisplayName)
		return scim.Resource{}, errors.ScimErrorUniqueness
		// Otherwise, we assume that we are replacing the displayName with this operation.
	}

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
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return errors.ScimErrorResourceNotFound(id)
	}
	err = g.ds.DeleteScimGroup(r.Context(), idUint)
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

// Patch
// Only supporting replacing the "displayName" attribute.
// Note: Okta does not use PATCH endpoint to update groups (2025/04/01)
func (g *GroupHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		level.Info(g.logger).Log("msg", "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	group, err := g.ds.ScimGroupByID(r.Context(), idUint)
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
			if len(newValues) != 1 {
				level.Info(g.logger).Log("msg", "too many patch values", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			displayName, err := getRequiredResource[string](newValues, displayNameAttr)
			if err != nil {
				level.Info(g.logger).Log("msg", "failed to get active value", "value", op.Value)
				return scim.Resource{}, err
			}
			group.DisplayName = displayName
		case op.Path.String() == displayNameAttr:
			displayName, ok := op.Value.(string)
			if !ok {
				level.Error(g.logger).Log("msg", "unsupported 'displayName' patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			group.DisplayName = displayName
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

func scimGroupID(groupID uint) string {
	return fmt.Sprintf("group-%d", groupID)
}

// extractGroupIDFromValue extracts the group ID from a value like "group-123"
func extractGroupIDFromValue(value string) (uint, error) {
	if !strings.HasPrefix(value, "group-") {
		return 0, fmt.Errorf("value %q does not match the expected format 'group-<id>'", value)
	}

	idStr := strings.TrimPrefix(value, "group-")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse group ID from value %q: %w", value, err)
	}

	return uint(id), nil
}
