package scim

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/scim2/filter-parser/v2"
)

const (
	// Group attributes: https://datatracker.ietf.org/doc/html/rfc7643#section-4.2
	displayNameAttr = "displayName"
	membersAttr     = "members"
)

type GroupHandler struct {
	ds     fleet.Datastore
	logger *slog.Logger
}

// Compile-time check
var _ scim.ResourceHandler = &GroupHandler{}

func NewGroupHandler(ds fleet.Datastore, logger *slog.Logger) scim.ResourceHandler {
	return &GroupHandler{ds: ds, logger: logger}
}

// Create creates a SCIM group
func (g *GroupHandler) Create(r *http.Request, attributes scim.ResourceAttributes) (scim.Resource, error) {
	displayName, err := getRequiredResource[string](attributes, displayNameAttr)
	if err != nil {
		g.logger.ErrorContext(r.Context(), "failed to get displayName", "err", err)
		return scim.Resource{}, err
	}

	// Microsoft’s SCIM implementation (Entra ID) imposes additional constraints—like enforcing uniqueness on a group’s
	// displayName—that the SCIM spec itself does not mandate.
	// In effect, Microsoft’s implementation diverges from strict SCIM compliance by making displayName behave like a unique key.
	// SCIM only mandates that each group’s "id" is unique
	_, err = g.ds.ScimGroupByDisplayName(r.Context(), displayName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		g.logger.ErrorContext(r.Context(), "failed to check for displayName uniqueness", displayNameAttr, displayName, "err", err)
		return scim.Resource{}, err
	case err == nil:
		g.logger.InfoContext(r.Context(), "group already exists", displayNameAttr, displayName)
		return scim.Resource{}, errors.ScimErrorUniqueness
	}

	group, err := createGroupFromAttributes(attributes)
	if err != nil {
		g.logger.ErrorContext(r.Context(), "failed to create group from attributes", displayNameAttr, displayName, "err", err)
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

// areMembersExcluded checks if the members attribute is excluded in the request
func areMembersExcluded(r *http.Request) bool {
	excludedAttrs := r.URL.Query().Get("excludedAttributes")
	if excludedAttrs == "" {
		return false
	}

	// Split the excluded attributes by comma
	attrs := strings.Split(excludedAttrs, ",")
	for _, attr := range attrs {
		// Trim spaces and check if it's "members"
		if strings.TrimSpace(attr) == membersAttr {
			return true
		}
	}

	return false
}

// Get the Scim group by ID. The group id is of the format: group-123
// SCIM resource IDs must be unique across all resources.
func (g *GroupHandler) Get(r *http.Request, id string) (scim.Resource, error) {
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		g.logger.InfoContext(r.Context(), "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := g.ds.ScimGroupByID(r.Context(), idUint, areMembersExcluded(r))
	switch {
	case fleet.IsNotFound(err):
		g.logger.InfoContext(r.Context(), "failed to find group", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		g.logger.ErrorContext(r.Context(), "failed to get group", "id", id, "err", err)
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

	opts := fleet.ScimGroupsListOptions{
		ScimListOptions: fleet.ScimListOptions{
			StartIndex: uint(startIndex), // nolint:gosec // ignore G115
			PerPage:    uint(count),      // nolint:gosec // ignore G115
		},
		ExcludeUsers: areMembersExcluded(r),
	}

	resourceFilter := r.URL.Query().Get("filter")
	if resourceFilter != "" {
		expr, err := filter.ParseAttrExp([]byte(resourceFilter))
		if err != nil {
			g.logger.ErrorContext(r.Context(), "failed to parse filter", "filter", resourceFilter, "err", err)
			return scim.Page{}, errors.ScimErrorInvalidFilter
		}
		if !strings.EqualFold(expr.AttributePath.String(), "displayName") || expr.Operator != "eq" {
			g.logger.InfoContext(r.Context(), "unsupported filter", "filter", resourceFilter)
			return scim.Page{}, nil
		}
		displayName, ok := expr.CompareValue.(string)
		if !ok {
			g.logger.ErrorContext(r.Context(), "unsupported value", "value", expr.CompareValue)
			return scim.Page{}, nil
		}

		// Decode URL-encoded characters
		displayName, err = url.QueryUnescape(displayName)
		if err != nil {
			g.logger.ErrorContext(r.Context(), "failed to decode displayName", "displayName", displayName, "err", err)
			return scim.Page{}, nil
		}
		opts.DisplayNameFilter = &displayName
	}

	groups, totalResults, err := g.ds.ListScimGroups(r.Context(), opts)
	if err != nil {
		g.logger.ErrorContext(r.Context(), "failed to list groups", "err", err)
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
		g.logger.InfoContext(r.Context(), "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}

	group, err := createGroupFromAttributes(attributes)
	if err != nil {
		g.logger.ErrorContext(r.Context(), "failed to create group from attributes", "id", id, "err", err)
		return scim.Resource{}, err
	}
	group.ID = idUint
	// Display name is unique to comply with Entra ID requirements,
	// so we must check if another group already exists with that display name to return a clear error
	groupWithSameDisplayName, err := g.ds.ScimGroupByDisplayName(r.Context(), group.DisplayName)
	switch {
	case err != nil && !fleet.IsNotFound(err):
		g.logger.ErrorContext(r.Context(), "failed to check for displayName uniqueness", displayNameAttr, group.DisplayName, "err", err)
		return scim.Resource{}, err
	case err == nil && group.ID != groupWithSameDisplayName.ID:
		g.logger.InfoContext(r.Context(), "group already exists with this displayName", displayNameAttr, group.DisplayName)
		return scim.Resource{}, errors.ScimErrorUniqueness
		// Otherwise, we assume that we are replacing the displayName with this operation.
	}

	err = g.ds.ReplaceScimGroup(r.Context(), group)
	switch {
	case fleet.IsNotFound(err):
		g.logger.InfoContext(r.Context(), "failed to find group to replace", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		g.logger.ErrorContext(r.Context(), "failed to replace group", "id", id, "err", err)
		return scim.Resource{}, err
	}

	return createGroupResource(group), nil
}

func (g *GroupHandler) Delete(r *http.Request, id string) error {
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		g.logger.InfoContext(r.Context(), "failed to parse id", "id", id, "err", err)
		return errors.ScimErrorResourceNotFound(id)
	}
	err = g.ds.DeleteScimGroup(r.Context(), idUint)
	switch {
	case fleet.IsNotFound(err):
		g.logger.InfoContext(r.Context(), "failed to find group to delete", "id", id)
		return errors.ScimErrorResourceNotFound(id)
	case err != nil:
		g.logger.ErrorContext(r.Context(), "failed to delete group", "id", id, "err", err)
		return err
	}
	return nil
}

// Patch
// Supporting add/replace/remove operations for "displayName", "externalId", and "members" attributes.
func (g *GroupHandler) Patch(r *http.Request, id string, operations []scim.PatchOperation) (scim.Resource, error) {
	ctx := r.Context()
	idUint, err := extractGroupIDFromValue(id)
	if err != nil {
		g.logger.InfoContext(ctx, "failed to parse id", "id", id, "err", err)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	}
	group, err := g.ds.ScimGroupByID(ctx, idUint, false)
	switch {
	case fleet.IsNotFound(err):
		g.logger.InfoContext(ctx, "failed to find group to patch", "id", id)
		return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
	case err != nil:
		g.logger.ErrorContext(ctx, "failed to get group to patch", "id", id, "err", err)
		return scim.Resource{}, err
	}

	for _, op := range operations {
		if op.Op != scim.PatchOperationAdd && op.Op != scim.PatchOperationReplace && op.Op != scim.PatchOperationRemove {
			g.logger.InfoContext(ctx, "unsupported patch operation", "op", op.Op)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
		switch {
		case op.Path == nil:
			if op.Op == scim.PatchOperationRemove {
				g.logger.InfoContext(ctx, "the 'path' attribute is REQUIRED for 'remove' operations", "op", op.Op)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			newValues, ok := op.Value.(map[string]interface{})
			if !ok {
				g.logger.InfoContext(ctx, "unsupported patch value", "value", op.Value)
				return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
			}
			for k, v := range newValues {
				switch k {
				case externalIdAttr:
					err = g.patchExternalId(ctx, op.Op, v, group)
					if err != nil {
						return scim.Resource{}, err
					}
				case displayNameAttr:
					err = g.patchDisplayName(ctx, op.Op, v, group)
					if err != nil {
						return scim.Resource{}, err
					}
				case membersAttr:
					err = g.patchMembers(ctx, op.Op, v, group)
					if err != nil {
						return scim.Resource{}, err
					}
				default:
					g.logger.InfoContext(ctx, "unsupported patch value field", "field", k)
					return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
				}
			}
		case op.Path.String() == externalIdAttr:
			err = g.patchExternalId(ctx, op.Op, op.Value, group)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == displayNameAttr:
			err = g.patchDisplayName(ctx, op.Op, op.Value, group)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.String() == membersAttr:
			err = g.patchMembers(ctx, op.Op, op.Value, group)
			if err != nil {
				return scim.Resource{}, err
			}
		case op.Path.AttributePath.String() == membersAttr:
			err = g.patchMembersWithPathFiltering(ctx, op, group)
			if err != nil {
				return scim.Resource{}, err
			}
		default:
			g.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path)
			return scim.Resource{}, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}
	}

	if len(operations) != 0 {
		err = g.ds.ReplaceScimGroup(ctx, group)
		switch {
		case fleet.IsNotFound(err):
			g.logger.InfoContext(ctx, "failed to find group to patch", "id", id)
			return scim.Resource{}, errors.ScimErrorResourceNotFound(id)
		case err != nil:
			g.logger.ErrorContext(ctx, "failed to patch group", "id", id, "err", err)
			return scim.Resource{}, err
		}
	}

	return createGroupResource(group), nil
}

func (g *GroupHandler) patchExternalId(ctx context.Context, op string, v interface{}, group *fleet.ScimGroup) error {
	if op == scim.PatchOperationRemove || v == nil {
		group.ExternalID = nil
		return nil
	}
	externalId, ok := v.(string)
	if !ok {
		g.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' value", externalIdAttr), "value", v)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}
	group.ExternalID = &externalId
	return nil
}

func (g *GroupHandler) patchDisplayName(ctx context.Context, op string, v interface{}, group *fleet.ScimGroup) error {
	if op == scim.PatchOperationRemove {
		g.logger.InfoContext(ctx, "cannot remove required attribute", "attribute", displayNameAttr)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}
	displayName, ok := v.(string)
	if !ok {
		g.logger.InfoContext(ctx, fmt.Sprintf("unsupported '%s' value", displayNameAttr), "value", v)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}
	if displayName == "" {
		g.logger.InfoContext(ctx, fmt.Sprintf("'%s' cannot be empty", displayNameAttr), "value", v)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}
	group.DisplayName = displayName
	return nil
}

// patchMembers handles add/replace/remove operations for the members attribute
func (g *GroupHandler) patchMembers(ctx context.Context, op string, v interface{}, group *fleet.ScimGroup) error {
	if op == scim.PatchOperationRemove {
		// Remove all members
		group.ScimUsers = []uint{}
		return nil
	}

	// For add and replace operations, we need to extract the member IDs
	var membersList []interface{}

	// Handle different value formats
	switch val := v.(type) {
	case []interface{}:
		// Direct array of members
		membersList = val
	case map[string]interface{}:
		// Single member as a map
		membersList = []interface{}{val}
	case []map[string]interface{}:
		// Array of member maps
		for _, m := range val {
			membersList = append(membersList, m)
		}
	default:
		g.logger.InfoContext(ctx, "unsupported members value format", "value", v)
		return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", v)})
	}

	// Process the members
	userIDs := make([]uint, 0, len(membersList))
	valueStrings := make([]string, 0, len(membersList))

	for _, memberIntf := range membersList {
		member, ok := memberIntf.(map[string]interface{})
		if !ok {
			g.logger.InfoContext(ctx, "member must be an object", "member", memberIntf)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", memberIntf)})
		}

		// Get the value attribute which contains the user ID
		valueIntf, ok := member["value"]
		if !ok || valueIntf == nil {
			g.logger.InfoContext(ctx, "member missing value attribute", "member", member)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", member)})
		}

		valueStr, ok := valueIntf.(string)
		if !ok {
			g.logger.InfoContext(ctx, "member value must be a string", "value", valueIntf)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", valueIntf)})
		}
		valueStrings = append(valueStrings, valueStr)

		// Extract user ID from the value
		userID, err := extractUserIDFromValue(valueStr)
		if err != nil {
			g.logger.InfoContext(ctx, "invalid user ID format", "value", valueStr, "err", err)
			return errors.ScimErrorBadParams([]string{valueStr})
		}

		userIDs = append(userIDs, userID)
	}

	// Verify all users exist in a single database call
	if len(userIDs) > 0 {
		allExist, err := g.ds.ScimUsersExist(ctx, userIDs)
		if err != nil {
			g.logger.ErrorContext(ctx, "error checking users existence", "err", err)
			return err
		}
		if !allExist {
			g.logger.InfoContext(ctx, "one or more users not found", "userIDs", userIDs)
			return errors.ScimErrorBadParams(valueStrings)
		}
	}

	// For add operation, append to existing members
	if op == scim.PatchOperationAdd {
		// Create a map to track existing user IDs to avoid duplicates
		existingUsers := make(map[uint]bool)
		for _, id := range group.ScimUsers {
			existingUsers[id] = true
		}

		// Add new users that don't already exist in the group
		for _, id := range userIDs {
			if !existingUsers[id] {
				group.ScimUsers = append(group.ScimUsers, id)
				existingUsers[id] = true
			}
		}
	} else {
		// For replace operation, replace all members
		group.ScimUsers = userIDs // FIXME: List should be deduplicated by us? See https://github.com/fleetdm/fleet/issues/30086
	}

	return nil
}

// patchMembersWithPathFiltering handles patch operations with path filtering for members
// This supports paths like members[value eq "422"] for add/replace/remove operations
func (g *GroupHandler) patchMembersWithPathFiltering(ctx context.Context, op scim.PatchOperation, group *fleet.ScimGroup) error {
	memberID, err := g.getMemberID(ctx, op)
	if err != nil {
		return err
	}

	// Check if the member exists in the group
	memberFound := false
	var memberIndex int
	for i, id := range group.ScimUsers {
		if id == memberID {
			memberIndex = i
			memberFound = true
			break
		}
	}

	// For remove operations, remove the member if found
	if op.Op == scim.PatchOperationRemove {
		if !memberFound {
			g.logger.InfoContext(ctx, "member not found in group", "member_id", memberID, "op", fmt.Sprintf("%v", op))
			// The member may have been removed already from this group. For example, if the member was deleted.
			return nil
		}
		group.ScimUsers = append(group.ScimUsers[:memberIndex], group.ScimUsers[memberIndex+1:]...)
		return nil
	}

	// For add operations, add the member if not found
	if op.Op == scim.PatchOperationAdd && !memberFound {
		// Verify the user exists
		userExists, err := g.ds.ScimUsersExist(ctx, []uint{memberID})
		if err != nil {
			g.logger.ErrorContext(ctx, "error checking user existence", "err", err)
			return err
		}
		if !userExists {
			g.logger.InfoContext(ctx, "user not found", "user_id", memberID)
			return errors.ScimErrorBadParams([]string{scimUserID(memberID)})
		}
		group.ScimUsers = append(group.ScimUsers, memberID)
		return nil
	}

	// For replace operations with a value
	if op.Op == scim.PatchOperationReplace {
		if !memberFound {
			g.logger.InfoContext(
				ctx, "member not found for replace operation", "members.value", memberID, "op", fmt.Sprintf("%v", op),
			)
			return errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
		}

		// If the value is nil or an empty object, remove the member
		if op.Value == nil {
			group.ScimUsers = append(group.ScimUsers[:memberIndex], group.ScimUsers[memberIndex+1:]...)
			return nil
		}

		// Otherwise, we don't change anything since we're already filtering by the member ID
		// and there are no other attributes to modify for a member
		return nil
	}

	return nil
}

// getMemberID extracts the member ID from a path expression like members[value eq "422"]
func (g *GroupHandler) getMemberID(ctx context.Context, op scim.PatchOperation) (uint, error) {
	attrExpression, ok := op.Path.ValueExpression.(*filter.AttributeExpression)
	if !ok {
		g.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path)
		return 0, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	// Only matching by member value (user ID) is supported
	if attrExpression.AttributePath.String() != valueAttr || attrExpression.Operator != filter.EQ {
		g.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path, "expression", attrExpression.AttributePath.String())
		return 0, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	memberIDStr, ok := attrExpression.CompareValue.(string)
	if !ok {
		g.logger.InfoContext(ctx, "unsupported patch path", "path", op.Path, "compare_value", attrExpression.CompareValue)
		return 0, errors.ScimErrorBadParams([]string{fmt.Sprintf("%v", op)})
	}

	// Extract user ID from the value
	userID, err := extractUserIDFromValue(memberIDStr)
	if err != nil {
		g.logger.InfoContext(ctx, "invalid user ID format", "value", memberIDStr, "err", err)
		return 0, errors.ScimErrorBadParams([]string{memberIDStr})
	}

	return userID, nil
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
