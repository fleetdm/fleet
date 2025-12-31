package mysql

import (
	"context"
	"testing"
	"time"

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

// TestUpgradeCodeDuplicateEntryError tests the scenario from GitHub issue #37494:
// Two titles with different names but the same upgrade_code. When reconciliation
// tries to update one title's upgrade_code, it conflicts with the other title's
// unique_identifier (which is based on upgrade_code when set).
//
// Scenario:
// 1. Title A: "Visual Studio 2022" exists with upgrade_code="{guid}"
// 2. Title B: "Visual Studio Community 2022" exists with upgrade_code=""
// 3. Host reports "Visual Studio Community 2022" with upgrade_code="{guid}"
// 4. Name matching finds Title B, but upgrade_code conflicts with Title A
// 5. Fix: Redirect mapping to Title A, software gets linked to Title A
func TestUpgradeCodeDuplicateEntryError(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// Create a host
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-duplicate-uc",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("test-host-duplicate-uc"),
		NodeKey:         ptr.String("test-host-duplicate-uc"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	upgradeCode := "{EB6B8302-C06E-4BEC-ADAC-932C68A3A98D}"
	emptyUc := ""

	// Create Title A with upgrade_code set (simulates Host A already checked in)
	resultA, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software_titles (name, source, upgrade_code)
		VALUES (?, ?, ?)`,
		"Visual Studio 2022", "programs", upgradeCode,
	)
	require.NoError(t, err)
	titleAID, _ := resultA.LastInsertId()

	// Create Title B with empty upgrade_code (simulates pre-existing title)
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software_titles (name, source, upgrade_code)
		VALUES (?, ?, ?)`,
		"Visual Studio Community 2022", "programs", emptyUc,
	)
	require.NoError(t, err)

	// Host reports software with Title B's name but same upgrade_code as Title A
	incomingSoftware := []fleet.Software{
		{
			Name:        "Visual Studio Community 2022",
			Version:     "17.0.0",
			Source:      "programs",
			UpgradeCode: &upgradeCode,
		},
	}

	// This should NOT error - the fix redirects the software to Title A
	_, err = ds.UpdateHostSoftware(ctx, host.ID, incomingSoftware)
	require.NoError(t, err)

	// Verify the software was linked to Title A (the one with matching upgrade_code)
	var softwareEntries []struct {
		ID      uint   `db:"id"`
		Name    string `db:"name"`
		TitleID *uint  `db:"title_id"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &softwareEntries, `
		SELECT s.id, s.name, s.title_id
		FROM software s
		WHERE s.name = 'Visual Studio Community 2022' AND s.source = 'programs'
	`)
	require.NoError(t, err)
	require.Len(t, softwareEntries, 1)
	require.NotNil(t, softwareEntries[0].TitleID)
	assert.Equal(t, uint(titleAID), *softwareEntries[0].TitleID,
		"Software should be linked to Title A (the one with matching upgrade_code)")
}

// TestUpgradeCodeCaseSensitivityWithExistingUpgradeCode tests that case differences
// in software names are handled correctly when matching with existing titles.
//
// Scenario:
// 1. Title "QEMU Guest Agent" exists with upgrade_code="{guid}" (from earlier ingestion)
// 2. Host reports "QEMU guest agent" (different case) with same upgrade_code
// 3. Case-insensitive matching finds the existing title
// 4. Software is linked to the existing title
func TestUpgradeCodeCaseSensitivityWithExistingUpgradeCode(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// Create a host
	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:        "test-host-case-with-uc",
		Platform:        "windows",
		OsqueryHostID:   ptr.String("test-host-case-with-uc"),
		NodeKey:         ptr.String("test-host-case-with-uc"),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	upgradeCode := "{EB6B8302-C06E-4BEC-ADAC-932C68A3A98D}"

	// Create title with upgrade_code already set and different casing
	result, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO software_titles (name, source, upgrade_code)
		VALUES (?, ?, ?)`,
		"QEMU Guest Agent", "programs", upgradeCode,
	)
	require.NoError(t, err)
	titleID, _ := result.LastInsertId()

	// Host reports software with different casing but same upgrade_code
	incomingSoftware := []fleet.Software{
		{
			Name:        "QEMU guest agent", // lowercase
			Version:     "107.0.1",
			Source:      "programs",
			UpgradeCode: &upgradeCode,
		},
	}

	_, err = ds.UpdateHostSoftware(ctx, host.ID, incomingSoftware)
	require.NoError(t, err)

	// Verify the software was linked to the existing title (not orphaned)
	var softwareEntries []struct {
		ID      uint   `db:"id"`
		Name    string `db:"name"`
		TitleID *uint  `db:"title_id"`
	}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &softwareEntries, `
		SELECT s.id, s.name, s.title_id
		FROM software s
		WHERE s.name LIKE '%QEMU%' AND s.source = 'programs'
	`)
	require.NoError(t, err)
	require.Len(t, softwareEntries, 1)

	// Software should be linked to the existing title despite case difference
	require.NotNil(t, softwareEntries[0].TitleID, "Software should have title_id set")
	assert.Equal(t, uint(titleID), *softwareEntries[0].TitleID,
		"Software should be linked to existing title")
}
