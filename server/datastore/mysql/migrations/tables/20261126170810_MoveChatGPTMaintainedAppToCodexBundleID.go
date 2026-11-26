package tables

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20261126170810, Down_20261126170810)
}

// Up_20261126170810 moves ChatGPT installers onto the com.openai.codex title. OpenAI rebuilt
// ChatGPT for macOS on its now discontinued Codex app, so current builds report com.openai.codex,
// and the maintained apps sync now stores that identifier for chatgpt/darwin. The published
// catalog keeps com.openai.chat, because older Fleet servers would move the installers without
// the references this migration carries along.
func Up_20261126170810(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Codex was removed from the catalog. Deleting it now instead of on the next sync keeps
	// com.openai.codex mapped to a single app, which the name reconcile job requires.
	if _, err := tx.Exec(`DELETE FROM fleet_maintained_apps WHERE slug = 'codex-app/darwin'`); err != nil {
		return fmt.Errorf("deleting Codex maintained app: %w", err)
	}

	var chatgpt struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}
	switch err := txx.Get(&chatgpt, `SELECT id, name FROM fleet_maintained_apps WHERE slug = 'chatgpt/darwin'`); {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("selecting ChatGPT maintained app: %w", err)
	}
	if _, err := tx.Exec(`UPDATE fleet_maintained_apps SET unique_identifier = 'com.openai.codex' WHERE id = ?`, chatgpt.ID); err != nil {
		return fmt.Errorf("updating ChatGPT maintained app identifier: %w", err)
	}

	var installers []struct {
		ID      uint `db:"id"`
		FleetID uint `db:"global_or_team_id"`
		TitleID uint `db:"title_id"`
	}
	if err := txx.Select(&installers, `
		SELECT id, global_or_team_id, title_id FROM software_installers
		WHERE fleet_maintained_app_id = ? AND title_id IS NOT NULL`, chatgpt.ID); err != nil {
		return fmt.Errorf("selecting ChatGPT installers: %w", err)
	}
	if len(installers) == 0 {
		return nil
	}

	var targetID uint
	switch err := txx.Get(&targetID, `
		SELECT id FROM software_titles
		WHERE bundle_identifier = 'com.openai.codex' AND additional_identifier = 0`); {
	case errors.Is(err, sql.ErrNoRows):
		res, err := tx.Exec(`
			INSERT INTO software_titles (name, source, bundle_identifier, extension_for)
			VALUES (?, 'apps', 'com.openai.codex', '')`, chatgpt.Name)
		if err != nil {
			return fmt.Errorf("creating com.openai.codex software title: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return fmt.Errorf("reading com.openai.codex software title id: %w", err)
		}
		targetID = uint(id) //nolint:gosec // auto-increment ids are positive
	case err != nil:
		return fmt.Errorf("selecting com.openai.codex software title: %w", err)
	}

	type fleetInstallers struct {
		ids       []uint
		oldTitles map[uint]struct{}
	}
	// A fleet's installers for one app share a title. If they don't, the (fleet, title) rows
	// below can't be merged without picking a winner, so the fleet is left alone.
	byFleet := make(map[uint]*fleetInstallers)
	for _, inst := range installers {
		if inst.TitleID == targetID {
			continue
		}
		g := byFleet[inst.FleetID]
		if g == nil {
			g = &fleetInstallers{oldTitles: make(map[uint]struct{})}
			byFleet[inst.FleetID] = g
		}
		g.ids = append(g.ids, inst.ID)
		g.oldTitles[inst.TitleID] = struct{}{}
	}

	var moved, skipped int
	for fleetID, g := range byFleet {
		if len(g.oldTitles) > 1 {
			logger.Info.Printf("Left the ChatGPT installers of fleet ID %d as they are: they are on more than one software title.\n", fleetID)
			skipped++
			continue
		}

		// A fleet gets one installer per title. When something already installs com.openai.codex
		// there (an uploaded ChatGPT package, the former Codex app), leave the fleet as it is rather
		// than deleting an admin's installer. A fleet has one patch policy per title, so an existing
		// one on the target is a conflict too.
		var conflict bool
		if err := tx.QueryRow(`
			SELECT
				EXISTS (SELECT 1 FROM software_installers WHERE global_or_team_id = ? AND title_id = ?)
				OR EXISTS (SELECT 1 FROM vpp_apps va
					JOIN vpp_apps_teams vat ON vat.adam_id = va.adam_id AND vat.platform = va.platform
					WHERE va.title_id = ? AND vat.global_or_team_id = ?)
				OR EXISTS (SELECT 1 FROM policies WHERE team_id = ? AND patch_software_title_id = ?)`,
			fleetID, targetID, targetID, fleetID, fleetID, targetID,
		).Scan(&conflict); err != nil {
			return fmt.Errorf("checking com.openai.codex title conflicts for fleet %d: %w", fleetID, err)
		}
		if conflict {
			logger.Info.Printf("Left the ChatGPT installer of fleet ID %d on its current software title: the com.openai.codex title "+
				"already has software or a patch policy in that fleet. To move it, delete that software or policy, then delete "+
				"ChatGPT and add it again.\n", fleetID)
			skipped++
			continue
		}

		// updated_at auto-updates, and host_software_installs.updated_at is the install time shown in
		// self-service and compared by patch notifications, so the moves keep it.
		byInstaller := []struct{ label, stmt string }{
			{"installers", `UPDATE software_installers SET title_id = ?, updated_at = updated_at WHERE id IN (?)`},
			{"host installs", `UPDATE host_software_installs FORCE INDEX (fk_host_software_installs_installer_id)
				SET software_title_id = ?, updated_at = updated_at WHERE software_installer_id IN (?)`},
			{"upcoming installs", `UPDATE software_install_upcoming_activities SET software_title_id = ?, updated_at = updated_at
				WHERE software_installer_id IN (?)`},
			{"patch notifications", `UPDATE IGNORE patch_notification_apps SET software_title_id = ? WHERE software_installer_id IN (?)`},
		}
		for _, u := range byInstaller {
			stmt, args, err := sqlx.In(u.stmt, targetID, g.ids)
			if err != nil {
				return fmt.Errorf("building ChatGPT %s statement: %w", u.label, err)
			}
			if _, err := tx.Exec(stmt, args...); err != nil {
				return fmt.Errorf("moving ChatGPT %s for fleet %d: %w", u.label, fleetID, err)
			}
		}

		var oldTitleID uint
		for id := range g.oldTitles {
			oldTitleID = id
		}
		if _, err := tx.Exec(`UPDATE policies SET patch_software_title_id = ?, updated_at = updated_at
			WHERE team_id = ? AND patch_software_title_id = ?`,
			targetID, fleetID, oldTitleID); err != nil {
			return fmt.Errorf("moving ChatGPT patch policies for fleet %d: %w", fleetID, err)
		}
		// These are keyed on (fleet, title). Nothing installs the target title in this fleet (checked
		// above), so a row it already has was left behind by a removed installer and is dropped
		// before the moved installer's rows take its place.
		byFleetTitle := []struct{ label, table, titleColumn, keepUpdatedAt string }{
			{"display names", "software_title_display_names", "software_title_id", ""},
			{"icons", "software_title_icons", "software_title_id", ""},
			{"version pins", "software_title_team_pins", "title_id", ", updated_at = updated_at"},
		}
		for _, u := range byFleetTitle {
			//nolint:gosec // table and column names are constants
			if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE team_id = ? AND %s = ?`, u.table, u.titleColumn),
				fleetID, targetID); err != nil {
				return fmt.Errorf("clearing stale ChatGPT %s for fleet %d: %w", u.label, fleetID, err)
			}
			//nolint:gosec // table and column names are constants
			if _, err := tx.Exec(fmt.Sprintf(`UPDATE %s SET %s = ?%s WHERE team_id = ? AND %[2]s = ?`, u.table, u.titleColumn, u.keepUpdatedAt),
				targetID, fleetID, oldTitleID); err != nil {
				return fmt.Errorf("moving ChatGPT %s for fleet %d: %w", u.label, fleetID, err)
			}
		}
		moved++
	}

	if moved > 0 || skipped > 0 {
		logger.Info.Printf("Moved ChatGPT installers to the com.openai.codex software title in %d fleets, skipped %d.\n", moved, skipped)
	}
	return nil
}

func Down_20261126170810(tx *sql.Tx) error {
	return nil
}
