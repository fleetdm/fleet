package ghapi

import (
	"fmt"
	"strings"
)

// SizePair joins the same issue's items in two projects.
type SizePair struct {
	Local  ProjectItem // item in the selected project
	Remote ProjectItem // item in the release planning project
}

// SizeSyncPlan describes how to reconcile size values between two projects.
type SizeSyncPlan struct {
	SetLocal  []SizePair // remote has a size, local doesn't: copy remote -> local
	SetRemote []SizePair // local has a size, remote doesn't: copy local -> remote
	Conflicts []SizePair // both set, different values: needs a human decision
}

// PlanSizeSync matches local items to remote items by issue number and
// classifies each pair by how their size values need to be reconciled. Local
// items without a remote counterpart (and vice versa) are ignored.
func PlanSizeSync(local, remote []ProjectItem) SizeSyncPlan {
	remoteByNumber := make(map[int]ProjectItem, len(remote))
	for _, it := range remote {
		if it.Content.Number != 0 {
			remoteByNumber[it.Content.Number] = it
		}
	}

	var plan SizeSyncPlan
	for _, it := range local {
		if it.Content.Number == 0 {
			continue
		}
		rp, ok := remoteByNumber[it.Content.Number]
		if !ok {
			continue
		}
		ls, rs := it.SizeValue(), rp.SizeValue()
		pair := SizePair{Local: it, Remote: rp}
		switch {
		case strings.EqualFold(ls, rs):
			// In sync (including both unset).
		case ls == "":
			plan.SetLocal = append(plan.SetLocal, pair)
		case rs == "":
			plan.SetRemote = append(plan.SetRemote, pair)
		default:
			plan.Conflicts = append(plan.Conflicts, pair)
		}
	}
	return plan
}

// LookupSizeFieldName finds a project's size field, which is named "Size" on
// product group boards and "T-shirt size" on release planning.
func LookupSizeFieldName(projectID int) (string, error) {
	fields, err := LoadProjectFields(projectID)
	if err != nil {
		return "", err
	}
	if _, ok := fields["Size"]; ok {
		return "Size", nil
	}
	for name, field := range fields {
		if strings.Contains(strings.ToLower(name), "size") && len(field.Options) > 0 {
			return name, nil
		}
	}
	return "", fmt.Errorf("no size field found in project %d", projectID)
}

// SetItemSize sets a project item's size field, resolving the project's name
// for that field first.
func SetItemSize(projectID int, itemID, value string) error {
	fieldName, err := LookupSizeFieldName(projectID)
	if err != nil {
		return err
	}
	return SetProjectItemFieldValue(itemID, projectID, fieldName, value)
}
