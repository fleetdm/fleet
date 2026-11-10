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
	_, err := loadLabelsToAssociate(ctx, ds, entityTeamID, labelNames, user)
	return err
}

// loadLabelsToAssociate is verifyLabelsToAssociate that also returns the verified labels by name.
func loadLabelsToAssociate(ctx context.Context, ds fleet.Datastore, entityTeamID *uint, labelNames []string, user *fleet.User) (map[string]*fleet.Label, error) {
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

// verifyNoBuiltInLabels rejects built-in labels used as custom targets; platform targeting should be used instead.
func verifyNoBuiltInLabels(ctx context.Context, labelNames []string, labels map[string]*fleet.Label) error {
	for _, name := range labelNames {
		if label, ok := labels[name]; ok && label.LabelType == fleet.LabelTypeBuiltIn {
			return ctxerr.Wrap(ctx, badRequestf(
				"'%s' label is built-in. Only custom labels are supported. If you want to target a specific platform please use 'platform' instead. If not, please create a custom label and try again.",
				label.Name,
			))
		}
	}
	return nil
}
