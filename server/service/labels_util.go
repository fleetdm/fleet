package service

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// Any fleet the user can see counts here; whether the entity can use the label is checked separately. Names match
// case-insensitively, as in the database, so case variants on different fleets count as one name.
func foundLabelNames(ctx context.Context, ds fleet.Datastore, names []string, user *fleet.User) (map[string]struct{}, error) {
	labels, err := ds.AllLabelIDsByName(ctx, names, fleet.TeamFilter{User: user})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}
	existing := make(map[string]struct{}, len(labels))
	for name := range labels {
		existing[strings.ToLower(name)] = struct{}{}
	}
	found := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, ok := existing[strings.ToLower(name)]; ok {
			found[name] = struct{}{}
		}
	}
	return found, nil
}

func loadLabelsFromNames(ctx context.Context, ds fleet.Datastore, labelNames []string, filter fleet.TeamFilter) (map[string]*fleet.Label, error) {
	labelsMap, err := ds.LabelsByName(ctx, labelNames, filter)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels by name")
	}
	// Make sure that all labels were found
	for _, labelName := range labelNames {
		if _, ok := labelsMap[labelName]; !ok {
			return nil, ctxerr.Wrap(ctx, badRequestf("label %q not found", labelName))
		}
	}
	return labelsMap, nil
}

func verifyLabelsToAssociate(ctx context.Context, ds fleet.Datastore, entityTeamID *uint, labelNames []string, user *fleet.User) error {
	_, err := labelsToAssociate(ctx, ds, entityTeamID, labelNames, user)
	return err
}

// labelsToAssociate resolves label names the way the entity sees them (global labels plus its fleet's labels). Fleets
// can share label names, so IDs must come from this lookup rather than one across every fleet the user can see.
func labelsToAssociate(ctx context.Context, ds fleet.Datastore, entityTeamID *uint, labelNames []string, user *fleet.User) (map[string]*fleet.Label, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}
	if user == nil {
		return nil, ctxerr.New(ctx, "Authentication required")
	}

	// Remove duplicate names.
	seen := make(map[string]struct{})
	uniqueLabelNames := make([]string, 0, len(labelNames))
	for _, s := range labelNames {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		uniqueLabelNames = append(uniqueLabelNames, s)
	}

	if entityTeamID == nil { // no-team/all-teams entities can only access global labels
		entityTeamID = ptr.Uint(0)
	}

	labels, err := loadLabelsFromNames(ctx, ds, uniqueLabelNames, fleet.TeamFilter{User: user, TeamID: entityTeamID})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "labels by name")
	}

	if len(labels) != len(uniqueLabelNames) {
		return nil, ctxerr.Wrap(ctx, badRequest("one or more labels specified do not exist, or cannot be applied to this entity"))
	}

	return labels, nil
}
