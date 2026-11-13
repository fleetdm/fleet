package tables

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

func init() {
	MigrationClient.AddMigration(Up_20261113214300, Down_20261113214300)
}

const fleetNameMaxLength20261113214300 = 255

// fleetNameKey20261113214300 is a frozen copy of fleet.TeamNameKey, so this migration keeps renaming the same fleets
// if that rule changes later.
func fleetNameKey20261113214300(name string) string {
	folded := norm.NFKD.String(cases.Fold().String(norm.NFKD.String(name)))
	var key strings.Builder
	for _, r := range folded {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			key.WriteRune(r)
		}
	}
	return key.String()
}

type renamedFleet20261113214300 struct {
	id      uint
	oldName string
	newName string
}

// Up_20261113214300 renames fleets that break the Fleet 5 name rules: a name needs at least one letter or number, and
// must not match another fleet's name by name key or under the teams.name collation.
//
// Fleets are processed in id order and each rename is written right away, so the teams table always holds the final
// names of the fleets already processed and the current names of the rest. A fleet whose name has no letter or number is
// numbered from 1. A fleet that matches a name an earlier fleet ended up with is numbered from 2, so the oldest fleet
// keeps its name. A number is skipped if the result would match any name in the table.
func Up_20261113214300(tx *sql.Tx) error {
	type fleetRow struct {
		id   uint
		name string
		key  string
	}
	var fleets []fleetRow
	liveKeys := make(map[string]int)
	err := forEachRow20261113214300(tx, `SELECT id, name FROM teams ORDER BY id`, func(rows *sql.Rows) error {
		var f fleetRow
		if err := rows.Scan(&f.id, &f.name); err != nil {
			return err
		}
		f.key = fleetNameKey20261113214300(f.name)
		liveKeys[f.key]++
		fleets = append(fleets, f)
		return nil
	})
	if err != nil {
		return fmt.Errorf("select fleets: %w", err)
	}

	// A new name never matches a current name, so a fleet can only match an earlier fleet under the collation if it
	// did before the migration. That limits the per-fleet collation query to these fleets.
	matchedEarlierFleet := make(map[uint]struct{})
	err = forEachRow20261113214300(tx, `
		SELECT DISTINCT later.id
		FROM teams later
		JOIN teams earlier ON earlier.name = later.name AND earlier.id < later.id`,
		func(rows *sql.Rows) error {
			var id uint
			if err := rows.Scan(&id); err != nil {
				return err
			}
			matchedEarlierFleet[id] = struct{}{}
			return nil
		})
	if err != nil {
		return fmt.Errorf("select fleets matching an earlier fleet: %w", err)
	}

	nameTaken := func(stmt, name string, id uint) (bool, error) {
		var taken bool
		if err := tx.QueryRow(stmt, name, id).Scan(&taken); err != nil {
			return false, fmt.Errorf("check fleet name %q under the teams.name collation: %w", name, err)
		}
		return taken, nil
	}

	finalKeys := make(map[string]struct{}, len(fleets))
	var renamed []renamedFleet20261113214300
	for _, f := range fleets {
		liveKeys[f.key]--

		_, keyTaken := finalKeys[f.key]
		_, mayMatchEarlierFleet := matchedEarlierFleet[f.id]
		firstNumber := 0
		switch {
		case f.key == "":
			firstNumber = 1
		case keyTaken:
			firstNumber = 2
		case mayMatchEarlierFleet:
			taken, err := nameTaken(`SELECT EXISTS (SELECT 1 FROM teams WHERE name = ? AND id < ?)`, f.name, f.id)
			if err != nil {
				return err
			}
			if taken {
				firstNumber = 2
			}
		}

		finalName, finalKey := f.name, f.key
		if firstNumber > 0 {
			for n := firstNumber; ; n++ {
				finalName = numberedFleetName20261113214300(f.name, n)
				finalKey = fleetNameKey20261113214300(finalName)
				if liveKeys[finalKey] > 0 {
					continue
				}
				taken, err := nameTaken(`SELECT EXISTS (SELECT 1 FROM teams WHERE name = ? AND id != ?)`, finalName, f.id)
				if err != nil {
					return err
				}
				if !taken {
					break
				}
			}
			if _, err := tx.Exec(`UPDATE teams SET name = ? WHERE id = ?`, finalName, f.id); err != nil {
				return fmt.Errorf("rename fleet %d: %w", f.id, err)
			}
			logger.Info.Printf("Renamed fleet ID %d from %q to %q to follow the Fleet 5 fleet name rules.\n", f.id, f.name, finalName)
			renamed = append(renamed, renamedFleet20261113214300{id: f.id, oldName: f.name, newName: finalName})
		}
		finalKeys[finalKey] = struct{}{}
		liveKeys[finalKey]++
	}

	if len(renamed) == 0 {
		return nil
	}
	return rewriteFleetNameCopies20261113214300(tx, renamed)
}

// forEachRow20261113214300 closes the rows before returning, so the transaction's connection is free for the next query.
func forEachRow20261113214300(tx *sql.Tx, query string, scan func(*sql.Rows) error) error {
	rows, err := tx.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// numberedFleetName20261113214300 appends " n" to name, shortening name so the result fits the teams.name column.
func numberedFleetName20261113214300(name string, n int) string {
	suffix := " " + strconv.Itoa(n)
	if runes := []rune(name); len(runes)+len(suffix) > fleetNameMaxLength20261113214300 {
		name = string(runes[:fleetNameMaxLength20261113214300-len(suffix)])
	}
	return name + suffix
}

// rewriteFleetNameCopies20261113214300 updates the copies of the renamed fleets' names that app config keeps for
// Apple Business and VPP tokens, the same copies a rename through the API updates. abm_tokens and vpp_token_teams are
// the source of truth, so only the entries of tokens assigned to a renamed fleet are touched. Keys are matched in both
// their Fleet 4 and Fleet 5 spellings.
func rewriteFleetNameCopies20261113214300(tx *sql.Tx, renamed []renamedFleet20261113214300) error {
	var raw []byte
	switch err := tx.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&raw); {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("select app config: %w", err)
	}
	var appConfig struct {
		MDM map[string]any `json:"mdm"`
	}
	if err := json.Unmarshal(raw, &appConfig); err != nil {
		return fmt.Errorf("unmarshal app config: %w", err)
	}

	// The app config copy is what the admin typed, which can be in a different Unicode normalization form than the
	// fleet's name.
	sameName := func(copied any, name string) bool {
		s, ok := copied.(string)
		return ok && s != "" && norm.NFC.String(s) == norm.NFC.String(name)
	}

	type nameCopy struct{ path, newName string }
	var copies []nameCopy
	set := func(path, newName string) {
		copies = append(copies, nameCopy{path: path, newName: newName})
	}

	abmFleetIDs := make(map[string]map[uint]struct{}) // organization name to default fleet IDs
	err := forEachRow20261113214300(tx, `
		SELECT organization_name, macos_default_team_id, ios_default_team_id, ipados_default_team_id, byod_default_team_id
		FROM abm_tokens`,
		func(rows *sql.Rows) error {
			var orgName string
			var ids [4]sql.Null[uint]
			if err := rows.Scan(&orgName, &ids[0], &ids[1], &ids[2], &ids[3]); err != nil {
				return err
			}
			abmFleetIDs[orgName] = make(map[uint]struct{})
			for _, id := range ids {
				if id.Valid {
					abmFleetIDs[orgName][id.V] = struct{}{}
				}
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("select Apple Business tokens: %w", err)
	}
	abmNameKeys := []string{
		"macos_team", "ios_team", "ipados_team", "byod_team",
		"macos_fleet", "ios_fleet", "ipados_fleet", "byod_fleet",
	}
	for _, listKey := range []string{"apple_business_manager", "apple_business"} {
		entries, _ := appConfig.MDM[listKey].([]any)
		for i, e := range entries {
			entry, _ := e.(map[string]any)
			orgName, _ := entry["organization_name"].(string)
			for _, nameKey := range abmNameKeys {
				for _, r := range renamed {
					if _, ok := abmFleetIDs[orgName][r.id]; ok && sameName(entry[nameKey], r.oldName) {
						set(fmt.Sprintf("$.mdm.%s[%d].%s", listKey, i, nameKey), r.newName)
						break
					}
				}
			}
		}
	}

	vppFleetIDs := make(map[string]map[uint]struct{}) // location to assigned fleet IDs
	err = forEachRow20261113214300(tx, `
		SELECT vt.location, vtt.team_id
		FROM vpp_token_teams vtt
		JOIN vpp_tokens vt ON vt.id = vtt.vpp_token_id
		WHERE vtt.team_id IS NOT NULL`,
		func(rows *sql.Rows) error {
			var location string
			var id uint
			if err := rows.Scan(&location, &id); err != nil {
				return err
			}
			if vppFleetIDs[location] == nil {
				vppFleetIDs[location] = make(map[uint]struct{})
			}
			vppFleetIDs[location][id] = struct{}{}
			return nil
		})
	if err != nil {
		return fmt.Errorf("select VPP token fleets: %w", err)
	}
	entries, _ := appConfig.MDM["volume_purchasing_program"].([]any)
	for i, e := range entries {
		entry, _ := e.(map[string]any)
		location, _ := entry["location"].(string)
		for _, listKey := range []string{"teams", "fleets"} {
			names, _ := entry[listKey].([]any)
			for j, name := range names {
				for _, r := range renamed {
					if _, ok := vppFleetIDs[location][r.id]; ok && sameName(name, r.oldName) {
						set(fmt.Sprintf("$.mdm.volume_purchasing_program[%d].%s[%d]", i, listKey, j), r.newName)
						break
					}
				}
			}
		}
	}

	// Deprecated, and not tied to a token.
	for _, r := range renamed {
		if sameName(appConfig.MDM["apple_bm_default_team"], r.oldName) {
			set("$.mdm.apple_bm_default_team", r.newName)
			break
		}
	}

	for _, c := range copies {
		if _, err := tx.Exec(`UPDATE app_config_json SET json_value = JSON_SET(json_value, ?, ?) WHERE id = 1`, c.path, c.newName); err != nil {
			return fmt.Errorf("update fleet name at %s in app config: %w", c.path, err)
		}
	}
	return nil
}

func Down_20261113214300(tx *sql.Tx) error {
	return nil
}
