package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateAndroidCACertificate(ctx context.Context, cert *fleet.AndroidCACertificate) (*fleet.AndroidCACertificate, error) {
	var createdCert *fleet.AndroidCACertificate
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO android_ca_certificates (
				team_id, name, certificate_pem, sha256_fingerprint, subject, issuer, not_valid_before, not_valid_after
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, cert.TeamID, cert.Name, cert.CertificatePEM, cert.SHA256Fingerprint, cert.Subject, cert.Issuer, cert.NotValidBefore, cert.NotValidAfter)
		if err != nil {
			if IsDuplicate(err) {
				return ctxerr.Wrap(ctx, alreadyExists("AndroidCACertificate", cert.Name), "inserting android_ca_certificate")
			}
			return ctxerr.Wrap(ctx, err, "inserting android_ca_certificate")
		}

		id, err := res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting last insert id for android_ca_certificate")
		}
		certID := uint(id)

		if err := insertAndroidCACertificateLabels(ctx, tx, certID, cert.LabelsIncludeAll, cert.LabelsIncludeAny, cert.LabelsExcludeAny); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting android ca certificate labels")
		}

		// Insert pending tracking rows for existing matching enrolled Android hosts
		if err := createPendingAndroidCACertificatesForExistingHostsTx(ctx, tx, certID, cert.TeamID); err != nil {
			return ctxerr.Wrap(ctx, err, "creating pending ca cert records for existing hosts")
		}

		createdCert = &fleet.AndroidCACertificate{
			ID:                certID,
			TeamID:            cert.TeamID,
			Name:              cert.Name,
			CertificatePEM:    cert.CertificatePEM,
			SHA256Fingerprint: cert.SHA256Fingerprint,
			Subject:           cert.Subject,
			Issuer:            cert.Issuer,
			NotValidBefore:    cert.NotValidBefore,
			NotValidAfter:     cert.NotValidAfter,
			LabelsIncludeAll:  cert.LabelsIncludeAll,
			LabelsIncludeAny:  cert.LabelsIncludeAny,
			LabelsExcludeAny:  cert.LabelsExcludeAny,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdCert, nil
}

func insertAndroidCACertificateLabels(ctx context.Context, tx sqlx.ExtContext, caCertID uint, includeAll, includeAny, excludeAny []string) error {
	var allNames []string
	allNames = append(allNames, includeAll...)
	allNames = append(allNames, includeAny...)
	allNames = append(allNames, excludeAny...)
	if len(allNames) == 0 {
		return nil
	}

	labelIDsByName := make(map[string]uint)
	query, args, err := sqlx.In(`SELECT id, name FROM labels WHERE name IN (?)`, allNames)
	if err == nil {
		var rows []struct {
			ID   uint   `db:"id"`
			Name string `db:"name"`
		}
		if err := sqlx.SelectContext(ctx, tx, &rows, query, args...); err == nil {
			for _, r := range rows {
				labelIDsByName[r.Name] = r.ID
			}
		}
	}

	insertLabel := func(name string, exclude, requireAll int) error {
		var labelID *uint
		if id, ok := labelIDsByName[name]; ok {
			labelID = &id
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO android_ca_certificate_labels (
				ca_certificate_id, label_name, label_id, exclude, require_all
			) VALUES (?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				label_id = VALUES(label_id),
				exclude = VALUES(exclude),
				require_all = VALUES(require_all)
		`, caCertID, name, labelID, exclude, requireAll)
		return err
	}

	for _, name := range includeAll {
		if err := insertLabel(name, 0, 1); err != nil {
			return ctxerr.Wrapf(ctx, err, "insert include_all label %s", name)
		}
	}
	for _, name := range includeAny {
		if err := insertLabel(name, 0, 0); err != nil {
			return ctxerr.Wrapf(ctx, err, "insert include_any label %s", name)
		}
	}
	for _, name := range excludeAny {
		if err := insertLabel(name, 1, 0); err != nil {
			return ctxerr.Wrapf(ctx, err, "insert exclude_any label %s", name)
		}
	}
	return nil
}

func loadAndroidCACertificateLabels(ctx context.Context, q sqlx.QueryerContext, certs []*fleet.AndroidCACertificate) error {
	if len(certs) == 0 {
		return nil
	}
	certMap := make(map[uint]*fleet.AndroidCACertificate, len(certs))
	var ids []uint
	for _, c := range certs {
		certMap[c.ID] = c
		ids = append(ids, c.ID)
	}

	query, args, err := sqlx.In(`
		SELECT ca_certificate_id, label_name, exclude, require_all
		FROM android_ca_certificate_labels
		WHERE ca_certificate_id IN (?)
		ORDER BY label_name ASC
	`, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query for ca cert labels")
	}

	var rows []struct {
		CACertificateID uint   `db:"ca_certificate_id"`
		LabelName       string `db:"label_name"`
		Exclude         int    `db:"exclude"`
		RequireAll      int    `db:"require_all"`
	}
	if err := sqlx.SelectContext(ctx, q, &rows, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting ca cert labels")
	}

	for _, r := range rows {
		c, ok := certMap[r.CACertificateID]
		if !ok {
			continue
		}
		if r.Exclude == 1 {
			c.LabelsExcludeAny = append(c.LabelsExcludeAny, r.LabelName)
		} else if r.RequireAll == 1 {
			c.LabelsIncludeAll = append(c.LabelsIncludeAll, r.LabelName)
		} else {
			c.LabelsIncludeAny = append(c.LabelsIncludeAny, r.LabelName)
		}
	}
	return nil
}

func (ds *Datastore) GetAndroidCACertificateByID(ctx context.Context, id uint) (*fleet.AndroidCACertificate, error) {
	stmt := `
		SELECT
			id, team_id, name, certificate_pem, sha256_fingerprint, subject, issuer,
			not_valid_before, not_valid_after, created_at, updated_at
		FROM android_ca_certificates
		WHERE id = ?
	`
	var cert fleet.AndroidCACertificate
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &cert, stmt, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("AndroidCACertificate").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting android_ca_certificate by id")
	}

	certs := []*fleet.AndroidCACertificate{&cert}
	if err := loadAndroidCACertificateLabels(ctx, ds.reader(ctx), certs); err != nil {
		return nil, err
	}
	return &cert, nil
}

func (ds *Datastore) GetAndroidCACertificateByName(ctx context.Context, teamID uint, name string) (*fleet.AndroidCACertificate, error) {
	stmt := `
		SELECT
			id, team_id, name, certificate_pem, sha256_fingerprint, subject, issuer,
			not_valid_before, not_valid_after, created_at, updated_at
		FROM android_ca_certificates
		WHERE team_id = ? AND name = ?
	`
	var cert fleet.AndroidCACertificate
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &cert, stmt, teamID, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("AndroidCACertificate").WithName(name)
		}
		return nil, ctxerr.Wrap(ctx, err, "getting android_ca_certificate by name")
	}

	certs := []*fleet.AndroidCACertificate{&cert}
	if err := loadAndroidCACertificateLabels(ctx, ds.reader(ctx), certs); err != nil {
		return nil, err
	}
	return &cert, nil
}

func (ds *Datastore) ListAndroidCACertificates(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.AndroidCACertificate, *fleet.PaginationMetadata, error) {
	stmt := `
		SELECT
			id, team_id, name, sha256_fingerprint, subject, issuer,
			not_valid_before, not_valid_after, created_at, updated_at
		FROM android_ca_certificates
		WHERE team_id = ?
		ORDER BY id ASC
	`
	var certs []*fleet.AndroidCACertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &certs, stmt, teamID); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "listing android ca certificates")
	}

	if err := loadAndroidCACertificateLabels(ctx, ds.reader(ctx), certs); err != nil {
		return nil, nil, err
	}

	meta := &fleet.PaginationMetadata{
		TotalResults:       uint(len(certs)),
		HasNextResults:     false,
		HasPreviousResults: false,
	}
	return certs, meta, nil
}

func (ds *Datastore) DeleteAndroidCACertificate(ctx context.Context, id uint) error {
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM android_ca_certificates WHERE id = ?`, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting android ca certificate")
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting rows affected for android ca certificate deletion")
	}
	if rowsAffected == 0 {
		return notFound("AndroidCACertificate").WithID(id)
	}
	return nil
}

func createPendingAndroidCACertificatesForExistingHostsTx(ctx context.Context, ext sqlx.ExtContext, caCertID uint, teamID uint) error {
	stmt := `
		INSERT INTO host_android_ca_certificates (
			host_id, host_uuid, ca_certificate_id, status
		)
		SELECT
			h.id, h.uuid, ?, 'pending'
		FROM hosts h
		JOIN host_mdm hm ON hm.host_id = h.id
		WHERE h.platform = 'android'
		  AND hm.enrolled = 1
		  AND (h.team_id = ? OR (? = 0 AND h.team_id IS NULL))
		  AND NOT EXISTS (
			  SELECT 1 FROM android_ca_certificate_labels acl
			  JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			  WHERE acl.ca_certificate_id = ? AND acl.exclude = 1
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1)
			  =
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1)
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 0) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 0) > 0
		  )
		ON DUPLICATE KEY UPDATE host_id = host_id
	`
	_, err := ext.ExecContext(ctx, stmt, caCertID, teamID, teamID, caCertID, caCertID, caCertID, caCertID, caCertID, caCertID)
	return err
}

func (ds *Datastore) CreatePendingAndroidCACertificatesForExistingHosts(ctx context.Context, caCertID uint, teamID uint) (int64, error) {
	var rows int64
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO host_android_ca_certificates (
				host_id, host_uuid, ca_certificate_id, status
			)
			SELECT
				h.id, h.uuid, ?, 'pending'
			FROM hosts h
			JOIN host_mdm hm ON hm.host_id = h.id
			WHERE h.platform = 'android'
			  AND hm.enrolled = 1
			  AND (h.team_id = ? OR (? = 0 AND h.team_id IS NULL))
			  AND NOT EXISTS (
				  SELECT 1 FROM android_ca_certificate_labels acl
				  JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
				  WHERE acl.ca_certificate_id = ? AND acl.exclude = 1
			  )
			  AND (
				  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1) = 0
				  OR
				  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
				   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
				   WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1)
				  =
				  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 1)
			  )
			  AND (
				  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 0) = 0
				  OR
				  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
				   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
				   WHERE acl.ca_certificate_id = ? AND acl.exclude = 0 AND acl.require_all = 0) > 0
			  )
			ON DUPLICATE KEY UPDATE host_id = host_id
		`, caCertID, teamID, teamID, caCertID, caCertID, caCertID, caCertID, caCertID, caCertID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating pending ca cert records for existing hosts")
		}
		r, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting rows affected")
		}
		rows = r
		return nil
	})
	return rows, err
}

func (ds *Datastore) CreatePendingAndroidCACertificatesForNewHost(ctx context.Context, host *fleet.Host) error {
	if host == nil || host.Platform != "android" {
		return nil
	}
	scopedCAs, err := ds.GetHostScopedAndroidCACertificates(ctx, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting scoped ca certs for new host")
	}
	for _, ca := range scopedCAs {
		if err := ds.UpsertHostAndroidCACertificateStatus(ctx, host.ID, host.UUID, ca.ID, fleet.MDMDeliveryPending, ""); err != nil {
			return ctxerr.Wrapf(ctx, err, "upserting pending ca cert for new host %d", host.ID)
		}
	}
	return nil
}

func (ds *Datastore) GetHostScopedAndroidCACertificates(ctx context.Context, hostID uint) ([]*fleet.AndroidCACertificate, error) {
	stmt := `
		SELECT
			cac.id,
			cac.team_id,
			cac.name,
			cac.certificate_pem,
			cac.sha256_fingerprint,
			cac.subject,
			cac.issuer,
			cac.not_valid_before,
			cac.not_valid_after,
			cac.created_at,
			cac.updated_at
		FROM android_ca_certificates cac
		JOIN hosts h ON (cac.team_id = COALESCE(h.team_id, 0))
		WHERE h.id = ?
		  AND h.platform = 'android'
		  AND NOT EXISTS (
			  SELECT 1 FROM android_ca_certificate_labels acl
			  JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			  WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 1
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1)
			  =
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1)
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 0) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 0) > 0
		  )
		ORDER BY cac.name ASC
	`
	var certs []*fleet.AndroidCACertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &certs, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host scoped ca certificates")
	}
	if err := loadAndroidCACertificateLabels(ctx, ds.reader(ctx), certs); err != nil {
		return nil, err
	}
	return certs, nil
}

func (ds *Datastore) GetHostScopedAndroidCACertificatesByUUID(ctx context.Context, hostUUID string) ([]*fleet.AndroidCACertificate, error) {
	stmt := `
		SELECT
			cac.id,
			cac.team_id,
			cac.name,
			cac.certificate_pem,
			cac.sha256_fingerprint,
			cac.subject,
			cac.issuer,
			cac.not_valid_before,
			cac.not_valid_after,
			cac.created_at,
			cac.updated_at
		FROM android_ca_certificates cac
		JOIN hosts h ON (cac.team_id = COALESCE(h.team_id, 0))
		WHERE h.uuid = ?
		  AND h.platform = 'android'
		  AND NOT EXISTS (
			  SELECT 1 FROM android_ca_certificate_labels acl
			  JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			  WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 1
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1)
			  =
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 1)
		  )
		  AND (
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 0) = 0
			  OR
			  (SELECT COUNT(*) FROM android_ca_certificate_labels acl
			   JOIN label_membership lm ON lm.label_id = acl.label_id AND lm.host_id = h.id
			   WHERE acl.ca_certificate_id = cac.id AND acl.exclude = 0 AND acl.require_all = 0) > 0
		  )
		ORDER BY cac.name ASC
	`
	var certs []*fleet.AndroidCACertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &certs, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host scoped ca certificates by uuid")
	}
	if err := loadAndroidCACertificateLabels(ctx, ds.reader(ctx), certs); err != nil {
		return nil, err
	}
	return certs, nil
}

func (ds *Datastore) UpsertHostAndroidCACertificateStatus(ctx context.Context, hostID uint, hostUUID string, caCertID uint, status fleet.MDMDeliveryStatus, detail string) error {
	stmt := `
		INSERT INTO host_android_ca_certificates (
			host_id, host_uuid, ca_certificate_id, status, detail
		) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			detail = VALUES(detail),
			updated_at = CURRENT_TIMESTAMP
	`
	var detailVal sql.NullString
	if detail != "" {
		detailVal = sql.NullString{String: detail, Valid: true}
	}
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, hostUUID, caCertID, status, detailVal)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upserting host android ca certificate status")
	}
	return nil
}

func (ds *Datastore) ListHostAndroidCACertificates(ctx context.Context, hostID uint) ([]*fleet.HostAndroidCACertificate, error) {
	stmt := `
		SELECT
			hac.id,
			hac.host_id,
			hac.host_uuid,
			hac.ca_certificate_id,
			cac.name AS certificate_name,
			cac.subject,
			cac.issuer,
			cac.sha256_fingerprint,
			cac.not_valid_before,
			cac.not_valid_after,
			hac.status,
			COALESCE(hac.detail, '') AS detail,
			hac.created_at,
			hac.updated_at
		FROM host_android_ca_certificates hac
		JOIN android_ca_certificates cac ON cac.id = hac.ca_certificate_id
		WHERE hac.host_id = ?
		ORDER BY cac.name ASC
	`
	var list []*fleet.HostAndroidCACertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &list, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing host android ca certificates")
	}
	return list, nil
}

func (ds *Datastore) ListHostAndroidCACertificatesByUUID(ctx context.Context, hostUUID string) ([]*fleet.HostAndroidCACertificate, error) {
	stmt := `
		SELECT
			hac.id,
			hac.host_id,
			hac.host_uuid,
			hac.ca_certificate_id,
			cac.name AS certificate_name,
			cac.subject,
			cac.issuer,
			cac.sha256_fingerprint,
			cac.not_valid_before,
			cac.not_valid_after,
			hac.status,
			COALESCE(hac.detail, '') AS detail,
			hac.created_at,
			hac.updated_at
		FROM host_android_ca_certificates hac
		JOIN android_ca_certificates cac ON cac.id = hac.ca_certificate_id
		WHERE hac.host_uuid = ?
		ORDER BY cac.name ASC
	`
	var list []*fleet.HostAndroidCACertificate
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &list, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing host android ca certificates by uuid")
	}
	return list, nil
}

func (ds *Datastore) ListAndroidHostUUIDsWithPendingCACertificates(ctx context.Context, offset int, limit int) ([]string, error) {
	stmt := `
		SELECT DISTINCT host_uuid
		FROM host_android_ca_certificates
		WHERE status = 'pending'
		ORDER BY host_uuid
		LIMIT ? OFFSET ?
	`
	var hostUUIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostUUIDs, stmt, limit, offset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing android host uuids with pending ca certificates")
	}
	return hostUUIDs, nil
}
