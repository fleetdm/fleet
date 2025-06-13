package mysql

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetmdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// addHostMDMCommandsBatchSize is the number of host MDM commands to add in a single batch. This is a var so that it can be modified in tests.
var addHostMDMCommandsBatchSize = 10000

func (ds *Datastore) NewMDMAppleConfigProfile(ctx context.Context, cp fleet.MDMAppleConfigProfile, usesFleetVars []string) (*fleet.MDMAppleConfigProfile, error) {
	profUUID := "a" + uuid.New().String()
	stmt := `
INSERT INTO
    mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, scope, mobileconfig, checksum, uploaded_at, secrets_updated_at)
(SELECT ?, ?, ?, ?, ?, ?, UNHEX(MD5(?)), CURRENT_TIMESTAMP(), ? FROM DUAL WHERE
	NOT EXISTS (
		SELECT 1 FROM mdm_windows_configuration_profiles WHERE name = ? AND team_id = ?
	) AND NOT EXISTS (
		SELECT 1 FROM mdm_apple_declarations WHERE name = ? AND team_id = ?
	)
)`

	var teamID uint
	if cp.TeamID != nil {
		teamID = *cp.TeamID
	}

	var profileID int64
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, stmt,
			profUUID, teamID, cp.Identifier, cp.Name, cp.Scope, cp.Mobileconfig, cp.Mobileconfig, cp.SecretsUpdatedAt, cp.Name, teamID, cp.Name,
			teamID)
		if err != nil {
			switch {
			case IsDuplicate(err):
				return ctxerr.Wrap(ctx, formatErrorDuplicateConfigProfile(err, &cp))
			default:
				return ctxerr.Wrap(ctx, err, "creating new apple mdm config profile")
			}
		}

		aff, _ := res.RowsAffected()
		if aff == 0 {
			return &existsError{
				ResourceType: "MDMAppleConfigProfile.PayloadDisplayName",
				Identifier:   cp.Name,
				TeamID:       cp.TeamID,
			}
		}

		// record the ID as we want to return a fleet.Profile instance with it
		// filled in.
		profileID, _ = res.LastInsertId()

		labels := make([]fleet.ConfigurationProfileLabel, 0, len(cp.LabelsIncludeAll)+len(cp.LabelsIncludeAny)+len(cp.LabelsExcludeAny))
		for i := range cp.LabelsIncludeAll {
			cp.LabelsIncludeAll[i].ProfileUUID = profUUID
			cp.LabelsIncludeAll[i].Exclude = false
			cp.LabelsIncludeAll[i].RequireAll = true
			labels = append(labels, cp.LabelsIncludeAll[i])
		}
		for i := range cp.LabelsIncludeAny {
			cp.LabelsIncludeAny[i].ProfileUUID = profUUID
			cp.LabelsIncludeAny[i].Exclude = false
			cp.LabelsIncludeAny[i].RequireAll = false
			labels = append(labels, cp.LabelsIncludeAny[i])
		}
		for i := range cp.LabelsExcludeAny {
			cp.LabelsExcludeAny[i].ProfileUUID = profUUID
			cp.LabelsExcludeAny[i].Exclude = true
			cp.LabelsExcludeAny[i].RequireAll = false
			labels = append(labels, cp.LabelsExcludeAny[i])
		}
		var profWithoutLabels []string
		if len(labels) == 0 {
			profWithoutLabels = append(profWithoutLabels, profUUID)
		}
		if _, err := batchSetProfileLabelAssociationsDB(ctx, tx, labels, profWithoutLabels, "darwin"); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting darwin profile label associations")
		}
		if err := batchSetProfileVariableAssociationsDB(ctx, tx, []fleet.MDMProfileUUIDFleetVariables{
			{ProfileUUID: profUUID, FleetVariables: usesFleetVars},
		}, "darwin"); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting darwin profile variable associations")
		}

		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting profile and label associations")
	}

	return &fleet.MDMAppleConfigProfile{
		ProfileUUID:  profUUID,
		ProfileID:    uint(profileID), //nolint:gosec // dismiss G115
		Identifier:   cp.Identifier,
		Name:         cp.Name,
		Scope:        cp.Scope,
		Mobileconfig: cp.Mobileconfig,
		TeamID:       cp.TeamID,
	}, nil
}

func formatErrorDuplicateConfigProfile(err error, cp *fleet.MDMAppleConfigProfile) error {
	switch {
	case strings.Contains(err.Error(), "idx_mdm_apple_config_prof_team_identifier"):
		return &existsError{
			ResourceType: "MDMAppleConfigProfile.PayloadIdentifier",
			Identifier:   cp.Identifier,
			TeamID:       cp.TeamID,
		}
	case strings.Contains(err.Error(), "idx_mdm_apple_config_prof_team_name"):
		return &existsError{
			ResourceType: "MDMAppleConfigProfile.PayloadDisplayName",
			Identifier:   cp.Name,
			TeamID:       cp.TeamID,
		}
	default:
		return err
	}
}

func formatErrorDuplicateDeclaration(err error, decl *fleet.MDMAppleDeclaration) error {
	switch {
	case strings.Contains(err.Error(), "idx_mdm_apple_declaration_team_identifier"):
		return &existsError{
			ResourceType: "MDMAppleDeclaration.Identifier",
			Identifier:   decl.Identifier,
			TeamID:       decl.TeamID,
		}
	case strings.Contains(err.Error(), "idx_mdm_apple_declaration_team_name"):
		return &existsError{
			ResourceType: "MDMAppleDeclaration.Name",
			Identifier:   decl.Name,
			TeamID:       decl.TeamID,
		}
	default:
		return err
	}
}

func (ds *Datastore) ListMDMAppleConfigProfiles(ctx context.Context, teamID *uint) ([]*fleet.MDMAppleConfigProfile, error) {
	stmt := `
SELECT
	profile_uuid,
	profile_id,
	team_id,
	name,
	scope,
	identifier,
	mobileconfig,
	created_at,
	uploaded_at,
	checksum
FROM
	mdm_apple_configuration_profiles
WHERE
	team_id=? AND identifier NOT IN (?)
ORDER BY name`

	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	fleetIdentifiers := []string{}
	for idf := range mobileconfig.FleetPayloadIdentifiers() {
		fleetIdentifiers = append(fleetIdentifiers, idf)
	}
	stmt, args, err := sqlx.In(stmt, teamID, fleetIdentifiers)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In ListMDMAppleConfigProfiles")
	}

	var res []*fleet.MDMAppleConfigProfile
	if err = sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, args...); err != nil {
		return nil, err
	}
	return res, nil
}

func (ds *Datastore) GetMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) (*fleet.MDMAppleConfigProfile, error) {
	return ds.getMDMAppleConfigProfileByIDOrUUID(ctx, profileID, "")
}

func (ds *Datastore) GetMDMAppleConfigProfile(ctx context.Context, profileUUID string) (*fleet.MDMAppleConfigProfile, error) {
	return ds.getMDMAppleConfigProfileByIDOrUUID(ctx, 0, profileUUID)
}

func (ds *Datastore) getMDMAppleConfigProfileByIDOrUUID(ctx context.Context, id uint, uuid string) (*fleet.MDMAppleConfigProfile, error) {
	stmt := `
SELECT
	profile_uuid,
	profile_id,
	team_id,
	name,
	scope,
	identifier,
	mobileconfig,
	checksum,
	created_at,
	uploaded_at,
	secrets_updated_at
FROM
	mdm_apple_configuration_profiles
WHERE
`
	var arg any
	if uuid != "" {
		arg = uuid
		stmt += `profile_uuid = ?`
	} else {
		arg = id
		stmt += `profile_id = ?`
	}

	var res fleet.MDMAppleConfigProfile
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, arg)
	if err != nil {
		if err == sql.ErrNoRows {
			if uuid != "" {
				return nil, ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithName(uuid))
			}
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get mdm apple config profile")
	}

	// get the labels for that profile, except if the profile was loaded by the
	// old (deprecated) endpoint.
	if uuid != "" {
		labels, err := ds.listProfileLabelsForProfiles(ctx, nil, []string{res.ProfileUUID}, nil)
		if err != nil {
			return nil, err
		}
		for _, lbl := range labels {
			switch {
			case lbl.Exclude && lbl.RequireAll:
				// this should never happen so log it for debugging
				level.Debug(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all",
					"profile_uuid", lbl.ProfileUUID,
					"label_name", lbl.LabelName,
				)
			case lbl.Exclude && !lbl.RequireAll:
				res.LabelsExcludeAny = append(res.LabelsExcludeAny, lbl)
			case !lbl.Exclude && !lbl.RequireAll:
				res.LabelsIncludeAny = append(res.LabelsIncludeAny, lbl)
			default:
				// default include all
				res.LabelsIncludeAll = append(res.LabelsIncludeAll, lbl)
			}
		}
	}

	return &res, nil
}

func (ds *Datastore) GetMDMAppleDeclaration(ctx context.Context, declUUID string) (*fleet.MDMAppleDeclaration, error) {
	stmt := `
SELECT
	declaration_uuid,
	team_id,
	name,
	identifier,
	raw_json,
	token,
	created_at,
	uploaded_at,
	secrets_updated_at
FROM
	mdm_apple_declarations
WHERE
	declaration_uuid = ?`

	var res fleet.MDMAppleDeclaration
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, declUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleDeclaration").WithName(declUUID))
		}

		return nil, ctxerr.Wrap(ctx, err, "get mdm apple declaration")
	}

	labels, err := ds.listProfileLabelsForProfiles(ctx, nil, nil, []string{res.DeclarationUUID})
	if err != nil {
		return nil, err
	}
	for _, lbl := range labels {
		switch {
		case lbl.Exclude && lbl.RequireAll:
			// this should never happen so log it for debugging
			level.Debug(ds.logger).Log("msg", "unsupported profile label: cannot be both exclude and require all",
				"profile_uuid", lbl.ProfileUUID,
				"label_name", lbl.LabelName,
			)
		case lbl.Exclude && !lbl.RequireAll:
			res.LabelsExcludeAny = append(res.LabelsExcludeAny, lbl)
		case !lbl.Exclude && !lbl.RequireAll:
			res.LabelsIncludeAny = append(res.LabelsIncludeAny, lbl)
		default:
			// default include all
			res.LabelsIncludeAll = append(res.LabelsIncludeAll, lbl)
		}
	}

	return &res, nil
}

func (ds *Datastore) DeleteMDMAppleConfigProfileByDeprecatedID(ctx context.Context, profileID uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return deleteMDMAppleConfigProfileByIDOrUUID(ctx, tx, profileID, "")
	})
}

func (ds *Datastore) DeleteMDMAppleDeclaration(ctx context.Context, declUUID string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := deleteMDMAppleDeclaration(ctx, tx, declUUID); err != nil {
			return err
		}

		// cancel any pending host installs immediately for this declaration
		if err := cancelAppleHostInstallsForDeletedMDMDeclarations(ctx, tx, []string{declUUID}); err != nil {
			return err
		}

		return nil
	})
}

func (ds *Datastore) DeleteMDMAppleConfigProfile(ctx context.Context, profileUUID string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := deleteMDMAppleConfigProfileByIDOrUUID(ctx, tx, 0, profileUUID); err != nil {
			return err
		}

		// cancel any pending host installs immediately for this profile
		if err := cancelAppleHostInstallsForDeletedMDMProfiles(ctx, tx, []string{profileUUID}); err != nil {
			return err
		}

		return nil
	})
}

func deleteMDMAppleConfigProfileByIDOrUUID(ctx context.Context, tx sqlx.ExtContext, id uint, uuid string) error {
	var arg any
	stmt := `DELETE FROM mdm_apple_configuration_profiles WHERE `
	if uuid != "" {
		arg = uuid
		stmt += `profile_uuid = ?`
	} else {
		arg = id
		stmt += `profile_id = ?`
	}
	res, err := tx.ExecContext(ctx, stmt, arg)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		if uuid != "" {
			return ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithName(uuid))
		}
		return ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithID(id))
	}

	return nil
}

func cancelAppleHostInstallsForDeletedMDMProfiles(ctx context.Context, tx sqlx.ExtContext, profileUUIDs []string) error {
	// For Apple profiles, we can safely delete the rows for hosts where status
	// is NULL and operation type is install, but if the status is not NULL, the
	// install command _may_ have been sent to the host, we need to change the
	// operation to remove and set the status to NULL. As a precaution to avoid
	// sending the Install command if possible at all, we deactivate the command
	// in the nano queue if the status is Pending (that is, not NULL, but no sign
	// that the host received it yet).

	if len(profileUUIDs) == 0 {
		return nil
	}

	const delStmt = `
	DELETE FROM
		host_mdm_apple_profiles
	WHERE
		profile_uuid IN (?) AND
		status IS NULL AND
		operation_type = ?`

	stmt, args, err := sqlx.In(delStmt, profileUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host_mdm_apple_profiles that have not been sent to host")
	}

	const deactivateNanoStmt = `
	UPDATE
		nano_enrollment_queue
		JOIN nano_enrollments ne
		    ON nano_enrollment_queue.id = ne.id
		JOIN host_mdm_apple_profiles hmap
			ON hmap.command_uuid = nano_enrollment_queue.command_uuid AND
				hmap.host_uuid = ne.device_id
	SET
		nano_enrollment_queue.active = 0
	WHERE
		hmap.profile_uuid IN (?) AND
		hmap.status = ? AND
		hmap.operation_type = ?`

	stmt, args, err = sqlx.In(deactivateNanoStmt, profileUUIDs, fleet.MDMDeliveryPending, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deactivating nano_enrollment_queue for commands that were pending send to host")
	}

	// we set the ignore_error flag if install status was "pending" or "failed"
	// because the profile may _not_ have been delivered, so if the remove
	// command fails, we don't want to show it.
	const updStmt = `
	UPDATE
		host_mdm_apple_profiles
	SET
		operation_type = ?,
		ignore_error = IF(status IN (?), 1, 0),
		status = NULL
	WHERE
		profile_uuid IN (?) AND
		status IS NOT NULL AND
		operation_type = ?`

	stmt, args, err = sqlx.In(updStmt, fleet.MDMOperationTypeRemove,
		[]fleet.MDMDeliveryStatus{fleet.MDMDeliveryPending, fleet.MDMDeliveryFailed}, profileUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host_mdm_apple_profiles to pending remove")
	}

	return nil
}

func cancelAppleHostInstallsForDeletedMDMDeclarations(ctx context.Context, tx sqlx.ExtContext, declUUIDs []string) error {
	// For Apple declarations, we can safely delete the rows for hosts where
	// status is NULL and operation type is install, but if the status is not
	// NULL, we need to change the operation to remove and set the status to NULL
	// (i.e. Enforcing removal (pending)) so that on the next reconcile there is
	// a DDM command issued to sync the host with the new declarative state
	// (which will not include the deleted declaration(s) anymore).

	if len(declUUIDs) == 0 {
		return nil
	}

	const delStmt = `
	DELETE FROM
		host_mdm_apple_declarations
	WHERE
		declaration_uuid IN (?) AND
		status IS NULL AND
		operation_type = ?`

	stmt, args, err := sqlx.In(delStmt, declUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host_mdm_apple_declarations that have not been sent to host")
	}

	const updStmt = `
	UPDATE
		host_mdm_apple_declarations
	SET
		status = NULL,
		operation_type = ?
	WHERE
		declaration_uuid IN (?) AND
		status IS NOT NULL AND
		operation_type = ?`

	stmt, args, err = sqlx.In(updStmt, fleet.MDMOperationTypeRemove, declUUIDs, fleet.MDMOperationTypeInstall)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building in statement")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host_mdm_apple_declarations that may have been sent to host")
	}

	return nil
}

func (ds *Datastore) DeleteMDMAppleDeclarationByName(ctx context.Context, teamID *uint, name string) error {
	const stmt = `DELETE FROM mdm_apple_declarations WHERE team_id = ? AND name = ?`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, globalOrTmID, name)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
}

func deleteMDMAppleDeclaration(ctx context.Context, tx sqlx.ExtContext, uuid string) error {
	stmt := `DELETE FROM mdm_apple_declarations WHERE declaration_uuid = ?`

	res, err := tx.ExecContext(ctx, stmt, uuid)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("MDMAppleDeclaration").WithName(uuid))
	}

	return nil
}

func (ds *Datastore) DeleteMDMAppleConfigProfileByTeamAndIdentifier(ctx context.Context, teamID *uint, profileIdentifier string) error {
	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM mdm_apple_configuration_profiles WHERE team_id = ? AND identifier = ?`, teamID, profileIdentifier)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if deleted, _ := res.RowsAffected(); deleted == 0 {
		message := fmt.Sprintf("identifier: %s, team_id: %d", profileIdentifier, teamID)
		return ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithMessage(message))
	}

	return nil
}

func (ds *Datastore) GetHostMDMAppleProfiles(ctx context.Context, hostUUID string) ([]fleet.HostMDMAppleProfile, error) {
	stmt := fmt.Sprintf(`
SELECT
profile_uuid,
profile_name AS name,
profile_identifier AS identifier,
-- internally, a NULL status implies that the cron needs to pick up
-- this profile, for the user that difference doesn't exist, the
-- profile is effectively pending. This is consistent with all our
-- aggregation functions.
COALESCE(status, '%s') AS status,
COALESCE(operation_type, '') AS operation_type,
COALESCE(detail, '') AS detail,
scope
FROM
  host_mdm_apple_profiles
WHERE
  host_uuid = ? AND NOT (operation_type = '%s' AND COALESCE(status, '%s') IN('%s', '%s'))

UNION ALL
SELECT
declaration_uuid AS profile_uuid,
declaration_name AS name,
declaration_identifier AS identifier,
-- internally, a NULL status implies that the cron needs to pick up
-- this profile, for the user that difference doesn't exist, the
-- profile is effectively pending. This is consistent with all our
-- aggregation functions.
COALESCE(status, '%s') AS status,
COALESCE(operation_type, '') AS operation_type,
COALESCE(detail, '') AS detail,
scope
FROM
  host_mdm_apple_declarations
WHERE
  host_uuid = ? AND declaration_name NOT IN (?) AND NOT (operation_type = '%s' AND COALESCE(status, '%s') IN('%s', '%s'))`,
		fleet.MDMDeliveryPending,
		fleet.MDMOperationTypeRemove,
		fleet.MDMDeliveryPending,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
		fleet.MDMDeliveryPending,
		fleet.MDMOperationTypeRemove,
		fleet.MDMDeliveryPending,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
	)

	stmt, args, err := sqlx.In(stmt, hostUUID, hostUUID, fleetmdm.ListFleetReservedMacOSDeclarationNames())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building in statement")
	}

	var profiles []fleet.HostMDMAppleProfile
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, stmt, args...); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (ds *Datastore) GetHostMDMCertificateProfile(ctx context.Context, hostUUID string,
	profileUUID string, caName string,
) (*fleet.HostMDMCertificateProfile, error) {
	stmt := `
	SELECT
		hmap.host_uuid,
		hmap.profile_uuid,
		hmap.status,
		hmmc.challenge_retrieved_at,
		hmmc.not_valid_before,
		hmmc.not_valid_after,
		hmmc.type,
		hmmc.ca_name,
		hmmc.serial
	FROM
		host_mdm_apple_profiles hmap
	JOIN host_mdm_managed_certificates hmmc
		ON hmap.host_uuid = hmmc.host_uuid AND hmap.profile_uuid = hmmc.profile_uuid
	WHERE
		hmmc.host_uuid = ? AND hmmc.profile_uuid = ? AND hmmc.ca_name = ?`
	var profile fleet.HostMDMCertificateProfile
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &profile, stmt, hostUUID, profileUUID, caName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (ds *Datastore) CleanUpMDMManagedCertificates(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
	DELETE hmmc FROM host_mdm_managed_certificates hmmc
		LEFT JOIN host_mdm_apple_profiles hmap ON hmmc.host_uuid = hmap.host_uuid AND hmmc.profile_uuid = hmap.profile_uuid
		WHERE hmap.host_uuid IS NULL`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clean up mdm certificate profiles")
	}
	return nil
}

// RenewMDMManagedCertificates marks managed certificate profiles for resend when renewal is required
func (ds *Datastore) RenewMDMManagedCertificates(ctx context.Context) error {
	// This will trigger a resend next time profiles are checked
	updateQuery := `UPDATE host_mdm_apple_profiles SET status = NULL WHERE status IS NOT NULL AND operation_type = ? AND (`
	hostProfileClause := ``
	values := []interface{}{fleet.MDMOperationTypeInstall}

	totalHostCertsToRenew := 0
	hostCertTypesToRenew := []fleet.CAConfigAssetType{fleet.CAConfigDigiCert, fleet.CAConfigCustomSCEPProxy, fleet.CAConfigNDES}
	for _, hostCertType := range hostCertTypesToRenew {
		hostCertsToRenew := []struct {
			HostUUID       string    `db:"host_uuid"`
			ProfileUUID    string    `db:"profile_uuid"`
			NotValidAfter  time.Time `db:"not_valid_after"`
			ValidityPeriod int       `db:"validity_period"`
		}{}
		// Fetch all MDM Managed certificates of the given type that aren't already queued for
		// resend(hmap.status=null) and which
		// * Have a validity period > 30 days and are expiring in the next 30 days
		// * Have a validity period <= 30 days and are within half the validity period of expiration
		// nb: we SELECT not_valid_after and validity_period here so we can use them in the HAVING clause, but
		// we don't actually need them for the update logic.
		err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostCertsToRenew, `
	SELECT
		hmmc.host_uuid,
		hmmc.profile_uuid,
		hmmc.not_valid_after,
		DATEDIFF(hmmc.not_valid_after, hmmc.not_valid_before) AS validity_period
	FROM
		host_mdm_managed_certificates hmmc
	INNER JOIN
		host_mdm_apple_profiles hmap
		ON hmmc.host_uuid = hmap.host_uuid AND hmmc.profile_uuid = hmap.profile_uuid
	WHERE
		hmmc.type = ? AND hmap.status IS NOT NULL AND hmap.operation_type = ?
	HAVING
		validity_period IS NOT NULL AND
		((validity_period > 30 AND not_valid_after < DATE_ADD(NOW(), INTERVAL 30 DAY)) OR
		(validity_period <= 30 AND not_valid_after < DATE_ADD(NOW(), INTERVAL validity_period/2 DAY)))
	LIMIT 1000`, hostCertType, fleet.MDMOperationTypeInstall)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving mdm managed certificates to renew")
		}
		if len(hostCertsToRenew) == 0 {
			level.Debug(ds.logger).Log("msg", "No "+hostCertType+" certificates to renew")
			continue
		}
		totalHostCertsToRenew += len(hostCertsToRenew)

		for _, hostCertToRenew := range hostCertsToRenew {
			hostProfileClause += `(host_uuid = ? AND profile_uuid = ?) OR `
			values = append(values, hostCertToRenew.HostUUID, hostCertToRenew.ProfileUUID)
		}
	}
	if totalHostCertsToRenew == 0 {
		return nil
	}

	hostProfileClause = strings.TrimSuffix(hostProfileClause, " OR ")

	level.Debug(ds.logger).Log("msg", "Renewing MDM managed digicert/SCEP certificates", "len(hostCertsToRenew)", totalHostCertsToRenew)
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, updateQuery+hostProfileClause+")", values...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating mdm managed certificates to renew")
		}
		return nil
	})
	return err
}

// ResendHostCustomSCEPProfile marks a custom SCEP profile to be resent to the host with the given UUID. It
// also deactivates prior nano commands for the profile UUID and host UUID.
//
// FIXME: We really should have a more generic function to handle this, but our existing methods
// for "resending" profiles don't reevaluate the profile variables so they aren't useful for
// custom SCEP profiles where we need to regenerate the SCEP challenge. The main difference between
// the existing flow and the implementation below is that we need to blank the command uuid in order
// get the reconcile cron to reevaluate the command template to generate the challenge. Otherwise,
// it just sends the old bytes again. It feels like we have some leaky abstrations somewhere that we need
// to clean up.
func (ds *Datastore) ResendHostCustomSCEPProfile(ctx context.Context, hostUUID string, profUUID string) error {
	deactivateNanoStmt := `
UPDATE
	nano_enrollment_queue
	JOIN nano_enrollments ne
		ON nano_enrollment_queue.id = ne.id
	JOIN host_mdm_apple_profiles hmap 
		ON hmap.command_uuid = nano_enrollment_queue.command_uuid AND 
			hmap.host_uuid = ne.device_id
SET 
	nano_enrollment_queue.active = 0
WHERE
	hmap.profile_uuid = ? AND
	hmap.host_uuid = ?`

	updateStmt := `
UPDATE
	host_mdm_apple_profiles
SET
	status = NULL,
	command_uuid = '',
	detail = '',
	retries = 0,
	variables_updated_at = NOW(6)
WHERE
	profile_uuid = ? AND
	host_uuid = ? AND
	operation_type = ?`

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, deactivateNanoStmt, profUUID, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deactivating nano_enrollment_queue for commands that were pending send to host")
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			// this should never happen, log for debugging
			level.Error(ds.logger).Log("msg", "resend custom scep profile: nano not deactivated", "host_uuid", hostUUID, "profile_uuid", profUUID)
		}

		res, err = tx.ExecContext(ctx, updateStmt, profUUID, hostUUID, fleet.MDMOperationTypeInstall)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resending host MDM profile")
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			// this should never happen, log for debugging
			level.Error(ds.logger).Log("msg", "resend custom scep profile: host mdm apple profiles not updated", "host_uuid", hostUUID, "profile_uuid", profUUID)
		}

		return nil
	})
}

func (ds *Datastore) NewMDMAppleEnrollmentProfile(
	ctx context.Context,
	payload fleet.MDMAppleEnrollmentProfilePayload,
) (*fleet.MDMAppleEnrollmentProfile, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`
INSERT INTO
    mdm_apple_enrollment_profiles (token, type, dep_profile)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    token = VALUES(token),
    type = VALUES(type),
    dep_profile = VALUES(dep_profile)
`,
		payload.Token, payload.Type, payload.DEPProfile,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleEnrollmentProfile{
		ID:         uint(id), //nolint:gosec // dismiss G115
		Token:      payload.Token,
		Type:       payload.Type,
		DEPProfile: payload.DEPProfile,
	}, nil
}

func (ds *Datastore) ListMDMAppleEnrollmentProfiles(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
	var enrollmentProfiles []*fleet.MDMAppleEnrollmentProfile
	if err := sqlx.SelectContext(
		ctx,
		ds.writer(ctx),
		&enrollmentProfiles,
		`
SELECT
    id,
    token,
    type,
    dep_profile,
    created_at,
    updated_at
FROM
    mdm_apple_enrollment_profiles
ORDER BY created_at DESC
`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list enrollment profiles")
	}
	return enrollmentProfiles, nil
}

func (ds *Datastore) GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string) (*fleet.MDMAppleEnrollmentProfile, error) {
	var enrollment fleet.MDMAppleEnrollmentProfile
	if err := sqlx.GetContext(ctx, ds.reader(ctx),
		&enrollment,
		`
SELECT
    id,
    token,
    type,
    dep_profile,
    created_at,
    updated_at
FROM
    mdm_apple_enrollment_profiles
WHERE
    token = ?
`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleEnrollmentProfile"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment profile by token")
	}
	return &enrollment, nil
}

func (ds *Datastore) GetMDMAppleEnrollmentProfileByType(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
	var enrollment fleet.MDMAppleEnrollmentProfile
	if err := sqlx.GetContext(ctx, ds.writer(ctx), // use writer as it is used just after creation in some cases
		&enrollment,
		`
SELECT
    id,
    token,
    type,
    dep_profile,
    created_at,
    updated_at
FROM
    mdm_apple_enrollment_profiles
WHERE
    type = ?
`,
		typ,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleEnrollmentProfile"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment profile by type")
	}
	return &enrollment, nil
}

func (ds *Datastore) GetMDMAppleCommandRequestType(ctx context.Context, commandUUID string) (string, error) {
	var rt string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &rt, `SELECT request_type FROM nano_commands WHERE command_uuid = ?`, commandUUID)
	if err == sql.ErrNoRows {
		return "", ctxerr.Wrap(ctx, notFound("MDMAppleCommand").WithName(commandUUID))
	}
	return rt, err
}

func (ds *Datastore) GetVPPCommandResults(ctx context.Context, commandUUID string, hostUUID string) ([]*fleet.MDMCommandResult, error) {
	var results []*fleet.MDMCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.reader(ctx),
		&results,
		`
SELECT
    ncr.id as host_uuid,
    ncr.command_uuid,
    ncr.status,
    ncr.result,
    ncr.updated_at,
    nc.request_type,
    nc.command as payload
FROM
    nano_command_results ncr
INNER JOIN
    nano_commands nc
ON
    ncr.command_uuid = nc.command_uuid AND ncr.id = ? AND ncr.command_uuid = ?
LEFT JOIN host_vpp_software_installs hvsi ON ncr.command_uuid = hvsi.command_uuid
LEFT JOIN upcoming_activities ua ON ncr.command_uuid = ua.execution_id AND ua.activity_type = 'software_install'
WHERE ua.id IS NOT NULL OR hvsi.id IS NOT NULL
`,
		hostUUID,
		commandUUID,
	)

	if err == sql.ErrNoRows || len(results) == 0 {
		var validCommandExists bool
		err = sqlx.GetContext(
			ctx,
			ds.reader(ctx),
			&validCommandExists,
			`SELECT COUNT(*) > 0 command_exists
					FROM hosts h
					LEFT JOIN host_vpp_software_installs hvsi ON h.id = hvsi.host_id AND hvsi.command_uuid = ?
					LEFT JOIN upcoming_activities ua ON h.id = ua.host_id AND ua.execution_id = ?
					WHERE h.uuid = ? AND (hvsi.id IS NOT NULL OR ua.id IS NOT NULL)`,
			commandUUID,
			commandUUID,
			hostUUID,
		)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get vpp command existence")
		}
		if validCommandExists {
			return nil, nil // no results, but command did reference a VPP install
		}
		return nil, notFound("HostMDMCommand").WithName(commandUUID)
	}

	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get vpp command results")
	}
	return results, nil
}

func (ds *Datastore) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	query := `
SELECT
    ncr.id as host_uuid,
    ncr.command_uuid,
    ncr.status,
    ncr.result,
    ncr.updated_at,
    nc.request_type,
    nc.command as payload
FROM
    nano_command_results ncr
INNER JOIN
    nano_commands nc
ON
    ncr.command_uuid = nc.command_uuid
WHERE
    ncr.command_uuid = ?
`

	var results []*fleet.MDMCommandResult
	err := sqlx.SelectContext(
		ctx,
		ds.reader(ctx),
		&results,
		query,
		commandUUID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get command results")
	}
	return results, nil
}

func (ds *Datastore) ListMDMAppleCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMAppleCommand, error) {
	// Note that right now we are explicitly filtering by ne.type = 'Device' but we may want to change
	// the API to allow and show user commands in the future
	stmt := fmt.Sprintf(`
SELECT
    ne.device_id as device_id,
    nvq.command_uuid,
    COALESCE(NULLIF(nvq.status, ''), 'Pending') as status,
    COALESCE(nvq.result_updated_at, nvq.created_at) as updated_at,
    nvq.request_type,
    h.hostname,
    h.team_id
FROM
    nano_view_queue nvq
INNER JOIN
	nano_enrollments ne
ON
	nvq.id = ne.id AND type = 'Device'
INNER JOIN
    hosts h
ON
    ne.id = h.uuid
WHERE
   nvq.active = 1 AND
    %s
`, ds.whereFilterHostsByTeams(tmFilter, "h"))
	stmt, params := appendListOptionsWithCursorToSQL(stmt, nil, &listOpts.ListOptions)

	var results []*fleet.MDMAppleCommand
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}

func (ds *Datastore) NewMDMAppleInstaller(ctx context.Context, name string, size int64, manifest string, installer []byte, urlToken string) (*fleet.MDMAppleInstaller, error) {
	res, err := ds.writer(ctx).ExecContext(
		ctx,
		`INSERT INTO mdm_apple_installers (name, size, manifest, installer, url_token) VALUES (?, ?, ?, ?, ?)`,
		name, size, manifest, installer, urlToken,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	id, _ := res.LastInsertId()
	return &fleet.MDMAppleInstaller{
		ID:        uint(id), //nolint:gosec // dismiss G115
		Size:      size,
		Name:      name,
		Manifest:  manifest,
		Installer: installer,
		URLToken:  urlToken,
	}, nil
}

func (ds *Datastore) MDMAppleInstaller(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer(ctx),
		&installer,
		`SELECT id, name, size, manifest, installer, url_token FROM mdm_apple_installers WHERE url_token = ?`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithName(token))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer by token")
	}
	return &installer, nil
}

func (ds *Datastore) MDMAppleInstallerDetailsByID(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer(ctx),
		&installer,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers WHERE id = ?`,
		id,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer details by id")
	}
	return &installer, nil
}

func (ds *Datastore) DeleteMDMAppleInstaller(ctx context.Context, id uint) error {
	if _, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM mdm_apple_installers WHERE id = ?`, id); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
}

func (ds *Datastore) MDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	var installer fleet.MDMAppleInstaller
	if err := sqlx.GetContext(
		ctx,
		ds.writer(ctx),
		&installer,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers WHERE url_token = ?`,
		token,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("AppleInstaller").WithName(token))
		}
		return nil, ctxerr.Wrap(ctx, err, "get installer details by id")
	}
	return &installer, nil
}

func (ds *Datastore) ListMDMAppleInstallers(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
	var installers []fleet.MDMAppleInstaller
	if err := sqlx.SelectContext(ctx, ds.writer(ctx),
		&installers,
		`SELECT id, name, size, manifest, url_token FROM mdm_apple_installers`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list installers")
	}
	return installers, nil
}

func (ds *Datastore) MDMAppleListDevices(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
	var devices []fleet.MDMAppleDevice
	if err := sqlx.SelectContext(
		ctx,
		ds.writer(ctx),
		&devices,
		`
SELECT
    d.id,
    d.serial_number,
    e.enabled
FROM
    nano_devices d
    JOIN nano_enrollments e ON d.id = e.device_id
WHERE
    type = "Device"
`,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list devices")
	}
	return devices, nil
}

func (ds *Datastore) MDMAppleUpsertHost(ctx context.Context, mdmHost *fleet.Host) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "mdm apple upsert host get app config")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ingestMDMAppleDeviceFromCheckinDB(ctx, tx, mdmHost, ds.logger, appCfg)
	})
}

func ingestMDMAppleDeviceFromCheckinDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	mdmHost *fleet.Host,
	logger log.Logger,
	appCfg *fleet.AppConfig,
) error {
	if mdmHost.HardwareSerial == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected device serial number but got empty string")
	}
	if mdmHost.UUID == "" {
		return ctxerr.New(ctx, "ingest mdm apple host from checkin expected unique device id but got empty string")
	}

	// MDM is necessarily enabled if this gets called, always pass true for that
	// parameter.
	enrolledHostInfo, err := matchHostDuringEnrollment(ctx, tx, mdmEnroll, true, "", mdmHost.UUID, mdmHost.HardwareSerial)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return insertMDMAppleHostDB(ctx, tx, mdmHost, logger, appCfg)

	case err != nil:
		return ctxerr.Wrap(ctx, err, "get mdm apple host by serial number or udid")

	default:
		return updateMDMAppleHostDB(ctx, tx, enrolledHostInfo.ID, mdmHost, appCfg)
	}
}

func mdmHostEnrollFields(mdmHost *fleet.Host) (refetchRequested bool, lastEnrolledAt time.Time) {
	supportsOsquery := mdmHost.SupportsOsquery()
	lastEnrolledAt, err := time.Parse("2006-01-02 15:04:05", server.NeverTimestamp)
	if err != nil {
		panic(err)
	}
	if !supportsOsquery {
		// Given the device does not have osquery, we set the last_enrolled_at as the MDM enroll time.
		lastEnrolledAt = time.Now()
	}
	return supportsOsquery, lastEnrolledAt
}

func updateMDMAppleHostDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostID uint,
	mdmHost *fleet.Host,
	appCfg *fleet.AppConfig,
) error {
	refetchRequested, lastEnrolledAt := mdmHostEnrollFields(mdmHost)

	args := []interface{}{
		mdmHost.HardwareSerial,
		mdmHost.UUID,
		mdmHost.HardwareModel,
		mdmHost.Platform,
		refetchRequested,
		// Set osquery_host_id to the device UUID only if it is not already set.
		mdmHost.UUID,
		hostID,
	}

	// Only update last_enrolled_at if this is a iOS/iPadOS device.
	// macOS should not update last_enrolled_at as it is set when osquery enrolls.
	lastEnrolledAtColumn := ""
	if mdmHost.Platform == "ios" || mdmHost.Platform == "ipados" {
		lastEnrolledAtColumn = "last_enrolled_at = ?,"
		args = append([]interface{}{lastEnrolledAt}, args...)
	}

	updateStmt := fmt.Sprintf(`
		UPDATE hosts SET
			%s
			hardware_serial = ?,
			uuid = ?,
			hardware_model = ?,
			platform =  ?,
			refetch_requested = ?,
			osquery_host_id = COALESCE(NULLIF(osquery_host_id, ''), ?)
		WHERE id = ?`, lastEnrolledAtColumn)

	if _, err := tx.ExecContext(ctx, updateStmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "update mdm apple host")
	}

	mdmHost.ID = hostID

	// load the enrolled team id for that host, will be needed later in the
	// HostActionReset lifecycle
	if err := sqlx.GetContext(ctx, tx, &mdmHost.TeamID, `SELECT team_id FROM hosts WHERE id = ?`, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "load host team id")
	}

	// clear any host_mdm_actions following re-enrollment here
	if _, err := tx.ExecContext(ctx, `DELETE FROM host_mdm_actions WHERE host_id = ?`, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "error clearing mdm apple host_mdm_actions")
	}

	if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, appCfg, false, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
	}

	return nil
}

func insertMDMAppleHostDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	mdmHost *fleet.Host,
	logger log.Logger,
	appCfg *fleet.AppConfig,
) error {
	refetchRequested, lastEnrolledAt := mdmHostEnrollFields(mdmHost)
	insertStmt := `
		INSERT INTO hosts (
			hardware_serial,
			uuid,
			hardware_model,
			platform,
			last_enrolled_at,
			detail_updated_at,
			osquery_host_id,
			refetch_requested
		) VALUES (?,?,?,?,?,?,?,?)`

	res, err := tx.ExecContext(
		ctx,
		insertStmt,
		mdmHost.HardwareSerial,
		mdmHost.UUID,
		mdmHost.HardwareModel,
		mdmHost.Platform,
		lastEnrolledAt,
		server.NeverTimestamp,
		mdmHost.UUID,
		refetchRequested,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "insert mdm apple host")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "last insert id mdm apple host")
	}
	if id < 1 {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host unexpected last insert id")
	}

	mdmHost.ID = uint(id)

	if err := upsertHostDisplayNames(ctx, tx, *mdmHost); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert display names")
	}

	if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, logger, *mdmHost); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert label membership")
	}

	if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, appCfg, false, mdmHost.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
	}
	return nil
}

// hostToCreateFromMDM defines a common set of parameters required to create
// host records without a pre-existing osquery enrollment from MDM flows like
// ADE ingestion or OTA enrollments
type hostToCreateFromMDM struct {
	// HardwareSerial should match the value for hosts.hardware_serial
	HardwareSerial string
	// HardwareModel should match the value for hosts.hardware_model
	HardwareModel string
	// PlatformHint is used to determine hosts.platform, if it:
	//
	// - contains "iphone" the platform is "ios"
	// - contains "ipad" the platform is "ipados"
	// - otherwise the platform is "darwin"
	PlatformHint string
}

func createHostFromMDMDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	logger log.Logger,
	devices []hostToCreateFromMDM,
	fromADE bool,
	macOSTeam, iosTeam, ipadTeam *uint,
) (int64, []fleet.Host, error) {
	// NOTE: order of arguments for teams is important, see statement.
	args := []any{iosTeam, ipadTeam, macOSTeam}
	us, unionArgs := unionSelectDevices(devices)
	args = append(args, unionArgs...)

	stmt := fmt.Sprintf(`
		INSERT INTO hosts (
			hardware_serial,
			hardware_model,
			platform,
			last_enrolled_at,
			detail_updated_at,
			osquery_host_id,
			refetch_requested,
			team_id
		) (
			SELECT
				us.hardware_serial,
				COALESCE(GROUP_CONCAT(DISTINCT us.hardware_model), ''),
				us.platform,
				'`+server.NeverTimestamp+`' AS last_enrolled_at,
				'`+server.NeverTimestamp+`' AS detail_updated_at,
				NULL AS osquery_host_id,
				IF(us.platform = 'ios' OR us.platform = 'ipados', 0, 1) AS refetch_requested,
				CASE
					WHEN us.platform = 'ios' THEN ?
					WHEN us.platform = 'ipados' THEN ?
					ELSE ?
				END AS team_id
			FROM (%s) us
			LEFT JOIN hosts h ON us.hardware_serial = h.hardware_serial
		WHERE
			h.id IS NULL
		GROUP BY
			us.hardware_serial, us.platform)`,
		us,
	)

	res, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "inserting new host in MDM ingestion")
	}

	n, _ := res.RowsAffected()
	// get new host ids
	args = []any{}
	parts := []string{}
	for _, d := range devices {
		args = append(args, d.HardwareSerial)
		parts = append(parts, "?")
	}

	var hostsWithEnrolled []struct {
		fleet.Host
		Enrolled *bool `db:"enrolled"`
	}
	err = sqlx.SelectContext(ctx, tx, &hostsWithEnrolled, fmt.Sprintf(`
			SELECT
				h.id,
				h.platform,
				h.hardware_model,
				h.hardware_serial,
				h.hostname,
				COALESCE(hmdm.enrolled, 0) as enrolled
			FROM hosts h
			LEFT JOIN host_mdm hmdm ON hmdm.host_id = h.id
			WHERE h.hardware_serial IN(%s)`,
		strings.Join(parts, ",")),
		args...)
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "ingest mdm apple host get host ids")
	}

	var hosts []fleet.Host
	var unmanagedHostIDs []uint
	for _, h := range hostsWithEnrolled {
		hosts = append(hosts, h.Host)
		if h.Enrolled == nil || !*h.Enrolled {
			unmanagedHostIDs = append(unmanagedHostIDs, h.ID)
		}
	}

	if err := upsertHostDisplayNames(ctx, tx, hosts...); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert display names")
	}

	if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, logger, hosts...); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert label membership")
	}

	appCfg, err := appConfigDB(ctx, tx)
	if err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "ingest mdm apple host get app config")
	}

	// only upsert MDM info for hosts that are unmanaged. This
	// prevents us from overriding valuable info with potentially
	// incorrect data. For example: if a host is enrolled in a
	// third-party MDM, but gets assigned in ABM to Fleet (during
	// migration) we'll get an 'added' event. In that case, we
	// expect that MDM info will be updated in due time as we ingest
	// future osquery data from the host
	if err := upsertMDMAppleHostMDMInfoDB(
		ctx,
		tx,
		appCfg,
		fromADE,
		unmanagedHostIDs...,
	); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
	}

	return n, hosts, nil
}

func (ds *Datastore) IngestMDMAppleDeviceFromOTAEnrollment(
	ctx context.Context,
	teamID *uint,
	deviceInfo fleet.MDMAppleMachineInfo,
) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		toInsert := []hostToCreateFromMDM{
			{
				HardwareSerial: deviceInfo.Serial,
				PlatformHint:   deviceInfo.Product,
				HardwareModel:  deviceInfo.Product,
			},
		}
		_, _, err := createHostFromMDMDB(ctx, tx, ds.logger, toInsert, false, teamID, teamID, teamID)
		return ctxerr.Wrap(ctx, err, "creating host from OTA enrollment")
	})
}

func (ds *Datastore) IngestMDMAppleDevicesFromDEPSync(
	ctx context.Context,
	devices []godep.Device,
	abmTokenID uint,
	macOSTeam, iosTeam, ipadTeam *fleet.Team,
) (createdCount int64, err error) {
	if len(devices) < 1 {
		level.Debug(ds.logger).Log("msg", "ingesting devices from DEP received < 1 device, skipping", "len(devices)", len(devices))
		return 0, nil
	}

	var teamIDs []*uint
	for _, team := range []*fleet.Team{macOSTeam, iosTeam, ipadTeam} {
		if team == nil {
			teamIDs = append(teamIDs, nil)
			continue
		}

		exists, err := ds.TeamExists(ctx, team.ID)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "ingest mdm apple host get team by name")
		}

		if exists {
			teamIDs = append(teamIDs, &team.ID)
			continue
		}

		// If the team doesn't exist, we still ingest the device, but it won't
		// belong to any team.
		level.Debug(ds.logger).Log(
			"msg",
			"ingesting devices from ABM: unable to find default team assigned in config, the devices won't be assigned to a team",
			"team_id",
			team,
		)
		teamIDs = append(teamIDs, nil)
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		htc := make([]hostToCreateFromMDM, len(devices))
		for i, d := range devices {
			htc[i] = hostToCreateFromMDM{
				HardwareSerial: d.SerialNumber,
				HardwareModel:  d.Model,
				PlatformHint:   d.DeviceFamily,
			}
		}

		n, hosts, err := createHostFromMDMDB(
			ctx,
			tx,
			ds.logger,
			htc,
			true,
			teamIDs[0], teamIDs[1], teamIDs[2],
		)
		if err != nil {
			return err
		}
		createdCount = n

		if err := upsertHostDEPAssignmentsDB(ctx, tx, hosts, abmTokenID); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert DEP assignments")
		}

		return nil
	})

	return createdCount, err
}

func (ds *Datastore) UpsertMDMAppleHostDEPAssignments(ctx context.Context, hosts []fleet.Host, abmTokenID uint) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		if err := upsertHostDEPAssignmentsDB(ctx, tx, hosts, abmTokenID); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert host DEP assignments")
		}

		return nil
	})
}

func upsertHostDEPAssignmentsDB(ctx context.Context, tx sqlx.ExtContext, hosts []fleet.Host, abmTokenID uint) error {
	if len(hosts) == 0 {
		return nil
	}

	stmt := `
		INSERT INTO host_dep_assignments (host_id, abm_token_id)
		VALUES %s
		ON DUPLICATE KEY UPDATE
		  added_at = CURRENT_TIMESTAMP,
		  deleted_at = NULL,
		  abm_token_id = VALUES(abm_token_id)`

	args := []interface{}{}
	values := []string{}
	for _, host := range hosts {
		args = append(args, host.ID, abmTokenID)
		values = append(values, "(?, ?)")
	}

	_, err := tx.ExecContext(ctx, fmt.Sprintf(stmt, strings.Join(values, ",")), args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host dep assignments")
	}

	return nil
}

func upsertHostDisplayNames(ctx context.Context, tx sqlx.ExtContext, hosts ...fleet.Host) error {
	var args []interface{}
	var parts []string
	for _, h := range hosts {
		args = append(args, h.ID, h.DisplayName())
		parts = append(parts, "(?, ?)")
	}

	_, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO host_display_names (host_id, display_name) VALUES %s
			ON DUPLICATE KEY UPDATE display_name = VALUES(display_name)`, strings.Join(parts, ",")),
		args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host display names")
	}

	return nil
}

func upsertMDMAppleHostMDMInfoDB(ctx context.Context, tx sqlx.ExtContext, appCfg *fleet.AppConfig, fromSync bool, hostIDs ...uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	serverURL, err := apple_mdm.ResolveAppleMDMURL(appCfg.MDMUrl())
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolve Fleet MDM URL")
	}

	// if the device is coming from the DEP sync, we don't consider it
	// enrolled yet.
	enrolled := !fromSync

	result, err := tx.ExecContext(ctx, `
		INSERT INTO mobile_device_management_solutions (name, server_url) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE server_url = VALUES(server_url)`,
		fleet.WellKnownMDMFleet, serverURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert mdm solution")
	}

	var mdmID int64
	if insertOnDuplicateDidInsertOrUpdate(result) {
		mdmID, _ = result.LastInsertId()
	} else {
		stmt := `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`
		if err := sqlx.GetContext(ctx, tx, &mdmID, stmt, fleet.WellKnownMDMFleet, serverURL); err != nil {
			return ctxerr.Wrap(ctx, err, "query mdm solution id")
		}
	}

	args := []interface{}{}
	parts := []string{}
	for _, id := range hostIDs {
		args = append(args, enrolled, serverURL, fromSync, mdmID, false, id)
		parts = append(parts, "(?, ?, ?, ?, ?, ?)")
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, is_server, host_id) VALUES %s
		ON DUPLICATE KEY UPDATE enrolled = VALUES(enrolled)`, strings.Join(parts, ",")), args...)

	return ctxerr.Wrap(ctx, err, "upsert host mdm info")
}

func upsertMDMAppleHostLabelMembershipDB(ctx context.Context, tx sqlx.ExtContext, logger log.Logger, hosts ...fleet.Host) error {
	// Builtin label memberships are usually inserted when the first distributed
	// query results are received; however, we want to insert pending MDM hosts
	// now because it may still be some time before osquery is running on these
	// devices. Because these are Apple devices, we're adding them to the "All
	// Hosts" and one of "macOS", "iOS", "iPadOS" labels.
	labels := []struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}{}
	err := sqlx.SelectContext(ctx, tx, &labels, `SELECT id, name FROM labels WHERE label_type = 1 AND (name = 'All Hosts' OR name = 'macOS' OR name = 'iOS' OR name = 'iPadOS')`)
	switch {
	case err != nil:
		return ctxerr.Wrap(ctx, err, "get builtin labels")
	case len(labels) != 4:
		// Builtin labels can get deleted so it is important that we check that
		// they still exist before we continue.
		level.Error(logger).Log("err", fmt.Sprintf("expected 4 builtin labels but got %d", len(labels)))
		return nil
	default:
		// continue
	}

	// We cannot assume IDs on labels, thus we look by name.
	var (
		allHostsLabelID uint
		macOSLabelID    uint
		iOSLabelID      uint
		iPadOSLabelID   uint
	)
	for _, label := range labels {
		switch label.Name {
		case "All Hosts":
			allHostsLabelID = label.ID
		case "macOS":
			macOSLabelID = label.ID
		case "iOS":
			iOSLabelID = label.ID
		case "iPadOS":
			iPadOSLabelID = label.ID
		}
	}

	parts := []string{}
	args := []interface{}{}
	for _, h := range hosts {
		var osLabelID uint
		switch h.Platform {
		case "ios":
			osLabelID = iOSLabelID
		case "ipados":
			osLabelID = iPadOSLabelID
		default: // at this point, assume "darwin"
			osLabelID = macOSLabelID
		}
		parts = append(parts, "(?,?),(?,?)")
		args = append(args, h.ID, allHostsLabelID, h.ID, osLabelID)
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO label_membership (host_id, label_id) VALUES %s
			ON DUPLICATE KEY UPDATE host_id = host_id`, strings.Join(parts, ",")), args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert label membership")
	}

	return nil
}

// deleteMDMOSCustomSettingsForHost deletes configuration profiles and
// declarations for a host based on its platform.
func (ds *Datastore) deleteMDMOSCustomSettingsForHost(ctx context.Context, tx sqlx.ExtContext, uuid, platform string) error {
	tableMap := map[string][]string{
		"darwin":  {"host_mdm_apple_profiles", "host_mdm_apple_declarations"},
		"ios":     {"host_mdm_apple_profiles", "host_mdm_apple_declarations"},
		"ipados":  {"host_mdm_apple_profiles", "host_mdm_apple_declarations"},
		"windows": {"host_mdm_windows_profiles"},
	}

	tables, ok := tableMap[platform]
	if !ok {
		return ctxerr.Errorf(ctx, "unsupported platform %s", platform)
	}

	for _, table := range tables {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`
                    DELETE FROM %s
                    WHERE host_uuid = ?`, table), uuid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "removing all %s from host %s", table, uuid)
		}
	}

	return nil
}

func (ds *Datastore) MDMTurnOff(ctx context.Context, uuid string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var host fleet.Host
		err := sqlx.GetContext(
			ctx, tx, &host,
			`SELECT id, platform FROM hosts WHERE uuid = ? LIMIT 1`, uuid,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting host info from UUID")
		}

		if !fleet.MDMSupported(host.Platform) {
			return ctxerr.Errorf(ctx, "unsupported host platform: %q", host.Platform)
		}

		// NOTE: set installed_from_dep = 0 so DEP host will not be
		// counted as pending after it unenrolls.
		_, err = tx.ExecContext(ctx, `
			UPDATE host_mdm
			SET
			  enrolled = 0,
			  installed_from_dep = 0,
			  server_url = '',
			  mdm_id = NULL
			WHERE
			  host_id = ?`, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clearing host_mdm for host")
		}

		// Since the host is unenrolled, delete all profiles assigned to the
		// host manually, the device won't Acknowledge any more requests (eg:
		// to delete profiles) and profiles are automatically removed on
		// unenrollment.
		if err := ds.deleteMDMOSCustomSettingsForHost(ctx, tx, uuid, host.Platform); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting profiles for host")
		}

		// clear up the MDM-dependent upcoming activities as it won't be able
		// to process them anymore
		if err := ds.ClearMDMUpcomingActivitiesDB(ctx, tx, uuid); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting mdm-related upcoming activities for host")
		}

		// NOTE: intentionally keeping disk encryption keys and bootstrap
		// package information.

		// iPhones and iPads have no osquery thus we don't need to refetch.
		if host.Platform == "ios" || host.Platform == "ipados" {
			return nil
		}

		// request a refetch to update any eventually consistent stale information.
		err = updateHostRefetchRequestedDB(ctx, tx, host.ID, true)
		return ctxerr.Wrap(ctx, err, "setting host refetch requested")
	})
}

func unionSelectDevices(devices []hostToCreateFromMDM) (stmt string, args []interface{}) {
	for i, d := range devices {
		if i == 0 {
			stmt = "SELECT ? hardware_serial, ? hardware_model, ? platform"
		} else {
			stmt += " UNION SELECT ?, ?, ?"
		}

		// map the platform hint to Fleet's hosts.platform field.
		normalizedHint := strings.ToLower(d.PlatformHint)
		platform := string(fleet.MacOSPlatform)
		switch {
		case strings.Contains(normalizedHint, "iphone"):
			platform = string(fleet.IOSPlatform)
		case strings.Contains(normalizedHint, "ipad"):
			platform = string(fleet.IPadOSPlatform)
		}
		args = append(args, d.HardwareSerial, d.HardwareModel, platform)
	}

	return stmt, args
}

func (ds *Datastore) GetHostDEPAssignment(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
	var res fleet.HostDEPAssignment
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, `
		SELECT host_id, added_at, deleted_at, abm_token_id FROM host_dep_assignments hdep WHERE hdep.host_id = ?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostDEPAssignment").WithID(hostID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting host dep assignments")
	}
	return &res, nil
}

func (ds *Datastore) DeleteHostDEPAssignmentsFromAnotherABM(ctx context.Context, abmTokenID uint, serials []string) error {
	if len(serials) == 0 {
		return nil
	}

	type depAssignment struct {
		HardwareSerial string `db:"hardware_serial"`
		ABMTokenID     uint   `db:"abm_token_id"`
	}
	selectStmt, selectArgs, err := sqlx.In(`
		SELECT h.hardware_serial, hdep.abm_token_id
		FROM hosts h
		JOIN host_dep_assignments hdep ON h.id = hdep.host_id
		WHERE hdep.abm_token_id != ? AND h.hardware_serial IN (?)`, abmTokenID, serials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN statement for selecting host serials")
	}
	var others []depAssignment
	if err = sqlx.SelectContext(ctx, ds.reader(ctx), &others, selectStmt, selectArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting host serials")
	}
	tokenToSerials := map[uint][]string{}
	for _, other := range others {
		tokenToSerials[other.ABMTokenID] = append(tokenToSerials[other.ABMTokenID], other.HardwareSerial)
	}
	for otherTokenID, otherSerials := range tokenToSerials {
		if err := ds.DeleteHostDEPAssignments(ctx, otherTokenID, otherSerials); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting DEP assignments for other ABM")
		}
	}
	return nil
}

func (ds *Datastore) DeleteHostDEPAssignments(ctx context.Context, abmTokenID uint, serials []string) error {
	if len(serials) == 0 {
		return nil
	}

	selectStmt, selectArgs, err := sqlx.In(`
		SELECT h.id, hmdm.enrollment_status
		FROM hosts h
		JOIN host_dep_assignments hdep ON h.id = hdep.host_id
		LEFT JOIN host_mdm hmdm ON h.id = hmdm.host_id
		WHERE hdep.abm_token_id = ? AND h.hardware_serial IN (?)`, abmTokenID, serials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN statement for selecting host IDs")
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		type hostWithEnrollmentStatus struct {
			ID               uint    `db:"id"`
			EnrollmentStatus *string `db:"enrollment_status"`
		}
		var hosts []hostWithEnrollmentStatus
		if err = sqlx.SelectContext(ctx, tx, &hosts, selectStmt, selectArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting host IDs")
		}
		if len(hosts) == 0 {
			// Nothing to delete. Hosts may have already been transferred to another ABM.
			return nil
		}
		var hostIDs []uint
		var hostIDsPending []uint
		for _, host := range hosts {
			hostIDs = append(hostIDs, host.ID)
			if host.EnrollmentStatus != nil && *host.EnrollmentStatus == "Pending" {
				hostIDsPending = append(hostIDsPending, host.ID)
			}
		}

		stmt, args, err := sqlx.In(`
          UPDATE host_dep_assignments
          SET deleted_at = NOW()
          WHERE host_id IN (?)`, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building IN statement")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting DEP assignment by host_id")
		}

		// If pending host is no longer in ABM, we should delete it because it will never enroll in Fleet.
		// If the host is later re-added to ABM, it will be re-created.
		if len(hostIDsPending) == 0 {
			return nil
		}

		return deleteHosts(ctx, tx, hostIDsPending)
	})
}

func (ds *Datastore) RestoreMDMApplePendingDEPHost(ctx context.Context, host *fleet.Host) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "restore pending dep host get app config")
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Insert a new host row with the same ID as the host that was deleted. We set only a
		// limited subset of fields just as if the host were initially ingested from DEP sync;
		// however, we also restore the UUID. Note that we are explicitly not restoring the
		// osquery_host_id.
		stmt := `
INSERT INTO hosts (
	id,
	uuid,
	hardware_serial,
	hardware_model,
	platform,
	last_enrolled_at,
	detail_updated_at,
	osquery_host_id,
	refetch_requested,
	team_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

		args := []interface{}{
			host.ID,
			host.UUID,
			host.HardwareSerial,
			host.HardwareModel,
			host.Platform,
			host.LastEnrolledAt,
			host.DetailUpdatedAt,
			nil, // osquery_host_id is not restored
			host.RefetchRequested,
			host.TeamID,
		}

		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "restore pending dep host")
		}

		// Upsert related host tables for the restored host just as if it were initially ingested
		// from DEP sync. Note we are not upserting host_dep_assignments in order to preserve the
		// existing timestamps.
		if err := upsertHostDisplayNames(ctx, tx, *host); err != nil {
			// TODO: Why didn't this work as expected?
			return ctxerr.Wrap(ctx, err, "restore pending dep host display name")
		}
		if err := upsertMDMAppleHostLabelMembershipDB(ctx, tx, ds.logger, *host); err != nil {
			return ctxerr.Wrap(ctx, err, "restore pending dep host label membership")
		}
		if err := upsertMDMAppleHostMDMInfoDB(ctx, tx, ac, true, host.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest mdm apple host upsert MDM info")
		}

		return nil
	})
}

func (ds *Datastore) GetNanoMDMEnrollment(ctx context.Context, id string) (*fleet.NanoEnrollment, error) {
	var nanoEnroll fleet.NanoEnrollment
	// use writer as it is used just after creation in some cases
	err := sqlx.GetContext(ctx, ds.writer(ctx), &nanoEnroll, `SELECT id, device_id, type, enabled, token_update_tally
		FROM nano_enrollments WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from nano_enrollments for id %s", id)
	}

	return &nanoEnroll, nil
}

func (ds *Datastore) GetNanoMDMUserEnrollment(ctx context.Context, deviceId string) (*fleet.NanoEnrollment, error) {
	var nanoEnroll fleet.NanoEnrollment
	// use writer as it is used just after creation in some cases
	err := sqlx.GetContext(ctx, ds.writer(ctx), &nanoEnroll, `SELECT id, device_id, type, enabled, token_update_tally
		FROM nano_enrollments WHERE type = 'User' AND enabled = 1 AND device_id = ? LIMIT 1`, deviceId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from nano_enrollments for device id %s", deviceId)
	}

	return &nanoEnroll, nil
}

// NOTE: this is only called from a deprecated endpoint that does not support
// Fleet variables in profiles.
func (ds *Datastore) BatchSetMDMAppleProfiles(ctx context.Context, tmID *uint, profiles []*fleet.MDMAppleConfigProfile) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := ds.batchSetMDMAppleProfilesDB(ctx, tx, tmID, profiles, nil)
		return err
	})
}

// batchSetMDMAppleProfilesDB must be called from inside a transaction.
func (ds *Datastore) batchSetMDMAppleProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	tmID *uint,
	profiles []*fleet.MDMAppleConfigProfile,
	profilesVariablesByIdentifier []fleet.MDMProfileIdentifierFleetVariables,
) (updatedDB bool, err error) {
	const loadExistingProfiles = `
SELECT
  identifier,
  profile_uuid,
  mobileconfig,
  secrets_updated_at
FROM
  mdm_apple_configuration_profiles
WHERE
  team_id = ? AND
  identifier IN (?)
`

	const deleteProfilesNotInList = `
DELETE FROM
  mdm_apple_configuration_profiles
WHERE
  team_id = ? AND
  identifier NOT IN (?)
`

	const loadToBeDeletedProfilesNotInList = `
SELECT
	profile_uuid
FROM
	mdm_apple_configuration_profiles
WHERE
	team_id = ? AND
	identifier NOT IN (?)
`

	const insertNewOrEditedProfile = `
INSERT INTO
  mdm_apple_configuration_profiles (
    profile_uuid, team_id, identifier, name, scope, mobileconfig, checksum, uploaded_at, secrets_updated_at
  )
VALUES
  -- see https://stackoverflow.com/a/51393124/1094941
  ( CONCAT('a', CONVERT(uuid() USING utf8mb4)), ?, ?, ?, ?, ?, UNHEX(MD5(mobileconfig)), CURRENT_TIMESTAMP(6), ?)
ON DUPLICATE KEY UPDATE
  uploaded_at = IF(checksum = VALUES(checksum) AND name = VALUES(name), uploaded_at, CURRENT_TIMESTAMP(6)),
  secrets_updated_at = VALUES(secrets_updated_at),
  scope = VALUES(scope),
  checksum = VALUES(checksum),
  name = VALUES(name),
  mobileconfig = VALUES(mobileconfig)
`

	// use a profile team id of 0 if no-team
	var profTeamID uint
	if tmID != nil {
		profTeamID = *tmID
	}

	// build a list of identifiers for the incoming profiles, will keep the
	// existing ones if there's a match and no change
	incomingIdents := make([]string, len(profiles))
	// at the same time, index the incoming profiles keyed by identifier for ease
	// or processing
	incomingProfs := make(map[string]*fleet.MDMAppleConfigProfile, len(profiles))
	for i, p := range profiles {
		incomingIdents[i] = p.Identifier
		incomingProfs[p.Identifier] = p
	}

	var existingProfiles []*fleet.MDMAppleConfigProfile

	if len(incomingIdents) > 0 {
		// load existing profiles that match the incoming profiles by identifiers
		stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, incomingIdents)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "inselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "build query to load existing profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "select") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "load existing profiles")
		}
	}

	// figure out if we need to delete any profiles
	keepIdents := make([]string, 0, len(incomingIdents))
	for _, p := range existingProfiles {
		if newP := incomingProfs[p.Identifier]; newP != nil {
			keepIdents = append(keepIdents, p.Identifier)
		}
	}

	// profiles that are managed and delivered by Fleet
	fleetIdents := []string{}
	for ident := range mobileconfig.FleetPayloadIdentifiers() {
		fleetIdents = append(fleetIdents, ident)
	}

	var (
		stmt                string
		args                []interface{}
		deletedProfileUUIDs []string
	)
	// load the profiles that will be deleted, so we can cancel the install immediately
	stmt, args, err = sqlx.In(loadToBeDeletedProfilesNotInList, profTeamID, append(keepIdents, fleetIdents...))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "build query to load profiles to be deleted")
	}
	if err := sqlx.SelectContext(ctx, tx, &deletedProfileUUIDs, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "load profiles to be deleted")
	}

	// delete the obsolete profiles (all those that are not in keepIdents or delivered by Fleet)
	var result sql.Result
	stmt, args, err = sqlx.In(deleteProfilesNotInList, profTeamID, append(keepIdents, fleetIdents...))
	if err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "indelete") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
		}
		return false, ctxerr.Wrap(ctx, err, "build statement to delete obsolete profiles")
	}
	if result, err = tx.ExecContext(ctx, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "delete") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
		}
		return false, ctxerr.Wrap(ctx, err, "delete obsolete profiles")
	}
	if result != nil {
		rows, _ := result.RowsAffected()
		updatedDB = rows > 0
	}
	if len(deletedProfileUUIDs) > 0 {
		// cancel installs of the deleted profiles immediately
		if err := cancelAppleHostInstallsForDeletedMDMProfiles(ctx, tx, deletedProfileUUIDs); err != nil {
			return false, ctxerr.Wrap(ctx, err, "cancel installs of deleted profiles")
		}
	}

	// insert the new profiles and the ones that have changed, and track those
	// that did insert or update, as we will need to update their variables (if
	// the profile did not change, its variables cannot have changed since the
	// contents is the same as it was already).
	var profileIdentsInsertedOrUpdated []string
	for _, p := range incomingProfs {
		if result, err = tx.ExecContext(ctx, insertNewOrEditedProfile, profTeamID, p.Identifier, p.Name, p.Scope,
			p.Mobileconfig, p.SecretsUpdatedAt); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "insert") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrapf(ctx, err, "insert new/edited profile with identifier %q", p.Identifier)
		}
		didInsertOrUpdate := insertOnDuplicateDidInsertOrUpdate(result)
		updatedDB = updatedDB || didInsertOrUpdate

		if didInsertOrUpdate {
			profileIdentsInsertedOrUpdated = append(profileIdentsInsertedOrUpdated, p.Identifier)
		}
	}

	if len(profileIdentsInsertedOrUpdated) > 0 {
		var insertedUpdatedProfs []*fleet.MDMAppleConfigProfile
		// load the profiles that match the inserted/updated profiles by identifier
		// to grab their uuids
		stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, profileIdentsInsertedOrUpdated)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "build query to load inserted/updated profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &insertedUpdatedProfs, stmt, args...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "load inserted/updated profiles")
		}

		lookupVariablesByIdentifier := make(map[string][]string, len(profilesVariablesByIdentifier))
		for _, p := range profilesVariablesByIdentifier {
			lookupVariablesByIdentifier[p.Identifier] = p.FleetVariables
		}

		// collect the profiles+variables that need to be upserted, keyed by
		// profile UUID as needed by the batch-set profile variables function.
		profilesVarsToUpsert := make([]fleet.MDMProfileUUIDFleetVariables, 0, len(insertedUpdatedProfs))
		for _, p := range insertedUpdatedProfs {
			profilesVarsToUpsert = append(profilesVarsToUpsert, fleet.MDMProfileUUIDFleetVariables{
				ProfileUUID:    p.ProfileUUID,
				FleetVariables: lookupVariablesByIdentifier[p.Identifier],
			})
		}

		if err := batchSetProfileVariableAssociationsDB(ctx, tx, profilesVarsToUpsert, "darwin"); err != nil {
			return false, ctxerr.Wrap(ctx, err, "inserting apple profile variable associations")
		}
	}

	// build a list of labels so the associations can be batch-set all at once
	// TODO: with minor changes this chunk of code could be shared
	// between macOS and Windows, but at the time of this
	// implementation we're under tight time constraints.
	incomingLabels := []fleet.ConfigurationProfileLabel{}
	var profsWithoutLabels []string
	if len(incomingIdents) > 0 {
		var newlyInsertedProfs []*fleet.MDMAppleConfigProfile
		// load current profiles (again) that match the incoming profiles by name to grab their uuids
		stmt, args, err := sqlx.In(loadExistingProfiles, profTeamID, incomingIdents)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "inreselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "build query to load newly inserted profiles")
		}
		if err := sqlx.SelectContext(ctx, tx, &newlyInsertedProfs, stmt, args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "reselect") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrap(ctx, err, "load newly inserted profiles")
		}

		for _, newlyInsertedProf := range newlyInsertedProfs {
			incomingProf, ok := incomingProfs[newlyInsertedProf.Identifier]
			if !ok {
				return false, ctxerr.Wrapf(ctx, err, "profile %q is in the database but was not incoming", newlyInsertedProf.Identifier)
			}

			var profHasLabel bool
			for _, label := range incomingProf.LabelsIncludeAll {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = false
				label.RequireAll = true
				incomingLabels = append(incomingLabels, label)
				profHasLabel = true
			}
			for _, label := range incomingProf.LabelsIncludeAny {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = false
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
				profHasLabel = true
			}
			for _, label := range incomingProf.LabelsExcludeAny {
				label.ProfileUUID = newlyInsertedProf.ProfileUUID
				label.Exclude = true
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
				profHasLabel = true
			}
			if !profHasLabel {
				profsWithoutLabels = append(profsWithoutLabels, newlyInsertedProf.ProfileUUID)
			}
		}
	}

	// insert label associations
	var updatedLabels bool
	if updatedLabels, err = batchSetProfileLabelAssociationsDB(ctx, tx, incomingLabels, profsWithoutLabels,
		"darwin"); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "labels") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
		}
		return false, ctxerr.Wrap(ctx, err, "inserting apple profile label associations")
	}
	return updatedDB || updatedLabels, nil
}

func (ds *Datastore) BulkDeleteMDMAppleHostsConfigProfiles(ctx context.Context, profs []*fleet.MDMAppleProfilePayload) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return ds.bulkDeleteMDMAppleHostsConfigProfilesDB(ctx, tx, profs)
	})
}

func (ds *Datastore) bulkDeleteMDMAppleHostsConfigProfilesDB(ctx context.Context, tx sqlx.ExtContext, profs []*fleet.MDMAppleProfilePayload) error {
	if len(profs) == 0 {
		return nil
	}

	executeDeleteBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`DELETE FROM host_mdm_apple_profiles WHERE (profile_identifier, host_uuid) IN (%s)`, strings.TrimSuffix(valuePart, ","))
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "error deleting host_mdm_apple_profiles")
		}
		return nil
	}

	var (
		args       []any
		sb         strings.Builder
		batchCount int
	)

	const defaultBatchSize = 1000 // results in this times 2 placeholders
	batchSize := defaultBatchSize
	if ds.testDeleteMDMProfilesBatchSize > 0 {
		batchSize = ds.testDeleteMDMProfilesBatchSize
	}

	resetBatch := func() {
		batchCount = 0
		args = args[:0]
		sb.Reset()
	}

	for _, p := range profs {
		args = append(args, p.ProfileIdentifier, p.HostUUID)
		sb.WriteString("(?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := executeDeleteBatch(sb.String(), args); err != nil {
				return err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeDeleteBatch(sb.String(), args); err != nil {
			return err
		}
	}
	return nil
}

// NOTE If onlyProfileUUIDs is provided (not nil), only profiles with
// those UUIDs will be update instead of rebuilding all pending
// profiles for hosts
func (ds *Datastore) bulkSetPendingMDMAppleHostProfilesDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	hostUUIDs []string,
	onlyProfileUUIDs []string,
) (updatedDB bool, err error) {
	if len(hostUUIDs) == 0 {
		return false, nil
	}

	appleMDMProfilesDesiredStateQuery := generateDesiredStateQuery("profile")

	// TODO(mna): the conditions here (and in toRemoveStmt) are subtly different
	// than the ones in ListMDMAppleProfilesToInstall/Remove, so I'm keeping
	// those statements distinct to avoid introducing a subtle bug, but we should
	// take the time to properly analyze this and try to reuse
	// ListMDMAppleProfilesToInstall/Remove as we do in the Windows equivalent
	// method.
	//
	// I.e. for toInstallStmt, this is missing:
	// 	-- profiles in A and B with operation type "install" and NULL status
	// but I believe it would be a no-op and no harm in adding (status is
	// already NULL).
	//
	// And for toRemoveStmt, this is different:
	// 	-- except "remove" operations in any state
	// vs
	// 	-- except "remove" operations in a terminal state or already pending
	// but again I believe it would be a no-op and no harm in making them the
	// same (if I'm understanding correctly, the only difference is that it
	// considers "remove" operations that have NULL status, which it would
	// update to make its status to NULL).

	profileHostIn := "h.uuid IN (?)"
	if len(onlyProfileUUIDs) > 0 {
		profileHostIn = "mae.profile_uuid IN (?) AND " + profileHostIn
	}

	toInstallStmt := fmt.Sprintf(`
	SELECT
		ds.profile_uuid as profile_uuid,
		ds.host_uuid as host_uuid,
		ds.host_platform as host_platform,
		ds.profile_identifier as profile_identifier,
		ds.profile_name as profile_name,
		ds.checksum as checksum,
		ds.secrets_updated_at as secrets_updated_at,
		ds.scope as scope
	FROM ( %s ) as ds
		LEFT JOIN host_mdm_apple_profiles hmap
			ON hmap.profile_uuid = ds.profile_uuid AND hmap.host_uuid = ds.host_uuid
	WHERE
		-- profile or secret variables have been updated
		( hmap.checksum != ds.checksum ) OR IFNULL(hmap.secrets_updated_at < ds.secrets_updated_at, FALSE) OR
		-- profiles in A but not in B
		( hmap.profile_uuid IS NULL AND hmap.host_uuid IS NULL ) OR
		-- profiles in A and B but with operation type "remove"
		( hmap.host_uuid IS NOT NULL AND ( hmap.operation_type = ? OR hmap.operation_type IS NULL ) )
`, fmt.Sprintf(appleMDMProfilesDesiredStateQuery, profileHostIn, profileHostIn, profileHostIn, profileHostIn))

	// batches of 10K hosts because h.uuid appears three times in the
	// query, and the max number of prepared statements is 65K, this was
	// good enough during a load test and gives us wiggle room if we add
	// more arguments and we forget to update the batch size.
	selectProfilesBatchSize := 10_000
	if ds.testSelectMDMProfilesBatchSize > 0 {
		selectProfilesBatchSize = ds.testSelectMDMProfilesBatchSize
	}
	selectProfilesTotalBatches := int(math.Ceil(float64(len(hostUUIDs)) / float64(selectProfilesBatchSize)))

	var wantedProfiles []*fleet.MDMAppleProfilePayload
	for i := 0; i < selectProfilesTotalBatches; i++ {
		start := i * selectProfilesBatchSize
		end := start + selectProfilesBatchSize
		if end > len(hostUUIDs) {
			end = len(hostUUIDs)
		}

		batchUUIDs := hostUUIDs[start:end]

		var stmt string
		var args []any
		var err error

		if len(onlyProfileUUIDs) > 0 {
			stmt, args, err = sqlx.In(
				toInstallStmt,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				fleet.MDMOperationTypeRemove,
			)
		} else {
			stmt, args, err = sqlx.In(toInstallStmt, batchUUIDs, batchUUIDs, batchUUIDs, batchUUIDs, fleet.MDMOperationTypeRemove)
		}

		if err != nil {
			return false, ctxerr.Wrapf(ctx, err, "building statement to select profiles to install, batch %d of %d", i,
				selectProfilesTotalBatches)
		}

		var partialResult []*fleet.MDMAppleProfilePayload
		err = sqlx.SelectContext(ctx, tx, &partialResult, stmt, args...)
		if err != nil {
			return false, ctxerr.Wrapf(ctx, err, "selecting profiles to install, batch %d of %d", i, selectProfilesTotalBatches)
		}

		wantedProfiles = append(wantedProfiles, partialResult...)
	}

	// Exclude macOS only profiles from iPhones/iPads.
	wantedProfiles = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(wantedProfiles)

	narrowByProfiles := ""
	if len(onlyProfileUUIDs) > 0 {
		narrowByProfiles = "AND hmap.profile_uuid IN (?)"
	}

	// Note: We do not need secrets_updated_at in the remove statement
	toRemoveStmt := fmt.Sprintf(`
	SELECT
		hmap.profile_uuid as profile_uuid,
		hmap.host_uuid as host_uuid,
		hmap.profile_identifier as profile_identifier,
		hmap.profile_name as profile_name,
		hmap.checksum as checksum,
		hmap.status as status,
		hmap.operation_type as operation_type,
		COALESCE(hmap.detail, '') as detail,
		hmap.command_uuid as command_uuid,
		hmap.scope as scope
	FROM ( %s ) as ds
		RIGHT JOIN host_mdm_apple_profiles hmap
			ON hmap.profile_uuid = ds.profile_uuid AND hmap.host_uuid = ds.host_uuid
	WHERE
		hmap.host_uuid IN (?) %s AND
		-- profiles that are in B but not in A
		ds.profile_uuid IS NULL AND ds.host_uuid IS NULL AND
		-- except "remove" operations in any state
		( hmap.operation_type IS NULL OR hmap.operation_type != ? ) AND
		-- except "would be removed" profiles if they are a broken label-based profile
		-- (regardless of if it is an include-all or exclude-any label)
		NOT EXISTS (
			SELECT 1
			FROM mdm_configuration_profile_labels mcpl
			WHERE
			mcpl.apple_profile_uuid = hmap.profile_uuid AND
			mcpl.label_id IS NULL
		)
`, fmt.Sprintf(appleMDMProfilesDesiredStateQuery, profileHostIn, profileHostIn, profileHostIn, profileHostIn), narrowByProfiles)

	var currentProfiles []*fleet.MDMAppleProfilePayload
	for i := 0; i < selectProfilesTotalBatches; i++ {
		start := i * selectProfilesBatchSize
		end := start + selectProfilesBatchSize
		if end > len(hostUUIDs) {
			end = len(hostUUIDs)
		}

		batchUUIDs := hostUUIDs[start:end]

		var stmt string
		var args []any
		var err error

		if len(onlyProfileUUIDs) > 0 {
			stmt, args, err = sqlx.In(
				toRemoveStmt,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				onlyProfileUUIDs, batchUUIDs,
				batchUUIDs, onlyProfileUUIDs,
				fleet.MDMOperationTypeRemove,
			)
		} else {
			stmt, args, err = sqlx.In(toRemoveStmt, batchUUIDs, batchUUIDs, batchUUIDs, batchUUIDs, batchUUIDs, fleet.MDMOperationTypeRemove)
		}

		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "building profiles to remove statement")
		}
		var partialResult []*fleet.MDMAppleProfilePayload
		err = sqlx.SelectContext(ctx, tx, &partialResult, stmt, args...)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "fetching profiles to remove")
		}

		currentProfiles = append(currentProfiles, partialResult...)
	}

	if len(wantedProfiles) == 0 && len(currentProfiles) == 0 {
		return false, nil
	}

	// delete all host profiles to start from a clean slate, new entries will be added next
	// TODO(roberto): is this really necessary? this was pre-existing
	// behavior but I think it can be refactored. For now leaving it as-is.
	//
	// TODO part II(roberto): we found this call to be a major bottleneck during load testing
	// https://github.com/fleetdm/fleet/issues/21338
	if len(wantedProfiles) > 0 {
		if err := ds.bulkDeleteMDMAppleHostsConfigProfilesDB(ctx, tx, wantedProfiles); err != nil {
			return false, ctxerr.Wrap(ctx, err, "bulk delete all profiles")
		}
		updatedDB = true
	}

	// profileIntersection tracks profilesToAdd ∩ profilesToRemove, this is used to avoid:
	//
	// - Sending a RemoveProfile followed by an InstallProfile for a
	// profile with an identifier that's already installed, which can cause
	// racy behaviors.
	// - Sending a InstallProfile command for a profile that's exactly the
	// same as the one installed. Customers have reported that sending the
	// command causes unwanted behavior.
	profileIntersection := apple_mdm.NewProfileBimap()
	profileIntersection.IntersectByIdentifierAndHostUUID(wantedProfiles, currentProfiles)

	// start by deleting any that are already in the desired state
	var hostProfilesToClean []*fleet.MDMAppleProfilePayload
	for _, p := range currentProfiles {
		if _, ok := profileIntersection.GetMatchingProfileInDesiredState(p); ok {
			hostProfilesToClean = append(hostProfilesToClean, p)
		}
	}
	if len(hostProfilesToClean) > 0 {
		if err := ds.bulkDeleteMDMAppleHostsConfigProfilesDB(ctx, tx, hostProfilesToClean); err != nil {
			return false, ctxerr.Wrap(ctx, err, "bulk delete profiles to clean")
		}
		updatedDB = true
	}

	profilesToInsert := make(map[string]*fleet.MDMAppleProfilePayload)

	executeUpsertBatch := func(valuePart string, args []any) error {
		// If this call is made, we assume the update must be done -- a new profile was added or existing one modified.
		updatedDB = true
		baseStmt := fmt.Sprintf(`
				INSERT INTO host_mdm_apple_profiles (
					profile_uuid,
					host_uuid,
					profile_identifier,
					profile_name,
					checksum,
					secrets_updated_at,
					operation_type,
					status,
					command_uuid,
					detail,
					scope
				)
				VALUES %s
				ON DUPLICATE KEY UPDATE
					operation_type = VALUES(operation_type),
					status = VALUES(status),
					command_uuid = VALUES(command_uuid),
					checksum = VALUES(checksum),
				    secrets_updated_at = VALUES(secrets_updated_at),
					detail = VALUES(detail),
					scope = VALUES(scope)
			`, strings.TrimSuffix(valuePart, ","))

		_, err := tx.ExecContext(ctx, baseStmt, args...)
		return ctxerr.Wrap(ctx, err, "bulk set pending profile status execute batch")
	}

	var (
		pargs      []any
		psb        strings.Builder
		batchCount int
	)

	const defaultBatchSize = 1000 // results in this times 9 placeholders
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	resetBatch := func() {
		batchCount = 0
		clear(profilesToInsert)
		pargs = pargs[:0]
		psb.Reset()
	}

	for _, p := range wantedProfiles {
		if pp, ok := profileIntersection.GetMatchingProfileInCurrentState(p); ok {
			if pp.Status != &fleet.MDMDeliveryFailed && bytes.Equal(pp.Checksum, p.Checksum) {
				profilesToInsert[fmt.Sprintf("%s\n%s", p.HostUUID, p.ProfileUUID)] = &fleet.MDMAppleProfilePayload{
					ProfileUUID:       p.ProfileUUID,
					ProfileIdentifier: p.ProfileIdentifier,
					ProfileName:       p.ProfileName,
					HostUUID:          p.HostUUID,
					HostPlatform:      p.HostPlatform,
					Checksum:          p.Checksum,
					SecretsUpdatedAt:  p.SecretsUpdatedAt,
					Status:            pp.Status,
					OperationType:     pp.OperationType,
					Detail:            pp.Detail,
					CommandUUID:       pp.CommandUUID,
					Scope:             pp.Scope,
				}
				pargs = append(pargs, p.ProfileUUID, p.HostUUID, p.ProfileIdentifier, p.ProfileName, p.Checksum, p.SecretsUpdatedAt,
					pp.OperationType, pp.Status, pp.CommandUUID, pp.Detail, p.Scope)
				psb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),")
				batchCount++

				if batchCount >= batchSize {
					if err := executeUpsertBatch(psb.String(), pargs); err != nil {
						return false, err
					}
					resetBatch()
				}
				continue
			}
		}

		profilesToInsert[fmt.Sprintf("%s\n%s", p.HostUUID, p.ProfileUUID)] = &fleet.MDMAppleProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			HostUUID:          p.HostUUID,
			HostPlatform:      p.HostPlatform,
			Checksum:          p.Checksum,
			SecretsUpdatedAt:  p.SecretsUpdatedAt,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            nil,
			CommandUUID:       "",
			Detail:            "",
			Scope:             p.Scope,
		}
		pargs = append(pargs, p.ProfileUUID, p.HostUUID, p.ProfileIdentifier, p.ProfileName, p.Checksum, p.SecretsUpdatedAt,
			fleet.MDMOperationTypeInstall, nil, "", "", p.Scope)
		psb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := executeUpsertBatch(psb.String(), pargs); err != nil {
				return false, err
			}
			resetBatch()
		}
	}

	for _, p := range currentProfiles {
		if _, ok := profileIntersection.GetMatchingProfileInDesiredState(p); ok {
			continue
		}
		// If the profile wasn't installed, then we do not want to change the operation to "Remove".
		// Doing so will result in Fleet attempting to remove a profile that doesn't exist on the
		// host (since the installation failed). Skipping it here will lead to it being removed from
		// the host in Fleet during profile reconciliation, which is what we want.
		if p.DidNotInstallOnHost() {
			continue
		}
		profilesToInsert[fmt.Sprintf("%s\n%s", p.HostUUID, p.ProfileUUID)] = &fleet.MDMAppleProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			HostUUID:          p.HostUUID,
			HostPlatform:      p.HostPlatform,
			Checksum:          p.Checksum,
			SecretsUpdatedAt:  p.SecretsUpdatedAt,
			OperationType:     fleet.MDMOperationTypeRemove,
			Status:            nil,
			CommandUUID:       "",
			Detail:            "",
			Scope:             p.Scope,
		}
		pargs = append(pargs, p.ProfileUUID, p.HostUUID, p.ProfileIdentifier, p.ProfileName, p.Checksum, p.SecretsUpdatedAt,
			fleet.MDMOperationTypeRemove, nil, "", "", p.Scope)
		psb.WriteString("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),")
		batchCount++

		if batchCount >= batchSize {
			if err := executeUpsertBatch(psb.String(), pargs); err != nil {
				return false, err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeUpsertBatch(psb.String(), pargs); err != nil {
			return false, err
		}
	}
	return updatedDB, nil
}

// mdmEntityTypeToDynamicNames tracks what names should be used in the
// templates for SQL statements based on the given entity type. The dynamic
// names are deliberately spelled out in full (instead of using an fmt.Sprintf
// approach) so that they are greppable in the codebase.
var mdmEntityTypeToDynamicNames = map[string]map[string]string{
	"declaration": {
		"entityUUIDColumn":        "declaration_uuid",
		"entityIdentifierColumn":  "declaration_identifier",
		"entityNameColumn":        "declaration_name",
		"countEntityLabelsColumn": "count_declaration_labels",
		"mdmAppleEntityTable":     "mdm_apple_declarations",
		"mdmEntityLabelsTable":    "mdm_declaration_labels",
		"appleEntityUUIDColumn":   "apple_declaration_uuid",
		"hostMDMAppleEntityTable": "host_mdm_apple_declarations",
		"checksumColumn":          "token",
	},
	"profile": {
		"entityUUIDColumn":        "profile_uuid",
		"entityIdentifierColumn":  "profile_identifier",
		"entityNameColumn":        "profile_name",
		"countEntityLabelsColumn": "count_profile_labels",
		"mdmAppleEntityTable":     "mdm_apple_configuration_profiles",
		"mdmEntityLabelsTable":    "mdm_configuration_profile_labels",
		"appleEntityUUIDColumn":   "apple_profile_uuid",
		"hostMDMAppleEntityTable": "host_mdm_apple_profiles",
		"checksumColumn":          "checksum",
	},
}

// generateDesiredStateQuery generates a query string that represents the
// desired state of an Apple entity based on its type (profile or declaration)
func generateDesiredStateQuery(entityType string) string {
	dynamicNames := mdmEntityTypeToDynamicNames[entityType]
	if dynamicNames == nil {
		panic(fmt.Sprintf("unknown entity type %q", entityType))
	}

	return os.Expand(`
	-- non label-based entities
	SELECT
		mae.${entityUUIDColumn},
		h.uuid as host_uuid,
		h.platform as host_platform,
		mae.identifier as ${entityIdentifierColumn},
		mae.name as ${entityNameColumn},
		mae.${checksumColumn} as ${checksumColumn},
		mae.secrets_updated_at as secrets_updated_at,
		mae.scope as scope,
		0 as ${countEntityLabelsColumn},
		0 as count_non_broken_labels,
		0 as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		${mdmAppleEntityTable} mae
			JOIN hosts h
				ON h.team_id = mae.team_id OR (h.team_id IS NULL AND mae.team_id = 0)
			JOIN nano_enrollments ne
				ON ne.device_id = h.uuid
	WHERE
		(h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados') AND
		ne.enabled = 1 AND
		ne.type = 'Device' AND
		NOT EXISTS (
			SELECT 1
			FROM ${mdmEntityLabelsTable} mel
			WHERE mel.${appleEntityUUIDColumn} = mae.${entityUUIDColumn}
		) AND
		( %s )

	UNION

	-- label-based entities where the host is a member of all the labels (include-all).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		mae.${entityUUIDColumn},
		h.uuid as host_uuid,
		h.platform as host_platform,
		mae.identifier as ${entityIdentifierColumn},
		mae.name as ${entityNameColumn},
		mae.${checksumColumn} as ${checksumColumn},
		mae.secrets_updated_at as secrets_updated_at,
		mae.scope as scope,
		COUNT(*) as ${countEntityLabelsColumn},
		COUNT(mel.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		${mdmAppleEntityTable} mae
			JOIN hosts h
				ON h.team_id = mae.team_id OR (h.team_id IS NULL AND mae.team_id = 0)
			JOIN nano_enrollments ne
				ON ne.device_id = h.uuid
			JOIN ${mdmEntityLabelsTable} mel
				ON mel.${appleEntityUUIDColumn} = mae.${entityUUIDColumn} AND mel.exclude = 0 AND mel.require_all = 1
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mel.label_id AND lm.host_id = h.id
	WHERE
		(h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados') AND
		ne.enabled = 1 AND
		ne.type = 'Device' AND
		( %s )
	GROUP BY
		mae.${entityUUIDColumn}, h.uuid, h.platform, mae.identifier, mae.name, mae.${checksumColumn}, mae.secrets_updated_at, mae.scope
	HAVING
		${countEntityLabelsColumn} > 0 AND count_host_labels = ${countEntityLabelsColumn}

	UNION

	-- label-based entities where the host is NOT a member of any of the labels (exclude-any).
	-- explicitly ignore profiles with broken excluded labels so that they are never applied,
	-- and ignore profiles that depend on labels created _after_ the label_updated_at timestamp
	-- of the host (because we don't have results for that label yet, the host may or may not be
	-- a member).
	SELECT
		mae.${entityUUIDColumn},
		h.uuid as host_uuid,
		h.platform as host_platform,
		mae.identifier as ${entityIdentifierColumn},
		mae.name as ${entityNameColumn},
		mae.${checksumColumn} as ${checksumColumn},
		mae.secrets_updated_at as secrets_updated_at,
		mae.scope as scope,
		COUNT(*) as ${countEntityLabelsColumn},
		COUNT(mel.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		-- this helps avoid the case where the host is not a member of a label
		-- just because it hasn't reported results for that label yet.
		SUM(CASE WHEN lbl.created_at IS NOT NULL AND h.label_updated_at >= lbl.created_at THEN 1 ELSE 0 END) as count_host_updated_after_labels
	FROM
		${mdmAppleEntityTable} mae
			JOIN hosts h
				ON h.team_id = mae.team_id OR (h.team_id IS NULL AND mae.team_id = 0)
			JOIN nano_enrollments ne
				ON ne.device_id = h.uuid
			JOIN ${mdmEntityLabelsTable} mel
				ON mel.${appleEntityUUIDColumn} = mae.${entityUUIDColumn} AND mel.exclude = 1 AND mel.require_all = 0
			LEFT OUTER JOIN labels lbl
				ON lbl.id = mel.label_id
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mel.label_id AND lm.host_id = h.id
	WHERE
		(h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados') AND
		ne.enabled = 1 AND
		ne.type = 'Device' AND
		( %s )
	GROUP BY
		mae.${entityUUIDColumn}, h.uuid, h.platform, mae.identifier, mae.name, mae.${checksumColumn}, mae.secrets_updated_at, mae.scope
	HAVING
		-- considers only the profiles with labels, without any broken label, with results reported after all labels were created and with the host not in any label
		${countEntityLabelsColumn} > 0 AND ${countEntityLabelsColumn} = count_non_broken_labels AND ${countEntityLabelsColumn} = count_host_updated_after_labels AND count_host_labels = 0

	UNION

	-- label-based entities where the host is a member of any the labels (include-any).
	-- by design, "include" labels cannot match if they are broken (the host cannot be
	-- a member of a deleted label).
	SELECT
		mae.${entityUUIDColumn},
		h.uuid as host_uuid,
		h.platform as host_platform,
		mae.identifier as ${entityIdentifierColumn},
		mae.name as ${entityNameColumn},
		mae.${checksumColumn} as ${checksumColumn},
		mae.secrets_updated_at as secrets_updated_at,
		mae.scope as scope,
		COUNT(*) as ${countEntityLabelsColumn},
		COUNT(mel.label_id) as count_non_broken_labels,
		COUNT(lm.label_id) as count_host_labels,
		0 as count_host_updated_after_labels
	FROM
		${mdmAppleEntityTable} mae
			JOIN hosts h
				ON h.team_id = mae.team_id OR (h.team_id IS NULL AND mae.team_id = 0)
			JOIN nano_enrollments ne
				ON ne.device_id = h.uuid
			JOIN ${mdmEntityLabelsTable} mel
				ON mel.${appleEntityUUIDColumn} = mae.${entityUUIDColumn} AND mel.exclude = 0 AND mel.require_all = 0
			LEFT OUTER JOIN label_membership lm
				ON lm.label_id = mel.label_id AND lm.host_id = h.id
	WHERE
		(h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados') AND
		ne.enabled = 1 AND
		ne.type = 'Device' AND
		( %s )
	GROUP BY
		mae.${entityUUIDColumn}, h.uuid, h.platform, mae.identifier, mae.name, mae.${checksumColumn}, mae.secrets_updated_at, mae.scope
	HAVING
		${countEntityLabelsColumn} > 0 AND count_host_labels >= 1
	`, func(s string) string { return dynamicNames[s] })
}

// generateEntitiesToInstallQuery is a set difference between:
//
//   - Set A (ds), the "desired state", can be obtained from a JOIN between
//     mdm_apple_x and hosts.
//
// - Set B, the "current state" given by host_mdm_apple_x.
//
// A - B gives us the entities that need to be installed:
//
//   - entities that are in A but not in B
//
//   - entities which contents have changed, but their identifier are
//     the same (by checking the checksums)
//
//   - entities that are in A and in B, but with an operation type of
//     "remove", regardless of the status. (technically, if status is NULL then
//     the entity should be already installed - it has not been queued for
//     remove yet -, and same if status is failed, but the proper thing to do
//     with it would be to remove the row, not return it as "to install". For
//     simplicity of implementation here (and to err on the safer side - the
//     entity's content could've changed), we'll return it as "to install" for
//     now, which will cause the row to be updated with the correct operation
//     type and status).
//
//   - entities that are in A and in B, with an operation type of "install"
//     and a NULL status. Other statuses mean that the operation is already in
//     flight (pending), the operation has been completed but is still subject
//     to independent verification by Fleet (verifying), or has reached a terminal
//     state (failed or verified). If the entity's content is edited, all
//     relevant hosts will be marked as status NULL so that it gets
//     re-installed.
//
// Note that for label-based entities, only fully-satisfied entities are
// considered for installation. This means that a broken label-based entity,
// where one of the labels does not exist anymore, will not be considered for
// installation.
func generateEntitiesToInstallQuery(entityType string) string {
	dynamicNames := mdmEntityTypeToDynamicNames[entityType]
	if dynamicNames == nil {
		panic(fmt.Sprintf("unknown entity type %q", entityType))
	}

	return fmt.Sprintf(os.Expand(`
	( %s ) as ds
		LEFT JOIN ${hostMDMAppleEntityTable} hmae
			ON hmae.${entityUUIDColumn} = ds.${entityUUIDColumn} AND hmae.host_uuid = ds.host_uuid
	WHERE
		-- entity has been updated
		( hmae.${checksumColumn} != ds.${checksumColumn} ) OR IFNULL(hmae.secrets_updated_at < ds.secrets_updated_at, FALSE) OR
		-- entity in A but not in B
		( hmae.${entityUUIDColumn} IS NULL AND hmae.host_uuid IS NULL ) OR
		-- entities in A and B but with operation type "remove"
		( hmae.host_uuid IS NOT NULL AND ( hmae.operation_type = ? OR hmae.operation_type IS NULL ) ) OR
		-- entities in A and B with operation type "install" and NULL status
		( hmae.host_uuid IS NOT NULL AND hmae.operation_type = ? AND hmae.status IS NULL )
`, func(s string) string { return dynamicNames[s] }), fmt.Sprintf(generateDesiredStateQuery(entityType), "TRUE", "TRUE", "TRUE", "TRUE"))
}

// generateEntitiesToRemoveQuery is a set difference between:
//
// - Set A (ds), the "desired state", can be obtained from a JOIN between
// mdm_apple_configuration_x and hosts.
//
// - Set B, the "current state" given by host_mdm_apple_x.
//
// B - A gives us the entities that need to be removed:
//
//   - entities that are in B but not in A, except those with operation type
//     "remove" and a terminal state (failed) or a state indicating
//     that the operation is in flight (pending) or the operation has been completed
//     but is still subject to independent verification by Fleet (verifying)
//     or the operation has been completed and independenly verified by Fleet (verified).
//
// Any other case are entities that are in both B and A, and as such are
// processed by the generateEntitiesToInstallQuery query (since they are in
// both, their desired state is necessarily to be installed).
//
// Note that for label-based entities, only those that are fully-sastisfied
// by the host are considered for install (are part of the desired state used
// to compute the ones to remove). However, as a special case, a broken
// label-based entity will NOT be removed from a host where it was
// previously installed. However, if a host used to satisfy a label-based
// entity but no longer does (and that label-based entity is not "broken"),
// the entity will be removed from the host.
func generateEntitiesToRemoveQuery(entityType string) string {
	dynamicNames := mdmEntityTypeToDynamicNames[entityType]
	if dynamicNames == nil {
		panic(fmt.Sprintf("unknown entity type %q", entityType))
	}

	return fmt.Sprintf(os.Expand(`
	( %s ) as ds
		RIGHT JOIN ${hostMDMAppleEntityTable} hmae
			ON hmae.${entityUUIDColumn} = ds.${entityUUIDColumn} AND hmae.host_uuid = ds.host_uuid
	WHERE
		-- entities that are in B but not in A
		ds.${entityUUIDColumn} IS NULL AND ds.host_uuid IS NULL AND
		-- except "remove" operations in a terminal state or already pending
		( hmae.operation_type IS NULL OR hmae.operation_type != ? OR hmae.status IS NULL ) AND
		-- except "would be removed" entities if they are a broken label-based entities
		-- (regardless of if it is an include-all or exclude-any label)
		NOT EXISTS (
			SELECT 1
			FROM ${mdmEntityLabelsTable} mcpl
			WHERE
				mcpl.${appleEntityUUIDColumn} = hmae.${entityUUIDColumn} AND
				mcpl.label_id IS NULL
		)
`, func(s string) string { return dynamicNames[s] }), fmt.Sprintf(generateDesiredStateQuery(entityType), "TRUE", "TRUE", "TRUE", "TRUE"))
}

func (ds *Datastore) ListMDMAppleProfilesToInstall(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
	query := fmt.Sprintf(`
	SELECT
		ds.profile_uuid,
		ds.host_uuid,
		ds.host_platform,
		ds.profile_identifier,
		ds.profile_name,
		ds.checksum,
		ds.secrets_updated_at,
		ds.scope
	FROM %s `,
		generateEntitiesToInstallQuery("profile"))
	var profiles []*fleet.MDMAppleProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, query, fleet.MDMOperationTypeRemove, fleet.MDMOperationTypeInstall)
	return profiles, err
}

func (ds *Datastore) ListMDMAppleProfilesToRemove(ctx context.Context) ([]*fleet.MDMAppleProfilePayload, error) {
	// Note: although some of these values (like secrets_updated_at) are not strictly necessary for profile removal,
	// we are keeping them here for consistency.
	query := fmt.Sprintf(`
	SELECT
		hmae.profile_uuid,
		hmae.profile_identifier,
		hmae.profile_name,
		hmae.host_uuid,
		hmae.checksum,
		hmae.secrets_updated_at,
		hmae.operation_type,
		COALESCE(hmae.detail, '') as detail,
		hmae.status,
		hmae.command_uuid,
		hmae.scope
	FROM %s`, generateEntitiesToRemoveQuery("profile"))
	var profiles []*fleet.MDMAppleProfilePayload
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &profiles, query, fleet.MDMOperationTypeRemove)

	return profiles, err
}

func (ds *Datastore) GetMDMAppleProfilesContents(ctx context.Context, uuids []string) (map[string]mobileconfig.Mobileconfig, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt := `
          SELECT profile_uuid, mobileconfig as mobileconfig
          FROM mdm_apple_configuration_profiles WHERE profile_uuid IN (?)
	`
	query, args, err := sqlx.In(stmt, uuids)
	if err != nil {
		return nil, err
	}
	rows, err := ds.reader(ctx).QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := make(map[string]mobileconfig.Mobileconfig)
	for rows.Next() {
		var uid string
		var mobileconfig mobileconfig.Mobileconfig
		if err := rows.Scan(&uid, &mobileconfig); err != nil {
			return nil, err
		}
		results[uid] = mobileconfig
	}
	return results, nil
}

// BulkUpsertMDMAppleHostProfiles is used to update the status of profile delivery to hosts.
func (ds *Datastore) BulkUpsertMDMAppleHostProfiles(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
	if len(payload) == 0 {
		return nil
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`
	    INSERT INTO host_mdm_apple_profiles (
              profile_uuid,
              profile_identifier,
              profile_name,
              host_uuid,
              status,
              operation_type,
              detail,
              command_uuid,
              checksum,
              secrets_updated_at,
              ignore_error,
              variables_updated_at,
			  scope
            )
            VALUES %s
            ON DUPLICATE KEY UPDATE
              status = VALUES(status),
              operation_type = VALUES(operation_type),
              detail = VALUES(detail),
              checksum = VALUES(checksum),
              secrets_updated_at = VALUES(secrets_updated_at),
              -- keep ignore error flag if the operation is still a remove
              ignore_error = IF(VALUES(operation_type) = '%s', ignore_error, VALUES(ignore_error)),
              profile_identifier = VALUES(profile_identifier),
              profile_name = VALUES(profile_name),
              command_uuid = VALUES(command_uuid),
              variables_updated_at = VALUES(variables_updated_at),
			  scope = VALUES(scope)`,
			strings.TrimSuffix(valuePart, ","), fleet.MDMOperationTypeRemove,
		)

		// We need to run with retry due to deadlocks.
		// The INSERT/ON DUPLICATE KEY UPDATE pattern is prone to deadlocks when multiple
		// threads are modifying nearby rows. That's because this statement uses gap locks.
		// When two transactions acquire the same gap lock, they may deadlock.
		// Two simultaneous transactions may happen when cron job runs and the user is updating via the UI at the same time.
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
		return err
	}

	generateValueArgs := func(p *fleet.MDMAppleBulkUpsertHostProfilePayload) (string, []any) {
		valuePart := "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),"
		args := []any{
			p.ProfileUUID, p.ProfileIdentifier, p.ProfileName, p.HostUUID, p.Status, p.OperationType, p.Detail, p.CommandUUID,
			p.Checksum, p.SecretsUpdatedAt, p.IgnoreError, p.VariablesUpdatedAt, p.Scope,
		}
		return valuePart, args
	}

	const defaultBatchSize = 1000 // number of parameters is this times number of placeholders
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	if err := batchProcessDB(payload, batchSize, generateValueArgs, executeUpsertBatch); err != nil {
		return err
	}

	return nil
}

func (ds *Datastore) UpdateOrDeleteHostMDMAppleProfile(ctx context.Context, profile *fleet.HostMDMAppleProfile) error {
	if profile.OperationType == fleet.MDMOperationTypeRemove && profile.Status != nil {
		var ignoreError bool
		if *profile.Status == fleet.MDMDeliveryFailed {
			// Check whether we should ignore the error.
			err := sqlx.GetContext(ctx, ds.reader(ctx), &ignoreError, `
				SELECT ignore_error FROM host_mdm_apple_profiles WHERE host_uuid = ? AND command_uuid = ?`,
				profile.HostUUID, profile.CommandUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "get ignore error")
			}
		}
		if ignoreError ||
			(*profile.Status == fleet.MDMDeliveryVerifying || *profile.Status == fleet.MDMDeliveryVerified) {
			_, err := ds.writer(ctx).ExecContext(ctx, `
          DELETE FROM host_mdm_apple_profiles
          WHERE host_uuid = ? AND command_uuid = ?
        `, profile.HostUUID, profile.CommandUUID)
			return err
		}
	}

	detail := profile.Detail

	if profile.OperationType == fleet.MDMOperationTypeRemove && profile.Status != nil && *profile.Status == fleet.MDMDeliveryFailed {
		detail = fmt.Sprintf("Failed to remove: %s", detail)
	}

	// Check whether we want to set a install operation as 'verifying' for an iOS/iPadOS device.
	var isIOSIPadOSInstallVerifiying bool
	if profile.OperationType == fleet.MDMOperationTypeInstall && profile.Status != nil && *profile.Status == fleet.MDMDeliveryVerifying {
		if err := ds.writer(ctx).GetContext(ctx, &isIOSIPadOSInstallVerifiying, `
          SELECT platform = 'ios' OR platform = 'ipados' FROM hosts WHERE uuid = ?`,
			profile.HostUUID,
		); err != nil {
			return err
		}
	}

	status := profile.Status
	if isIOSIPadOSInstallVerifiying {
		// iOS/iPadOS devices do not have osquery,
		// thus they go from 'pending' straight to 'verified'
		status = &fleet.MDMDeliveryVerified
	}

	// We need to run with retry due to potential deadlocks with BulkSetPendingMDMHostProfiles.
	// Deadlock seen in 2024/12/12 loadtest: https://docs.google.com/document/d/1-Q6qFTd7CDm-lh7MVRgpNlNNJijk6JZ4KO49R1fp80U

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		if profile.VariablesUpdatedAt != nil {
			// Only update variable_updated_at if the caller supplied it. Not all callers care but
			// we don't want to "lose" it since the install time should never need to be earlier
			// than it currently is set to.
			_, err = tx.ExecContext(ctx, `
          UPDATE host_mdm_apple_profiles
          SET status = ?, operation_type = ?, detail = ?, variables_updated_at = ?
          WHERE host_uuid = ? AND command_uuid = ?
        `, status, profile.OperationType, detail, profile.VariablesUpdatedAt, profile.HostUUID, profile.CommandUUID)
		} else {
			_, err = tx.ExecContext(ctx, `
          UPDATE host_mdm_apple_profiles
          SET status = ?, operation_type = ?, detail = ?
          WHERE host_uuid = ? AND command_uuid = ?
        `, status, profile.OperationType, detail, profile.HostUUID, profile.CommandUUID)
		}
		return err
	})
	return err
}

// sqlCaseMDMAppleStatus returns a SQL snippet that can be used to determine the status of a host
// based on the status of its profiles and declarations and filevault status. It should be used in
// conjunction with sqlJoinMDMAppleProfilesStatus and sqlJoinMDMAppleDeclarationsStatus. It assumes the
// hosts table to be aliased as 'h' and the host_disk_encryption_keys table to be aliased as 'hdek'.
func sqlCaseMDMAppleStatus() string {
	// NOTE: To make this snippet reusable, we're not using sqlx.Named here because it would
	// complicate usage in other queries (e.g., list hosts).
	var (
		failed    = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryFailed))
		pending   = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryPending))
		verifying = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerifying))
		verified  = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerified))
	)
	return `
	CASE WHEN (prof_failed
		OR decl_failed
		OR fv_failed) THEN
		` + failed + `
	WHEN (prof_pending
		OR decl_pending
		-- special case for filevault, it's pending if the profile is
		-- pending OR the profile is verified or verifying but we still
		-- don't have an encryption key.
		OR(fv_pending
			OR((fv_verifying
				OR fv_verified)
			AND (hdek.base64_encrypted IS NULL OR (hdek.decryptable IS NOT NULL AND hdek.decryptable != 1))))) THEN
		` + pending + `
	WHEN (prof_verifying
		OR decl_verifying
		-- special case when fv profile is verifying, and we already have an encryption key, in any state, we treat as verifying
		OR(fv_verifying
			AND hdek.base64_encrypted IS NOT NULL AND (hdek.decryptable IS NULL OR hdek.decryptable = 1))
		-- special case when fv profile is verified, but we didn't verify the encryption key, we treat as verifying
		OR(fv_verified
			AND hdek.base64_encrypted IS NOT NULL AND hdek.decryptable IS NULL)) THEN
		` + verifying + `
	WHEN (prof_verified
		OR decl_verified
		OR(fv_verified
			AND hdek.base64_encrypted IS NOT NULL AND hdek.decryptable = 1)) THEN
		` + verified + `
	END
`
}

// sqlJoinMDMAppleProfilesStatus returns a SQL snippet that can be used to join a table derived from
// host_mdm_apple_profiles (grouped by host_uuid and status) and the hosts table. For each host_uuid,
// it derives a boolean value for each status category. The value will be 1 if the host has any
// profile in the given status category. Separate columns are used for status of the filevault profile
// vs. all other profiles. The snippet assumes the hosts table to be aliased as 'h'.
func sqlJoinMDMAppleProfilesStatus() string {
	// NOTE: To make this snippet reusable, we're not using sqlx.Named here because it would
	// complicate usage in other queries (e.g., list hosts).
	var (
		failed    = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryFailed))
		pending   = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryPending))
		verifying = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerifying))
		verified  = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerified))
		install   = fmt.Sprintf("'%s'", string(fleet.MDMOperationTypeInstall))
		filevault = fmt.Sprintf("'%s'", mobileconfig.FleetFileVaultPayloadIdentifier)
	)
	return `
	LEFT JOIN (
		-- profile statuses grouped by host uuid, boolean value will be 1 if host has any profile with the given status
		-- filevault profiles are treated separately
		SELECT
			host_uuid,
			MAX( IF((status IS NULL OR status = ` + pending + `) AND profile_identifier != ` + filevault + `, 1, 0)) AS prof_pending,
			MAX( IF(status = ` + failed + ` AND profile_identifier != ` + filevault + `, 1, 0)) AS prof_failed,
			MAX( IF(status = ` + verifying + ` AND profile_identifier != ` + filevault + ` AND operation_type = ` + install + `, 1, 0)) AS prof_verifying,
			MAX( IF(status = ` + verified + `  AND profile_identifier != ` + filevault + ` AND operation_type = ` + install + `, 1, 0)) AS prof_verified,
			MAX( IF((status IS NULL OR status = ` + pending + `) AND profile_identifier = ` + filevault + `, 1, 0)) AS fv_pending,
			MAX( IF(status = ` + failed + ` AND profile_identifier = ` + filevault + `, 1, 0)) AS fv_failed,
			MAX( IF(status = ` + verifying + ` AND profile_identifier = ` + filevault + ` AND operation_type = ` + install + `, 1, 0)) AS fv_verifying,
			MAX( IF(status = ` + verified + `  AND profile_identifier = ` + filevault + ` AND operation_type = ` + install + `, 1, 0)) AS fv_verified
		FROM
			host_mdm_apple_profiles
		GROUP BY
			host_uuid) hmap ON h.uuid = hmap.host_uuid
`
}

// sqlJoinMDMAppleDeclarationsStatus returns a SQL snippet that can be used to join a table derived from
// host_mdm_apple_declarations (grouped by host_uuid and status) and the hosts table. For each host_uuid,
// it derives a boolean value for each status category. The value will be 1 if the host has any
// declaration in the given status category. The snippet assumes the hosts table to be aliased as 'h'.
func sqlJoinMDMAppleDeclarationsStatus() string {
	// NOTE: To make this snippet reusable, we're not using sqlx.Named here because it would
	// complicate usage in other queries (e.g., list hosts).
	var (
		failed            = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryFailed))
		pending           = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryPending))
		verifying         = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerifying))
		verified          = fmt.Sprintf("'%s'", string(fleet.MDMDeliveryVerified))
		install           = fmt.Sprintf("'%s'", string(fleet.MDMOperationTypeInstall))
		reservedDeclNames = fmt.Sprintf("'%s', '%s', '%s'", fleetmdm.FleetMacOSUpdatesProfileName, fleetmdm.FleetIOSUpdatesProfileName, fleetmdm.FleetIPadOSUpdatesProfileName)
	)
	return `
	LEFT JOIN (
		-- declaration statuses grouped by host uuid, boolean value will be 1 if host has any declaration with the given status
		SELECT
			host_uuid,
			MAX( IF((status IS NULL OR status = ` + pending + `), 1, 0)) AS decl_pending,
			MAX( IF(status = ` + failed + `, 1, 0)) AS decl_failed,
			MAX( IF(status = ` + verifying + ` , 1, 0)) AS decl_verifying,
			MAX( IF(status = ` + verified + ` , 1, 0)) AS decl_verified
		FROM
			host_mdm_apple_declarations
		WHERE
			operation_type = ` + install + ` AND declaration_name NOT IN(` + reservedDeclNames + `)
		GROUP BY
			host_uuid) hmad ON h.uuid = hmad.host_uuid
`
}

func (ds *Datastore) GetMDMAppleProfilesSummary(ctx context.Context, teamID *uint) (*fleet.MDMProfilesSummary, error) {
	stmt := `
SELECT
	COUNT(id) AS count,
	%s AS status
FROM
	hosts h
	%s
	%s
	LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
WHERE
	platform IN('darwin', 'ios', 'ipados') AND %s
GROUP BY
	status HAVING status IS NOT NULL`

	teamFilter := "team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = fmt.Sprintf("team_id = %d", *teamID)
	}

	stmt = fmt.Sprintf(stmt, sqlCaseMDMAppleStatus(), sqlJoinMDMAppleProfilesStatus(), sqlJoinMDMAppleDeclarationsStatus(), teamFilter)

	var dest []struct {
		Count  uint   `db:"count"`
		Status string `db:"status"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &dest, stmt); err != nil {
		return nil, err
	}

	byStatus := make(map[string]uint)
	for _, s := range dest {
		if _, ok := byStatus[s.Status]; ok {
			return nil, fmt.Errorf("duplicate status %s", s.Status)
		}
		byStatus[s.Status] = s.Count
	}

	var res fleet.MDMProfilesSummary
	for s, c := range byStatus {
		switch fleet.MDMDeliveryStatus(s) {
		case fleet.MDMDeliveryFailed:
			res.Failed = c
		case fleet.MDMDeliveryPending:
			res.Pending = c
		case fleet.MDMDeliveryVerifying:
			res.Verifying = c
		case fleet.MDMDeliveryVerified:
			res.Verified = c
		default:
			return nil, fmt.Errorf("unknown status %s", s)
		}
	}

	return &res, nil
}

func (ds *Datastore) InsertMDMIdPAccount(ctx context.Context, account *fleet.MDMIdPAccount) error {
	stmt := `
      INSERT INTO mdm_idp_accounts
        (uuid, username, fullname, email)
      VALUES
        (UUID(), ?, ?, ?)
      ON DUPLICATE KEY UPDATE
        username   = VALUES(username),
        fullname   = VALUES(fullname)`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, account.Username, account.Fullname, account.Email)
	return ctxerr.Wrap(ctx, err, "creating new MDM IdP account")
}

func (ds *Datastore) GetMDMIdPAccountByEmail(ctx context.Context, email string) (*fleet.MDMIdPAccount, error) {
	stmt := `SELECT uuid, username, fullname, email FROM mdm_idp_accounts WHERE email = ?`
	var acct fleet.MDMIdPAccount
	err := sqlx.GetContext(ctx, ds.reader(ctx), &acct, stmt, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMIdPAccount").WithMessage(fmt.Sprintf("with email %s", email)))
		}
		return nil, ctxerr.Wrap(ctx, err, "select mdm_idp_accounts by email")
	}
	return &acct, nil
}

func (ds *Datastore) GetMDMIdPAccountByUUID(ctx context.Context, uuid string) (*fleet.MDMIdPAccount, error) {
	stmt := `SELECT uuid, username, fullname, email FROM mdm_idp_accounts WHERE uuid = ?`
	var acct fleet.MDMIdPAccount
	err := sqlx.GetContext(ctx, ds.reader(ctx), &acct, stmt, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMIdPAccount").WithMessage(fmt.Sprintf("with uuid %s", uuid)))
		}
		return nil, ctxerr.Wrap(ctx, err, "select mdm_idp_accounts")
	}
	return &acct, nil
}

func subqueryFileVaultVerifying() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.profile_identifier = ?
                AND hmap.operation_type = ?
                AND (
		  (hmap.status = ? AND hdek.decryptable IS NULL AND hdek.host_id IS NOT NULL)
		  OR
		  (hmap.status = ? AND hdek.decryptable = 1)
		)`
	args := []interface{}{
		mobileconfig.FleetFileVaultPayloadIdentifier,
		fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerified,
		fleet.MDMDeliveryVerifying,
	}
	return sql, args
}

func subqueryFileVaultVerified() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hdek.decryptable = 1
                AND hmap.profile_identifier = ?
                AND hmap.status = ?
                AND hmap.operation_type = ?`
	args := []interface{}{
		mobileconfig.FleetFileVaultPayloadIdentifier,
		fleet.MDMDeliveryVerified,
		fleet.MDMOperationTypeInstall,
	}
	return sql, args
}

func subqueryFileVaultActionRequired() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND(hdek.decryptable = 0
                    OR (hdek.host_id IS NULL AND hdek.decryptable IS NULL))
                AND hmap.profile_identifier = ?
                AND (hmap.status = ? OR hmap.status = ?)
                AND hmap.operation_type = ?`
	args := []interface{}{
		mobileconfig.FleetFileVaultPayloadIdentifier,
		fleet.MDMDeliveryVerifying,
		fleet.MDMDeliveryVerified,
		fleet.MDMOperationTypeInstall,
	}
	return sql, args
}

func subqueryFileVaultEnforcing() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.profile_identifier = ?
                AND (hmap.status IS NULL OR hmap.status = ?)
                AND hmap.operation_type = ?
		`
	args := []interface{}{
		mobileconfig.FleetFileVaultPayloadIdentifier,
		fleet.MDMDeliveryPending,
		fleet.MDMOperationTypeInstall,
	}
	return sql, args
}

func subqueryFileVaultFailed() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
			    h.uuid = hmap.host_uuid
                AND hmap.profile_identifier = ?
                AND hmap.status = ?`
	args := []interface{}{mobileconfig.FleetFileVaultPayloadIdentifier, fleet.MDMDeliveryFailed}
	return sql, args
}

func subqueryFileVaultRemovingEnforcement() (string, []interface{}) {
	sql := `
            SELECT
                1 FROM host_mdm_apple_profiles hmap
            WHERE
                h.uuid = hmap.host_uuid
                AND hmap.profile_identifier = ?
                AND (hmap.status IS NULL OR hmap.status = ?)
                AND hmap.operation_type = ?`
	args := []interface{}{mobileconfig.FleetFileVaultPayloadIdentifier, fleet.MDMDeliveryPending, fleet.MDMOperationTypeRemove}
	return sql, args
}

func (ds *Datastore) GetMDMAppleFileVaultSummary(ctx context.Context, teamID *uint) (*fleet.MDMAppleFileVaultSummary, error) {
	sqlFmt := `
SELECT
    COUNT(
        CASE WHEN EXISTS (%s)
            THEN 1
        END) AS verified,
    COUNT(
        CASE WHEN EXISTS (%s)
            THEN 1
        END) AS verifying,
    COUNT(
        CASE WHEN EXISTS (%s)
            THEN 1
        END) AS action_required,
    COUNT(
        CASE WHEN EXISTS (%s)
            THEN 1
        END) AS enforcing,
    COUNT(
		CASE WHEN EXISTS (%s)
            THEN 1
        END) AS failed,
    COUNT(
        CASE WHEN EXISTS (%s)
            THEN 1
        END) AS removing_enforcement
FROM
    hosts h
    LEFT JOIN host_disk_encryption_keys hdek ON h.id = hdek.host_id
WHERE
    h.platform = 'darwin' AND %s`

	var args []interface{}
	subqueryVerified, subqueryVerifiedArgs := subqueryFileVaultVerified()
	args = append(args, subqueryVerifiedArgs...)
	subqueryVerifying, subqueryVerifyingArgs := subqueryFileVaultVerifying()
	args = append(args, subqueryVerifyingArgs...)
	subqueryActionRequired, subqueryActionRequiredArgs := subqueryFileVaultActionRequired()
	args = append(args, subqueryActionRequiredArgs...)
	subqueryEnforcing, subqueryEnforcingArgs := subqueryFileVaultEnforcing()
	args = append(args, subqueryEnforcingArgs...)
	subqueryFailed, subqueryFailedArgs := subqueryFileVaultFailed()
	args = append(args, subqueryFailedArgs...)
	subqueryRemovingEnforcement, subqueryRemovingEnforcementArgs := subqueryFileVaultRemovingEnforcement()
	args = append(args, subqueryRemovingEnforcementArgs...)

	teamFilter := "h.team_id IS NULL"
	if teamID != nil && *teamID > 0 {
		teamFilter = "h.team_id = ?"
		args = append(args, *teamID)
	}

	stmt := fmt.Sprintf(sqlFmt, subqueryVerified, subqueryVerifying, subqueryActionRequired, subqueryEnforcing, subqueryFailed, subqueryRemovingEnforcement, teamFilter)

	var res fleet.MDMAppleFileVaultSummary
	err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, args...)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// NOTE: this method is only used internally to upsert Fleet-controlled
// profiles (called in service.ensureFleetProfiles with the Fleet configuration
// profiles). Those are known not to use any Fleet variables. This method must
// not be used for any profile that uses Fleet variables.
func (ds *Datastore) BulkUpsertMDMAppleConfigProfiles(ctx context.Context, payload []*fleet.MDMAppleConfigProfile) error {
	if len(payload) == 0 {
		return nil
	}

	var args []any
	var sb strings.Builder
	for _, cp := range payload {
		var teamID uint
		if cp.TeamID != nil {
			teamID = *cp.TeamID
		}

		args = append(args, teamID, cp.Identifier, cp.Name, cp.Scope, cp.Mobileconfig, cp.SecretsUpdatedAt)
		// see https://stackoverflow.com/a/51393124/1094941
		sb.WriteString("( CONCAT('a', CONVERT(uuid() USING utf8mb4)), ?, ?, ?, ?, ?, UNHEX(MD5(mobileconfig)), CURRENT_TIMESTAMP(), ?),")
	}

	stmt := fmt.Sprintf(`
          INSERT INTO
              mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, scope, mobileconfig, checksum, uploaded_at, secrets_updated_at)
          VALUES %s
          ON DUPLICATE KEY UPDATE
            uploaded_at = IF(checksum = VALUES(checksum) AND name = VALUES(name), uploaded_at, CURRENT_TIMESTAMP()),
            mobileconfig = VALUES(mobileconfig),
			scope = VALUES(scope),
            checksum = VALUES(checksum),
		    secrets_updated_at = VALUES(secrets_updated_at)
`, strings.TrimSuffix(sb.String(), ","))

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "upsert mdm config profiles")
	}

	return nil
}

func isMDMAppleBootstrapPackageInDB(ctx context.Context, q sqlx.QueryerContext, teamID uint) (isInDB, existsForTeam bool, err error) {
	const stmt = `SELECT COALESCE(LENGTH(bytes), 0) FROM mdm_apple_bootstrap_packages WHERE team_id = ?`
	var pkgLen int
	if err := sqlx.GetContext(ctx, q, &pkgLen, stmt, teamID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, false, nil
		}
		return false, false, ctxerr.Wrapf(ctx, err, "check for bootstrap package content in database for team %d", teamID)
	}
	return pkgLen > 0, true, nil
}

func (ds *Datastore) InsertMDMAppleBootstrapPackage(ctx context.Context, bp *fleet.MDMAppleBootstrapPackage, pkgStore fleet.MDMBootstrapPackageStore) error {
	const insStmt = `INSERT INTO mdm_apple_bootstrap_packages (team_id, name, sha256, bytes, token) VALUES (?, ?, ?, ?, ?)`
	execInsert := func(args ...any) error {
		if _, err := ds.writer(ctx).ExecContext(ctx, insStmt, args...); err != nil {
			if IsDuplicate(err) {
				return ctxerr.Wrap(ctx, alreadyExists("BootstrapPackage", fmt.Sprintf("for team %d", bp.TeamID)))
			}
			return ctxerr.Wrap(ctx, err, "create bootstrap package")
		}
		return nil
	}

	if pkgStore == nil {
		// no S3 storage configured, insert the metadata and the content in the DB
		return execInsert(bp.TeamID, bp.Name, bp.Sha256, bp.Bytes, bp.Token)
	}

	// using distinct storages for content and metadata introduces an
	// intractable problem: the operation cannot be atomic (all succeed or all
	// fail together), so what we do instead is to minimize the risk of data
	// inconsistency:
	//
	//   1. check if the row exists in the DB, if so fail immediately with a
	//   duplicate error (which would happen at the INSERT stage anyway
	//   otherwise).
	//   2. if it does not exist in the DB, check if the package is already on
	//   S3, to avoid a costly upload if it is.
	//   3. if it is not already on S3, upload the package - if this fails,
	//   return and the DB was not touched and data is still consistent.
	//   4. after upload, insert the metadata in the DB - if this fails, the
	//   only possible inconsistency is an unused package stored on S3, which a
	//   cron job will eventually cleanup.
	//   5. if everything succeeds, data is consistent and the S3 package
	//   cannot be used before it is uploaded (since the DB row is inserted
	//   after upload).
	_, existsInDB, err := isMDMAppleBootstrapPackageInDB(ctx, ds.writer(ctx), bp.TeamID)
	if err != nil {
		return err
	}
	if existsInDB {
		return ctxerr.Wrap(ctx, alreadyExists("BootstrapPackage", fmt.Sprintf("for team %d", bp.TeamID)))
	}

	pkgID := hex.EncodeToString(bp.Sha256)
	ok, err := pkgStore.Exists(ctx, pkgID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "check if bootstrap package %s already exists", pkgID)
	}
	if !ok {
		if err := pkgStore.Put(ctx, pkgID, bytes.NewReader(bp.Bytes)); err != nil {
			return ctxerr.Wrapf(ctx, err, "upload bootstrap package %s to S3", pkgID)
		}
	}

	// insert in the DB with a NULL bytes content (to indicate it is on S3)
	return execInsert(bp.TeamID, bp.Name, bp.Sha256, nil, bp.Token)
}

func (ds *Datastore) CopyDefaultMDMAppleBootstrapPackage(ctx context.Context, ac *fleet.AppConfig, toTeamID uint) error {
	if ac == nil {
		return ctxerr.New(ctx, "app config must not be nil")
	}
	if toTeamID == 0 {
		return ctxerr.New(ctx, "team id must not be zero")
	}

	// NOTE: if the bootstrap package is stored in S3, nothing needs to happen on
	// S3 for a copy of it since the bytes are the same and the stored contents
	// is the same (the sha256 is copied, so it points to the same file on S3).
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Copy the bytes for the default bootstrap package to the specified team
		insertStmt := `
INSERT INTO mdm_apple_bootstrap_packages (team_id, name, sha256, bytes, token)
SELECT ?, name, sha256, bytes, ?
FROM mdm_apple_bootstrap_packages
WHERE team_id = 0
`
		_, err := tx.ExecContext(ctx, insertStmt, toTeamID, uuid.New().String())
		if err != nil {
			if IsDuplicate(err) {
				return ctxerr.Wrap(ctx, &existsError{
					ResourceType: "BootstrapPackage",
					TeamID:       &toTeamID,
				})
			}
			return ctxerr.Wrap(ctx, err, fmt.Sprintf("copy default bootstrap package to team %d", toTeamID))
		}

		// Update the team config json with the default bootstrap package url
		//
		// NOTE: The bytes copied above may not match the bytes at the url because it is possible to
		// upload a new bootrap pacakge via the UI, which replaces the bytes but does not change
		// the configured URL. This was a deliberate product design choice and it is intended that
		// the bytes would be replaced again the next time the team config is applied (i.e. via
		// fleetctl in a gitops workflow).
		url := ac.MDM.MacOSSetup.BootstrapPackage.Value
		if url != "" {
			updateConfigStmt := `
UPDATE teams
SET config = JSON_SET(config, '$.mdm.macos_setup.bootstrap_package', '%s')
WHERE id = ?
`
			_, err = tx.ExecContext(ctx, fmt.Sprintf(updateConfigStmt, url), toTeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("update bootstrap package config for team %d", toTeamID))
			}
		}

		return nil
	})
}

func (ds *Datastore) DeleteMDMAppleBootstrapPackage(ctx context.Context, teamID uint) error {
	// NOTE: if S3 storage is used for the bootstrap package, we don't delete it
	// here. The reason for this is that other teams may be using the same
	// package, so it would use the same S3 key (based on its hash). Instead we
	// rely on the cron job to clear unused packages from S3. Outside of using up
	// space in the bucket, an unused package on S3 is not a problem.

	stmt := "DELETE FROM mdm_apple_bootstrap_packages WHERE team_id = ?"
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete bootstrap package")
	}

	deleted, _ := res.RowsAffected()
	if deleted != 1 {
		return ctxerr.Wrap(ctx, notFound("BootstrapPackage").WithID(teamID))
	}
	return nil
}

func (ds *Datastore) GetMDMAppleBootstrapPackageBytes(ctx context.Context, token string, pkgStore fleet.MDMBootstrapPackageStore) (*fleet.MDMAppleBootstrapPackage, error) {
	const stmt = `SELECT name, bytes, sha256 FROM mdm_apple_bootstrap_packages WHERE token = ?`

	var bp fleet.MDMAppleBootstrapPackage
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &bp, stmt, token); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("BootstrapPackage").WithMessage(token))
		}
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package bytes")
	}

	if pkgStore != nil && len(bp.Bytes) == 0 {
		// bootstrap package is stored on S3, retrieve it
		pkgID := hex.EncodeToString(bp.Sha256)
		rc, _, err := pkgStore.Get(ctx, pkgID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "get bootstrap package %s from S3", pkgID)
		}
		defer rc.Close()

		// TODO: optimize memory usage by supporting a streaming approach
		// throughout the API (we have a similar issue with software installers).
		// Currently we load everything in memory and those can be quite big.
		bp.Bytes, err = io.ReadAll(rc)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "reading bootstrap package %s from S3", pkgID)
		}
	}
	return &bp, nil
}

func (ds *Datastore) GetMDMAppleBootstrapPackageSummary(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackageSummary, error) {
	// NOTE: Consider joining on host_dep_assignments instead of host_mdm so DEP hosts that
	// manually enroll or re-enroll are included in the results so long as they are not unassigned
	// in Apple Business Manager. The problem with using host_dep_assignments is that a host can be
	// assigned to Fleet in ABM but still manually enroll. We should probably keep using host_mdm,
	// but be better at updating the table with the right values when a host enrolls (perhaps adding
	// a query param to the enroll endpoint).
	stmt := `
          SELECT
              COUNT(IF(ncr.status = 'Acknowledged', 1, NULL)) AS installed,
              COUNT(IF(ncr.status = 'Error', 1, NULL)) AS failed,
              COUNT(IF(ncr.status IS NULL OR (ncr.status != 'Acknowledged' AND ncr.status != 'Error'), 1, NULL)) AS pending
          FROM
              hosts h
          LEFT JOIN host_mdm_apple_bootstrap_packages hmabp ON
              hmabp.host_uuid = h.uuid
          LEFT JOIN nano_command_results ncr ON
              ncr.command_uuid  = hmabp.command_uuid
          JOIN host_mdm hm ON
              hm.host_id = h.id
          WHERE
              hm.installed_from_dep = 1 AND COALESCE(h.team_id, 0) = ? AND h.platform = 'darwin'`

	var bp fleet.MDMAppleBootstrapPackageSummary
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &bp, stmt, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package summary")
	}
	return &bp, nil
}

func (ds *Datastore) RecordHostBootstrapPackage(ctx context.Context, commandUUID string, hostUUID string) error {
	stmt := `INSERT INTO host_mdm_apple_bootstrap_packages (command_uuid, host_uuid) VALUES (?, ?)
        ON DUPLICATE KEY UPDATE command_uuid = command_uuid`
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, commandUUID, hostUUID)
	return ctxerr.Wrap(ctx, err, "record bootstrap package command")
}

func (ds *Datastore) GetHostBootstrapPackageCommand(ctx context.Context, hostUUID string) (string, error) {
	var cmdUUID string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &cmdUUID, `SELECT command_uuid FROM host_mdm_apple_bootstrap_packages WHERE host_uuid = ?`, hostUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ctxerr.Wrap(ctx, notFound("HostMDMBootstrapPackage").WithName(hostUUID))
		}
		return "", ctxerr.Wrap(ctx, err, "get bootstrap package command")
	}
	return cmdUUID, nil
}

func (ds *Datastore) GetHostMDMMacOSSetup(ctx context.Context, hostID uint) (*fleet.HostMDMMacOSSetup, error) {
	// NOTE: Consider joining on host_dep_assignments instead of host_mdm so DEP hosts that
	// manually enroll or re-enroll are included in the results so long as they are not unassigned
	// in Apple Business Manager. The problem with using host_dep_assignments is that a host can be
	// assigned to Fleet in ABM but still manually enroll. We should probably keep using host_mdm,
	// but be better at updating the table with the right values when a host enrolls (perhaps adding
	// a query param to the enroll endpoint).
	stmt := `
SELECT
    CASE
        WHEN ncr.status = 'Acknowledged' THEN ?
        WHEN ncr.status = 'Error' THEN ?
        ELSE ?
    END AS bootstrap_package_status,
    COALESCE(ncr.result, '') AS result,
		mabs.name AS bootstrap_package_name
FROM
    hosts h
JOIN host_mdm_apple_bootstrap_packages hmabp ON
    hmabp.host_uuid = h.uuid
LEFT JOIN nano_command_results ncr ON
    ncr.command_uuid = hmabp.command_uuid
JOIN host_mdm hm ON
    hm.host_id = h.id
JOIN mdm_apple_bootstrap_packages mabs ON
		COALESCE(h.team_id, 0) = mabs.team_id
WHERE
    h.id = ? AND hm.installed_from_dep = 1`

	args := []interface{}{fleet.MDMBootstrapPackageInstalled, fleet.MDMBootstrapPackageFailed, fleet.MDMBootstrapPackagePending, hostID}

	var dest fleet.HostMDMMacOSSetup
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostMDMMacOSSetup").WithID(hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host mdm macos setup")
	}

	if dest.BootstrapPackageStatus == fleet.MDMBootstrapPackageFailed {
		decoded, err := mdm.DecodeCommandResults(dest.Result)
		if err != nil {
			dest.Detail = "Unable to decode command result"
		} else {
			dest.Detail = apple_mdm.FmtErrorChain(decoded.ErrorChain)
		}
	}
	return &dest, nil
}

func (ds *Datastore) GetMDMAppleBootstrapPackageMeta(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
	stmt := "SELECT team_id, name, sha256, token, created_at, updated_at FROM mdm_apple_bootstrap_packages WHERE team_id = ?"
	var bp fleet.MDMAppleBootstrapPackage
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &bp, stmt, teamID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("BootstrapPackage").WithID(teamID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get bootstrap package meta")
	}
	return &bp, nil
}

func (ds *Datastore) CleanupUnusedBootstrapPackages(ctx context.Context, pkgStore fleet.MDMBootstrapPackageStore, removeCreatedBefore time.Time) error {
	if pkgStore == nil {
		// no-op in this case, possible if not running with a Premium license or
		// configured S3 storage
		return nil
	}

	// get the list of bootstrap package hashes that are in use
	var shaIDs [][]byte
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &shaIDs, `SELECT DISTINCT sha256 FROM mdm_apple_bootstrap_packages`); err != nil {
		return ctxerr.Wrap(ctx, err, "get list of bootstrap packages in use")
	}
	var pkgIDs []string
	for _, sha := range shaIDs {
		pkgIDs = append(pkgIDs, hex.EncodeToString(sha))
	}

	_, err := pkgStore.Cleanup(ctx, pkgIDs, removeCreatedBefore)
	return ctxerr.Wrap(ctx, err, "cleanup unused bootstrap packages")
}

func getMDMAppleConfigProfileByTeamAndIdentifierDB(ctx context.Context, tx sqlx.QueryerContext, teamID *uint, profileIdentifier string) (*fleet.MDMAppleConfigProfile, error) {
	if teamID == nil {
		teamID = ptr.Uint(0)
	}

	stmt := `
SELECT
	profile_uuid,
	profile_id,
	team_id,
	name,
	scope,
	identifier,
	mobileconfig,
	created_at,
	uploaded_at
FROM
	mdm_apple_configuration_profiles
WHERE
	team_id=? AND identifier=?`

	var profile fleet.MDMAppleConfigProfile
	err := sqlx.GetContext(ctx, tx, &profile, stmt, teamID, profileIdentifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return &fleet.MDMAppleConfigProfile{}, ctxerr.Wrap(ctx, notFound("MDMAppleConfigProfile").WithName(profileIdentifier))
		}
		return &fleet.MDMAppleConfigProfile{}, ctxerr.Wrap(ctx, err, "get mdm apple config profile by team and identifier")
	}
	return &profile, nil
}

func (ds *Datastore) SetOrUpdateMDMAppleSetupAssistant(ctx context.Context, asst *fleet.MDMAppleSetupAssistant) (*fleet.MDMAppleSetupAssistant, error) {
	const stmt = `
		INSERT INTO
			mdm_apple_setup_assistants (team_id, global_or_team_id, name, profile)
		VALUES
			(?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			updated_at = IF(profile = VALUES(profile) AND name = VALUES(name), updated_at, CURRENT_TIMESTAMP),
			name = VALUES(name),
			profile = VALUES(profile)
`
	var globalOrTmID uint
	if asst.TeamID != nil {
		globalOrTmID = *asst.TeamID
	}
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, asst.TeamID, globalOrTmID, asst.Name, asst.Profile)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upsert mdm apple setup assistant")
	}

	// TODO(mna): to improve, previously we only cleared the profile UUIDs if the
	// profile/name did change, but from tests it seems we can't rely on
	// insertOnDuplicateDidUpdate to handle this case properly (presumably
	// because the updated_at update condition is too complex?), so at the moment
	// this clears the profile uuids at all times, even if the profile did not
	// change.
	if insertOnDuplicateDidInsertOrUpdate(res) {
		// profile was updated, need to clear the profile uuids
		if err := ds.SetMDMAppleSetupAssistantProfileUUID(ctx, asst.TeamID, "", ""); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "clear mdm apple setup assistant profiles")
		}
	}

	// reload to return the proper timestamp and id
	return ds.getMDMAppleSetupAssistant(ctx, ds.writer(ctx), asst.TeamID)
}

func (ds *Datastore) SetMDMAppleSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID, abmTokenOrgName string) error {
	const clearStmt = `
		DELETE FROM mdm_apple_setup_assistant_profiles
			WHERE setup_assistant_id = (
				SELECT
					id
				FROM
					mdm_apple_setup_assistants
				WHERE
					global_or_team_id = ?
			)`

	const upsertStmt = `
	INSERT INTO mdm_apple_setup_assistant_profiles (
		setup_assistant_id, abm_token_id, profile_uuid
	) (
		SELECT
			mas.id, abt.id, ?
		FROM
			mdm_apple_setup_assistants mas,
			abm_tokens abt
		WHERE
			mas.global_or_team_id = ? AND
			abt.organization_name = ? AND
			mas.id IS NOT NULL AND
			abt.id IS NOT NULL
	)
	ON DUPLICATE KEY UPDATE
		profile_uuid = VALUES(profile_uuid)
	`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}

	if profileUUID == "" && abmTokenOrgName == "" {
		// delete all profile uuids for that team, regardless of ABM token
		_, err := ds.writer(ctx).ExecContext(ctx, clearStmt, globalOrTmID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete mdm apple setup assistant profiles")
		}
		return nil
	}

	_, err := ds.writer(ctx).ExecContext(ctx, upsertStmt, profileUUID, globalOrTmID, abmTokenOrgName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set mdm apple setup assistant profile uuid")
	}
	return nil
}

func (ds *Datastore) GetMDMAppleSetupAssistantProfileForABMToken(ctx context.Context, teamID *uint, abmTokenOrgName string) (string, time.Time, error) {
	// to preserve previous behavior, the updated_at we use is the one of the
	// setup assistant (what we refer to as "uploaded_at" in the API responses),
	// as the important signal is when the content of the setup assistant
	// changes.
	const stmt = `
	SELECT
		msap.profile_uuid,
		mas.updated_at
	FROM
		mdm_apple_setup_assistants mas
	INNER JOIN
		mdm_apple_setup_assistant_profiles msap ON mas.id = msap.setup_assistant_id
	INNER JOIN
		abm_tokens abt ON abt.id = msap.abm_token_id
	WHERE
		mas.global_or_team_id = ? AND
		abt.organization_name = ?
`
	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}
	var asstProf struct {
		ProfileUUID string    `db:"profile_uuid"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	if err := sqlx.GetContext(ctx, ds.writer(ctx) /* needs to read recent writes */, &asstProf, stmt, globalOrTmID, abmTokenOrgName); err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, ctxerr.Wrap(ctx, notFound("MDMAppleSetupAssistant").WithID(globalOrTmID))
		}
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "get mdm apple setup assistant")
	}
	return asstProf.ProfileUUID, asstProf.UpdatedAt, nil
}

func (ds *Datastore) GetMDMAppleSetupAssistant(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
	return ds.getMDMAppleSetupAssistant(ctx, ds.reader(ctx), teamID)
}

func (ds *Datastore) getMDMAppleSetupAssistant(ctx context.Context, q sqlx.QueryerContext, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
	const stmt = `
	SELECT
		id,
		team_id,
		name,
		profile,
		updated_at as uploaded_at
	FROM
		mdm_apple_setup_assistants
	WHERE global_or_team_id = ?`

	var asst fleet.MDMAppleSetupAssistant
	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}
	if err := sqlx.GetContext(ctx, q, &asst, stmt, globalOrTmID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMAppleSetupAssistant").WithID(globalOrTmID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get mdm apple setup assistant")
	}
	return &asst, nil
}

func (ds *Datastore) DeleteMDMAppleSetupAssistant(ctx context.Context, teamID *uint) error {
	// this deletes the setup assistant for that team, and via foreign key
	// cascade also the profiles associated with it.
	const stmt = `
		DELETE FROM mdm_apple_setup_assistants
		WHERE global_or_team_id = ?`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, globalOrTmID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete mdm apple setup assistant")
	}
	return nil
}

func (ds *Datastore) ListMDMAppleDEPSerialsInTeam(ctx context.Context, teamID *uint) ([]string, error) {
	var args []any
	teamCond := `h.team_id IS NULL`
	if teamID != nil {
		teamCond = `h.team_id = ?`
		args = append(args, *teamID)
	}

	stmt := fmt.Sprintf(`
SELECT
	hardware_serial
FROM
	hosts h
	JOIN host_dep_assignments hda ON hda.host_id = h.id
WHERE
	h.hardware_serial != '' AND
	-- team_id condition
	%s AND
	hda.deleted_at IS NULL
`, teamCond)

	var serials []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &serials, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list mdm apple dep serials")
	}
	return serials, nil
}

func (ds *Datastore) ListMDMAppleDEPSerialsInHostIDs(ctx context.Context, hostIDs []uint) ([]string, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}

	stmt := `
SELECT
	hardware_serial
FROM
	hosts h
	JOIN host_dep_assignments hda ON hda.host_id = h.id
WHERE
	h.hardware_serial != '' AND
	h.id IN (?) AND
	hda.deleted_at IS NULL
`

	stmt, args, err := sqlx.In(stmt, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "prepare statement arguments")
	}

	var serials []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &serials, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list mdm apple dep serials")
	}
	return serials, nil
}

func (ds *Datastore) SetMDMAppleDefaultSetupAssistantProfileUUID(ctx context.Context, teamID *uint, profileUUID, abmTokenOrgName string) error {
	const clearStmt = `
		DELETE FROM mdm_apple_default_setup_assistants
			WHERE global_or_team_id = ?`

	const upsertStmt = `
		INSERT INTO
			mdm_apple_default_setup_assistants (team_id, global_or_team_id, profile_uuid, abm_token_id)
		SELECT
			?, ?, ?, abt.id
		FROM
			abm_tokens abt
		WHERE
			abt.organization_name = ?
		ON DUPLICATE KEY UPDATE
			profile_uuid = VALUES(profile_uuid)
`
	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}

	if profileUUID == "" && abmTokenOrgName == "" {
		// delete all profile uuids for that team, regardless of ABM token
		_, err := ds.writer(ctx).ExecContext(ctx, clearStmt, globalOrTmID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete mdm apple default setup assistant")
		}
		return nil
	}

	// upsert the profile uuid for the provided token
	_, err := ds.writer(ctx).ExecContext(ctx, upsertStmt, teamID, globalOrTmID, profileUUID, abmTokenOrgName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert mdm apple default setup assistant")
	}
	return nil
}

func (ds *Datastore) GetMDMAppleDefaultSetupAssistant(ctx context.Context, teamID *uint, abmTokenOrgName string) (profileUUID string, updatedAt time.Time, err error) {
	const stmt = `
	SELECT
		mad.profile_uuid,
		mad.updated_at
	FROM
		mdm_apple_default_setup_assistants mad
	INNER JOIN
		abm_tokens abt ON mad.abm_token_id = abt.id
	WHERE
		mad.global_or_team_id = ? AND
		abt.organization_name = ?
	`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}
	var asstProf struct {
		ProfileUUID string    `db:"profile_uuid"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	if err := sqlx.GetContext(ctx, ds.writer(ctx) /* needs to read recent writes */, &asstProf, stmt, globalOrTmID, abmTokenOrgName); err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, ctxerr.Wrap(ctx, notFound("MDMAppleDefaultSetupAssistant").WithID(globalOrTmID))
		}
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "get mdm apple default setup assistant")
	}
	return asstProf.ProfileUUID, asstProf.UpdatedAt, nil
}

func (ds *Datastore) UpdateHostDEPAssignProfileResponses(ctx context.Context, payload *godep.ProfileResponse, abmTokenID uint) error {
	return ds.updateHostDEPAssignProfileResponses(ctx, payload, &abmTokenID)
}

func (ds *Datastore) UpdateHostDEPAssignProfileResponsesSameABM(ctx context.Context, payload *godep.ProfileResponse) error {
	return ds.updateHostDEPAssignProfileResponses(ctx, payload, nil)
}

func (ds *Datastore) updateHostDEPAssignProfileResponses(ctx context.Context, payload *godep.ProfileResponse, abmTokenID *uint) error {
	if payload == nil {
		// caller should ensure this does not happen
		level.Debug(ds.logger).Log("msg", "update host dep assign profiles responses received nil payload")
		return nil
	}

	// we expect all devices to success so pre-allocate just the success slice
	success := make([]string, 0, len(payload.Devices))
	var (
		notAccessible []string
		failed        []string
	)

	for serial, status := range payload.Devices {
		switch status {
		case string(fleet.DEPAssignProfileResponseSuccess):
			success = append(success, serial)
		case string(fleet.DEPAssignProfileResponseNotAccessible):
			notAccessible = append(notAccessible, serial)
		case string(fleet.DEPAssignProfileResponseFailed):
			failed = append(failed, serial)
		default:
			// this should never happen unless Apple changes the response format, so we log it for
			// future debugging
			level.Debug(ds.logger).Log("msg", "unrecognized assign profile response", "serial", serial, "status", status)
		}
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := updateHostDEPAssignProfileResponses(ctx, tx, ds.logger, payload.ProfileUUID, success,
			string(fleet.DEPAssignProfileResponseSuccess), abmTokenID); err != nil {
			return err
		}
		if err := updateHostDEPAssignProfileResponses(ctx, tx, ds.logger, payload.ProfileUUID, notAccessible,
			string(fleet.DEPAssignProfileResponseNotAccessible), abmTokenID); err != nil {
			return err
		}
		if err := updateHostDEPAssignProfileResponses(ctx, tx, ds.logger, payload.ProfileUUID, failed,
			string(fleet.DEPAssignProfileResponseFailed), abmTokenID); err != nil {
			return err
		}
		return nil
	})
}

func updateHostDEPAssignProfileResponses(ctx context.Context, tx sqlx.ExtContext, logger log.Logger, profileUUID string, serials []string,
	status string, abmTokenID *uint,
) error {
	if len(serials) == 0 {
		return nil
	}

	setABMTokenID := ""
	if abmTokenID != nil {
		setABMTokenID = "abm_token_id = ?," //nolint:gosec // G101 false positive
	}
	stmt := fmt.Sprintf(`
UPDATE
	host_dep_assignments
JOIN
	hosts ON id = host_id
SET
	%s
	profile_uuid = ?,
	assign_profile_response = ?,
	response_updated_at = CURRENT_TIMESTAMP,
	retry_job_id = 0
WHERE
	hardware_serial IN (?)
`, setABMTokenID)
	var args []interface{}
	var err error
	if abmTokenID != nil {
		stmt, args, err = sqlx.In(stmt, abmTokenID, profileUUID, status, serials)
	} else {
		stmt, args, err = sqlx.In(stmt, profileUUID, status, serials)
	}
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement arguments")
	}
	res, err := tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update host dep assignments")
	}

	n, _ := res.RowsAffected()
	level.Info(logger).Log("msg", "update host dep assign profile responses", "profile_uuid", profileUUID, "status", status, "devices", n,
		"serials", fmt.Sprintf("%s", serials), "abm_token_id", abmTokenID)

	return nil
}

// depCooldownPeriod is the waiting period following a failed DEP assign profile request for a host.
const depCooldownPeriod = 1 * time.Hour // TODO: Make this a test config option?

func (ds *Datastore) ScreenDEPAssignProfileSerialsForCooldown(ctx context.Context, serials []string) (skipSerialsByOrgName map[string][]string, serialsByOrgName map[string][]string, err error) {
	if len(serials) == 0 {
		return skipSerialsByOrgName, serialsByOrgName, nil
	}

	stmt := `
SELECT
	CASE WHEN assign_profile_response = ? AND (response_updated_at > DATE_SUB(NOW(), INTERVAL ? SECOND) OR retry_job_id != 0) THEN
		'skip'
	ELSE
		'assign'
	END AS status,
	h.hardware_serial,
	abm.organization_name
FROM
	host_dep_assignments hda
	JOIN hosts h ON h.id = hda.host_id
	JOIN abm_tokens abm ON abm.id = hda.abm_token_id
WHERE
	h.hardware_serial IN (?)
`

	stmt, args, err := sqlx.In(stmt, string(fleet.DEPAssignProfileResponseFailed), depCooldownPeriod.Seconds(), serials)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "screen dep serials: prepare statement arguments")
	}

	var rows []struct {
		Status         string `db:"status"`
		HardwareSerial string `db:"hardware_serial"`
		ABMOrgName     string `db:"organization_name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "screen dep serials: get rows")
	}

	serialsByOrgName = make(map[string][]string)
	skipSerialsByOrgName = make(map[string][]string)

	for _, r := range rows {
		switch r.Status {
		case "assign":
			serialsByOrgName[r.ABMOrgName] = append(serialsByOrgName[r.ABMOrgName], r.HardwareSerial)
		case "skip":
			skipSerialsByOrgName[r.ABMOrgName] = append(skipSerialsByOrgName[r.ABMOrgName], r.HardwareSerial)
		default:
			return nil, nil, ctxerr.New(ctx, fmt.Sprintf("screen dep serials: %s unrecognized status: %s", r.HardwareSerial, r.Status))
		}
	}

	return skipSerialsByOrgName, serialsByOrgName, nil
}

func (ds *Datastore) GetDEPAssignProfileExpiredCooldowns(ctx context.Context) (map[uint][]string, error) {
	const stmt = `
SELECT
	COALESCE(team_id, 0) AS team_id,
	hardware_serial
FROM
	host_dep_assignments
	JOIN hosts h ON h.id = host_id
	LEFT JOIN jobs j ON j.id = retry_job_id
WHERE
	assign_profile_response = ?
	AND(retry_job_id = 0 OR j.state = ?)
	AND(response_updated_at IS NULL
		OR response_updated_at <= DATE_SUB(NOW(), INTERVAL ? SECOND))`

	var rows []struct {
		TeamID         uint   `db:"team_id"`
		HardwareSerial string `db:"hardware_serial"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, string(fleet.DEPAssignProfileResponseFailed), string(fleet.JobStateFailure), depCooldownPeriod.Seconds()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host dep assign profile expired cooldowns")
	}

	serialsByTeamID := make(map[uint][]string, len(rows))
	for _, r := range rows {
		serialsByTeamID[r.TeamID] = append(serialsByTeamID[r.TeamID], r.HardwareSerial)
	}
	return serialsByTeamID, nil
}

func (ds *Datastore) UpdateDEPAssignProfileRetryPending(ctx context.Context, jobID uint, serials []string) error {
	if len(serials) == 0 {
		return nil
	}

	stmt := `
UPDATE
	host_dep_assignments
JOIN
	hosts ON id = host_id
SET
	retry_job_id = ?
WHERE
	hardware_serial IN (?)`

	stmt, args, err := sqlx.In(stmt, jobID, serials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement arguments")
	}

	res, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update dep assign profile retry pending")
	}

	n, _ := res.RowsAffected()
	level.Info(ds.logger).Log("msg", "update dep assign profile retry pending", "job_id", jobID, "devices", n, "serials", fmt.Sprintf("%s", serials))

	return nil
}

func (ds *Datastore) MDMResetEnrollment(ctx context.Context, hostUUID string, scepRenewalInProgress bool) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var hosts []fleet.Host
		err := sqlx.SelectContext(
			ctx, tx, &hosts,
			`SELECT id, platform, team_id FROM hosts WHERE uuid = ?`, hostUUID,
		)
		switch {
		case err != nil:
			return ctxerr.Wrap(ctx, err, "resetting mdm enrollment: getting host info from UUID")
		case len(hosts) == 0:
			return ctxerr.Wrap(ctx, notFound("Host").WithName(hostUUID), "resetting mdm enrollment: getting host info from UUID")
		case len(hosts) > 1:
			// This shouldn't happen, but if it does, we log the IDs of the hosts
			// with the same UUID for debugging purposes.
			ids := make([]uint, len(hosts))
			for _, host := range hosts {
				ids = append(ids, host.ID)
			}
			level.Info(ds.logger).Log("msg", "multiple hosts found with the same uuid", "host_uuid", "host_ids", fmt.Sprintf("%v", ids))
		}
		host := hosts[0]

		if !fleet.MDMSupported(host.Platform) {
			return ctxerr.Errorf(ctx, "unsupported host platform: %q", host.Platform)
		}

		// Reconcile host_emails and host_scim_users sourced from mdm_idp_accounts.
		//
		// Note that we aren't deleting the mdm_idp_accounts themselves, just prior associations
		// for the host. This ensures that hosts that reenroll their emails up-to-date.
		switch host.Platform {
		case "darwin", "ios", "ipados":
			idp, err := reconcileHostEmailsFromMdmIdpAccountsDB(ctx, tx, ds.logger, host.ID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "resetting host_emails sourced from mdm_idp_accounts")
			}
			// TODO: confirm approach with Victor
			if _, err := tx.ExecContext(ctx, `DELETE FROM host_scim_user WHERE host_id = ?`, host.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "resetting host_scim_users")
			}
			if err := maybeAssociateHostMDMIdPWithScimUser(ctx, tx, ds.logger, host.ID, idp); err != nil {
				return ctxerr.Wrap(ctx, err, "resetting host_emails sourced from mdm_idp_accounts")
			}
		}

		// TODO: Add test coverage for scepRenewalInProgress branching logic.
		if scepRenewalInProgress {
			// FIXME: We need to revisit this flow. Short-circuiting in random places means it is
			// much more difficult to reason about the state of the host. We should try instead
			// to centralize the flow control in the lifecycle methods.
			level.Info(ds.logger).Log("host lifecycle action received for a SCEP renewal in process, skipping additional reset actions", "host_uuid", hostUUID)
			return nil
		}

		// Deleting profiles from this table will cause all profiles to
		// be re-delivered on the next cron run.
		if err := ds.deleteMDMOSCustomSettingsForHost(ctx, tx, hostUUID, host.Platform); err != nil {
			return ctxerr.Wrap(ctx, err, "resetting profiles status")
		}

		// Delete any stored disk encryption keys. This covers cases
		// where hosts re-enroll without sending a CheckOut message
		// first, for example:
		//
		// - IT admin wiping the host locally
		// - Host restoring from a back-up
		//
		// This also means that somebody running `sudo profiles renew
		// --type enrollment` will report disk encryption as "pending"
		// for a short period of time.
		_, err = tx.ExecContext(ctx, `
                    DELETE FROM host_disk_encryption_keys
                    WHERE host_id = ?`, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "resetting disk encryption key information for host")
		}

		if host.Platform == "darwin" {
			// Deleting the matching entry on this table will cause
			// the aggregate report to show this host as 'pending' to
			// install the bootstrap package.
			_, err = tx.ExecContext(ctx, `DELETE FROM host_mdm_apple_bootstrap_packages WHERE host_uuid = ?`, hostUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "resetting host_mdm_apple_bootstrap_packages")
			}
		}

		// reset the enrolled_from_migration value. We only get to this
		// stage if the host is enrolling with Fleet, SCEP renewals are
		// short-circuited before this.
		_, err = tx.ExecContext(
			ctx,
			"UPDATE nano_enrollments SET enrolled_from_migration = 0 WHERE id = ? AND enabled = 1",
			hostUUID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "setting enrolled_from_migration value")
		}
		_, err = tx.ExecContext(
			ctx,
			`UPDATE nano_devices SET platform = ?, enroll_team_id = ? WHERE id = ?`,
			host.Platform, host.TeamID, hostUUID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "setting platform and enroll_team_id")
		}

		return nil
	})
}

func (ds *Datastore) CleanMacOSMDMLock(ctx context.Context, hostUUID string) error {
	const stmt = `
UPDATE host_mdm_actions hma
JOIN hosts h ON hma.host_id = h.id
SET hma.unlock_ref = NULL,
    hma.lock_ref = NULL,
    hma.unlock_pin = NULL
WHERE h.uuid = ?
  AND hma.unlock_ref IS NOT NULL
  AND hma.unlock_pin IS NOT NULL
  `

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up macOS lock")
	}

	return nil
}

func (ds *Datastore) batchSetMDMAppleDeclarations(ctx context.Context, tx sqlx.ExtContext, tmID *uint,
	incomingDeclarations []*fleet.MDMAppleDeclaration,
) (updatedDB bool, err error) {
	// First, build a list of names (which are usually filenames) for the incoming declarations.
	// We will keep the existing ones if there's a match and no change.
	// At the same time, index the incoming declarations keyed by name for ease of processing.
	incomingNames := make([]string, len(incomingDeclarations))
	incomingDeclarationsMap := make(map[string]*fleet.MDMAppleDeclaration, len(incomingDeclarations))
	for i, p := range incomingDeclarations {
		incomingNames[i] = p.Name
		incomingDeclarationsMap[p.Name] = p
	}

	teamIDOrZero := ds.teamIDPtrToUint(tmID)
	existingDecls, err := ds.getExistingDeclarations(ctx, tx, incomingNames, teamIDOrZero)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "load existing declarations")
	}

	// figure out which declarations we should not delete, and put those into keepNames list
	keepNames := make([]string, 0, len(existingDecls)+len(fleetmdm.ListFleetReservedMacOSDeclarationNames()))
	for _, p := range existingDecls {
		if newP := incomingDeclarationsMap[p.Name]; newP != nil {
			keepNames = append(keepNames, p.Name)
		}
	}
	keepNames = append(keepNames, fleetmdm.ListFleetReservedMacOSDeclarationNames()...)

	deletedDeclUUIDs, deletedDeclarations, err := ds.deleteObsoleteDeclarations(ctx, tx, keepNames, teamIDOrZero)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "delete obsolete declarations")
	}
	if len(deletedDeclUUIDs) > 0 {
		// cancel installs of the deleted declarations immediately
		if err := cancelAppleHostInstallsForDeletedMDMDeclarations(ctx, tx, deletedDeclUUIDs); err != nil {
			return false, ctxerr.Wrap(ctx, err, "cancel installs of deleted declarations")
		}
	}

	insertedOrUpdatedDeclarations, err := ds.insertOrUpdateDeclarations(ctx, tx, incomingDeclarations, teamIDOrZero)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "insert/update declarations")
	}

	updatedLabels, err := ds.updateDeclarationsLabelAssociations(ctx, tx, incomingDeclarationsMap, teamIDOrZero)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "update declaration label associations")
	}

	return deletedDeclarations || insertedOrUpdatedDeclarations || updatedLabels, nil
}

func (ds *Datastore) updateDeclarationsLabelAssociations(ctx context.Context, tx sqlx.ExtContext,
	incomingDeclarationsMap map[string]*fleet.MDMAppleDeclaration, teamID uint,
) (updatedDB bool, err error) {
	var incomingLabels []fleet.ConfigurationProfileLabel
	var declWithoutLabels []string
	if len(incomingDeclarationsMap) > 0 {
		incomingNames := make([]string, 0, len(incomingDeclarationsMap))
		for _, p := range incomingDeclarationsMap {
			incomingNames = append(incomingNames, p.Name)
		}

		// load current declarations (again) that match the incoming declarations by name to grab their uuids
		// this is an easy way to grab the identifiers for both the existing declarations and the new ones we generated.
		//
		// TODO(roberto): if we're a bit careful, we can harvest this
		// information without this extra request in the previous DB
		// calls. Due to time constraints, I'm leaving that
		// optimization for a later iteration.
		newlyInsertedDecls, err := ds.getExistingDeclarations(ctx, tx, incomingNames, teamID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "load newly inserted declarations")
		}

		for _, newlyInsertedDecl := range newlyInsertedDecls {
			incomingDecl, ok := incomingDeclarationsMap[newlyInsertedDecl.Name]
			if !ok {
				return false, ctxerr.Wrapf(ctx, err, "declaration %q is in the database but was not incoming",
					newlyInsertedDecl.Name)
			}

			var declHasLabel bool
			for _, label := range incomingDecl.LabelsIncludeAll {
				label.ProfileUUID = newlyInsertedDecl.DeclarationUUID
				label.Exclude = false
				label.RequireAll = true
				incomingLabels = append(incomingLabels, label)
				declHasLabel = true
			}
			for _, label := range incomingDecl.LabelsIncludeAny {
				label.ProfileUUID = newlyInsertedDecl.DeclarationUUID
				label.Exclude = false
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
				declHasLabel = true
			}
			for _, label := range incomingDecl.LabelsExcludeAny {
				label.ProfileUUID = newlyInsertedDecl.DeclarationUUID
				label.Exclude = true
				label.RequireAll = false
				incomingLabels = append(incomingLabels, label)
				declHasLabel = true
			}
			if !declHasLabel {
				declWithoutLabels = append(declWithoutLabels, newlyInsertedDecl.DeclarationUUID)
			}
		}
	}

	if updatedDB, err = batchSetDeclarationLabelAssociationsDB(ctx, tx,
		incomingLabels, declWithoutLabels); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "labels") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
		}
		return false, ctxerr.Wrap(ctx, err, "inserting apple declaration label associations")
	}
	return updatedDB, err
}

func (ds *Datastore) insertOrUpdateDeclarations(ctx context.Context, tx sqlx.ExtContext, incomingDeclarations []*fleet.MDMAppleDeclaration,
	teamID uint,
) (updatedDB bool, err error) {
	const insertStmt = `
INSERT INTO mdm_apple_declarations (
	declaration_uuid,
	identifier,
	name,
	raw_json,
	secrets_updated_at,
	uploaded_at,
	team_id
)
VALUES (
	?,?,?,?,?,NOW(6),?
)
ON DUPLICATE KEY UPDATE
  uploaded_at = IF(raw_json = VALUES(raw_json) AND name = VALUES(name) AND IFNULL(secrets_updated_at = VALUES(secrets_updated_at), TRUE), uploaded_at, NOW(6)),
  secrets_updated_at = VALUES(secrets_updated_at),
  name = VALUES(name),
  identifier = VALUES(identifier),
  raw_json = VALUES(raw_json)
`

	for _, d := range incomingDeclarations {
		declUUID := fleet.MDMAppleDeclarationUUIDPrefix + uuid.NewString()
		var result sql.Result
		if result, err = tx.ExecContext(ctx, insertStmt,
			declUUID,
			d.Identifier,
			d.Name,
			d.RawJSON,
			d.SecretsUpdatedAt,
			teamID); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "insert") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return false, ctxerr.Wrapf(ctx, err, "insert new/edited declaration with identifier %q", d.Identifier)
		}
		updatedDB = updatedDB || insertOnDuplicateDidInsertOrUpdate(result)
	}
	return updatedDB, nil
}

// deleteObsoleteDeclarations deletes all declarations that are not in the keepNames list.
func (ds *Datastore) deleteObsoleteDeclarations(ctx context.Context, tx sqlx.ExtContext, keepNames []string, teamID uint) (
	deletedDeclUUIDs []string, updatedDB bool, err error,
) {
	const loadToBeDeletedProfilesStmt = `
SELECT
	declaration_uuid
FROM
  mdm_apple_declarations
WHERE
  team_id = ? AND name NOT IN (?)
`

	const fmtDeleteStmt = `
DELETE FROM
  mdm_apple_declarations
WHERE
  team_id = ? AND name NOT IN (?)
`

	selStmt, selArgs, err := sqlx.In(loadToBeDeletedProfilesStmt, teamID, keepNames)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "build query to load deleted declarations")
	}
	if err := sqlx.SelectContext(ctx, tx, &deletedDeclUUIDs, selStmt, selArgs...); err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "load deleted declarations")
	}

	delStmt, delArgs, err := sqlx.In(fmtDeleteStmt, teamID, keepNames)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "build query to delete obsolete profiles")
	}

	var result sql.Result
	if result, err = tx.ExecContext(ctx, delStmt, delArgs...); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr,
		"delete") {
		if err == nil {
			err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
		}
		return nil, false, ctxerr.Wrap(ctx, err, "delete obsolete declarations")
	}
	if result != nil {
		rows, _ := result.RowsAffected()
		updatedDB = rows > 0
	}
	return deletedDeclUUIDs, updatedDB, nil
}

func (ds *Datastore) getExistingDeclarations(ctx context.Context, tx sqlx.ExtContext, incomingNames []string,
	teamID uint,
) ([]*fleet.MDMAppleDeclaration, error) {
	const loadExistingDecls = `
SELECT
  name,
  declaration_uuid,
  raw_json
FROM
  mdm_apple_declarations
WHERE
  team_id = ? AND
  name IN (?)
`
	var existingDecls []*fleet.MDMAppleDeclaration
	if len(incomingNames) > 0 {
		// load existing declarations that match the incoming declarations by names
		loadExistingDeclsStmt, args, err := sqlx.In(loadExistingDecls, teamID, incomingNames)
		if err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr,
			"inselect") { // TODO(JVE): do we need to create similar errors for testing decls?
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return nil, ctxerr.Wrap(ctx, err, "build query to load existing declarations")
		}
		if err := sqlx.SelectContext(ctx, tx, &existingDecls, loadExistingDeclsStmt,
			args...); err != nil || strings.HasPrefix(ds.testBatchSetMDMAppleProfilesErr, "select") {
			if err == nil {
				err = errors.New(ds.testBatchSetMDMAppleProfilesErr)
			}
			return nil, ctxerr.Wrap(ctx, err, "load existing declarations")
		}
	}
	return existingDecls, nil
}

func (ds *Datastore) teamIDPtrToUint(tmID *uint) uint {
	if tmID != nil {
		return *tmID
	}
	return 0
}

func (ds *Datastore) NewMDMAppleDeclaration(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
	const stmt = `
INSERT INTO mdm_apple_declarations (
	declaration_uuid,
	team_id,
	identifier,
	name,
	raw_json,
	secrets_updated_at,
	uploaded_at)
(SELECT ?,?,?,?,?,?,CURRENT_TIMESTAMP() FROM DUAL WHERE
	NOT EXISTS (
 		SELECT 1 FROM mdm_windows_configuration_profiles WHERE name = ? AND team_id = ?
 	) AND NOT EXISTS (
 		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
 	)
)`

	return ds.insertOrUpsertMDMAppleDeclaration(ctx, stmt, declaration)
}

func (ds *Datastore) SetOrUpdateMDMAppleDeclaration(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
	const stmt = `
INSERT INTO mdm_apple_declarations (
	declaration_uuid,
	team_id,
	identifier,
	name,
	raw_json,
	secrets_updated_at,
	uploaded_at)
(SELECT ?,?,?,?,?,?,NOW(6) FROM DUAL WHERE
	NOT EXISTS (
 		SELECT 1 FROM mdm_windows_configuration_profiles WHERE name = ? AND team_id = ?
 	) AND NOT EXISTS (
 		SELECT 1 FROM mdm_apple_configuration_profiles WHERE name = ? AND team_id = ?
 	)
)
ON DUPLICATE KEY UPDATE
	identifier = VALUES(identifier),
	uploaded_at = IF(raw_json = VALUES(raw_json) AND name = VALUES(name) AND IFNULL(secrets_updated_at = VALUES(secrets_updated_at), TRUE), uploaded_at, NOW(6)),
	raw_json = VALUES(raw_json)`

	return ds.insertOrUpsertMDMAppleDeclaration(ctx, stmt, declaration)
}

func (ds *Datastore) insertOrUpsertMDMAppleDeclaration(ctx context.Context, insOrUpsertStmt string, declaration *fleet.MDMAppleDeclaration) (*fleet.MDMAppleDeclaration, error) {
	declUUID := fleet.MDMAppleDeclarationUUIDPrefix + uuid.NewString()

	var tmID uint
	if declaration.TeamID != nil {
		tmID = *declaration.TeamID
	}

	const reloadStmt = `SELECT declaration_uuid FROM mdm_apple_declarations WHERE name = ? AND team_id = ?`

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insOrUpsertStmt,
			declUUID, tmID, declaration.Identifier, declaration.Name, declaration.RawJSON,
			declaration.SecretsUpdatedAt,
			declaration.Name, tmID, declaration.Name, tmID)
		if err != nil {
			switch {
			case IsDuplicate(err):
				return ctxerr.Wrap(ctx, formatErrorDuplicateDeclaration(err, declaration))
			default:
				return ctxerr.Wrap(ctx, err, "creating new apple mdm declaration")
			}
		}

		aff, _ := res.RowsAffected()
		if aff == 0 {
			return &existsError{
				ResourceType: "MDMAppleDeclaration.Name",
				Identifier:   declaration.Name,
				TeamID:       declaration.TeamID,
			}
		}

		if err := sqlx.GetContext(ctx, tx, &declUUID, reloadStmt, declaration.Name, tmID); err != nil {
			return ctxerr.Wrap(ctx, err, "reload apple mdm declaration")
		}

		labels := make([]fleet.ConfigurationProfileLabel, 0,
			len(declaration.LabelsIncludeAll)+len(declaration.LabelsIncludeAny)+len(declaration.LabelsExcludeAny))
		for i := range declaration.LabelsIncludeAll {
			declaration.LabelsIncludeAll[i].ProfileUUID = declUUID
			declaration.LabelsIncludeAll[i].Exclude = false
			declaration.LabelsIncludeAll[i].RequireAll = true
			labels = append(labels, declaration.LabelsIncludeAll[i])
		}
		for i := range declaration.LabelsIncludeAny {
			declaration.LabelsIncludeAny[i].ProfileUUID = declUUID
			declaration.LabelsIncludeAny[i].Exclude = false
			declaration.LabelsIncludeAny[i].RequireAll = false
			labels = append(labels, declaration.LabelsIncludeAny[i])
		}
		for i := range declaration.LabelsExcludeAny {
			declaration.LabelsExcludeAny[i].ProfileUUID = declUUID
			declaration.LabelsExcludeAny[i].Exclude = true
			declaration.LabelsExcludeAny[i].RequireAll = false
			labels = append(labels, declaration.LabelsExcludeAny[i])
		}
		var declWithoutLabels []string
		if len(labels) == 0 {
			declWithoutLabels = []string{declUUID}
		}
		if _, err := batchSetDeclarationLabelAssociationsDB(ctx, tx, labels, declWithoutLabels); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting mdm declaration label associations")
		}

		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting declaration and label associations")
	}

	declaration.DeclarationUUID = declUUID
	return declaration, nil
}

func batchSetDeclarationLabelAssociationsDB(ctx context.Context, tx sqlx.ExtContext,
	declarationLabels []fleet.ConfigurationProfileLabel, declUUIDsWithoutLabels []string,
) (updatedDB bool, err error) {
	if len(declarationLabels)+len(declUUIDsWithoutLabels) == 0 {
		return false, nil
	}

	// delete any decl+label tuple that is NOT in the list of provided tuples
	// but are associated with the provided declarations (so we don't delete
	// unrelated decl+label tuples)
	deleteStmt := `
	  DELETE FROM mdm_declaration_labels
	  WHERE (apple_declaration_uuid, label_id) NOT IN (%s) AND
	  apple_declaration_uuid IN (?)
	`

	// used when only declUUIDsWithoutLabels is provided, there are no
	// labels to keep, delete all labels for declarations in this list.
	deleteNoLabelStmt := `
	  DELETE FROM mdm_declaration_labels
	  WHERE apple_declaration_uuid IN (?)
	`

	upsertStmt := `
	  INSERT INTO mdm_declaration_labels
              (apple_declaration_uuid, label_id, label_name, exclude, require_all)
          VALUES
              %s
          ON DUPLICATE KEY UPDATE
              label_id = VALUES(label_id),
              exclude = VALUES(exclude),
			  require_all = VALUES(require_all)
	`

	selectStmt := `
		SELECT apple_declaration_uuid as profile_uuid, label_name, label_id, exclude, require_all FROM mdm_declaration_labels
		WHERE (apple_declaration_uuid, label_name) IN (%s)
	`

	if len(declarationLabels) == 0 {
		deleteNoLabelStmt, args, err := sqlx.In(deleteNoLabelStmt, declUUIDsWithoutLabels)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "sqlx.In delete labels for declarations without labels")
		}

		var result sql.Result
		if result, err = tx.ExecContext(ctx, deleteNoLabelStmt, args...); err != nil {
			return false, ctxerr.Wrap(ctx, err, "deleting labels for declarations without labels")
		}
		if result != nil {
			rows, _ := result.RowsAffected()
			updatedDB = rows > 0
		}
		return updatedDB, nil
	}

	var (
		insertBuilder         strings.Builder
		selectOrDeleteBuilder strings.Builder
		selectParams          []any
		insertParams          []any
		deleteParams          []any

		setProfileUUIDs = make(map[string]struct{})
		labelsToInsert  = make(map[string]*fleet.ConfigurationProfileLabel, len(declarationLabels))
	)
	for i, pl := range declarationLabels {
		labelsToInsert[fmt.Sprintf("%s\n%s", pl.ProfileUUID, pl.LabelName)] = &declarationLabels[i]
		if i > 0 {
			insertBuilder.WriteString(",")
			selectOrDeleteBuilder.WriteString(",")
		}
		insertBuilder.WriteString("(?, ?, ?, ?, ?)")
		selectOrDeleteBuilder.WriteString("(?, ?)")
		selectParams = append(selectParams, pl.ProfileUUID, pl.LabelName)
		insertParams = append(insertParams, pl.ProfileUUID, pl.LabelID, pl.LabelName, pl.Exclude, pl.RequireAll)
		deleteParams = append(deleteParams, pl.ProfileUUID, pl.LabelID)

		setProfileUUIDs[pl.ProfileUUID] = struct{}{}
	}

	// Determine if we need to update the database
	var existingProfileLabels []fleet.ConfigurationProfileLabel
	err = sqlx.SelectContext(ctx, tx, &existingProfileLabels,
		fmt.Sprintf(selectStmt, selectOrDeleteBuilder.String()), selectParams...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "selecting existing profile labels")
	}

	updateNeeded := false
	if len(existingProfileLabels) == len(labelsToInsert) {
		for _, existing := range existingProfileLabels {
			toInsert, ok := labelsToInsert[fmt.Sprintf("%s\n%s", existing.ProfileUUID, existing.LabelName)]
			// The fleet.ConfigurationProfileLabel struct has no pointers, so we can use standard cmp.Equal
			if !ok || !cmp.Equal(existing, *toInsert) {
				updateNeeded = true
				break
			}
		}
	} else {
		updateNeeded = true
	}

	if updateNeeded {
		_, err = tx.ExecContext(ctx, fmt.Sprintf(upsertStmt, insertBuilder.String()), insertParams...)
		if err != nil {
			if isChildForeignKeyError(err) {
				// one of the provided labels doesn't exist
				return false, foreignKey("mdm_declaration_labels", fmt.Sprintf("(declaration, label)=(%v)", insertParams))
			}

			return false, ctxerr.Wrap(ctx, err, "setting label associations for declarations")
		}
		updatedDB = true
	}

	deleteStmt = fmt.Sprintf(deleteStmt, selectOrDeleteBuilder.String())

	profUUIDs := make([]string, 0, len(setProfileUUIDs)+len(declUUIDsWithoutLabels))
	for k := range setProfileUUIDs {
		profUUIDs = append(profUUIDs, k)
	}
	profUUIDs = append(profUUIDs, declUUIDsWithoutLabels...)
	deleteArgs := deleteParams
	deleteArgs = append(deleteArgs, profUUIDs)

	deleteStmt, args, err := sqlx.In(deleteStmt, deleteArgs...)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "sqlx.In delete labels for declarations")
	}
	var result sql.Result
	if result, err = tx.ExecContext(ctx, deleteStmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "deleting labels for declarations")
	}
	if result != nil {
		rows, err := result.RowsAffected()
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "count rows affected by insert")
		}
		updatedDB = updatedDB || rows > 0
	}

	return updatedDB, nil
}

func (ds *Datastore) MDMAppleDDMDeclarationsToken(ctx context.Context, hostUUID string) (*fleet.MDMAppleDDMDeclarationsToken, error) {
	const stmt = `
SELECT
	COALESCE(MD5(CONCAT(COUNT(0), GROUP_CONCAT(HEX(mad.token)
		ORDER BY
			mad.uploaded_at DESC, mad.declaration_uuid ASC separator ''))), '') AS token,
	COALESCE(MAX(mad.created_at), NOW()) AS latest_created_timestamp
FROM
	host_mdm_apple_declarations hmad
	JOIN mdm_apple_declarations mad ON hmad.declaration_uuid = mad.declaration_uuid
WHERE
	hmad.host_uuid = ? AND hmad.operation_type = ?`

	// NOTE: the token generated as part of this query decides if the DDM session
	// proceeds with sending the declarations - if the token differs from what
	// the host last applied, it will proceed. That's why we use only the "to be
	// installed" declarations for the token generation. If some declarations get
	// removed, then they will be ignored in the token generation, which will
	// change the token and make the DDM session proceed (and declarations not
	// sent get removed).

	var res fleet.MDMAppleDDMDeclarationsToken
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, hostUUID, fleet.MDMOperationTypeInstall); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get DDM declarations token")
	}

	return &res, nil
}

func (ds *Datastore) MDMAppleDDMDeclarationItems(ctx context.Context, hostUUID string) ([]fleet.MDMAppleDDMDeclarationItem, error) {
	const stmt = `
SELECT
	HEX(mad.token) as token,
	mad.identifier, mad.declaration_uuid, status, operation_type, mad.uploaded_at
FROM
	host_mdm_apple_declarations hmad
	JOIN mdm_apple_declarations mad ON mad.declaration_uuid = hmad.declaration_uuid
WHERE
	hmad.host_uuid = ?`

	var res []fleet.MDMAppleDDMDeclarationItem
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get DDM declaration items")
	}

	return res, nil
}

func (ds *Datastore) MDMAppleDDMDeclarationsResponse(ctx context.Context, identifier string, hostUUID string) (*fleet.MDMAppleDeclaration, error) {
	// TODO: When hosts table is indexed by uuid, consider joining on hosts to ensure that the
	// declaration for the host's current team is returned. In the case where the specified
	// identifier is not unique to the team, the cron should ensure that any conflicting
	// declarations are removed, but the join would provide an extra layer of safety.
	const stmt = `
SELECT
	mad.raw_json, HEX(mad.token) as token
FROM
	host_mdm_apple_declarations hmad
	JOIN mdm_apple_declarations mad ON hmad.declaration_uuid = mad.declaration_uuid
WHERE
	host_uuid = ? AND identifier = ? AND operation_type = ?`

	var res fleet.MDMAppleDeclaration
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, hostUUID, identifier, fleet.MDMOperationTypeInstall); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("MDMAppleDeclaration").WithName(identifier)
		}
		return nil, ctxerr.Wrap(ctx, err, "get ddm declarations response")
	}

	return &res, nil
}

func (ds *Datastore) MDMAppleHostDeclarationsGetAndClearResync(ctx context.Context) (hostUUIDs []string, err error) {
	stmt := `
	SELECT DISTINCT host_uuid
	FROM host_mdm_apple_declarations
	WHERE resync = '1'
	`
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get resync host uuids")
	}

	err = common_mysql.BatchProcessSimple(hostUUIDs, 1000, func(uuids []string) error {
		clearStmt := `
		UPDATE host_mdm_apple_declarations
		SET resync = '0'
		WHERE host_uuid IN (?) AND resync = '1'
		`
		clearStmt, args, err := sqlx.In(clearStmt, uuids)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "sqlx.In clear resync host uuids")
		}
		_, err = ds.writer(ctx).ExecContext(ctx, clearStmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clear resync host uuids")
		}
		return nil
	})
	return hostUUIDs, err
}

func (ds *Datastore) MDMAppleBatchSetHostDeclarationState(ctx context.Context) ([]string, error) {
	var uuids []string

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		uuids, _, err = mdmAppleBatchSetHostDeclarationStateDB(ctx, tx, batchSize, &fleet.MDMDeliveryPending)
		return err
	})

	return uuids, ctxerr.Wrap(ctx, err, "upserting host declaration state")
}

func mdmAppleBatchSetHostDeclarationStateDB(ctx context.Context, tx sqlx.ExtContext, batchSize int,
	status *fleet.MDMDeliveryStatus,
) ([]string, bool, error) {
	// once all the declarations are in place, compute the desired state
	// and find which hosts need a DDM sync.
	changedDeclarations, err := mdmAppleGetHostsWithChangedDeclarationsDB(ctx, tx)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "find hosts with changed declarations")
	}

	if len(changedDeclarations) == 0 {
		return []string{}, false, nil
	}

	// a host might have more than one declaration to sync, we do this to
	// collect unique host UUIDs in order to send a single command to each
	// host in the next step
	uuidMap := map[string]struct{}{}
	for _, d := range changedDeclarations {
		uuidMap[d.HostUUID] = struct{}{}
	}
	uuids := make([]string, 0, len(uuidMap))
	for uuid := range uuidMap {
		uuids = append(uuids, uuid)
	}

	// mark the host declarations as pending, this serves two purposes:
	//
	// - support the APIs/methods that track host status (summaries, filters, etc)
	//
	// - support the DDM endpoints, which use data from the
	//   `host_mdm_apple_declarations` table to compute which declarations to
	//   serve
	var updatedDB bool
	if updatedDB, err = mdmAppleBatchSetPendingHostDeclarationsDB(ctx, tx, batchSize, changedDeclarations, status); err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "batch insert mdm apple host declarations")
	}

	return uuids, updatedDB, nil
}

// mdmAppleBatchSetPendingHostDeclarationsDB tracks the current status of all
// the host declarations provided.
func mdmAppleBatchSetPendingHostDeclarationsDB(
	ctx context.Context,
	tx sqlx.ExtContext,
	batchSize int,
	changedDeclarations []*fleet.MDMAppleHostDeclaration,
	status *fleet.MDMDeliveryStatus,
) (updatedDB bool, err error) {
	baseStmt := `
	  INSERT INTO host_mdm_apple_declarations
	    (host_uuid, status, operation_type, token, secrets_updated_at, declaration_uuid, declaration_identifier, declaration_name)
	  VALUES
	    %s
	  ON DUPLICATE KEY UPDATE
	    status = VALUES(status),
	    operation_type = VALUES(operation_type),
	    token = VALUES(token),
	    secrets_updated_at = VALUES(secrets_updated_at)
	  `

	profilesToInsert := make(map[string]*fleet.MDMAppleHostDeclaration)

	executeUpsertBatch := func(valuePart string, args []any) error {
		// Check if the update needs to be done at all.
		selectStmt := fmt.Sprintf(`
			SELECT
				host_uuid,
				declaration_uuid,
				status,
				COALESCE(operation_type, '') AS operation_type,
				COALESCE(detail, '') AS detail,
				token,
				secrets_updated_at,
				declaration_uuid,
				declaration_identifier,
				declaration_name
			FROM host_mdm_apple_declarations WHERE (host_uuid, declaration_uuid) IN (%s)`,
			strings.TrimSuffix(strings.Repeat("(?,?),", len(profilesToInsert)), ","))
		var selectArgs []any
		for _, p := range profilesToInsert {
			selectArgs = append(selectArgs, p.HostUUID, p.DeclarationUUID)
		}
		var existingProfiles []fleet.MDMAppleHostDeclaration
		if err := sqlx.SelectContext(ctx, tx, &existingProfiles, selectStmt, selectArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending declarations select existing")
		}
		var updateNeeded bool
		if len(existingProfiles) == len(profilesToInsert) {
			for _, exist := range existingProfiles {
				insert, ok := profilesToInsert[fmt.Sprintf("%s\n%s", exist.HostUUID, exist.DeclarationUUID)]
				if !ok || !exist.Equal(*insert) {
					updateNeeded = true
					break
				}
			}
		} else {
			updateNeeded = true
		}
		defer clear(profilesToInsert)
		if !updateNeeded {
			// All profiles are already in the database, no need to update.
			return nil
		}

		updatedDB = true
		_, err := tx.ExecContext(
			ctx,
			fmt.Sprintf(baseStmt, strings.TrimSuffix(valuePart, ",")),
			args...,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending declarations insert")
		}

		// We do this cleanup as part of the main insert flow (as opposed to the clean up job)
		// because the IT admin expects to see the update relatively quickly after they upload a new declaration
		// If this becomes a bottleneck, we can move this to a separate job.
		err = cleanUpDuplicateRemoveInstall(ctx, tx, profilesToInsert)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clean up duplicate remove/install declarations")
		}
		return nil
	}

	generateValueArgs := func(d *fleet.MDMAppleHostDeclaration) (string, []any) {
		profilesToInsert[fmt.Sprintf("%s\n%s", d.HostUUID, d.DeclarationUUID)] = &fleet.MDMAppleHostDeclaration{
			HostUUID:         d.HostUUID,
			DeclarationUUID:  d.DeclarationUUID,
			Name:             d.Name,
			Identifier:       d.Identifier,
			Status:           status,
			OperationType:    d.OperationType,
			Detail:           d.Detail,
			Token:            d.Token,
			SecretsUpdatedAt: d.SecretsUpdatedAt,
		}
		valuePart := "(?, ?, ?, ?, ?, ?, ?, ?),"
		args := []any{d.HostUUID, status, d.OperationType, d.Token, d.SecretsUpdatedAt, d.DeclarationUUID, d.Identifier, d.Name}
		return valuePart, args
	}

	err = batchProcessDB(changedDeclarations, batchSize, generateValueArgs, executeUpsertBatch)
	return updatedDB, ctxerr.Wrap(ctx, err, "inserting changed host declaration state")
}

func cleanUpDuplicateRemoveInstall(ctx context.Context, tx sqlx.ExtContext, profilesToInsert map[string]*fleet.MDMAppleHostDeclaration) error {
	// If we are inserting a declaration that has a matching pending "remove" declaration (same hash),
	// we will mark the insert as verified. Why? Because there is nothing for the host to do if the same
	// declaration was removed and then immediately added back. This is a corner case that should rarely happen
	// except in testing.
	// However, it is possible that the profile was actually removed on the device, but we did not get a status update.
	// Because of that small possibility, we flag this declaration with resync=1 to make sure we do a DeclarativeManagement sync.
	if len(profilesToInsert) == 0 {
		return nil
	}
	var findRemoveProfilesArgs []any
	var foundInstall bool
	for _, p := range profilesToInsert {
		if p.OperationType == fleet.MDMOperationTypeInstall {
			findRemoveProfilesArgs = append(findRemoveProfilesArgs, p.HostUUID, p.Token)
			foundInstall = true
		}
	}
	if !foundInstall {
		return nil
	}
	findRemoveProfiles := fmt.Sprintf(`
		SELECT host_uuid, token
		FROM host_mdm_apple_declarations
		WHERE (host_uuid, token) IN (%s)
		AND operation_type = ?
		AND status = ?
		`, strings.TrimSuffix(strings.Repeat("(?,?),", len(findRemoveProfilesArgs)/2), ","))
	findRemoveProfilesArgs = append(findRemoveProfilesArgs, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending)
	type tokensToMark struct {
		HostUUID string `db:"host_uuid"`
		Token    string `db:"token"`
	}
	var tokensToMarkVerified []tokensToMark
	err := sqlx.SelectContext(ctx, tx, &tokensToMarkVerified, findRemoveProfiles, findRemoveProfilesArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "selecting tokens to mark verified")
	}

	if len(tokensToMarkVerified) > 0 {
		deleteRemoveProfiles := fmt.Sprintf(`
			DELETE FROM host_mdm_apple_declarations
			WHERE (host_uuid, token) IN (%s)
			AND operation_type = ?
			AND status = ?
			`, strings.TrimSuffix(strings.Repeat("(?,?),", len(tokensToMarkVerified)), ","))
		var removeProfilesArgs []any
		var markVerifiedArgs []any
		markVerifiedArgs = append(markVerifiedArgs, fleet.MDMDeliveryVerified)
		for _, t := range tokensToMarkVerified {
			removeProfilesArgs = append(removeProfilesArgs, t.HostUUID, t.Token)
			markVerifiedArgs = append(markVerifiedArgs, t.HostUUID, t.Token)
		}
		removeProfilesArgs = append(removeProfilesArgs, fleet.MDMOperationTypeRemove, fleet.MDMDeliveryPending)
		_, err = tx.ExecContext(ctx, deleteRemoveProfiles, removeProfilesArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk delete pending declarations")
		}
		markInstallProfilesVerified := fmt.Sprintf(`
			UPDATE host_mdm_apple_declarations
			SET status = ?, resync = 1
			WHERE (host_uuid, token) IN (%s)
			AND operation_type = ?
			`, strings.TrimSuffix(strings.Repeat("(?,?),", len(tokensToMarkVerified)), ","))
		markVerifiedArgs = append(markVerifiedArgs, fleet.MDMOperationTypeInstall)
		_, err = tx.ExecContext(ctx, markInstallProfilesVerified, markVerifiedArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set declarations mark verified")
		}
	}
	return nil
}

// mdmAppleGetHostsWithChangedDeclarationsDB returns a
// MDMAppleHostDeclaration item for each (host x declaration) pair that
// needs a status change, this includes declarations to install and
// declarations to be removed. Those can be differentiated by the
// OperationType field on each struct.
//
// Note (2024/12/24): This method returns some rows that DO NOT NEED TO BE UPDATED.
// We should optimize this method to only return the rows that need to be updated.
// Then we can eliminate the subsequent check for updates in the caller.
// The check for updates is needed to log the correct activity item -- whether declarations were updated or not.
func mdmAppleGetHostsWithChangedDeclarationsDB(ctx context.Context, tx sqlx.ExtContext) ([]*fleet.MDMAppleHostDeclaration, error) {
	stmt := fmt.Sprintf(`
        (
            SELECT
                ds.host_uuid,
                'install' as operation_type,
                ds.token,
				ds.secrets_updated_at,
                ds.declaration_uuid,
                ds.declaration_identifier,
                ds.declaration_name
            FROM
                %s
        )
        UNION ALL
        (
            SELECT
                hmae.host_uuid,
                'remove' as operation_type,
                hmae.token,
				hmae.secrets_updated_at,
                hmae.declaration_uuid,
                hmae.declaration_identifier,
                hmae.declaration_name
            FROM
                %s
        )
    `,
		generateEntitiesToInstallQuery("declaration"),
		generateEntitiesToRemoveQuery("declaration"),
	)

	var decls []*fleet.MDMAppleHostDeclaration
	if err := sqlx.SelectContext(ctx, tx, &decls, stmt, fleet.MDMOperationTypeRemove, fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeRemove); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "running sql statement")
	}
	return decls, nil
}

// MDMAppleStoreDDMStatusReport updates the status of the host's declarations.
func (ds *Datastore) MDMAppleStoreDDMStatusReport(ctx context.Context, hostUUID string, updates []*fleet.MDMAppleHostDeclaration) error {
	getHostDeclarationsStmt := `
    SELECT host_uuid, status, operation_type, HEX(token) as token, secrets_updated_at, declaration_uuid, declaration_identifier, declaration_name
    FROM host_mdm_apple_declarations
    WHERE host_uuid = ?
  `

	updateHostDeclarationsStmt := `
INSERT INTO host_mdm_apple_declarations
    (host_uuid, declaration_uuid, status, operation_type, detail, declaration_name, declaration_identifier, token, secrets_updated_at)
VALUES
  %s
ON DUPLICATE KEY UPDATE
  status = VALUES(status),
  operation_type = VALUES(operation_type),
  detail = VALUES(detail)
  `

	deletePendingRemovesStmt := `
  DELETE FROM host_mdm_apple_declarations
  WHERE host_uuid = ? AND operation_type = 'remove' AND (status = 'pending' OR status IS NULL)
  `

	var current []*fleet.MDMAppleHostDeclaration
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &current, getHostDeclarationsStmt, hostUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "getting current host declarations")
	}

	updatesByToken := make(map[string]*fleet.MDMAppleHostDeclaration, len(updates))
	for _, u := range updates {
		updatesByToken[u.Token] = u
	}

	var args []any
	var insertVals strings.Builder
	for _, c := range current {
		// Skip updates for 'remove' operations because it is possible that IT admin removed a profile and then re-added it.
		// Pending removes are cleaned up after we update status of installs.
		if u, ok := updatesByToken[c.Token]; ok && u.OperationType != fleet.MDMOperationTypeRemove {
			insertVals.WriteString("(?, ?, ?, ?, ?, ?, ?, UNHEX(?), ?),")
			args = append(args, hostUUID, c.DeclarationUUID, u.Status, u.OperationType, u.Detail, c.Identifier, c.Name, c.Token,
				c.SecretsUpdatedAt)
		}
	}

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if len(args) != 0 {
			stmt := fmt.Sprintf(updateHostDeclarationsStmt, strings.TrimSuffix(insertVals.String(), ","))
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "updating existing declarations")
			}
		}

		if _, err := tx.ExecContext(ctx, deletePendingRemovesStmt, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting pending removals")
		}

		return nil
	})

	return ctxerr.Wrap(ctx, err, "updating host declarations")
}

func (ds *Datastore) MDMAppleSetPendingDeclarationsAs(ctx context.Context, hostUUID string, status *fleet.MDMDeliveryStatus, detail string) error {
	stmt := `
  UPDATE host_mdm_apple_declarations
  SET
    status = ?,
    detail = ?
  WHERE
    operation_type = ?
    AND status = ?
    AND host_uuid = ?
  `

	_, err := ds.writer(ctx).ExecContext(
		ctx, stmt,
		// SET ...
		status, detail,
		// WHERE ...
		fleet.MDMOperationTypeInstall, fleet.MDMDeliveryPending, hostUUID,
	)
	return ctxerr.Wrap(ctx, err, "updating host declaration status to verifying")
}

func (ds *Datastore) MDMAppleSetRemoveDeclarationsAsPending(ctx context.Context, hostUUID string, declarationUUIDs []string) error {
	stmt := `
  UPDATE host_mdm_apple_declarations
  SET
    status = ?
  WHERE
    host_uuid = ?
    AND declaration_uuid IN (?)
    AND operation_type = ?
    AND status IS NULL
  `

	stmt, args, err := sqlx.In(stmt, fleet.MDMDeliveryPending, hostUUID, declarationUUIDs, fleet.MDMOperationTypeRemove)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building IN clause")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
	return ctxerr.Wrap(ctx, err, "updating host declaration status to pending")
}

func (ds *Datastore) InsertMDMAppleDDMRequest(ctx context.Context, hostUUID, messageType string, rawJSON json.RawMessage) error {
	const stmt = `
INSERT INTO
    mdm_apple_declarative_requests (
        enrollment_id,
        message_type,
        raw_json
    )
VALUES
    (?, ?, ?)
`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostUUID, messageType, rawJSON); err != nil {
		return ctxerr.Wrap(ctx, err, "writing apple declarative request to db")
	}

	return nil
}

func encrypt(plainText []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

func decrypt(encrypted []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()

	// Extract the nonce from the encrypted data
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return decrypted, nil
}

func (ds *Datastore) InsertMDMConfigAssets(ctx context.Context, assets []fleet.MDMConfigAsset, tx sqlx.ExtContext) error {
	insertFunc := func(tx sqlx.ExtContext) error {
		if err := insertMDMConfigAssets(ctx, tx, assets, ds.serverPrivateKey); err != nil {
			return ctxerr.Wrap(ctx, err, "insert mdm config assets")
		}
		return nil
	}
	if tx != nil {
		return insertFunc(tx)
	}
	return ds.withRetryTxx(ctx, insertFunc)
}

func (ds *Datastore) GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName,
	queryerContext sqlx.QueryerContext,
) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	if len(assetNames) == 0 {
		return nil, nil
	}

	stmt := `
SELECT
    name, value
FROM
   mdm_config_assets
WHERE
    name IN (?)
	AND deletion_uuid = ''
	`

	stmt, args, err := sqlx.In(stmt, assetNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building sqlx.In statement")
	}

	var res []fleet.MDMConfigAsset
	if queryerContext == nil {
		queryerContext = ds.reader(ctx)
	}
	if err := sqlx.SelectContext(ctx, queryerContext, &res, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get mdm config assets by name")
	}

	if len(res) == 0 {
		return nil, notFound("MDMConfigAsset")
	}

	assetMap := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset, len(res))
	for _, asset := range res {
		decryptedVal, err := decrypt(asset.Value, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decrypting mdm config asset %s", asset.Name)
		}

		assetMap[asset.Name] = fleet.MDMConfigAsset{Name: asset.Name, Value: decryptedVal}
	}

	if len(res) < len(assetNames) {
		return assetMap, ErrPartialResult
	}

	return assetMap, nil
}

func (ds *Datastore) GetAllMDMConfigAssetsHashes(ctx context.Context, assetNames []fleet.MDMAssetName) (map[fleet.MDMAssetName]string, error) {
	if len(assetNames) == 0 {
		return nil, nil
	}

	stmt := `
SELECT name, HEX(md5_checksum) as md5_checksum
FROM mdm_config_assets
WHERE name IN (?) AND deletion_uuid = ''`

	stmt, args, err := sqlx.In(stmt, assetNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building sqlx.In statement")
	}

	var res []fleet.MDMConfigAsset
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get mdm config checksums by name")
	}

	if len(res) == 0 {
		return nil, notFound("MDMConfigAsset")
	}

	assetMap := make(map[fleet.MDMAssetName]string, len(res))
	for _, asset := range res {
		assetMap[asset.Name] = asset.MD5Checksum
	}

	if len(res) < len(assetNames) {
		return assetMap, ErrPartialResult
	}

	return assetMap, nil
}

func (ds *Datastore) DeleteMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := softDeleteMDMConfigAssetsByName(ctx, tx, assetNames); err != nil {
			return ctxerr.Wrap(ctx, err, "delete mdm config assets by name")
		}

		return nil
	})
}

func (ds *Datastore) HardDeleteMDMConfigAsset(ctx context.Context, assetName fleet.MDMAssetName) error {
	stmt := `
DELETE FROM mdm_config_assets
WHERE name = ?`
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, assetName)
	// ctxerr.Wrap returns nil if err is nil
	return ctxerr.Wrap(ctx, err, "hard delete mdm config asset")
}

func softDeleteMDMConfigAssetsByName(ctx context.Context, tx sqlx.ExtContext, assetNames []fleet.MDMAssetName) error {
	stmt := `
UPDATE
    mdm_config_assets
SET
    deleted_at = CURRENT_TIMESTAMP(),
	deletion_uuid = ?
WHERE
    name IN (?) AND deletion_uuid = ''
	`

	deletionUUID := uuid.New().String()

	stmt, args, err := sqlx.In(stmt, deletionUUID, assetNames)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In softDeleteMDMConfigAssetsByName")
	}

	_, err = tx.ExecContext(ctx, stmt, args...)
	return ctxerr.Wrap(ctx, err, "deleting mdm config assets")
}

func insertMDMConfigAssets(ctx context.Context, tx sqlx.ExtContext, assets []fleet.MDMConfigAsset, privateKey string) error {
	stmt := `
INSERT INTO mdm_config_assets
  (name, value, md5_checksum)
VALUES
  %s`

	var args []any
	var insertVals strings.Builder

	for _, a := range assets {
		encryptedVal, err := encrypt(a.Value, privateKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, fmt.Sprintf("encrypting mdm config asset %s", a.Name))
		}

		hexChecksum := md5ChecksumBytes(encryptedVal)
		insertVals.WriteString(`(?, ?, UNHEX(?)),`)
		args = append(args, a.Name, encryptedVal, hexChecksum)
	}

	stmt = fmt.Sprintf(stmt, strings.TrimSuffix(insertVals.String(), ","))

	_, err := tx.ExecContext(ctx, stmt, args...)

	return ctxerr.Wrap(ctx, err, "writing mdm config assets to db")
}

func (ds *Datastore) ReplaceMDMConfigAssets(ctx context.Context, assets []fleet.MDMConfigAsset, tx sqlx.ExtContext) error {
	replaceFunc := func(tx sqlx.ExtContext) error {
		var names []fleet.MDMAssetName
		for _, a := range assets {
			names = append(names, a.Name)
		}

		if err := softDeleteMDMConfigAssetsByName(ctx, tx, names); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert mdm config assets soft delete")
		}

		if err := insertMDMConfigAssets(ctx, tx, assets, ds.serverPrivateKey); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert mdm config assets insert")
		}
		return nil
	}
	if tx != nil {
		return replaceFunc(tx)
	}
	return ds.withRetryTxx(ctx, replaceFunc)
}

// ListIOSAndIPadOSToRefetch returns the UUIDs of iPhones/iPads that should be refetched
// (their details haven't been updated in the given `interval`).
func (ds *Datastore) ListIOSAndIPadOSToRefetch(ctx context.Context, interval time.Duration) (devices []fleet.AppleDevicesToRefetch,
	err error,
) {
	hostsStmt := `
SELECT h.id as host_id, h.uuid as uuid, JSON_ARRAYAGG(hmc.command_type) as commands_already_sent FROM hosts h
INNER JOIN host_mdm hmdm ON hmdm.host_id = h.id
LEFT JOIN host_mdm_commands hmc ON hmc.host_id = h.id
WHERE (h.platform = 'ios' OR h.platform = 'ipados')
AND TRIM(h.uuid) != ''
AND TIMESTAMPDIFF(SECOND, h.detail_updated_at, NOW()) > ?
GROUP BY h.id`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &devices, hostsStmt, interval.Seconds()); err != nil {
		return nil, err
	}

	return devices, nil
}

func (ds *Datastore) GetEnrollmentIDsWithPendingMDMAppleCommands(ctx context.Context) (uuids []string, err error) {
	const stmt = `
SELECT DISTINCT neq.id
FROM nano_enrollment_queue neq
INNER JOIN nano_enrollments ne ON ne.id = neq.id
LEFT JOIN nano_command_results ncr ON ncr.command_uuid = neq.command_uuid AND ncr.id = neq.id
WHERE neq.active = 1 AND ne.enabled=1 AND ncr.status IS NULL
AND neq.created_at >= NOW() - INTERVAL 7 DAY
LIMIT 500
`

	var deviceUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &deviceUUIDs, stmt); err != nil {
		return nil, err
	}

	return deviceUUIDs, nil
}

func (ds *Datastore) GetABMTokenByOrgName(ctx context.Context, orgName string) (*fleet.ABMToken, error) {
	tok, err := ds.getABMToken(ctx, 0, orgName)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get ABM token by org name")
	}

	return tok, nil
}

func (ds *Datastore) SaveABMToken(ctx context.Context, tok *fleet.ABMToken) error {
	const stmt = `
UPDATE
	abm_tokens
SET
	organization_name = ?,
	apple_id = ?,
	terms_expired = ?,
	renew_at = ?,
	token = ?,
	macos_default_team_id = ?,
	ios_default_team_id = ?,
	ipados_default_team_id = ?
WHERE
	id = ?`

	doubleEncTok, err := encrypt(tok.EncryptedToken, ds.serverPrivateKey)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encrypt with datastore.serverPrivateKey")
	}

	_, err = ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		tok.OrganizationName,
		tok.AppleID,
		tok.TermsExpired,
		tok.RenewAt.UTC(),
		doubleEncTok,
		tok.MacOSDefaultTeamID,
		tok.IOSDefaultTeamID,
		tok.IPadOSDefaultTeamID,
		tok.ID)
	return ctxerr.Wrap(ctx, err, "updating abm_token")
}

func (ds *Datastore) InsertABMToken(ctx context.Context, tok *fleet.ABMToken) (*fleet.ABMToken, error) {
	const stmt = `
INSERT INTO
	abm_tokens
	(organization_name, apple_id, terms_expired, renew_at, token, macos_default_team_id, ios_default_team_id, ipados_default_team_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`
	doubleEncTok, err := encrypt(tok.EncryptedToken, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "encrypt abm_token with datastore.serverPrivateKey")
	}

	res, err := ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		tok.OrganizationName,
		tok.AppleID,
		tok.TermsExpired,
		tok.RenewAt,
		doubleEncTok,
		tok.MacOSDefaultTeamID,
		tok.IOSDefaultTeamID,
		tok.IPadOSDefaultTeamID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting abm_token")
	}

	tokenID, _ := res.LastInsertId()

	tok.ID = uint(tokenID) //nolint:gosec // dismiss G115

	cfg, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	url, err := apple_mdm.ResolveAppleMDMURL(cfg.MDMUrl())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting ABM token MDM server url")
	}

	tok.MDMServerURL = url

	return tok, nil
}

func (ds *Datastore) ListABMTokens(ctx context.Context) ([]*fleet.ABMToken, error) {
	stmt := `
SELECT
	abt.id,
	abt.organization_name,
	abt.apple_id,
	abt.terms_expired,
	abt.renew_at,
	abt.token,
	abt.macos_default_team_id,
	abt.ios_default_team_id,
	abt.ipados_default_team_id,
	COALESCE(t1.name, :no_team) as macos_team,
	COALESCE(t2.name, :no_team) as ios_team,
	COALESCE(t3.name, :no_team) as ipados_team
FROM
	abm_tokens abt
LEFT OUTER JOIN
	teams t1 ON t1.id = abt.macos_default_team_id
LEFT OUTER JOIN
	teams t2 ON t2.id = abt.ios_default_team_id
LEFT OUTER JOIN
	teams t3 ON t3.id = abt.ipados_default_team_id

	`

	stmt, args, err := sqlx.Named(stmt, map[string]any{"no_team": fleet.TeamNameNoTeam})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build list ABM tokens query from named args")
	}

	var tokens []*fleet.ABMToken
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &tokens, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list ABM tokens")
	}

	cfg, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	url, err := apple_mdm.ResolveAppleMDMURL(cfg.MDMUrl())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting ABM token MDM server url")
	}

	for _, tok := range tokens {
		tok.MDMServerURL = url

		// Promote DB fields into respective objects
		var macOSTeamID, iOSTeamID, iPadIOSTeamID uint
		if tok.MacOSDefaultTeamID != nil {
			macOSTeamID = *tok.MacOSDefaultTeamID
		}
		if tok.IOSDefaultTeamID != nil {
			iOSTeamID = *tok.IOSDefaultTeamID
		}
		if tok.IPadOSDefaultTeamID != nil {
			iPadIOSTeamID = *tok.IPadOSDefaultTeamID
		}

		tok.MacOSTeam = fleet.ABMTokenTeam{Name: tok.MacOSTeamName, ID: macOSTeamID}
		tok.IOSTeam = fleet.ABMTokenTeam{Name: tok.IOSTeamName, ID: iOSTeamID}
		tok.IPadOSTeam = fleet.ABMTokenTeam{Name: tok.IPadOSTeamName, ID: iPadIOSTeamID}

		// decrypt the token with the serverPrivateKey, the resulting value will be
		// the token still encrypted, but just with the ABM cert and key (it is that
		// encrypted value that is stored with another layer of encryption with the
		// serverPrivateKey).
		decrypted, err := decrypt(tok.EncryptedToken, ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decrypting abm token with datastore.serverPrivateKey")
		}
		tok.EncryptedToken = decrypted
	}

	return tokens, nil
}

func (ds *Datastore) DeleteABMToken(ctx context.Context, tokenID uint) error {
	const stmt = `
DELETE FROM
	abm_tokens
WHERE ID = ?
		`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, tokenID)

	return ctxerr.Wrap(ctx, err, "deleting ABM token")
}

func (ds *Datastore) GetABMTokenByID(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
	tok, err := ds.getABMToken(ctx, tokenID, "")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get ABM token by id")
	}

	return tok, nil
}

func (ds *Datastore) getABMToken(ctx context.Context, tokenID uint, orgName string) (*fleet.ABMToken, error) {
	stmt := `
SELECT
	abt.id,
	abt.organization_name,
	abt.apple_id,
	abt.terms_expired,
	abt.renew_at,
	abt.token,
	abt.macos_default_team_id,
	abt.ios_default_team_id,
	abt.ipados_default_team_id,
	COALESCE(t1.name, :no_team) as macos_team,
	COALESCE(t2.name, :no_team) as ios_team,
	COALESCE(t3.name, :no_team) as ipados_team
FROM
	abm_tokens abt
LEFT OUTER JOIN
	teams t1 ON t1.id = abt.macos_default_team_id
LEFT OUTER JOIN
	teams t2 ON t2.id = abt.ios_default_team_id
LEFT OUTER JOIN
	teams t3 ON t3.id = abt.ipados_default_team_id
%s
	`

	stmt, args, err := sqlx.Named(stmt, map[string]any{"no_team": fleet.TeamNameNoTeam})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build list ABM tokens query from named args")
	}

	var ident any = orgName
	clause := "WHERE abt.organization_name = ?"
	if tokenID != 0 {
		clause = "WHERE abt.id = ?"
		ident = tokenID
	}

	stmt = fmt.Sprintf(stmt, clause)

	args = append(args, ident)

	var tok fleet.ABMToken
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &tok, stmt, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("ABMToken"))
		}

		return nil, ctxerr.Wrap(ctx, err, "get ABM token")
	}

	// decrypt the token with the serverPrivateKey, the resulting value will be
	// the token still encrypted, but just with the ABM cert and key (it is that
	// encrypted value that is stored with another layer of encryption with the
	// serverPrivateKey).
	decrypted, err := decrypt(tok.EncryptedToken, ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "decrypting abm token with datastore.serverPrivateKey")
	}
	tok.EncryptedToken = decrypted

	cfg, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	url, err := apple_mdm.ResolveAppleMDMURL(cfg.MDMUrl())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting ABM token MDM server url")
	}

	tok.MDMServerURL = url

	// Promote DB fields into respective objects
	var macOSTeamID, iOSTeamID, iPadIOSTeamID uint
	if tok.MacOSDefaultTeamID != nil {
		macOSTeamID = *tok.MacOSDefaultTeamID
	}
	if tok.IOSDefaultTeamID != nil {
		iOSTeamID = *tok.IOSDefaultTeamID
	}
	if tok.IPadOSDefaultTeamID != nil {
		iPadIOSTeamID = *tok.IPadOSDefaultTeamID
	}

	tok.MacOSTeam = fleet.ABMTokenTeam{Name: tok.MacOSTeamName, ID: macOSTeamID}
	tok.IOSTeam = fleet.ABMTokenTeam{Name: tok.IOSTeamName, ID: iOSTeamID}
	tok.IPadOSTeam = fleet.ABMTokenTeam{Name: tok.IPadOSTeamName, ID: iPadIOSTeamID}

	return &tok, nil
}

func (ds *Datastore) GetABMTokenCount(ctx context.Context) (int, error) {
	var count int
	const countStmt = `SELECT COUNT(*) FROM abm_tokens`

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, countStmt); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting existing ABM tokens")
	}

	return count, nil
}

func (ds *Datastore) SetABMTokenTermsExpiredForOrgName(ctx context.Context, orgName string, expired bool) (wasSet bool, err error) {
	const stmt = `UPDATE abm_tokens SET terms_expired = ? WHERE organization_name = ? AND terms_expired != ?`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, expired, orgName, expired)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "update abm_tokens terms_expired")
	}
	affRows, _ := res.RowsAffected()

	if affRows > 0 {
		// if it did update the row, then the previous value was the opposite of
		// expired
		wasSet = !expired
	} else {
		// if it did not update any row, then the previous value was the same
		wasSet = expired
	}
	return wasSet, nil
}

func (ds *Datastore) CountABMTokensWithTermsExpired(ctx context.Context) (int, error) {
	// The expectation is that abm_tokens will have few rows (we don't even
	// support pagination on the "list ABM tokens" endpoint), so this query
	// should be very fast even without index on terms_expired.
	const stmt = `SELECT COUNT(*) FROM abm_tokens WHERE terms_expired = 1`

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, stmt); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count ABM tokens with terms expired")
	}
	return count, nil
}

func (ds *Datastore) GetABMTokenOrgNamesAssociatedWithTeam(ctx context.Context, teamID *uint) ([]string, error) {
	stmt := `
SELECT DISTINCT
	abmt.organization_name
FROM
	abm_tokens abmt
	JOIN host_dep_assignments hda ON hda.abm_token_id = abmt.id
	JOIN hosts h ON hda.host_id = h.id
WHERE
	%s
UNION
SELECT DISTINCT
	abmt.organization_name
FROM
	abm_tokens abmt
WHERE
	%s
`
	var args []any
	teamFilter := `h.team_id IS NULL`
	abmtFilter := `abmt.macos_default_team_id IS NULL OR abmt.ios_default_team_id IS NULL OR abmt.ipados_default_team_id IS NULL`
	if teamID != nil {
		teamFilter = `h.team_id = ?`
		abmtFilter = `abmt.macos_default_team_id = ? OR abmt.ios_default_team_id = ? OR abmt.ipados_default_team_id = ?`
		args = append(args, *teamID, *teamID, *teamID, *teamID)
	}

	stmt = fmt.Sprintf(stmt, teamFilter, abmtFilter)

	var orgNames []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &orgNames, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting org names for team from db")
	}

	return orgNames, nil
}

func (ds *Datastore) AddHostMDMCommands(ctx context.Context, commands []fleet.HostMDMCommand) error {
	const baseStmt = `
		INSERT INTO host_mdm_commands (host_id, command_type)
		VALUES %s
		ON DUPLICATE KEY UPDATE
		command_type = VALUES(command_type)`

	for i := 0; i < len(commands); i += addHostMDMCommandsBatchSize {
		start := i
		end := i + hostIssuesInsertBatchSize
		if end > len(commands) {
			end = len(commands)
		}
		totalToProcess := end - start
		const numberOfArgsPerInsert = 2 // number of ? in each VALUES clause
		values := strings.TrimSuffix(
			strings.Repeat("(?,?),", totalToProcess), ",",
		)
		stmt := fmt.Sprintf(baseStmt, values)
		args := make([]interface{}, 0, totalToProcess*numberOfArgsPerInsert)
		for j := start; j < end; j++ {
			item := commands[j]
			args = append(
				args, item.HostID, item.CommandType,
			)
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert into host_mdm_commands")
		}
	}

	return nil
}

func (ds *Datastore) GetHostMDMCommands(ctx context.Context, hostID uint) (commands []fleet.HostMDMCommand, err error) {
	const stmt = `SELECT host_id, command_type FROM host_mdm_commands WHERE host_id = ?`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &commands, stmt, hostID); err != nil {
		return nil, err
	}
	return commands, nil
}

func (ds *Datastore) RemoveHostMDMCommand(ctx context.Context, command fleet.HostMDMCommand) error {
	const stmt = `
		DELETE FROM host_mdm_commands
		WHERE host_id = ? AND command_type = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, command.HostID, command.CommandType); err != nil {
		return ctxerr.Wrap(ctx, err, "delete from host_mdm_commands")
	}
	return nil
}

func (ds *Datastore) CleanupHostMDMCommands(ctx context.Context) error {
	// Delete commands that don't have a corresponding host or have been sent over 1 day ago.
	// We are using 1 day instead of 7 days in case MDM commands fail to be sent or fail to process. They can be resent the next day.
	const stmt = `
		DELETE hmc FROM host_mdm_commands AS hmc
		LEFT JOIN hosts h ON h.id = hmc.host_id
		WHERE h.id IS NULL OR hmc.updated_at < NOW() - INTERVAL 1 DAY`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete from host_mdm_commands")
	}
	return nil
}

func (ds *Datastore) CleanupHostMDMAppleProfiles(ctx context.Context) error {
	// Delete pending commands that don't have a corresponding entry in nano_enrollment_queue.
	// This could occur when the host re-enrolls in MDM with outstanding Pending commands.
	// This could also occur due to errors (i.e., large server/DB load) or server being stopped while processing the profiles.
	// After the entry is deleted, the mdm_apple_profile_manager job will try to requeue the profile.
	stmt := fmt.Sprintf(`
		DELETE hmap FROM host_mdm_apple_profiles AS hmap
        -- ANTIJOIN: Delete rows that don't have a corresponding entry in nano_enrollment_queue
		-- Note that a given host may have multiple nano_enrollments(device and user) and thus we
		-- need to account for the use of either of them in the join
		LEFT JOIN (nano_enrollment_queue neq INNER JOIN nano_enrollments ne ON neq.id = ne.id AND ne.enabled = 1)
		ON ne.device_id = hmap.host_uuid AND hmap.command_uuid = neq.command_uuid AND neq.active = 1
		WHERE neq.id IS NULL AND (hmap.status IS NULL OR hmap.status = '%s') AND hmap.updated_at < NOW() - INTERVAL 1 HOUR`,
		fleet.MDMDeliveryPending)
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "delete from host_mdm_apple_profiles")
	}
	return nil
}

func (ds *Datastore) GetMDMAppleOSUpdatesSettingsByHostSerial(ctx context.Context, serial string) (*fleet.AppleOSUpdateSettings, error) {
	stmt := `
SELECT
	team_id, platform
FROM
	hosts h
JOIN
	host_dep_assignments hdep ON h.id = host_id
WHERE
	hardware_serial = ? AND deleted_at IS NULL
LIMIT 1`

	var dest struct {
		TeamID   *uint  `db:"team_id"`
		Platform string `db:"platform"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, serial); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting team id for host")
	}

	var settings fleet.AppleOSUpdateSettings
	if dest.TeamID == nil {
		// use the global settings
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting app config for os update settings")
		}
		switch dest.Platform {
		case "ios":
			settings = ac.MDM.IOSUpdates
		case "ipados":
			settings = ac.MDM.IPadOSUpdates
		case "darwin":
			settings = ac.MDM.MacOSUpdates
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unsupported platform %s", dest.Platform))
		}
	} else {
		// use the team settings
		tm, err := ds.TeamWithoutExtras(ctx, *dest.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting team os update settings")
		}
		switch dest.Platform {
		case "ios":
			settings = tm.Config.MDM.IOSUpdates
		case "ipados":
			settings = tm.Config.MDM.IPadOSUpdates
		case "darwin":
			settings = tm.Config.MDM.MacOSUpdates
		default:
			return nil, ctxerr.New(ctx, fmt.Sprintf("unsupported platform %s", dest.Platform))
		}
	}

	return &settings, nil
}

func (ds *Datastore) BulkUpsertMDMManagedCertificates(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
	if len(payload) == 0 {
		return nil
	}

	executeUpsertBatch := func(valuePart string, args []any) error {
		stmt := fmt.Sprintf(`
	    INSERT INTO host_mdm_managed_certificates (
              host_uuid,
              profile_uuid,
              challenge_retrieved_at,
			  not_valid_before,
	          not_valid_after,
			  type,
			  ca_name,
			  serial
            )
            VALUES %s
            ON DUPLICATE KEY UPDATE
              challenge_retrieved_at = VALUES(challenge_retrieved_at),
			  not_valid_before = VALUES(not_valid_before),
			  not_valid_after = VALUES(not_valid_after),
			  type = VALUES(type),
			  ca_name = VALUES(ca_name),
			  serial = VALUES(serial)`,
			strings.TrimSuffix(valuePart, ","),
		)

		_, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
		return err
	}

	generateValueArgs := func(p *fleet.MDMManagedCertificate) (string, []any) {
		valuePart := "(?, ?, ?, ?, ?, ?, ?, ?),"
		args := []any{p.HostUUID, p.ProfileUUID, p.ChallengeRetrievedAt, p.NotValidBefore, p.NotValidAfter, p.Type, p.CAName, p.Serial}
		return valuePart, args
	}

	const defaultBatchSize = 1000
	batchSize := defaultBatchSize
	if ds.testUpsertMDMDesiredProfilesBatchSize > 0 {
		batchSize = ds.testUpsertMDMDesiredProfilesBatchSize
	}

	if err := batchProcessDB(payload, batchSize, generateValueArgs, executeUpsertBatch); err != nil {
		return err
	}

	return nil
}

func (ds *Datastore) ListHostMDMManagedCertificates(ctx context.Context, hostUUID string) ([]*fleet.MDMManagedCertificate, error) {
	hostCertsToRenew := []*fleet.MDMManagedCertificate{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostCertsToRenew, `
	SELECT profile_uuid, host_uuid, challenge_retrieved_at, not_valid_before, not_valid_after, type, ca_name, serial
	FROM host_mdm_managed_certificates
	WHERE host_uuid = ?
	`, hostUUID)
	return hostCertsToRenew, ctxerr.Wrap(ctx, err, "get mdm managed certificates for host")
}

// ClearMDMUpcomingActivitiesDB clears the upcoming activities of the host that
// require MDM to be processed, for when MDM is turned off for the host (or
// when it turns on again, e.g. after removing the enrollment profile - it may
// not necessarily report as "turned off" in that scenario).
func (ds *Datastore) ClearMDMUpcomingActivitiesDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	// NOTE: must be updated if new activity types that require MDM are added to
	// the upcoming activities.
	const deleteUpcomingMDMActivities = `
DELETE FROM upcoming_activities
	USING upcoming_activities
		JOIN hosts h ON upcoming_activities.host_id = h.id
WHERE
	h.uuid = ? AND
	upcoming_activities.activity_type IN ('vpp_app_install')
`
	_, err := tx.ExecContext(ctx, deleteUpcomingMDMActivities, hostUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clearing upcoming activities")
	}

	// get the host ID for that uuid
	var hostID uint
	if err := sqlx.GetContext(ctx, tx, &hostID, "SELECT id FROM hosts WHERE uuid = ?", hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "load host ID by UUID")
	}

	// ensure that if there's an activity waiting to be activated, it does get
	// activated
	if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
		return ctxerr.Wrap(ctx, err, "activate next upcoming activity")
	}
	return nil
}

func (ds *Datastore) GetMDMAppleEnrolledDeviceDeletedFromFleet(ctx context.Context, hostUUID string) (*fleet.MDMAppleEnrolledDeviceInfo, error) {
	const stmt = `
SELECT
	d.id,
	COALESCE(d.serial_number, '') as serial_number,
	d.authenticate,
	d.platform,
	d.enroll_team_id
FROM
	nano_devices d
	JOIN nano_enrollments e ON d.id = e.device_id
	LEFT OUTER JOIN hosts h ON h.uuid = d.id
WHERE
	e.type = 'Device' AND
	e.enabled = 1 AND
	d.id = ? AND
	h.id IS NULL
`

	var res fleet.MDMAppleEnrolledDeviceInfo
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, stmt, hostUUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("NanoDevice").WithName(hostUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get mdm apple enrolled device info")
	}
	return &res, nil
}

func (ds *Datastore) ListMDMAppleEnrolledIPhoneIpadDeletedFromFleet(ctx context.Context, limit int) ([]string, error) {
	const stmt = `
SELECT
	d.id
FROM
	nano_devices d
	JOIN nano_enrollments e ON d.id = e.device_id
	LEFT OUTER JOIN hosts h ON h.uuid = d.id
WHERE
	e.type = 'Device' AND
	e.enabled = 1 AND
	d.platform IN ('ios', 'ipados') AND
	h.id IS NULL
LIMIT ?
`

	var res []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, stmt, limit); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list mdm apple enrolled but deleted iDevices")
	}
	return res, nil
}

func (ds *Datastore) GetNanoMDMEnrollmentTimes(ctx context.Context, hostUUID string) (*time.Time, *time.Time, error) {
	res := []struct {
		LastMDMEnrollmentTime *time.Time `db:"authenticate_at"`
		LastMDMSeenTime       *time.Time `db:"last_seen_at"`
	}{}
	// We are specifically only looking at the singular device enrollment row and not the
	// potentially many user enrollment rows that will exist for a given device. The device
	// enrollment row is the only one that gets regularly updated with the last seen time. Along
	// those same lines authenticate_at gets updated only at the authenticate step during the
	// enroll process and as such is a good indicator of the last enrollment or reenrollment.
	query := `
	SELECT nd.authenticate_at, ne.last_seen_at
	FROM nano_devices nd
	  INNER JOIN nano_enrollments ne ON ne.id = nd.id
	WHERE ne.type = 'Device' AND nd.id = ?`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &res, query, hostUUID)

	if err == sql.ErrNoRows || len(res) == 0 {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get mdm enrollment times")
	}
	return res[0].LastMDMEnrollmentTime, res[0].LastMDMSeenTime, nil
}

func (ds *Datastore) ReconcileMDMAppleEnrollRef(ctx context.Context, enrollRef string, machineInfo *fleet.MDMAppleMachineInfo) (string, error) {
	if machineInfo == nil {
		level.Info(ds.logger).Log("msg", "reconcile mdm apple enroll ref: machine info is nil")
		return "", ctxerr.New(ctx, "machine info is nil")
	}

	var result string
	// TODO: maybe we don't need a transaction here?
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := associateHostMDMIdPAccountDB(ctx, tx, machineInfo.UDID, enrollRef); err != nil {
			return ctxerr.Wrap(ctx, err, "associate host mdm idp account")
		}
		legacyRef, err := getMDMAppleLegacyEnrollRefDB(ctx, tx, ds.logger, machineInfo.UDID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get mdm apple legacy enroll ref")
		}
		result = legacyRef
		return nil
	})

	// TODO: when should we delete from host_mdm_idp_accounts?

	return result, err
}

func associateHostMDMIdPAccountDB(ctx context.Context, tx sqlx.ExtContext, hostUUID, acctUUID string) error {
	const stmt = `
INSERT INTO host_mdm_idp_accounts (host_uuid, account_uuid)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
	account_uuid = VALUES(account_uuid)`

	_, err := tx.ExecContext(ctx, stmt, hostUUID, acctUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "associate host mdm idp account")
	}
	return nil
}

func getMDMAppleLegacyEnrollRefDB(ctx context.Context, tx sqlx.ExtContext, logger log.Logger, hostUUID string) (string, error) {
	const stmt = `SELECT enroll_ref FROM legacy_host_mdm_enroll_refs WHERE host_uuid = ?`

	var enrollRefs []string
	if err := sqlx.SelectContext(ctx, tx, &enrollRefs, stmt, hostUUID); err != nil {
		return "", ctxerr.Wrap(ctx, err, "get mdm apple legacy enroll ref")
	}

	var result string
	if len(enrollRefs) > 0 {
		// if we have more than one ref, we'll just use the first one
		result = enrollRefs[0]
	}
	if len(enrollRefs) > 1 {
		// this should not happen, but if it does we want to know about it
		level.Info(logger).Log("msg", "host uuid has multiple legacy enroll refs", "host_uuid", hostUUID, "enroll_refs", fmt.Sprintf("%+v", enrollRefs))
	}

	return result, nil
}

func reconcileHostEmailsFromMdmIdpAccountsDB(ctx context.Context, tx sqlx.ExtContext, logger log.Logger, hostID uint) (*fleet.MDMIdPAccount, error) {
	idp, err := getMDMIdPAccountByHostID(ctx, tx, logger, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host mdm idp account email")
	}

	var hostEmails []fleet.HostDeviceMapping
	selectStmt := `SELECT id, host_id, email, source FROM host_emails WHERE host_id = ? AND source = ?`
	if err := sqlx.SelectContext(ctx, tx, &hostEmails, selectStmt, hostID, fleet.DeviceMappingMDMIdpAccounts); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host_emails")
	}

	// TODO: discuss email vs. username with Victor
	var idpEmail, idpAccountUUID string
	if idp != nil {
		idpEmail = idp.Email
		idpAccountUUID = idp.UUID
	}

	// if we don't have idp info, we can just delete any prior host mdm idp account emails
	if idpEmail == "" {
		if len(hostEmails) == 0 {
			// nothing to do
			level.Info(logger).Log("msg", "reconcile host emails: no mdm idp account and no host emails", "host_id", hostID, "account_uuid", idpAccountUUID)
			return nil, nil
		}
		// delete any prior host mdm idp account emails
		delStmt := "DELETE FROM host_emails WHERE host_id = ? AND source = ?"
		if _, err := tx.ExecContext(ctx, delStmt, hostID, fleet.DeviceMappingMDMIdpAccounts); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete host_emails")
		}
		return idp, nil
	}

	// analyze existing host emails to see if we have a match; we also want to handle potential
	// duplicates because we don't have good constraints on the host_emails table
	hits, misses := []fleet.HostDeviceMapping{}, []fleet.HostDeviceMapping{}
	for _, he := range hostEmails {
		if he.Email == idp.Email {
			hits = append(hits, he)
		} else {
			misses = append(misses, he)
		}
	}

	idsToDelete := []uint{}
	if len(misses) > 0 {
		// log the emails we'll be deleting
		msg := "reconcile host emails: deleting emails"
		for _, m := range misses {
			idsToDelete = append(idsToDelete, m.ID)
			msg += fmt.Sprintf(" %s", m.Email)
		}
		level.Info(logger).Log("msg", msg, "host_id", hostID)
	}

	var idToUpdate uint
	if len(hits) > 0 {
		// if we have more than one hit, we'll just update the first one
		idToUpdate = hits[0].ID
	}
	if len(hits) > 1 {
		// this should not happen, but if it does we want to know about it
		level.Info(logger).Log("msg", "reconcile host emails: found duplicate mdm idp account host email", "host_id", hostID, "email", idp.Email, "account_uuid", idp.UUID)
		for _, h := range hits[1:] {
			idsToDelete = append(idsToDelete, h.ID) // we'll delete all but the first
		}
	}

	if len(idsToDelete) > 0 {
		// perform the delete
		stmt, args, err := sqlx.In("DELETE FROM host_emails WHERE id IN (?)", idsToDelete)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "prepare delete host_emails arguments")
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete host_emails")
		}
	}

	if idToUpdate != 0 {
		// perform the update
		level.Info(logger).Log("msg", "reconcile host emails: update host mdm idp account email", "host_id", hostID, "email", idpEmail)
		updateStmt := "UPDATE host_emails SET email = ? WHERE id = ?"
		if _, err := tx.ExecContext(ctx, updateStmt, idpEmail, idToUpdate); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update host_emails")
		}
		return idp, nil
	}

	// insert new email
	level.Info(logger).Log("msg", "reconcile host emails: insert host mdm idp account email", "host_id", hostID, "email", idpEmail, "account_uuid", idp.UUID)
	insStmt := "INSERT INTO host_emails (email, host_id, source) VALUES (?, ?, ?)"
	if _, err := tx.ExecContext(ctx, insStmt, idpEmail, hostID, fleet.DeviceMappingMDMIdpAccounts); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert host_emails")
	}

	return idp, nil
}

func getMDMIdPAccountByHostID(ctx context.Context, q sqlx.QueryerContext, logger log.Logger, hostID uint) (*fleet.MDMIdPAccount, error) {
	stmt := `SELECT account_uuid FROM host_mdm_idp_accounts WHERE host_uuid = (SELECT uuid FROM hosts WHERE id = ?)`
	var dest []string
	if err := sqlx.SelectContext(ctx, q, &dest, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host_mdm_idp_accounts")
	}

	var acctUUID string
	switch {
	case len(dest) == 0:
		// TODO: consider falling back to the legacy enroll ref
		level.Info(logger).Log("msg", "get host mdm idp accounts: no account found", "host_id", hostID)
	default:
		if len(dest) > 1 {
			// this should not happen, but if it does we want to know about it
			level.Info(logger).Log("msg", "get host mdm idp accounts: found multiple accounts", "host_id", hostID, "acct_uuids", fmt.Sprintf("%+v", dest))
		}
		acctUUID = dest[0]
	}

	var idp fleet.MDMIdPAccount
	if acctUUID != "" {
		stmt := `SELECT uuid, username, fullname, email FROM mdm_idp_accounts WHERE uuid = ?`
		if err := sqlx.GetContext(ctx, q, &idp, stmt, acctUUID); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil // TODO: maybe return a not found error?
			}
			return nil, ctxerr.Wrap(ctx, err, "get host mdm idp account")
		}
	}

	return &idp, nil
}

func (ds *Datastore) GetMDMIdPAccountByHostUUID(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
	return getMDMIdPAccountByHostUUID(ctx, ds.reader(ctx), hostUUID)
}

func getMDMIdPAccountByHostUUID(ctx context.Context, q sqlx.QueryerContext, hostUUID string) (*fleet.MDMIdPAccount, error) {
	stmt := `
SELECT
	uuid, username, fullname, email
FROM
	mdm_idp_accounts
WHERE
	uuid = (SELECT account_uuid FROM host_mdm_idp_accounts WHERE host_uuid = ?)`

	var idp fleet.MDMIdPAccount
	if err := sqlx.GetContext(ctx, q, &idp, stmt, hostUUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // TODO: maybe return a not found error?
		}
		return nil, ctxerr.Wrap(ctx, err, "get host mdm idp account")
	}

	return &idp, nil
}
