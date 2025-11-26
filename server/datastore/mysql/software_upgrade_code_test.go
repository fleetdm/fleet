package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSoftwareTitlesUpgradeCode(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	nonEmptyUc := "{A681CB20-907E-428A-9B14-2D3C1AFED244}"
	emptyUc := ""

	testCases := []struct {
		name                              string
		incomingSwChecksumToSw            map[string]fleet.Software
		incomingSwChecksumToMatchingTitle map[string]fleet.SoftwareTitleSummary
		expectedUpdates                   map[uint]string // titleID -> value of expected upgrade_code
		expectedInfoLogs                  int
		expectedWarningLogs               int
	}{
		{
			name: "update empty upgrade_code with non-empty value",
			incomingSwChecksumToSw: map[string]fleet.Software{
				"checksum1": {
					Name:        "Software 1",
					Version:     "1.0",
					Source:      "programs",
					UpgradeCode: &nonEmptyUc,
				},
			},
			incomingSwChecksumToMatchingTitle: map[string]fleet.SoftwareTitleSummary{
				"checksum1": {
					ID:          1,
					Name:        "Software 1",
					Source:      "programs",
					UpgradeCode: &emptyUc,
				},
			},
			expectedUpdates: map[uint]string{
				1: nonEmptyUc,
			},
			expectedInfoLogs: 1,
		},
		{
			name: "skip when incoming software has empty upgrade_code",
			incomingSwChecksumToSw: map[string]fleet.Software{
				"checksum2": {
					Name:        "Software 2",
					Version:     "3.0",
					Source:      "programs",
					UpgradeCode: &emptyUc,
				},
			},
			incomingSwChecksumToMatchingTitle: map[string]fleet.SoftwareTitleSummary{
				"checksum2": {
					ID:          2,
					Name:        "Software 2",
					Source:      "programs",
					UpgradeCode: &emptyUc,
				},
			},
			expectedUpdates: map[uint]string{},
		},
		{
			name: "skip and log warning when existing title has different, non-empty upgrade_code",
			incomingSwChecksumToSw: map[string]fleet.Software{
				"checksum3": {
					Name:        "Software 3",
					Version:     "3.0",
					Source:      "programs",
					UpgradeCode: &nonEmptyUc,
				},
			},
			incomingSwChecksumToMatchingTitle: map[string]fleet.SoftwareTitleSummary{
				"checksum3": {
					ID:          3,
					Name:        "Software 3",
					Source:      "programs",
					UpgradeCode: ptr.String("different_uc_value"),
				},
			},
			expectedUpdates:     map[uint]string{},
			expectedWarningLogs: 1,
		},
		{
			name: "log warning and replace when existing title has NULL upgrade_code",
			incomingSwChecksumToSw: map[string]fleet.Software{
				"checksum4": {
					Name:        "Software 4",
					Version:     "5.0",
					Source:      "programs",
					UpgradeCode: &nonEmptyUc,
				},
			},
			incomingSwChecksumToMatchingTitle: map[string]fleet.SoftwareTitleSummary{
				"checksum4": {
					ID:          4,
					Name:        "Software 4",
					Source:      "programs",
					UpgradeCode: nil, // shouldn't happen, warn and replace
				},
			},
			expectedUpdates: map[uint]string{
				4: nonEmptyUc,
			},
			expectedWarningLogs: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, existingTitle := range tc.incomingSwChecksumToMatchingTitle {

				_, err := ds.writer(ctx).ExecContext(ctx, `
					INSERT INTO software_titles (id, name, source, upgrade_code)
					VALUES (?, ?, ?, ?)`,
					existingTitle.ID, existingTitle.Name, existingTitle.Source, existingTitle.UpgradeCode,
				)
				require.NoError(t, err)
			}

			err := ds.reconcileExistingTitleEmptyUpgradeCodes(ctx, tc.incomingSwChecksumToSw, tc.incomingSwChecksumToMatchingTitle)
			require.NoError(t, err)

			for titleID, expectedUpgradeCode := range tc.expectedUpdates {
				var updatedUC *string
				err := sqlx.GetContext(ctx, ds.reader(ctx), &updatedUC,
					`SELECT upgrade_code FROM software_titles WHERE id = ?`, titleID)
				require.NoError(t, err)
				assert.Equal(t, expectedUpgradeCode, *updatedUC,
					"Title %d should have upgrade_code updated", titleID)
			}

			// Verify non-updated titles remain unchanged
			for _, title := range tc.incomingSwChecksumToMatchingTitle {
				if _, shouldUpdate := tc.expectedUpdates[title.ID]; !shouldUpdate {
					var actualUpgradeCode *string
					err := sqlx.GetContext(ctx, ds.reader(ctx), &actualUpgradeCode,
						`SELECT upgrade_code FROM software_titles WHERE id = ?`, title.ID)
					require.NoError(t, err)

					if title.UpgradeCode == nil {
						assert.Nil(t, actualUpgradeCode, "Title %d should still have NULL upgrade_code", title.ID)
					} else {
						require.NotNil(t, actualUpgradeCode)
						assert.Equal(t, *title.UpgradeCode, *actualUpgradeCode,
							"Title %d upgrade_code should remain unchanged", title.ID)
					}
				}

				_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM software_titles WHERE id = ?`, title.ID)
				require.NoError(t, err)
			}
		})
	}
}
