package mysql

import (
	"context"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListHostCertificates(ctx context.Context, hostID uint, opts fleet.ListOptions) ([]*fleet.HostCertificateRecord, *fleet.PaginationMetadata, error) {
	return listHostCertsDB(ctx, ds.reader(ctx), hostID, opts)
}

func (ds *Datastore) UpdateHostCertificates(ctx context.Context, hostID uint, hostUUID string, certs []*fleet.HostCertificateRecord) error {
	type certSourceToSet struct {
		Source   fleet.HostCertificateSource
		Username string
	}

	incomingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(certs))
	incomingSourcesBySHA1 := make(map[string][]certSourceToSet, len(certs))
	for _, cert := range certs {
		if cert.HostID != hostID {
			// caller should ensure this does not happen
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("host certificates: host ID does not match provided certificate: %d %d", hostID, cert.HostID))
		}

		normalizedSHA1 := strings.ToUpper(hex.EncodeToString(cert.SHA1Sum))
		incomingSourcesBySHA1[normalizedSHA1] = append(incomingSourcesBySHA1[normalizedSHA1], certSourceToSet{
			Source:   cert.Source,
			Username: cert.Username,
		})
		incomingBySHA1[normalizedSHA1] = cert
	}

	// get existing certs for this host; we'll use the reader because we expect certs to change
	// infrequently and they will be eventually consistent
	existingCerts, _, err := listHostCertsDB(ctx, ds.reader(ctx), hostID, fleet.ListOptions{}) // requesting unpaginated results with default limit of 1 million
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host certificates for update")
	}

	existingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(existingCerts))
	existingSourcesBySHA1 := make(map[string][]certSourceToSet, len(existingCerts))
	for _, ec := range existingCerts {
		normalizedSHA1 := strings.ToUpper(hex.EncodeToString(ec.SHA1Sum))
		existingBySHA1[normalizedSHA1] = ec
		existingSourcesBySHA1[normalizedSHA1] = append(existingSourcesBySHA1[normalizedSHA1], certSourceToSet{
			Source:   ec.Source,
			Username: ec.Username,
		})
	}

	toInsert := make([]*fleet.HostCertificateRecord, 0, len(incomingBySHA1))
	toSetSourcesBySHA1 := make(map[string][]certSourceToSet, len(incomingBySHA1))
	for sha1, incoming := range incomingBySHA1 {
		incomingSources := incomingSourcesBySHA1[sha1]
		existingSources := existingSourcesBySHA1[sha1]
		if !slices.Equal(incomingSources, existingSources) {
			toSetSourcesBySHA1[sha1] = incomingSources
		}

		if _, ok := existingBySHA1[sha1]; ok {
			// TODO: should we always update existing records? skipping updates reduces db load but
			// osquery is using sha1 so we consider subtleties
			level.Debug(ds.logger).Log("msg", fmt.Sprintf("host certificates: already exists: %s", sha1), "host_id", hostID) // TODO: silence this log after initial rollout period
		} else {
			toInsert = append(toInsert, incoming)
		}
	}

	// Check if any of the certs to insert are managed by Fleet; if so, update the associated host_mdm_managed_certificates rows
	hostMDMManagedCertsToUpdate := make([]*fleet.MDMManagedCertificate, 0, len(toInsert))
	if len(toInsert) > 0 {
		hostMDMManagedCerts, err := ds.ListHostMDMManagedCertificates(ctx, hostUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list host mdm managed certs for update")
		}
		for _, hostMDMManagedCert := range hostMDMManagedCerts {
			// Note that we only care about proxied SCEP certificates because DigiCert are requested
			// by Fleet and stored in the DB directly, so we need not fetch them via osquery/MDM
			if hostMDMManagedCert.Type != fleet.CAConfigCustomSCEPProxy && hostMDMManagedCert.Type != fleet.CAConfigNDES {
				continue
			}
			for _, certToInsert := range toInsert {
				if strings.Contains(certToInsert.SubjectCommonName, "fleet-"+hostMDMManagedCert.ProfileUUID) {
					managedCertToUpdate := &fleet.MDMManagedCertificate{
						ProfileUUID:          hostMDMManagedCert.ProfileUUID,
						HostUUID:             hostMDMManagedCert.HostUUID,
						ChallengeRetrievedAt: hostMDMManagedCert.ChallengeRetrievedAt,
						NotValidBefore:       &certToInsert.NotValidBefore,
						NotValidAfter:        &certToInsert.NotValidAfter,
						Type:                 hostMDMManagedCert.Type,
						CAName:               hostMDMManagedCert.CAName,
						Serial:               ptr.String(fmt.Sprintf("%040s", certToInsert.Serial)),
					}
					// To reduce DB load, we only write to datastore if the managed cert is different
					// However, they should never be the same because we check the certificate SHA1 above and only insert new certs
					if !hostMDMManagedCert.Equal(*managedCertToUpdate) {
						hostMDMManagedCertsToUpdate = append(hostMDMManagedCertsToUpdate, managedCertToUpdate)
					}
					// We found a matching cert from host certs; move on to the next managed cert
					break
				}
			}
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

		if len(toSetSourcesBySHA1) > 0 {
			// must reload the DB IDs to insert the host_certificates_sources rows
			certIDsBySHA1, err := loadHostCertIDsForSHA1DB(ctx, tx, slices.Collect(maps.Keys(toSetSourcesBySHA1)))
			if err != nil {
				return ctxerr.Wrap(ctx, err, "load host certs ids")
			}

			toReplaceSources := make([]*fleet.HostCertificateRecord, 0, len(toSetSourcesBySHA1))
			for sha1, sources := range toSetSourcesBySHA1 {
				for _, source := range sources {
					toReplaceSources = append(toReplaceSources, &fleet.HostCertificateRecord{
						ID:       certIDsBySHA1[sha1],
						Source:   source.Source,
						Username: source.Username,
					})
				}
			}
			if err := replaceHostCertsSourcesDB(ctx, tx, toReplaceSources); err != nil {
				return ctxerr.Wrap(ctx, err, "replace host certs sources")
			}
		}

		if err := softDeleteHostCertsDB(ctx, tx, hostID, toDelete); err != nil {
			return ctxerr.Wrap(ctx, err, "soft delete host certs")
		}

		if err := updateHostMDMManagedCertDetailsDB(ctx, tx, hostMDMManagedCertsToUpdate); err != nil {
			return ctxerr.Wrap(ctx, err, "update host mdm managed cert details")
		}
		return nil
	})
}

func loadHostCertIDsForSHA1DB(ctx context.Context, tx sqlx.QueryerContext, sha1s []string) (map[string]uint, error) {
	if len(sha1s) == 0 {
		return nil, nil
	}

	binarySHA1s := make([][]byte, 0, len(sha1s))
	for _, sha1 := range sha1s {
		binarySHA1, _ := hex.DecodeString(sha1)
		binarySHA1s = append(binarySHA1s, binarySHA1)
	}

	stmt := `
	SELECT
		hc.id,
		hc.sha1_sum
	FROM
		host_certificates hc
	WHERE
		hc.sha1_sum IN (?)`

	var certs []*fleet.HostCertificateRecord
	stmt, args, err := sqlx.In(stmt, binarySHA1s)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building load host cert ids query")
	}
	if err := sqlx.SelectContext(ctx, tx, &certs, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting host cert ids")
	}

	certIDsBySHA1 := make(map[string]uint, len(certs))
	for _, cert := range certs {
		normalizedSHA1 := strings.ToUpper(hex.EncodeToString(cert.SHA1Sum))
		certIDsBySHA1[normalizedSHA1] = cert.ID
	}
	return certIDsBySHA1, nil
}

func listHostCertsDB(ctx context.Context, tx sqlx.QueryerContext, hostID uint, opts fleet.ListOptions) ([]*fleet.HostCertificateRecord, *fleet.PaginationMetadata, error) {
	stmt := `
SELECT
	hc.id,
	hc.sha1_sum,
	hc.host_id,
	hc.created_at,
	hc.deleted_at,
	hc.not_valid_before,
	hc.not_valid_after,
	hc.certificate_authority,
	hc.common_name,
	hc.key_algorithm,
	hc.key_strength,
	hc.key_usage,
	hc.serial,
	hc.signing_algorithm,
	hc.subject_country,
	hc.subject_org,
	hc.subject_org_unit,
	hc.subject_common_name,
	hc.issuer_country,
	hc.issuer_org,
	hc.issuer_org_unit,
	hc.issuer_common_name,
	hcs.source,
	hcs.username
FROM
	host_certificates hc
	INNER JOIN host_certificate_sources hcs ON hc.id = hcs.host_certificate_id
WHERE
	hc.host_id = ?
	AND hc.deleted_at IS NULL`

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

func replaceHostCertsSourcesDB(ctx context.Context, tx sqlx.ExtContext, toReplaceSources []*fleet.HostCertificateRecord) error {
	if len(toReplaceSources) == 0 {
		return nil
	}

	certIDs := make([]uint, 0, len(toReplaceSources))
	for _, source := range toReplaceSources {
		certIDs = append(certIDs, source.ID)
	}

	// delete existing sources
	stmtDelete := `DELETE FROM host_certificate_sources WHERE host_certificate_id IN (?)`
	stmtDelete, args, err := sqlx.In(stmtDelete, certIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building delete host cert sources query")
	}
	if _, err := tx.ExecContext(ctx, stmtDelete, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host cert sources")
	}

	// create incoming sources
	stmtInsert := `
	INSERT INTO host_certificate_sources (
		host_certificate_id,
		source,
		username
	) VALUES %s`

	const singleRowPlaceholderCount = 3
	placeholders := make([]string, 0, len(toReplaceSources))
	args = make([]any, 0, len(toReplaceSources)*singleRowPlaceholderCount)
	for _, source := range toReplaceSources {
		placeholders = append(placeholders, "("+strings.Repeat("?,", singleRowPlaceholderCount-1)+"?)")
		args = append(args, source.ID, source.Source, source.Username)
	}

	stmtInsert = fmt.Sprintf(stmtInsert, strings.Join(placeholders, ","))
	if _, err := tx.ExecContext(ctx, stmtInsert, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting host cert sources")
	}
	return nil
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
	const singleRowPlaceholderCount = 19
	args := make([]interface{}, 0, len(certs)*singleRowPlaceholderCount)
	for _, cert := range certs {
		placeholders = append(placeholders, "("+strings.Repeat("?,", singleRowPlaceholderCount-1)+"?)")
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

func updateHostMDMManagedCertDetailsDB(ctx context.Context, tx sqlx.ExtContext, certs []*fleet.MDMManagedCertificate) error {
	if len(certs) == 0 {
		return nil
	}

	for _, certToUpdate := range certs {
		stmt := `UPDATE host_mdm_managed_certificates SET serial=?, not_valid_before=?, not_valid_after=? WHERE host_uuid = ? AND profile_uuid = ? AND ca_name=?`
		args := []interface{}{
			certToUpdate.Serial,
			certToUpdate.NotValidBefore,
			certToUpdate.NotValidAfter,
			certToUpdate.HostUUID,
			certToUpdate.ProfileUUID,
			certToUpdate.CAName,
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "updating host mdm managed certificates")
		}
	}
	return nil
}
