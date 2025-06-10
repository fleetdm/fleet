//go:build windows
// +build windows

// based on github.com/kolide/launcher/pkg/osquery/tables
package windowsupdatetable

import (
	"context"
	"github.com/fleetdm/fleet/v4/orbit/pkg/windows/windowsupdate"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/tablehelpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type windowsUpdatesSearcherMock struct {
	searchCriteria string
	SearchResult   *windowsupdate.ISearchResult
	History        []*windowsupdate.IUpdateHistoryEntry
}

func (wus *windowsUpdatesSearcherMock) Search(criteria string) (*windowsupdate.ISearchResult, error) {
	wus.searchCriteria = criteria
	return wus.SearchResult, nil
}

func (wus *windowsUpdatesSearcherMock) QueryHistoryAll() ([]*windowsupdate.IUpdateHistoryEntry, error) {
	return wus.History, nil
}

func TestQueryUpdates(t *testing.T) {
	t.Run("the right criteria is used", func(t *testing.T) {
		searcher := &windowsUpdatesSearcherMock{}
		_, err := queryUpdates(searcher)
		require.NoError(t, err)

		criteriaParts := strings.Split(searcher.searchCriteria, " AND ")
		require.Contains(t, criteriaParts, "Type='Software'")
		require.Contains(t, criteriaParts, "IsInstalled=0")
	})

	t.Run("only return results iff any updates", func(t *testing.T) {
		testCases := []struct {
			updates []*windowsupdate.IUpdate
			isNil   bool
		}{
			{updates: nil, isNil: true},
			{updates: []*windowsupdate.IUpdate{}, isNil: true},
			{updates: []*windowsupdate.IUpdate{{}}, isNil: false},
		}

		for _, tt := range testCases {
			searcher := &windowsUpdatesSearcherMock{
				SearchResult: &windowsupdate.ISearchResult{
					Updates: tt.updates,
				},
			}
			r, err := queryUpdates(searcher)
			require.NoError(t, err)
			require.Equal(t, r == nil, tt.isNil)
		}
	})
}

func TestTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		queryFunc queryFuncType
	}{
		{name: "updates", queryFunc: queryUpdates},
		{name: "history", queryFunc: queryHistory},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := Table{
				logger:    zerolog.Nop(),
				queryFunc: tt.queryFunc,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			// ci doesn't return data, but we can, at least, check that the underlying API doesn't error.
			_, err := table.generate(ctx, tablehelpers.MockQueryContext(nil))
			require.NoError(t, err, "generate")
		})
	}
}
