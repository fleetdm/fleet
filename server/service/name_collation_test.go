package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
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

func TestFindDuplicateNameUnderCollation(t *testing.T) {
	ds := new(mock.Store)
	mockDuplicateStringGroupsExactMatch(ds)

	testCases := []struct {
		name               string
		names              []string
		wantFound          bool
		wantDuplicateIndex int
	}{
		{name: "distinct names", names: []string{"a", "b"}},
		{name: "one duplicate", names: []string{"a", "b", "a"}, wantFound: true, wantDuplicateIndex: 2},
		{name: "earliest repeat is in a later group", names: []string{"a", "b", "b", "a"}, wantFound: true, wantDuplicateIndex: 2},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			duplicateIndex, found, err := findDuplicateNameUnderCollation(t.Context(), ds, tc.names)
			require.NoError(t, err)
			require.Equal(t, tc.wantFound, found)
			require.Equal(t, tc.wantDuplicateIndex, duplicateIndex)
		})
	}
}
