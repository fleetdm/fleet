package mysql

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListHostCertificates(ctx context.Context, hostID uint, opts fleet.ListOptions) ([]*fleet.HostCertificateRecord, *fleet.PaginationMetadata, error) {
	return listHostCertsDB(ctx, ds.reader(ctx), hostID, opts)
}

func (ds *Datastore) UpdateHostCertificates(ctx context.Context, hostID uint, certs []*fleet.HostCertificateRecord) error {
	incomingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(certs))
	for _, cert := range certs {
		if cert.HostID != hostID {
			// caller should ensure this does not happen
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("host certificates: host ID does not match provided certificate: %d %d", hostID, cert.HostID))
		}
		if _, ok := incomingBySHA1[strings.ToUpper(hex.EncodeToString(cert.SHA1Sum))]; ok {
			// TODO: sha1 is broken so this could be a sign of a problem, how should we handle?
			level.Info(ds.logger).Log("msg", "host certificates: host has multiple certificates with the same SHA1, only the first will be recorded", "host_id", hostID, "sha1", string(cert.SHA1Sum))
			continue
		}
		incomingBySHA1[strings.ToUpper(hex.EncodeToString(cert.SHA1Sum))] = cert
	}

	// get existing certs for this host; we'll use the reader because we expect certs to change
	// infrequently and they will be eventually consistent
	existingCerts, _, err := listHostCertsDB(ctx, ds.reader(ctx), hostID, fleet.ListOptions{}) // requesting unpaginated results with default limit of 1 million
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host certificates for update")
	}
	existingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(existingCerts))
	for _, ec := range existingCerts {
		existingBySHA1[strings.ToUpper(hex.EncodeToString(ec.SHA1Sum))] = ec
	}

	toInsert := make([]*fleet.HostCertificateRecord, 0, len(incomingBySHA1))
	// toUpdate := make([]*fleet.HostCertificateRecord, 0, len(incomingBySHA1))
	for sha1, incoming := range incomingBySHA1 {
		if _, ok := existingBySHA1[sha1]; ok {
			// TODO: should we always update existing records? skipping updates reduces db load but
			// osquery is using sha1 so we consider subtleties
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("host certificates: already exists: %s", sha1), "host_id", hostID) // TODO: silence this log after initial rollout period
		} else {
			toInsert = append(toInsert, incoming)
		}
	}

	toDelete := make([]uint, 0, len(existingBySHA1))
	for sha1, existing := range existingBySHA1 {
		if _, ok := incomingBySHA1[sha1]; !ok {
			toDelete = append(toDelete, existing.ID)
		}
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := insertHostCertsDB(ctx, tx, toInsert); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host certs")
		}
		if err := softDeleteHostCertsDB(ctx, tx, hostID, toDelete); err != nil {
			return ctxerr.Wrap(ctx, err, "soft delete host certs")
		}
		return nil
	})
}

func listHostCertsDB(ctx context.Context, tx sqlx.QueryerContext, hostID uint, opts fleet.ListOptions) ([]*fleet.HostCertificateRecord, *fleet.PaginationMetadata, error) {
	stmt := `
SELECT
	id,
	sha1_sum,
	host_id,
	created_at,
	deleted_at,
	not_valid_before,
	not_valid_after,
	certificate_authority,
	common_name,
	key_algorithm,
	key_strength,
	key_usage,
	serial,
	signing_algorithm,
	subject_country,
	subject_org,
	subject_org_unit,
	subject_common_name,
	issuer_country,
	issuer_org,
	issuer_org_unit,
	issuer_common_name
FROM
	host_certificates
WHERE
	host_id = ?
	AND deleted_at IS NULL`

	args := []interface{}{hostID}
	stmtPaged, args := appendListOptionsWithCursorToSQL(stmt, args, &opts)

	var certs []*fleet.HostCertificateRecord
	if err := sqlx.SelectContext(ctx, tx, &certs, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting host certificates")
	}

	var metaData *fleet.PaginationMetadata
	if opts.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opts.Page > 0}
		if len(certs) > int(opts.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			certs = certs[:len(certs)-1]
		}
	}
	return certs, metaData, nil
}

func insertHostCertsDB(ctx context.Context, tx sqlx.ExtContext, certs []*fleet.HostCertificateRecord) error {
	if len(certs) == 0 {
		return nil
	}

	stmt := `
INSERT INTO host_certificates (
	host_id,
	sha1_sum,
	not_valid_before,
	not_valid_after,
	certificate_authority,
	common_name,
	key_algorithm,
	key_strength,
	key_usage,
	serial,
	signing_algorithm,
	subject_country,
	subject_org,
	subject_org_unit,
	subject_common_name,
	issuer_country,
	issuer_org,
	issuer_org_unit,
	issuer_common_name
) VALUES %s`

	placeholders := make([]string, 0, len(certs))
	args := make([]interface{}, 0, len(certs)*19)
	for _, cert := range certs {
		placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args,
			cert.HostID, cert.SHA1Sum, cert.NotValidBefore, cert.NotValidAfter, cert.CertificateAuthority, cert.CommonName,
			cert.KeyAlgorithm, cert.KeyStrength, cert.KeyUsage, cert.Serial, cert.SigningAlgorithm,
			cert.SubjectCountry, cert.SubjectOrganization, cert.SubjectOrganizationalUnit, cert.SubjectCommonName,
			cert.IssuerCountry, cert.IssuerOrganization, cert.IssuerOrganizationalUnit, cert.IssuerCommonName)
	}

	stmt = fmt.Sprintf(stmt, strings.Join(placeholders, ","))

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting host certificates")
	}

	return nil
}

func softDeleteHostCertsDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, toDelete []uint) error {
	// TODO: consider whether we should hard delete certs after a certain period of time if we are seeing
	// the table grow too large with soft deleted records

	if len(toDelete) == 0 {
		return nil
	}

	stmt := `UPDATE host_certificates SET deleted_at = NOW(6) WHERE host_id = ? AND id IN (?)`
	stmt, args, err := sqlx.In(stmt, hostID, toDelete)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building soft delete query")
	}

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "soft deleting host certificates")
	}

	return nil
}
