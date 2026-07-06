package mysql

import (
	"context"
	"database/sql"
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
		var exists bool
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists,
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
		if useErr, err := customHostVitalUsedBy(ctx, tx, id, name); err != nil {
			return ctxerr.Wrap(ctx, err, "checking custom host vital references")
		} else if useErr != nil {
			return ctxerr.Wrap(ctx, useErr, "found custom host vital in use")
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

// customHostVitalUsedBy scans script_contents, Apple configuration profiles,
// Apple declarations, and Windows configuration profiles for a
// $FLEET_HOST_VITAL_<id> (or ${FLEET_HOST_VITAL_<id>}) reference to the given
// vital id. It returns a *fleet.CustomHostVitalUsedError describing the first
// referencing entity found, or nil if unreferenced. Mirrors the scan structure
// of DeleteSecretVariable. The second return is a real DB error.
func customHostVitalUsedBy(ctx context.Context, tx sqlx.ExtContext, id uint, name string) (*fleet.CustomHostVitalUsedError, error) {
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
				COALESCE(t.name, 'No team') AS team_name, sc.contents
				FROM script_contents sc
				JOIN scripts s ON s.script_content_id = sc.id
				LEFT JOIN teams t ON t.id = s.team_id;`,
		},
		{
			desc: "get apple profile contents",
			stmt: `SELECT 'apple_profile' AS entity, p.name,
				COALESCE(t.name, 'No team') AS team_name, p.mobileconfig AS contents
				FROM mdm_apple_configuration_profiles p
				LEFT JOIN teams t ON t.id = p.team_id;`,
		},
		{
			desc: "get apple declaration contents",
			stmt: `SELECT 'apple_declaration' AS entity, d.name,
				COALESCE(t.name, 'No team') AS team_name, d.raw_json AS contents
				FROM mdm_apple_declarations d
				LEFT JOIN teams t ON t.id = d.team_id;`,
		},
		{
			desc: "get windows profile contents",
			stmt: `SELECT 'windows_profile' AS entity, p.name,
				COALESCE(t.name, 'No team') AS team_name, p.syncml AS contents
				FROM mdm_windows_configuration_profiles p
				LEFT JOIN teams t ON t.id = p.team_id;`,
		},
		// Software installer and setup-experience scripts exceed secret-variable
		// delete-protection (which doesn't scan them), so a vital can't be deleted
		// while a script that runs it would silently start failing on hosts.
		{
			desc: "get software installer script contents",
			stmt: `SELECT 'software_installer' AS entity, COALESCE(st.name, si.filename) AS name,
				COALESCE(t.name, 'No team') AS team_name, sc.contents
				FROM software_installers si
				JOIN script_contents sc ON sc.id IN (si.install_script_content_id, si.post_install_script_content_id, si.uninstall_script_content_id)
				LEFT JOIN software_titles st ON st.id = si.title_id
				LEFT JOIN teams t ON t.id = si.team_id;`,
		},
		{
			desc: "get setup experience script contents",
			stmt: `SELECT 'setup_experience_script' AS entity, ses.name,
				COALESCE(t.name, 'No team') AS team_name, sc.contents
				FROM setup_experience_scripts ses
				JOIN script_contents sc ON sc.id = ses.script_content_id
				LEFT JOIN teams t ON t.id = ses.team_id;`,
		},
	}

	for _, scan := range scans {
		var entities []customHostVitalRefEntity
		if err := sqlx.SelectContext(ctx, tx, &entities, scan.stmt); err != nil {
			return nil, ctxerr.Wrap(ctx, err, scan.desc)
		}
		for _, e := range entities {
			if fleet.ContainsVar(e.Contents, token) {
				return &fleet.CustomHostVitalUsedError{
					CustomHostVitalID:   id,
					CustomHostVitalName: name,
					Entity: fleet.EntityUsingCustomHostVital{
						Type:      e.Type,
						Name:      e.Name,
						FleetName: e.FleetName,
					},
				}, nil
			}
		}
	}

	return nil, nil
}

func (ds *Datastore) SetHostCustomHostVitalValue(ctx context.Context, hostID uint, vitalID uint, value string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE value = VALUES(value)`,
		hostID, vitalID, value,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set host custom host vital value")
	}
	return nil
}

func (ds *Datastore) GetHostCustomHostVitals(ctx context.Context, hostID uint) ([]fleet.HostCustomHostVital, error) {
	var vitals []fleet.HostCustomHostVital
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &vitals, `
		SELECT chv.id AS custom_host_vital_id, chv.name, hchv.value
		FROM host_custom_host_vitals hchv
		JOIN custom_host_vitals chv ON chv.id = hchv.custom_host_vital_id
		WHERE hchv.host_id = ?`,
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
	refIDs := fleet.ContainsCustomHostVitalIDs(document)
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
	for _, v := range vitals {
		if v.Value == "" {
			continue
		}
		valueByID[v.CustomHostVitalID] = v.Value
	}

	var missingIDs []uint
	for _, id := range refIDs {
		if _, ok := valueByID[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}
	if len(missingIDs) > 0 {
		// The vital exists (validated on upload); the host just has no value for it.
		return "", &fleet.MissingCustomHostVitalValueError{MissingIDs: missingIDs}
	}

	expanded := expandDocumentVars(document, func(s string) (string, bool) {
		if !strings.HasPrefix(s, fleet.CustomHostVitalPrefix) {
			return "", false
		}
		id, parseErr := strconv.ParseUint(strings.TrimPrefix(s, fleet.CustomHostVitalPrefix), 10, 64)
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
	for _, document := range documents {
		for _, id := range fleet.ContainsCustomHostVitalIDs(document) {
			wantIDs[id] = struct{}{}
		}
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
