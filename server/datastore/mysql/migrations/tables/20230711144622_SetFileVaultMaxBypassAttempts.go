package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
	"howett.net/plist"
)

func init() {
	MigrationClient.AddMigration(Up_20230711144622, Down_20230711144622)
}

// setBypassAttemptsToFileVaultProfile is used to add the
// `DeferForceAtUserLoginMaxBypassAttempts` to an existing FileVault profile at
// the right place without doing any other modifications to the profile.
//
// We intentionally use a map[string]interface{} to make sure we're fully
// unmarshalling and marshalling the profile without making additional changes.
func setBypassAttemptsToFileVaultProfile(original []byte) ([]byte, error) {
	var configuration map[string]interface{}
	if _, err := plist.Unmarshal(original, &configuration); err != nil {
		return nil, fmt.Errorf("unmarshalling configuration profile: %w", err)
	}

	payloadContent, ok := configuration["PayloadContent"].([]interface{})
	if !ok {
		return nil, errors.New("failed to access PayloadContent element")
	}

	for _, c := range payloadContent {
		payload, ok := c.(map[string]interface{})
		if !ok {
			return nil, errors.New("failed to access Payload element")
		}

		if payload["PayloadType"] == "com.apple.MCX.FileVault2" {
			payload["DeferForceAtUserLoginMaxBypassAttempts"] = 1
		}
	}

	out, err := plist.Marshal(configuration, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new payload: %w", err)
	}

	return out, nil
}

func Up_20230711144622(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	fvProfiles := []struct {
		ID           uint   `db:"profile_id"`
		Mobileconfig []byte `db:"mobileconfig"`
	}{}
	query := `
		SELECT profile_id, mobileconfig FROM mdm_apple_configuration_profiles macp WHERE identifier = 'com.fleetdm.fleet.mdm.filevault'
	`
	if err := txx.Select(&fvProfiles, query); err != nil {
		return fmt.Errorf("getting existing FileVault profiles: %w", err)
	}

	if len(fvProfiles) == 0 {
		return nil
	}

	for _, prof := range fvProfiles {
		newProf, err := setBypassAttemptsToFileVaultProfile(prof.Mobileconfig)
		if err != nil {
			return fmt.Errorf("adding new key to profile with ID %d: %w", prof.ID, err)
		}

		if _, err = txx.Exec(`
			UPDATE mdm_apple_configuration_profiles
			SET mobileconfig = ?, checksum = UNHEX(MD5(mobileconfig))
			WHERE profile_id = ?
		`, newProf, prof.ID); err != nil {
			return fmt.Errorf("updating FileVault profile with ID %d: %w", prof.ID, err)
		}
	}

	return nil
}

func Down_20230711144622(tx *sql.Tx) error {
	return nil
}
