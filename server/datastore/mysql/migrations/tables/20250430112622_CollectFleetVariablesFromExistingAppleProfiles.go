package tables

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250430112622, Down_20250430112622)
}

func Up_20250430112622(tx *sql.Tx) error {
	// For now, we only collect used fleet variables from existing Apple
	// configuration profiles, as those are the only ones that rely on the
	// variables lookup table for the time being.
	var varDefs []varDef
	const varsStmt = `SELECT id, name, is_prefix FROM fleet_variables`
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	if err := txx.Select(&varDefs, varsStmt); err != nil {
		return fmt.Errorf("failed to load fleet variables: %w", err)
	}

	// Configuration profiles can be up to 16MB in size (MEDIUMBLOB column type),
	// but are typically only a few KB. To be safe, load them in small-ish
	// batches of 100 which will generally be < 1MB (worse case is ~1.6GB). We
	// cannot use a cursor approach (sql.Rows) as we can't do anything else with
	// the connection until the cursor is closed and we need to insert the
	// mappings.
	const batchSize = 100
	const profilesStmt = `SELECT profile_uuid, mobileconfig FROM mdm_apple_configuration_profiles WHERE profile_uuid > ? ORDER BY profile_uuid LIMIT ?`
	var lastProfileUUID string
	var existingProfiles []struct {
		ProfileUUID  string `db:"profile_uuid"`
		MobileConfig string `db:"mobileconfig"`
	}
	for {
		if err := txx.Select(&existingProfiles, profilesStmt, lastProfileUUID, batchSize); err != nil {
			return fmt.Errorf("failed to load existing profiles: %w", err)
		}

		for _, profile := range existingProfiles {
			vars := findFleetVariables(profile.MobileConfig)
			if len(vars) > 0 {
				if err := saveConfigProfileVars(tx, profile.ProfileUUID, vars, varDefs); err != nil {
					return fmt.Errorf("failed to save fleet variables for profile %s: %w", profile.ProfileUUID, err)
				}
			}
			lastProfileUUID = profile.ProfileUUID
		}

		if len(existingProfiles) < batchSize {
			break
		}
	}

	return nil
}

type varDef struct {
	ID       uint   `db:"id"`
	Name     string `db:"name"`
	IsPrefix bool   `db:"is_prefix"`
}

func saveConfigProfileVars(tx *sql.Tx, profileUUID string, vars map[string]any, varDefs []varDef) error {
	// map the profiles' variables to the fleet variable IDs
	var varIDs []uint
	for v := range vars {
		for _, def := range varDefs {
			if !def.IsPrefix && def.Name == v {
				varIDs = append(varIDs, def.ID)
				break
			}
			if def.IsPrefix && strings.HasPrefix(v, def.Name) {
				varIDs = append(varIDs, def.ID)
				break
			}
		}
	}

	if len(varIDs) == 0 {
		return nil
	}

	stmt := strings.TrimSuffix(fmt.Sprintf(`INSERT INTO mdm_configuration_profile_variables (
		apple_profile_uuid, fleet_variable_id
	) VALUES %s`, strings.Repeat("(?, ?),", len(varIDs))), ",")

	args := make([]any, 0, len(varIDs)*2)
	for _, id := range varIDs {
		args = append(args, profileUUID, id)
	}

	if _, err := tx.Exec(stmt, args...); err != nil {
		return fmt.Errorf("failed to insert into mdm_configuration_profile_variables: %w", err)
	}
	return nil
}

// this is a copy of the code in the server/service package so that any future
// change to it does not impact the behavior of this DB migration.
func findFleetVariables(contents string) map[string]interface{} {
	resultSlice := findFleetVariablesKeepDuplicates(contents)
	if len(resultSlice) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(resultSlice))
	for _, v := range resultSlice {
		// re-add the "FLEET_VAR_" prefix as it is needed to map with the entries in
		// fleet_variables table.
		result["FLEET_VAR_"+v] = struct{}{}
	}
	return result
}

var profileVariableRegex = regexp.MustCompile(`(\$FLEET_VAR_(?P<name1>\w+))|(\${FLEET_VAR_(?P<name2>\w+)})`)

func findFleetVariablesKeepDuplicates(contents string) []string {
	var result []string
	matches := profileVariableRegex.FindAllStringSubmatch(contents, -1)
	if len(matches) == 0 {
		return nil
	}
	nameToIndex := make(map[string]int, 2)
	for i, name := range profileVariableRegex.SubexpNames() {
		if name == "" {
			continue
		}
		nameToIndex[name] = i
	}
	for _, match := range matches {
		for _, i := range nameToIndex {
			if match[i] != "" {
				result = append(result, match[i])
			}
		}
	}
	return result
}

func Down_20250430112622(tx *sql.Tx) error {
	return nil
}
