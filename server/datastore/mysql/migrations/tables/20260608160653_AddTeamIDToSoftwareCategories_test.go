package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260608160653(t *testing.T) {
	db := applyUpToPrev(t)

	type seededCategory struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	var seeded []seededCategory
	require.NoError(t, db.Select(&seeded, `SELECT id, name FROM software_categories ORDER BY id`))
	require.NotEmpty(t, seeded, "earlier migrations should have seeded default categories")

	preID := make(map[string]uint, len(seeded))
	for _, s := range seeded {
		preID[s.Name] = s.ID
	}

	// Two teams plus apps on each (and one on the Unassigned scope) so the
	// migration has to: backfill defaults per team, re-point team-scoped link
	// rows in all three linking tables, and leave Unassigned-scope link rows
	// untouched.
	teamA := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team-a")) //nolint:gosec // dismiss G115
	teamB := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "team-b")) //nolint:gosec // dismiss G115

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, ?)`, "demo", "apps")
	execNoErr(t, db, `INSERT INTO script_contents (id, md5_checksum, contents) VALUES (1, 'demo-checksum', 'demo')`)

	const insertInstaller = `
INSERT INTO software_installers (title_id, global_or_team_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	installerA := execNoErrLastID(t, db, insertInstaller, titleID, teamA, "a.pkg", "1.0", "darwin", 1, 1, "storage-a", "", "")
	installerUnassigned := execNoErrLastID(t, db, insertInstaller, titleID, 0, "unassigned.pkg", "1.0", "darwin", 1, 1, "storage-u", "", "")
	browsersID := preID["Browsers"]
	communicationID := preID["Communication"]
	execNoErr(t, db, `INSERT INTO software_installer_software_categories (software_installer_id, software_category_id) VALUES (?, ?)`, installerA, browsersID)
	execNoErr(t, db, `INSERT INTO software_installer_software_categories (software_installer_id, software_category_id) VALUES (?, ?)`, installerUnassigned, communicationID)

	// VPP app on team B linked to "Developer tools".
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?, ?)`, "1234567890", "darwin")
	vppAppTeamB := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, global_or_team_id, platform) VALUES (?, ?, ?)`, "1234567890", teamB, "darwin")
	devToolsID := preID["Developer tools"]
	execNoErr(t, db, `INSERT INTO vpp_app_team_software_categories (vpp_app_team_id, software_category_id) VALUES (?, ?)`, vppAppTeamB, devToolsID)

	// In-house app on team A linked to "Productivity".
	inHouseAppA := execNoErrLastID(t, db, `INSERT INTO in_house_apps (global_or_team_id, storage_id, platform) VALUES (?, ?, ?)`, teamA, "storage-ih", "darwin")
	productivityID := preID["Productivity"]
	execNoErr(t, db, `INSERT INTO in_house_app_software_categories (in_house_app_id, software_category_id) VALUES (?, ?)`, inHouseAppA, productivityID)

	applyNext(t, db)

	type categoryRow struct {
		ID     uint   `db:"id"`
		Name   string `db:"name"`
		TeamID uint   `db:"team_id"`
	}

	// Defaults renamed to emoji-prefixed forms; IDs preserved.
	expectedRenames := map[string]string{
		"Browsers":        "🌎 Browsers",
		"Communication":   "👬 Communication",
		"Developer tools": "🧰 Developer tools",
		"Productivity":    "🖥️ Productivity",
		"Security":        "🔐 Security",
		"Utilities":       "🛠️ Utilities",
	}
	for oldName, newName := range expectedRenames {
		oldID, ok := preID[oldName]
		if !ok {
			continue
		}
		var row categoryRow
		require.NoError(t, db.Get(&row, `SELECT id, name, team_id FROM software_categories WHERE id = ?`, oldID))
		require.Equal(t, newName, row.Name, "row %q should be renamed to %q", oldName, newName)
		require.Equal(t, uint(0), row.TeamID, "renamed row should be at team_id=0")
	}

	canonicalNames := []string{
		"🌎 Browsers",
		"👬 Communication",
		"🧰 Developer tools",
		"🖥️ Productivity",
		"🔐 Security",
		"🛠️ Utilities",
	}

	// Both teams have all 6 defaults in canonical order with sequential IDs.
	assertTeamCategories := func(teamID uint) []categoryRow {
		var rows []categoryRow
		require.NoError(t, db.Select(&rows,
			`SELECT id, name, team_id FROM software_categories WHERE team_id = ? ORDER BY id`, teamID))
		require.Len(t, rows, len(canonicalNames), "team %d should have all 6 default categories", teamID)
		for i, r := range rows {
			require.Equal(t, canonicalNames[i], r.Name, "team %d row %d should be %q", teamID, i, canonicalNames[i])
			if i > 0 {
				require.Equal(t, rows[i-1].ID+1, r.ID, "team %d rows should have sequential ids", teamID)
			}
		}
		return rows
	}
	teamARows := assertTeamCategories(teamA)
	teamBRows := assertTeamCategories(teamB)
	require.NotEqual(t, teamARows[0].ID, teamBRows[0].ID, "team A and team B should have distinct id blocks")

	// Team A's installer link now points at team A's "🌎 Browsers", not the
	// renamed team_id=0 row.
	var installerALinkedCatID uint
	require.NoError(t, db.Get(&installerALinkedCatID,
		`SELECT software_category_id FROM software_installer_software_categories WHERE software_installer_id = ?`,
		installerA))
	require.Equal(t, teamARows[0].ID, installerALinkedCatID, "team A installer should link to team A's 🌎 Browsers")
	require.Equal(t, "🌎 Browsers", teamARows[0].Name)

	// Unassigned installer link is unchanged — still pointing at the renamed
	// team_id=0 Communication row.
	var unassignedLinkedCatID uint
	require.NoError(t, db.Get(&unassignedLinkedCatID,
		`SELECT software_category_id FROM software_installer_software_categories WHERE software_installer_id = ?`,
		installerUnassigned))
	require.Equal(t, communicationID, unassignedLinkedCatID, "Unassigned installer link should be untouched")

	// VPP app on team B was re-pointed.
	var vppLinkedCatID uint
	require.NoError(t, db.Get(&vppLinkedCatID,
		`SELECT software_category_id FROM vpp_app_team_software_categories WHERE vpp_app_team_id = ?`,
		vppAppTeamB))
	require.Equal(t, teamBRows[2].ID, vppLinkedCatID, "team B VPP app should link to team B's 🧰 Developer tools")
	require.Equal(t, "🧰 Developer tools", teamBRows[2].Name)

	// In-house app on team A was re-pointed.
	var inHouseLinkedCatID uint
	require.NoError(t, db.Get(&inHouseLinkedCatID,
		`SELECT software_category_id FROM in_house_app_software_categories WHERE in_house_app_id = ?`,
		inHouseAppA))
	require.Equal(t, teamARows[3].ID, inHouseLinkedCatID, "team A in-house app should link to team A's 🖥️ Productivity")
	require.Equal(t, "🖥️ Productivity", teamARows[3].Name)

	assertLinkGone := func(query string, parentID uint, label string) {
		var dummy uint
		err := db.Get(&dummy, query, parentID)
		require.ErrorIs(t, err, sql.ErrNoRows, "%s link row should be gone after deleting its category", label)
	}
	execNoErr(t, db, `DELETE FROM software_categories WHERE id = ?`, teamARows[0].ID)
	assertLinkGone(`SELECT software_category_id FROM software_installer_software_categories WHERE software_installer_id = ?`,
		uint(installerA), "team A installer") //nolint:gosec // dismiss G115
	execNoErr(t, db, `DELETE FROM software_categories WHERE id = ?`, teamBRows[2].ID)
	assertLinkGone(`SELECT software_category_id FROM vpp_app_team_software_categories WHERE vpp_app_team_id = ?`,
		uint(vppAppTeamB), "team B VPP app") //nolint:gosec // dismiss G115
	execNoErr(t, db, `DELETE FROM software_categories WHERE id = ?`, teamARows[3].ID)
	assertLinkGone(`SELECT software_category_id FROM in_house_app_software_categories WHERE in_house_app_id = ?`,
		uint(inHouseAppA), "team A in-house app") //nolint:gosec // dismiss G115

	// Unique key on (team_id, name) rejects duplicates within a team.
	_, err := db.Exec(`INSERT INTO software_categories (name, team_id) VALUES (?, ?)`, "🌎 Browsers", uint(0))
	require.Error(t, err, "duplicate (team_id, name) should violate unique key")
}
