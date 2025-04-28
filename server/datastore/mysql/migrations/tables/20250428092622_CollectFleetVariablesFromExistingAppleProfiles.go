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
	MigrationClient.AddMigration(Up_20250428092622, Down_20250428092622)
}

func Up_20250428092622(tx *sql.Tx) error {
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
	// but are typically only a few KB. To be safe, we use a cursor to load them
	// efficiently (the exact behavior of sql.Rows depends on the driver, either
	// one row at a time or filling a small memory buffer, either way it is
	// memory-efficient).
	const profilesStmt = `SELECT profile_uuid, mobileconfig FROM mdm_apple_configuration_profiles`
	rows, err := tx.Query(profilesStmt)
	if err != nil {
		return fmt.Errorf("failed to query mdm_apple_configuration_profiles: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var profileUUID, mobileConfig string
		if err := rows.Scan(&profileUUID, &mobileConfig); err != nil {
			return fmt.Errorf("failed to scan mdm_apple_configuration_profiles row: %w", err)
		}
		vars := findFleetVariables(mobileConfig)
		if len(vars) > 0 {
			if err := saveConfigProfileVars(tx, profileUUID, vars, varDefs); err != nil {
				return fmt.Errorf("failed to save fleet variables for profile %s: %w", profileUUID, err)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate over mdm_apple_configuration_profiles: %w", err)
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
		result[v] = struct{}{}
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

func Down_20250428092622(tx *sql.Tx) error {
	return nil
}
