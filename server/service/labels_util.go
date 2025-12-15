package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func loadLabelsFromNames(ctx context.Context, ds fleet.Datastore, labelNames []string) (map[string]*fleet.Label, error) {
	labelsMap, err := ds.LabelsByName(ctx, labelNames)
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

func verifyLabelsToAssociate(ctx context.Context, ds fleet.Datastore, entityTeamID *uint, labelNames []string) error {
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

	// Load data of all labels.
	labels, err := loadLabelsFromNames(ctx, ds, uniqueLabelNames)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "labels by name")
	}

	// Perform team ID checks for "No team" or global entities.
	if entityTeamID == nil || *entityTeamID == 0 {
		// entityTeamID == nil: global entity (like "All teams" policies and "All team" queries)
		// entityTeamID == 0: "no team" entity.
		// For both cases, labels must be global because currently we don't support labels in "No team".
		for _, label := range labels {
			if label.TeamID != nil {
				return ctxerr.Wrap(ctx, badRequestf("label %q is a team label", label.Name))
			}
		}
		return nil
	}

	// Perform team ID checks for team entities.
	for _, label := range labels {
		// Team entities can reference global labels.
		if label.TeamID == nil {
			continue
		}
		// Team entities cannot reference labels that belong another team.
		if *label.TeamID != *entityTeamID {
			return ctxerr.Wrap(ctx, badRequestf("label %q belongs to a different team", label.Name))
		}
	}

	return nil
}
