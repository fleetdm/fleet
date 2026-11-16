package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// findDuplicateNameUnderCollation compares names the way the utf8mb4_unicode_ci name columns do, ignoring case, accents and
// trailing spaces, so a batch can't upsert two entries into one row. It returns the index in names of the first name that repeats an earlier one.
func findDuplicateNameUnderCollation(ctx context.Context, ds fleet.Datastore, names []string) (duplicateIndex int, found bool, err error) {
	if len(names) < 2 {
		return 0, false, nil
	}
	groups, err := ds.GetDuplicateStringGroupsUnderCollation(ctx, names)
	if err != nil {
		return 0, false, ctxerr.Wrap(ctx, err, "comparing names under collation")
	}
	if len(groups) == 0 {
		return 0, false, nil
	}
	// groups are ordered by first occurrence, so the earliest repeat can be in any group
	duplicateIndex = groups[0].Indices[1]
	for _, group := range groups[1:] {
		duplicateIndex = min(duplicateIndex, group.Indices[1])
	}
	return duplicateIndex, true, nil
}
