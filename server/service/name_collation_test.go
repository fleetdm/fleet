package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
)

// mockDuplicateStringGroupsExactMatch stands in for the MySQL collation with exact string equality, which is enough for mocked
// tests; integration tests cover the collation itself.
func mockDuplicateStringGroupsExactMatch(ds *mock.Store) {
	ds.GetDuplicateStringGroupsUnderCollationFunc = func(ctx context.Context, values []string) ([]fleet.DuplicateStringGroup, error) {
		indicesByValue := make(map[string][]int, len(values))
		var order []string
		for i, value := range values {
			if _, ok := indicesByValue[value]; !ok {
				order = append(order, value)
			}
			indicesByValue[value] = append(indicesByValue[value], i)
		}
		var groups []fleet.DuplicateStringGroup
		for _, value := range order {
			if len(indicesByValue[value]) > 1 {
				groups = append(groups, fleet.DuplicateStringGroup{Indices: indicesByValue[value]})
			}
		}
		return groups, nil
	}
}
