package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20261126170810(t *testing.T) {
	type fixture struct {
		db       *sqlx.DB
		scriptID int64
	}
	setup := func(t *testing.T) fixture {
		db := applyUpToPrev(t)
		scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('echo')), 'echo')`)
		return fixture{db: db, scriptID: scriptID}
	}
	addFMA := func(t *testing.T, f fixture, name, slug, identifier string) int64 {
		return execNoErrLastID(t, f.db, `INSERT INTO fleet_maintained_apps (name, slug, platform, unique_identifier) VALUES (?, ?, 'darwin', ?)`,
			name, slug, identifier)
	}
	addTitle := func(t *testing.T, f fixture, name, bundleID string) int64 {
		return execNoErrLastID(t, f.db, `INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, 'apps', ?)`, name, bundleID)
	}
	addTeam := func(t *testing.T, f fixture, name string) int64 {
		return execNoErrLastID(t, f.db, `INSERT INTO teams (name) VALUES (?)`, name)
	}
	addInstaller := func(t *testing.T, f fixture, teamID, titleID int64, fmaID *int64, version string) int64 {
		return execNoErrLastID(t, f.db, `
			INSERT INTO software_installers
				(team_id, global_or_team_id, title_id, fleet_maintained_app_id, filename, version, platform,
				 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
			VALUES (NULLIF(?, 0), ?, ?, ?, 'app.zip', ?, 'darwin', ?, ?, ?, '', '')`,
			teamID, teamID, titleID, fmaID, version, f.scriptID, f.scriptID, fmt.Sprintf("storage-%d-%s", teamID, version))
	}
	titleOf := func(t *testing.T, f fixture, installerID int64) int64 {
		var titleID int64
		require.NoError(t, f.db.Get(&titleID, `SELECT title_id FROM software_installers WHERE id = ?`, installerID))
		return titleID
	}
	identifierOf := func(t *testing.T, f fixture, slug string) string {
		var identifier string
		require.NoError(t, f.db.Get(&identifier, `SELECT unique_identifier FROM fleet_maintained_apps WHERE slug = ?`, slug))
		return identifier
	}
	runAgain := func(t *testing.T, f fixture) {
		tx, err := f.db.Begin()
		require.NoError(t, err)
		require.NoError(t, Up_20261126170810(tx))
		require.NoError(t, tx.Commit())
	}

	t.Run("relinks_chatgpt_installers", func(t *testing.T) {
		f := setup(t)

		chatgptFMA := addFMA(t, f, "ChatGPT Desktop", "chatgpt/darwin", "com.openai.chat")
		codexFMA := addFMA(t, f, "Codex", "codex-app/darwin", "com.openai.codex")
		firefoxFMA := addFMA(t, f, "Mozilla Firefox", "firefox/darwin", "org.mozilla.firefox")

		oldTitle := addTitle(t, f, "ChatGPT Desktop", "com.openai.chat")
		codexTitle := addTitle(t, f, "Codex", "com.openai.codex")
		firefoxTitle := addTitle(t, f, "Mozilla Firefox", "org.mozilla.firefox")

		teamA := addTeam(t, f, "A")
		teamB := addTeam(t, f, "B")
		teamC := addTeam(t, f, "C")
		teamD := addTeam(t, f, "D")
		teamE := addTeam(t, f, "E")
		teamF := addTeam(t, f, "F")
		const noTeam = 0

		// Fleet A: ChatGPT FMA with two cached versions, plus everything keyed on its title.
		installerA1 := addInstaller(t, f, teamA, oldTitle, &chatgptFMA, "1.0")
		installerA2 := addInstaller(t, f, teamA, oldTitle, &chatgptFMA, "2.0")
		firefoxA := addInstaller(t, f, teamA, firefoxTitle, &firefoxFMA, "150.0")
		execNoErr(t, f.db, `INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, 'ChatGPT')`, teamA, oldTitle)
		execNoErr(t, f.db, `INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename) VALUES (?, ?, 'icon', 'icon.png')`, teamA, oldTitle)
		execNoErr(t, f.db, `INSERT INTO software_title_team_pins (team_id, title_id, pinned_version) VALUES (?, ?, '1.0')`, teamA, oldTitle)
		patchPolicyA := execNoErrLastID(t, f.db, `
			INSERT INTO policies (team_id, name, query, description, checksum, patch_software_title_id)
			VALUES (?, 'Patch ChatGPT', 'SELECT 1', '', UNHEX(MD5('patch-a')), ?)`, teamA, oldTitle)
		execNoErr(t, f.db, `UPDATE policies SET updated_at = '2026-01-02 03:04:05' WHERE id = ?`, patchPolicyA)
		execNoErr(t, f.db, `UPDATE software_title_team_pins SET updated_at = '2026-01-02 03:04:05' WHERE team_id = ?`, teamA)
		execNoErr(t, f.db, `
			INSERT INTO host_software_installs (execution_id, host_id, software_installer_id, software_title_id, updated_at)
			VALUES ('install-a', 1, ?, ?, '2026-01-02 03:04:05')`, installerA2, oldTitle)
		upcomingA := execNoErrLastID(t, f.db, `
			INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
			VALUES (1, 'software_install', 'upcoming-a', '{}')`)
		execNoErr(t, f.db, `
			INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id, software_title_id, updated_at)
			VALUES (?, ?, ?, '2026-01-02 03:04:05')`, upcomingA, installerA2, oldTitle)
		execNoErr(t, f.db, `UPDATE software_installers SET updated_at = '2026-01-02 03:04:05' WHERE id = ?`, installerA2)

		// Fleet B: an uploaded package already owns the com.openai.codex title, so it's skipped.
		installerB := addInstaller(t, f, teamB, oldTitle, &chatgptFMA, "2.0")
		customB := addInstaller(t, f, teamB, codexTitle, nil, "custom")
		execNoErr(t, f.db, `INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, 'ChatGPT B')`, teamB, oldTitle)

		// Fleet C: only the Codex FMA, which is retired.
		installerC := addInstaller(t, f, teamC, codexTitle, &codexFMA, "26.623.141536")

		// No team: moved like any fleet, including its patch policy.
		installerNoTeam := addInstaller(t, f, noTeam, oldTitle, &chatgptFMA, "2.0")
		execNoErr(t, f.db, `
			INSERT INTO policies (team_id, name, query, description, checksum, patch_software_title_id)
			VALUES (?, 'Patch ChatGPT', 'SELECT 1', '', UNHEX(MD5('patch-no-team')), ?)`, noTeam, oldTitle)

		// Fleet D: a display name left on the com.openai.codex title by an installer that's gone is
		// replaced by the moved installer's.
		installerD := addInstaller(t, f, teamD, oldTitle, &chatgptFMA, "2.0")
		execNoErr(t, f.db, `INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, 'Stale')`, teamD, codexTitle)
		execNoErr(t, f.db, `INSERT INTO software_title_display_names (team_id, software_title_id, display_name) VALUES (?, ?, 'ChatGPT D')`, teamD, oldTitle)

		// Fleet E: a patch policy already targets the com.openai.codex title, so it's skipped.
		installerE := addInstaller(t, f, teamE, oldTitle, &chatgptFMA, "2.0")
		execNoErr(t, f.db, `
			INSERT INTO policies (team_id, name, query, description, checksum, patch_software_title_id)
			VALUES (?, 'Patch Codex', 'SELECT 1', '', UNHEX(MD5('patch-e')), ?)`, teamE, codexTitle)

		// Fleet F: an App Store app owns the com.openai.codex title, so it's skipped.
		installerF := addInstaller(t, f, teamF, oldTitle, &chatgptFMA, "2.0")
		execNoErr(t, f.db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES ('123', 'darwin', ?)`, codexTitle)
		execNoErr(t, f.db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, team_id, name) VALUES ('123', 'darwin', ?, ?, '')`, teamF, teamF)

		applyNext(t, f.db)

		check := func(t *testing.T) {
			var codexCount int
			require.NoError(t, f.db.Get(&codexCount, `SELECT COUNT(*) FROM fleet_maintained_apps WHERE slug = 'codex-app/darwin'`))
			require.Zero(t, codexCount)
			require.Equal(t, "com.openai.codex", identifierOf(t, f, "chatgpt/darwin"))
			require.Equal(t, "org.mozilla.firefox", identifierOf(t, f, "firefox/darwin"))

			require.Equal(t, codexTitle, titleOf(t, f, installerA1))
			require.Equal(t, codexTitle, titleOf(t, f, installerA2))
			require.Equal(t, firefoxTitle, titleOf(t, f, firefoxA))

			for _, stmt := range []string{
				`SELECT software_title_id FROM software_title_display_names WHERE team_id = ?`,
				`SELECT software_title_id FROM software_title_icons WHERE team_id = ?`,
				`SELECT title_id FROM software_title_team_pins WHERE team_id = ?`,
				`SELECT patch_software_title_id FROM policies WHERE team_id = ?`,
			} {
				var titleID int64
				require.NoError(t, f.db.Get(&titleID, stmt, teamA), stmt)
				require.Equal(t, codexTitle, titleID, stmt)
			}
			var policyID int64
			require.NoError(t, f.db.Get(&policyID, `SELECT id FROM policies WHERE team_id = ? AND patch_software_title_id = ?`, teamA, codexTitle))
			require.Equal(t, patchPolicyA, policyID)

			require.Equal(t, codexTitle, titleOf(t, f, installerNoTeam))
			var noTeamPolicyTitle int64
			require.NoError(t, f.db.Get(&noTeamPolicyTitle, `SELECT patch_software_title_id FROM policies WHERE team_id = ?`, noTeam))
			require.Equal(t, codexTitle, noTeamPolicyTitle)

			require.Equal(t, codexTitle, titleOf(t, f, installerD))
			var displayD []struct {
				TitleID int64  `db:"software_title_id"`
				Name    string `db:"display_name"`
			}
			require.NoError(t, f.db.Select(&displayD, `SELECT software_title_id, display_name FROM software_title_display_names WHERE team_id = ?`, teamD))
			require.Len(t, displayD, 1)
			require.Equal(t, codexTitle, displayD[0].TitleID)
			require.Equal(t, "ChatGPT D", displayD[0].Name)

			require.Equal(t, oldTitle, titleOf(t, f, installerE))
			require.Equal(t, oldTitle, titleOf(t, f, installerF))

			// The install time shown in self-service is updated_at, so moving rows must keep it.
			type moved struct {
				TitleID   int64     `db:"software_title_id"`
				UpdatedAt time.Time `db:"updated_at"`
			}
			wantUpdatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			var install, upcoming moved
			require.NoError(t, f.db.Get(&install, `SELECT software_title_id, updated_at FROM host_software_installs WHERE execution_id = 'install-a'`))
			require.Equal(t, moved{codexTitle, wantUpdatedAt}, install)
			require.NoError(t, f.db.Get(&upcoming, `SELECT software_title_id, updated_at FROM software_install_upcoming_activities WHERE upcoming_activity_id = ?`, upcomingA))
			require.Equal(t, moved{codexTitle, wantUpdatedAt}, upcoming)
			var installerUpdatedAt time.Time
			require.NoError(t, f.db.Get(&installerUpdatedAt, `SELECT updated_at FROM software_installers WHERE id = ?`, installerA2))
			require.Equal(t, wantUpdatedAt, installerUpdatedAt)
			for _, stmt := range []string{
				`SELECT updated_at FROM policies WHERE team_id = ?`,
				`SELECT updated_at FROM software_title_team_pins WHERE team_id = ?`,
			} {
				var updatedAt time.Time
				require.NoError(t, f.db.Get(&updatedAt, stmt, teamA), stmt)
				require.Equal(t, wantUpdatedAt, updatedAt, stmt)
			}

			require.Equal(t, oldTitle, titleOf(t, f, installerB))
			require.Equal(t, codexTitle, titleOf(t, f, customB))
			var displayB int64
			require.NoError(t, f.db.Get(&displayB, `SELECT software_title_id FROM software_title_display_names WHERE team_id = ?`, teamB))
			require.Equal(t, oldTitle, displayB)

			var fmaC *int64
			require.NoError(t, f.db.Get(&fmaC, `SELECT fleet_maintained_app_id FROM software_installers WHERE id = ?`, installerC))
			require.Nil(t, fmaC)
			require.Equal(t, codexTitle, titleOf(t, f, installerC))

			var oldTitleCount int
			require.NoError(t, f.db.Get(&oldTitleCount, `SELECT COUNT(*) FROM software_titles WHERE id = ?`, oldTitle))
			require.Equal(t, 1, oldTitleCount)
		}
		check(t)

		runAgain(t, f)
		check(t)
	})

	t.Run("creates_missing_title", func(t *testing.T) {
		f := setup(t)

		chatgptFMA := addFMA(t, f, "ChatGPT Desktop", "chatgpt/darwin", "com.openai.chat")
		oldTitle := addTitle(t, f, "ChatGPT Desktop", "com.openai.chat")
		team := addTeam(t, f, "A")
		installer := addInstaller(t, f, team, oldTitle, &chatgptFMA, "2.0")

		applyNext(t, f.db)

		var title struct {
			ID     int64  `db:"id"`
			Name   string `db:"name"`
			Source string `db:"source"`
		}
		require.NoError(t, f.db.Get(&title, `SELECT id, name, source FROM software_titles WHERE bundle_identifier = 'com.openai.codex'`))
		require.Equal(t, "ChatGPT Desktop", title.Name)
		require.Equal(t, "apps", title.Source)
		require.Equal(t, title.ID, titleOf(t, f, installer))
		require.Equal(t, "com.openai.codex", identifierOf(t, f, "chatgpt/darwin"))
	})

	t.Run("no_maintained_apps", func(t *testing.T) {
		f := setup(t)
		applyNext(t, f.db)

		var count int
		require.NoError(t, f.db.Get(&count, `SELECT COUNT(*) FROM software_titles`))
		require.Zero(t, count)
	})
}
