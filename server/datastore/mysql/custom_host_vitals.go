package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/text/unicode/norm"
)

var customHostVitalAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"name":       "name",
	"id":         "id",
	"updated_at": "updated_at",
}

func (ds *Datastore) CreateCustomHostVital(ctx context.Context, name string) (fleet.CustomHostVital, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO custom_host_vitals (name) VALUES (?)`,
		name,
	)
	if err != nil {
		if IsDuplicate(err) {
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, alreadyExists("name", name), "found duplicate")
		}
		return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "insert custom host vital")
	}
	id, _ := res.LastInsertId()
	return fleet.CustomHostVital{ID: uint(id), Name: name}, nil //nolint:gosec // dismiss G115
}

func (ds *Datastore) ListCustomHostVitals(ctx context.Context, opt fleet.ListOptions) (
	customHostVitals []fleet.CustomHostVital, meta *fleet.PaginationMetadata, count int, err error,
) {
	stmt := `SELECT id, name, created_at, updated_at FROM custom_host_vitals WHERE true`

	// normalize the name for full Unicode support (Unicode equivalence).
	// Search matches the name OR the variable name (the derived
	// `$FLEET_HOST_VITAL_<id>` token). The second column is a hardcoded SQL
	// expression (not user input); searchLike escapes the LIKE pattern.
	normMatch := norm.NFC.String(opt.MatchQuery)
	whereClauses, args := searchLike("", nil, normMatch, "name", `CONCAT('$FLEET_HOST_VITAL_', id)`)
	stmt += whereClauses

	// perform a second query to grab the count
	// build the count statement before adding pagination constraints
	countStmt := fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM (%s) AS s", stmt)

	stmt, args, err = appendListOptionsWithCursorToSQLSecure(stmt, args, &opt, customHostVitalAllowedOrderKeys)
	if err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "apply list options")
	}

	dbReader := ds.reader(ctx)
	if err := sqlx.SelectContext(ctx, dbReader, &customHostVitals, stmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "listing custom host vitals")
	}
	if err := sqlx.GetContext(ctx, dbReader, &count, countStmt, args...); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "get custom host vitals count")
	}

	if opt.IncludeMetadata {
		meta = &fleet.PaginationMetadata{
			HasPreviousResults: opt.Page > 0,
			TotalResults:       uint(count), //nolint:gosec // dismiss G115
		}
		// `appendListOptionsWithCursorToSQL` used above to build the query statement will cause this discrepancy.
		if len(customHostVitals) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			customHostVitals = customHostVitals[:len(customHostVitals)-1]
		}
	}

	return customHostVitals, meta, count, nil
}

func (ds *Datastore) UpdateCustomHostVital(ctx context.Context, id uint, name string) (fleet.CustomHostVital, error) {
	res, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE custom_host_vitals SET name = ? WHERE id = ?`,
		name, id,
	)
	if err != nil {
		if IsDuplicate(err) {
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, alreadyExists("name", name), "found duplicate")
		}
		return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "update custom host vital")
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		// No rows affected can mean the id was not found, or the name is unchanged.
		// Distinguish the two so a no-op rename doesn't surface as NotFound.
		// Check on the writer: the UPDATE above targeted the primary, so a replica
		// lagging behind it could otherwise report a false NotFound.
		var exists bool
		if err := sqlx.GetContext(ctx, ds.writer(ctx), &exists,
			`SELECT 1 FROM custom_host_vitals WHERE id = ?`, id); err != nil {
			if err == sql.ErrNoRows {
				return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, notFound("CustomHostVital").WithID(id))
			}
			return fleet.CustomHostVital{}, ctxerr.Wrap(ctx, err, "check custom host vital exists")
		}
	}
	return fleet.CustomHostVital{ID: id, Name: name}, nil
}

func (ds *Datastore) DeleteCustomHostVital(ctx context.Context, id uint) (name string, err error) {
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &name, `SELECT name FROM custom_host_vitals WHERE id = ?`, id)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("CustomHostVital").WithID(id))
			}
			return ctxerr.Wrap(ctx, err, "getting name of custom host vital to delete")
		}

		// Refuse to delete a definition still referenced by a script/profile.
		if usedByInfo, err := ds.customHostVitalUsedBy(ctx, tx, id, name); err != nil {
			return ctxerr.Wrap(ctx, err, "checking custom host vital references")
		} else if usedByInfo != nil {
			return ctxerr.Wrap(ctx, &fleet.CustomHostVitalUsedError{CustomHostVitalUsedInfo: *usedByInfo}, "found custom host vital in use")
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM custom_host_vitals WHERE id = ?`, id); err != nil {
			return ctxerr.Wrap(ctx, err, "delete custom host vital")
		}
		return nil
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "delete custom host vital")
	}

	return name, nil
}

// customHostVitalRefEntity is a script or profile scanned for $FLEET_HOST_VITAL_<id>
// references during delete-protection.
type customHostVitalRefEntity struct {
	// Type is the entity type, "script", "apple_profile", "apple_declaration", or "windows_profile".
	Type string `db:"entity"`
	// Name is the name of the entity.
	Name string `db:"name"`
	// FleetName is the name of the fleet (team) the entity belongs to.
	FleetName string `db:"team_name"`
	// Contents is the content of the entity (script's/profile's body).
	Contents string `db:"contents"`
}

// customHostVitalUsedBy scans scripts, Apple configuration profiles, Apple
// declarations, Windows configuration profiles, Android configuration
// profiles, software installer scripts, setup-experience scripts, and
// team/No-team host name templates for a $FLEET_HOST_VITAL_<id> (or
// ${FLEET_HOST_VITAL_<id>}) reference to the given vital id, then separately
// checks host-vitals labels (which reference the vital by id in their
// criteria JSON, not via the token). It returns a *fleet.CustomHostVitalUsedInfo
// describing the first referencing entity found, or nil if unreferenced.
// Mirrors the scan structure of DeleteSecretVariable. The second return is a
// real DB error.
func (ds *Datastore) customHostVitalUsedBy(ctx context.Context, tx sqlx.ExtContext, id uint, name string) (*fleet.CustomHostVitalUsedInfo, error) {
	// The token embeds the numeric id (survives renames), so match by id, not name.
	token := fmt.Sprintf("%s%d", fleet.CustomHostVitalPrefix, id)

	// Each scan mirrors DeleteSecretVariable: pull the content column of every
	// script/profile/declaration and check for the token in Go (os.Expand-based,
	// so it matches both $VAR and ${VAR} forms).
	scans := []struct {
		desc string
		stmt string
	}{
		{
			desc: "get script contents",
			stmt: `SELECT 'script' AS entity, s.name,
				COALESCE(t.name, 'Unassigned') AS team_name, sc.contents
				FROM script_contents sc
				JOIN scripts s ON s.script_content_id = sc.id
				LEFT JOIN teams t ON t.id = s.team_id;`,
		},
		{
			desc: "get apple profile contents",
			stmt: `SELECT 'apple_profile' AS entity, p.name,
				COALESCE(t.name, 'Unassigned') AS team_name, p.mobileconfig AS contents
				FROM mdm_apple_configuration_profiles p
				LEFT JOIN teams t ON t.id = p.team_id;`,
		},
		{
			desc: "get apple declaration contents",
			stmt: `SELECT 'apple_declaration' AS entity, d.name,
				COALESCE(t.name, 'Unassigned') AS team_name, d.raw_json AS contents
				FROM mdm_apple_declarations d
				LEFT JOIN teams t ON t.id = d.team_id;`,
		},
		{
			desc: "get windows profile contents",
			stmt: `SELECT 'windows_profile' AS entity, p.name,
				COALESCE(t.name, 'Unassigned') AS team_name, p.syncml AS contents
				FROM mdm_windows_configuration_profiles p
				LEFT JOIN teams t ON t.id = p.team_id;`,
		},
		{
			desc: "get android profile contents",
			stmt: `SELECT 'android_profile' AS entity, p.name,
				COALESCE(t.name, 'Unassigned') AS team_name, p.raw_json AS contents
				FROM mdm_android_configuration_profiles p
				LEFT JOIN teams t ON t.id = p.team_id;`,
		},
		// Software installer and setup-experience scripts exceed secret-variable
		// delete-protection (which doesn't scan them), so a vital can't be deleted
		// while a script that runs it would silently start failing on hosts.
		{
			desc: "get software installer script contents",
			stmt: `SELECT 'software_installer' AS entity, COALESCE(st.name, si.filename) AS name,
				COALESCE(t.name, 'Unassigned') AS team_name, sc.contents
				FROM software_installers si
				JOIN script_contents sc ON sc.id IN (si.install_script_content_id, si.post_install_script_content_id, si.uninstall_script_content_id)
				LEFT JOIN software_titles st ON st.id = si.title_id
				LEFT JOIN teams t ON t.id = si.team_id;`,
		},
		{
			desc: "get setup experience script contents",
			stmt: `SELECT 'setup_experience_script' AS entity, ses.name,
				COALESCE(t.name, 'Unassigned') AS team_name, sc.contents
				FROM setup_experience_scripts ses
				JOIN script_contents sc ON sc.id = ses.script_content_id
				LEFT JOIN teams t ON t.id = ses.team_id;`,
		},
		// Host name templates aren't scripts/profiles, but the token can appear in
		// a team's (or "No team"'s) name_template, same as DeleteSecretVariable
		// scans for $FLEET_SECRET_* there. A team's name_template is a plain
		// string that always serializes into the config JSON (as "" when unset),
		// and the No-team template is an optjson that serializes to null when
		// unset, so filter both on a non-empty resolved value rather than
		// IS NOT NULL (avoids scanning a NULL contents column).
		{
			desc: "get host name template contents",
			stmt: `SELECT 'host_name_template' AS entity, 'Host name' AS name,
				t.name AS team_name, t.config->>'$.mdm.name_template' AS contents
				FROM teams t
				WHERE COALESCE(t.config->>'$.mdm.name_template', '') != ''
				UNION ALL
				SELECT 'host_name_template' AS entity, 'Host name' AS name,
				'Unassigned' AS team_name, json_value->>'$.mdm.name_template' AS contents
				FROM app_config_json
				WHERE COALESCE(json_value->>'$.mdm.name_template', '') != '';`,
		},
	}

	for _, scan := range scans {
		var entities []customHostVitalRefEntity
		if err := sqlx.SelectContext(ctx, tx, &entities, scan.stmt); err != nil {
			return nil, ctxerr.Wrap(ctx, err, scan.desc)
		}
		for _, e := range entities {
			if fleet.ContainsVar(e.Contents, token) {
				return &fleet.CustomHostVitalUsedInfo{
					CustomHostVitalID:   id,
					CustomHostVitalName: name,
					Entity: fleet.EntityUsingCustomHostVital{
						Type:      fleet.CustomHostVitalEntity(e.Type),
						Name:      e.Name,
						FleetName: e.FleetName,
					},
				}, nil
			}
		}
	}

	// Host-vitals labels reference the vital by id inside their criteria JSON
	// (not by the $FLEET_HOST_VITAL_<id> token), so they need a structured check
	// rather than the content-token scan above.
	var labels []struct {
		Name      string          `db:"name"`
		FleetName string          `db:"team_name"`
		Criteria  json.RawMessage `db:"criteria"`
	}
	labelStmt := `SELECT l.name, COALESCE(t.name, 'Unassigned') AS team_name, l.criteria
		FROM labels l
		LEFT JOIN teams t ON t.id = l.team_id
		WHERE l.label_membership_type = ? AND l.criteria IS NOT NULL`
	if err := sqlx.SelectContext(ctx, tx, &labels, labelStmt, fleet.LabelMembershipTypeHostVitals); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host vitals label criteria")
	}
	for _, l := range labels {
		var criteria fleet.HostVitalCriteria
		// A label with malformed criteria is already broken; skip it rather than
		// block the delete on it.
		if err := json.Unmarshal(l.Criteria, &criteria); err != nil {
			ds.logger.WarnContext(ctx, "skipping host vitals label with unparseable criteria during custom host vital delete-protection scan",
				"label", l.Name, "error", err)
			continue
		}
		if criteria.CustomHostVitalID != nil && *criteria.CustomHostVitalID == id {
			return &fleet.CustomHostVitalUsedInfo{
				CustomHostVitalID:   id,
				CustomHostVitalName: name,
				Entity: fleet.EntityUsingCustomHostVital{
					Type:      fleet.CustomHostVitalEntityLabel,
					Name:      l.Name,
					FleetName: l.FleetName,
				},
			}, nil
		}
	}

	return nil, nil
}

func (ds *Datastore) SetHostCustomHostVitalValue(ctx context.Context, hostID uint, vitalID uint, value string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE value = VALUES(value)`,
			hostID, vitalID, value,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "set host custom host vital value")
		}

		// Re-queue any MDM profiles/declarations already delivered to this host
		// that reference the vital, so the reconcilers re-expand
		// $FLEET_HOST_VITAL_<id> with the new value. Runs in the same transaction
		// as the value write so the reconciler never reads a stale value.
		if err := resendMDMProfilesForCustomHostVital(ctx, tx, hostID, vitalID); err != nil {
			return ctxerr.Wrap(ctx, err, "resend mdm profiles for custom host vital value change")
		}

		// Re-queue the host's device-name enforcement row (if its name template
		// references the vital), so the cron re-resolves the name with the new
		// value. Same transaction as the value write, for the same reason as above.
		if err := resendDeviceNameForCustomHostVital(ctx, tx, hostID, vitalID); err != nil {
			return ctxerr.Wrap(ctx, err, "resend device name for custom host vital value change")
		}
		return nil
	})
}

// resendMDMProfilesForCustomHostVital resets the status of the Apple/Windows/
// Android configuration profiles and Apple DDM declarations already delivered
// to the host that reference $FLEET_HOST_VITAL_<vitalID>, so the reconcilers
// resend them with the host's newly-set value. Mirrors triggerResendProfilesUsingVariables,
// but matches by profile/declaration content because custom host vitals aren't
// tracked in mdm_configuration_profile_variables. Declarations only reset status
// (the DDM reconciler re-stamps variables_updated_at, cache-busting the token).
//
// Unlike the IdP resend, this deliberately omits certificate templates and
// Android managed app config: a vital can't reach cert templates (they only
// take fleet_variables), and Android managed app config isn't tracked by
// content-match resend like profiles are (its delivery is driven by the
// software worker, not the profile reconciler), so there's nothing on those
// two surfaces to resend here.
func resendMDMProfilesForCustomHostVital(ctx context.Context, tx sqlx.ExtContext, hostID, vitalID uint) error {
	var hostUUID string
	if err := sqlx.GetContext(ctx, tx, &hostUUID, `SELECT uuid FROM hosts WHERE id = ?`, hostID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get host uuid for custom host vital resend")
	}

	// varName is the exact token (id included) matched precisely, id-boundary and
	// ${...}-aware, by ContainsVar in Go. The INSTR prefix filter in SQL only
	// narrows candidates; it deliberately over-matches (ignores the id) so the
	// Go pass does the authoritative match.
	varName := fmt.Sprintf("%s%d", fleet.CustomHostVitalPrefix, vitalID)

	// These SELECTs filter on host_uuid first — the leftmost column of each
	// host-profile table's PRIMARY KEY (host_uuid, {profile,declaration}_uuid) —
	// so the INSTR content match only evaluates this one host's rows, not the
	// whole table; cost is independent of fleet size.
	const (
		customHostVitalResendAppleProfilesSelectStmt = `SELECT hmap.profile_uuid AS uuid, macp.mobileconfig AS contents
			FROM host_mdm_apple_profiles hmap
			JOIN mdm_apple_configuration_profiles macp ON macp.profile_uuid = hmap.profile_uuid
			WHERE hmap.host_uuid = ? AND hmap.operation_type = ? AND hmap.status IS NOT NULL AND INSTR(macp.mobileconfig, ?) > 0`

		customHostVitalResendWindowsProfilesSelectStmt = `SELECT hmwp.profile_uuid AS uuid, mwcp.syncml AS contents
			FROM host_mdm_windows_profiles hmwp
			JOIN mdm_windows_configuration_profiles mwcp ON mwcp.profile_uuid = hmwp.profile_uuid
			WHERE hmwp.host_uuid = ? AND hmwp.operation_type = ? AND hmwp.status IS NOT NULL AND INSTR(mwcp.syncml, ?) > 0`

		customHostVitalResendAppleDeclarationsSelectStmt = `SELECT hmad.declaration_uuid AS uuid, mad.raw_json AS contents
			FROM host_mdm_apple_declarations hmad
			JOIN mdm_apple_declarations mad ON mad.declaration_uuid = hmad.declaration_uuid
			WHERE hmad.host_uuid = ? AND hmad.operation_type = ? AND hmad.status IS NOT NULL AND INSTR(mad.raw_json, ?) > 0`

		customHostVitalResendAndroidProfilesSelectStmt = `SELECT hmap.profile_uuid AS uuid, macp.raw_json AS contents
			FROM host_mdm_android_profiles hmap
			JOIN mdm_android_configuration_profiles macp ON macp.profile_uuid = hmap.profile_uuid
			WHERE hmap.host_uuid = ? AND hmap.operation_type = ? AND hmap.status IS NOT NULL AND INSTR(macp.raw_json, ?) > 0`
	)

	targets := []struct {
		desc       string
		selectStmt string
		updateStmt string
	}{
		{
			desc:       "apple profiles",
			selectStmt: customHostVitalResendAppleProfilesSelectStmt,
			updateStmt: `UPDATE host_mdm_apple_profiles
				SET status = NULL, detail = NULL, command_uuid = ''
				WHERE host_uuid = ? AND operation_type = ? AND profile_uuid IN (?)`,
		},
		{
			desc:       "windows profiles",
			selectStmt: customHostVitalResendWindowsProfilesSelectStmt,
			updateStmt: `UPDATE host_mdm_windows_profiles
				SET status = NULL, detail = NULL, command_uuid = ''
				WHERE host_uuid = ? AND operation_type = ? AND profile_uuid IN (?)`,
		},
		{
			desc:       "apple declarations",
			selectStmt: customHostVitalResendAppleDeclarationsSelectStmt,
			updateStmt: `UPDATE host_mdm_apple_declarations
				SET status = NULL, detail = NULL
				WHERE host_uuid = ? AND operation_type = ? AND declaration_uuid IN (?)`,
		},
		{
			desc:       "android profiles",
			selectStmt: customHostVitalResendAndroidProfilesSelectStmt,
			updateStmt: `UPDATE host_mdm_android_profiles
				SET status = NULL, detail = NULL
				WHERE host_uuid = ? AND operation_type = ? AND profile_uuid IN (?)`,
		},
	}

	for _, tgt := range targets {
		var rows []struct {
			UUID     string `db:"uuid"`
			Contents string `db:"contents"`
		}
		if err := sqlx.SelectContext(ctx, tx, &rows, tgt.selectStmt,
			hostUUID, fleet.MDMOperationTypeInstall, fleet.CustomHostVitalPrefix); err != nil {
			return ctxerr.Wrap(ctx, err, "select "+tgt.desc+" referencing custom host vital")
		}

		var uuids []string
		for _, r := range rows {
			if fleet.ContainsVar(r.Contents, varName) {
				uuids = append(uuids, r.UUID)
			}
		}
		if len(uuids) == 0 {
			continue
		}

		stmt, args, err := sqlx.In(tgt.updateStmt, hostUUID, fleet.MDMOperationTypeInstall, uuids)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build resend update for "+tgt.desc)
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "reset "+tgt.desc+" for custom host vital resend")
		}
	}

	return nil
}

func (ds *Datastore) GetHostCustomHostVitals(ctx context.Context, hostID uint) ([]fleet.HostCustomHostVital, error) {
	var vitals []fleet.HostCustomHostVital
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &vitals, `
		SELECT chv.id AS custom_host_vital_id, chv.name, COALESCE(hchv.value, '') AS value
		FROM custom_host_vitals chv
		LEFT JOIN host_custom_host_vitals hchv
			ON hchv.custom_host_vital_id = chv.id AND hchv.host_id = ?
		ORDER BY chv.name`,
		hostID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host custom host vitals")
	}
	return vitals, nil
}

// ExpandCustomHostVitals substitutes $FLEET_HOST_VITAL_<id> tokens in the
// document with the given host's stored values, applying format-aware escaping
// (JSON/XML) like expandEmbeddedSecrets. If a referenced vital has no value for
// the host (no row, or an empty value), it returns a MissingCustomHostVitalValueError
// so delivery fails rather than substituting an empty value (product decision).
func (ds *Datastore) ExpandCustomHostVitals(ctx context.Context, hostID uint, document string) (string, error) {
	refIDs := fleet.FindCustomHostVitalIDs(document)
	if len(refIDs) == 0 {
		return document, nil
	}

	vitals, err := ds.GetHostCustomHostVitals(ctx, hostID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "expanding custom host vitals")
	}

	// A vital with an empty value counts as missing: we refuse to ship an empty
	// substitution.
	valueByID := make(map[uint]string, len(vitals))
	nameByID := make(map[uint]string, len(vitals))
	for _, v := range vitals {
		nameByID[v.CustomHostVitalID] = v.Name
		if v.Value == "" {
			continue
		}
		valueByID[v.CustomHostVitalID] = v.Value
	}

	var missingIDs []uint
	var missingNames []string
	for _, id := range refIDs {
		if _, ok := valueByID[id]; !ok {
			missingIDs = append(missingIDs, id)
			missingNames = append(missingNames, nameByID[id])
		}
	}
	if len(missingIDs) > 0 {
		// The vital exists (validated on upload); the host just has no value for it.
		return "", &fleet.MissingCustomHostVitalValueError{MissingIDs: missingIDs, MissingNames: missingNames}
	}

	expanded := expandDocumentVars(document, func(s string) (string, bool) {
		if !strings.HasPrefix(s, fleet.CustomHostVitalPrefix) {
			return "", false
		}
		id, parseErr := strconv.ParseUint(strings.TrimPrefix(s, fleet.CustomHostVitalPrefix), 10, strconv.IntSize)
		if parseErr != nil {
			return "", false
		}
		val, ok := valueByID[uint(id)]
		return val, ok
	})

	return expanded, nil
}

// ValidateReferencedCustomHostVitals parses $FLEET_HOST_VITAL_<id> tokens from
// the given documents and verifies every referenced id resolves to a definition.
// Mirrors ValidateEmbeddedSecrets. Returns a MissingCustomHostVitalsError listing
// any unknown ids.
func (ds *Datastore) ValidateReferencedCustomHostVitals(ctx context.Context, documents []string) error {
	wantIDs := make(map[uint]struct{})
	var malformed []string
	seenMalformed := make(map[string]struct{})
	for _, document := range documents {
		// A $FLEET_HOST_VITAL_<x> token whose <x> isn't a valid ID (e.g. a typo like
		// $FLEET_HOST_VITAL_asset_tag) is rejected rather than silently delivered as
		// a literal token, matching how $FLEET_VAR_*/$FLEET_SECRET_* reject unknowns.
		for _, ref := range fleet.ContainsMalformedCustomHostVitalRefs(document) {
			if _, ok := seenMalformed[ref]; ok {
				continue
			}
			seenMalformed[ref] = struct{}{}
			malformed = append(malformed, ref)
		}
		for _, id := range fleet.FindCustomHostVitalIDs(document) {
			wantIDs[id] = struct{}{}
		}
	}
	if len(malformed) > 0 {
		return &fleet.InvalidCustomHostVitalRefError{Refs: malformed}
	}
	if len(wantIDs) == 0 {
		return nil
	}

	wantIDsList := make([]uint, 0, len(wantIDs))
	for id := range wantIDs {
		wantIDsList = append(wantIDsList, id)
	}

	dbVitals, err := ds.GetCustomHostVitals(ctx, wantIDsList)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating document referenced custom host vitals")
	}

	haveIDs := make(map[uint]struct{}, len(dbVitals))
	for _, v := range dbVitals {
		haveIDs[v.ID] = struct{}{}
	}

	var missingIDs []uint
	for id := range wantIDs {
		if _, ok := haveIDs[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		return &fleet.MissingCustomHostVitalsError{MissingIDs: missingIDs}
	}
	return nil
}

func (ds *Datastore) GetCustomHostVitals(ctx context.Context, ids []uint) ([]fleet.CustomHostVital, error) {
	stmt, args, err := sqlx.In(`
		SELECT id, name, created_at, updated_at
		FROM custom_host_vitals
		WHERE id IN (?)`, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build custom host vitals query")
	}

	var vitals []fleet.CustomHostVital
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &vitals, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get custom host vitals")
	}
	return vitals, nil
}

func (ds *Datastore) UpsertCustomHostVitals(ctx context.Context, vitals []fleet.CustomHostVital) (created []fleet.CustomHostVital, deleted []fleet.CustomHostVital, err error) {
	incomingNames := make(map[string]struct{}, len(vitals))
	for _, v := range vitals {
		incomingNames[v.Name] = struct{}{}
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		created, deleted = nil, nil

		var existing []fleet.CustomHostVital
		if err := sqlx.SelectContext(ctx, tx, &existing, `SELECT id, name FROM custom_host_vitals`); err != nil {
			return ctxerr.Wrap(ctx, err, "list existing custom host vitals")
		}

		existingNames := make(map[string]struct{}, len(existing))
		for _, e := range existing {
			existingNames[e.Name] = struct{}{}
			if _, ok := incomingNames[e.Name]; !ok {
				deleted = append(deleted, e)
			}
		}

		var toInsert []string
		for _, v := range vitals {
			if _, ok := existingNames[v.Name]; !ok {
				toInsert = append(toInsert, v.Name)
			}
		}

		for _, v := range deleted {
			usedByInfo, err := ds.customHostVitalUsedBy(ctx, tx, v.ID, v.Name)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "checking custom host vital references")
			}
			if usedByInfo != nil {
				return ctxerr.Wrap(ctx, &fleet.CustomHostVitalUsedError{CustomHostVitalUsedInfo: *usedByInfo}, "found custom host vital in use")
			}
		}

		if len(deleted) > 0 {
			ids := make([]uint, 0, len(deleted))
			for _, v := range deleted {
				ids = append(ids, v.ID)
			}
			stmt, args, err := sqlx.In(`DELETE FROM custom_host_vitals WHERE id IN (?)`, ids)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build delete custom host vitals query")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete custom host vitals")
			}
		}

		// Inserted one at a time (rather than a single multi-row INSERT) so each
		// row's LastInsertId can be captured for the returned `created` list.
		for _, name := range toInsert {
			res, err := tx.ExecContext(ctx, `INSERT INTO custom_host_vitals (name) VALUES (?)`, name)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert custom host vital")
			}
			id, _ := res.LastInsertId()
			created = append(created, fleet.CustomHostVital{ID: uint(id), Name: name}) //nolint:gosec // dismiss G115
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return created, deleted, nil
}
