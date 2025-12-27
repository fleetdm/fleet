package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

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
	if len(labelNames) == 0 {
		return nil
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
		return ctxerr.Wrap(ctx, err, "labels by name")
	}

	if len(labels) != len(uniqueLabelNames) {
		return ctxerr.Wrap(ctx, badRequest("one or more labels specified do not exist, or cannot be applied to this entity"))
	}

	return nil
}
