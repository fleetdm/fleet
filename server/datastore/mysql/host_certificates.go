package mysql

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

// hostCertificateAllowedOrderKeys defines the allowed order keys for ListHostCertificates.
// SECURITY: This prevents information disclosure via arbitrary column sorting.
var hostCertificateAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"not_valid_after": "hc.not_valid_after",
	"common_name":     "hc.common_name",
}

func isSettledStatus(s *fleet.MDMDeliveryStatus) bool {
	if s == nil {
		return false
	}
	return *s == fleet.MDMDeliveryVerified || *s == fleet.MDMDeliveryFailed
}

// hmmcBackfillGrace separates an in-flight renewal (recently NULL'd by
// reconcile, may still be matched by the pre-renewal cert in
// host_certificates) from a stuck row that needs wide-pool recovery. See
// issue #44111.
const hmmcBackfillGrace = 4 * time.Hour

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
			ds.logger.DebugContext(ctx, fmt.Sprintf("host certificates: host ID does not match provided certificate: %d %d", hostID, cert.HostID))
		}

		// Validate and truncate certificate fields
		ds.validateAndTruncateCertificateFields(ctx, hostID, cert)

		// NOTE: it is SUPER important that the sha1 sum was created with
		// sha1.Sum(...) and NOT sha1.New().Sum(...), as the latter is a wrong use
		// of the hash.Hash interface and it creates a longer byte slice that gets
		// truncated when inserted in the DB table.
		//
		// That's because sha1.Sum takes the data to checksum as argument, while
		// hash.Hash.Sum takes a byte slice to *store the checksum* of whatever was
		// written to the hash.Hash interface! Subtle but critical difference.
		//
		// The correct usage (that would also work) of hash.Hash is:
		//   h := sha1.New()
		//   h.Write(data)
		//   sha1Sum := h.Sum(nil)
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
	toInsertBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(incomingBySHA1))
	toSetSourcesBySHA1 := make(map[string][]certSourceToSet, len(incomingBySHA1))
	for sha1, incoming := range incomingBySHA1 {
		incomingSources := incomingSourcesBySHA1[sha1]
		existingSources := existingSourcesBySHA1[sha1]

		// sort by keychain (user/system) and username to ensure consistent ordering
		sliceSortFunc := func(a, b certSourceToSet) int {
			if a.Source != b.Source {
				return strings.Compare(string(a.Source), string(b.Source))
			}
			return strings.Compare(a.Username, b.Username)
		}
		slices.SortFunc(incomingSources, sliceSortFunc)
		slices.SortFunc(existingSources, sliceSortFunc)

		if !slices.Equal(incomingSources, existingSources) {
			toSetSourcesBySHA1[sha1] = incomingSources
		}

		// Check by SHA but also validity dates, as certs with dynamic SCEP challenges, the profile contents does not change other than validity dates.
		if existing, ok := existingBySHA1[sha1]; ok && existing.NotValidBefore.Equal(incoming.NotValidBefore) && existing.NotValidAfter.Equal(incoming.NotValidAfter) {
			// TODO: should we always update existing records? skipping updates reduces db load but
			// osquery is using sha1 so we consider subtleties
			ds.logger.DebugContext(ctx, fmt.Sprintf("host certificates: already exists: %s", sha1), "host_id", hostID) // TODO: silence this log after initial rollout period
		} else {
			toInsert = append(toInsert, incoming)
			toInsertBySHA1[sha1] = incoming
		}
	}

	// Update host_mdm_managed_certificates from the host's reported certs.
	// Per hmmc row we pick a cert pool: toInsertBySHA1 in the steady/in-flight
	// case (matches today's behavior — react only to NEW certs), or
	// incomingBySHA1 when the row is stuck (NULL beyond hmmcBackfillGrace AND
	// the profile is in a settled state, so widening can't re-match a
	// pre-renewal cert mid-renewal). See issue #44111.
	hostMDMManagedCertsToUpdate := make([]*fleet.MDMManagedCertificate, 0, len(toInsert))
	if len(toInsert) > 0 {
		// JOINs to the per-platform profile tables surface delivery status so
		// we don't widen the pool while a renewal is genuinely in flight.
		type hmmcRow struct {
			fleet.MDMManagedCertificate
			AppleStatus   *fleet.MDMDeliveryStatus `db:"apple_status"`
			WindowsStatus *fleet.MDMDeliveryStatus `db:"windows_status"`
		}
		var hostMDMManagedCerts []*hmmcRow
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostMDMManagedCerts, `
			SELECT
				hmmc.profile_uuid, hmmc.host_uuid, hmmc.challenge_retrieved_at,
				hmmc.not_valid_before, hmmc.not_valid_after, hmmc.type,
				hmmc.ca_name, hmmc.serial, hmmc.updated_at,
				hmap.status AS apple_status,
				hwmp.status AS windows_status
			FROM host_mdm_managed_certificates hmmc
			LEFT JOIN host_mdm_apple_profiles hmap
				ON hmap.host_uuid = hmmc.host_uuid AND hmap.profile_uuid = hmmc.profile_uuid AND hmap.operation_type = ?
			LEFT JOIN host_mdm_windows_profiles hwmp
				ON hwmp.host_uuid = hmmc.host_uuid AND hwmp.profile_uuid = hmmc.profile_uuid AND hwmp.operation_type = ?
			WHERE hmmc.host_uuid = ?
		`, fleet.MDMOperationTypeInstall, fleet.MDMOperationTypeInstall, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "list host mdm managed certs for update")
		}
		now := time.Now()
		for _, row := range hostMDMManagedCerts {
			hostMDMManagedCert := &row.MDMManagedCertificate
			// Note that we only care about proxied SCEP certificates because DigiCert are requested
			// by Fleet and stored in the DB directly, so we need not fetch them via osquery/MDM
			if !hostMDMManagedCert.Type.SupportsRenewalID() {
				continue
			}

			var pool map[string]*fleet.HostCertificateRecord
			settled := isSettledStatus(row.AppleStatus) || isSettledStatus(row.WindowsStatus)
			stuck := hostMDMManagedCert.NotValidAfter == nil &&
				now.Sub(hostMDMManagedCert.UpdatedAt) > hmmcBackfillGrace &&
				settled
			if stuck {
				pool = incomingBySHA1
			} else {
				pool = toInsertBySHA1
			}

			renewalIDString := "fleet-" + hostMDMManagedCert.ProfileUUID
			var bestMatch *fleet.HostCertificateRecord
			for _, cert := range pool {
				if !strings.Contains(cert.SubjectCommonName, renewalIDString) && !strings.Contains(cert.SubjectOrganizationalUnit, renewalIDString) {
					continue
				}
				if cert.NotValidBefore.After(now) || cert.NotValidAfter.Before(now) {
					continue
				}
				if bestMatch == nil || cert.NotValidBefore.After(bestMatch.NotValidBefore) {
					bestMatch = cert
				}
			}
			if bestMatch == nil {
				continue
			}

			// Monotonic-forward: never regress hmmc with an older cert.
			if hostMDMManagedCert.NotValidAfter != nil && !hostMDMManagedCert.NotValidAfter.Before(bestMatch.NotValidAfter) {
				continue
			}

			managedCertToUpdate := &fleet.MDMManagedCertificate{
				ProfileUUID:          hostMDMManagedCert.ProfileUUID,
				HostUUID:             hostMDMManagedCert.HostUUID,
				ChallengeRetrievedAt: hostMDMManagedCert.ChallengeRetrievedAt,
				NotValidBefore:       &bestMatch.NotValidBefore,
				NotValidAfter:        &bestMatch.NotValidAfter,
				Type:                 hostMDMManagedCert.Type,
				CAName:               hostMDMManagedCert.CAName,
				Serial:               ptr.String(fmt.Sprintf("%040s", bestMatch.Serial)),
			}
			if !hostMDMManagedCert.Equal(*managedCertToUpdate) {
				hostMDMManagedCertsToUpdate = append(hostMDMManagedCertsToUpdate, managedCertToUpdate)
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
			certIDsBySHA1, err := loadHostCertIDsForSHA1DB(ctx, tx, hostID, slices.Collect(maps.Keys(toSetSourcesBySHA1)))
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

// validateAndTruncateCertificateFields validates and truncates certificate string fields to match database schema constraints
func (ds *Datastore) validateAndTruncateCertificateFields(ctx context.Context, hostID uint, cert *fleet.HostCertificateRecord) {
	// Field length limits based on schema
	const (
		maxVarchar255 = 255
		maxCountry    = 32
	)

	// Helper function to truncate strings and log if needed
	truncateString := func(field string, value string, maxLen int) string {
		if len(value) > maxLen {
			truncated := value[:maxLen]
			err := errors.New("certificate field too long")
			ctxerr.Handle(ctx, err)
			ds.logger.ErrorContext(ctx, "truncating certificate field",
				"err", err,
				"field", field,
				"host_id", hostID,
				"original_length", len(value),
				"max_length", maxLen,
				"truncated_value", truncated,
			)
			return truncated
		}
		return value
	}

	// Validate and truncate all string fields
	cert.CommonName = truncateString("common_name", cert.CommonName, maxVarchar255)
	cert.KeyAlgorithm = truncateString("key_algorithm", cert.KeyAlgorithm, maxVarchar255)
	cert.KeyUsage = truncateString("key_usage", cert.KeyUsage, maxVarchar255)
	cert.Serial = truncateString("serial", cert.Serial, maxVarchar255)
	cert.SigningAlgorithm = truncateString("signing_algorithm", cert.SigningAlgorithm, maxVarchar255)
	cert.SubjectCountry = truncateString("subject_country", cert.SubjectCountry, maxCountry)
	cert.SubjectOrganization = truncateString("subject_org", cert.SubjectOrganization, maxVarchar255)
	cert.SubjectOrganizationalUnit = truncateString("subject_org_unit", cert.SubjectOrganizationalUnit, maxVarchar255)
	cert.SubjectCommonName = truncateString("subject_common_name", cert.SubjectCommonName, maxVarchar255)
	cert.IssuerCountry = truncateString("issuer_country", cert.IssuerCountry, maxCountry)
	cert.IssuerOrganization = truncateString("issuer_org", cert.IssuerOrganization, maxVarchar255)
	cert.IssuerOrganizationalUnit = truncateString("issuer_org_unit", cert.IssuerOrganizationalUnit, maxVarchar255)
	cert.IssuerCommonName = truncateString("issuer_common_name", cert.IssuerCommonName, maxVarchar255)
	cert.Username = truncateString("username", cert.Username, maxVarchar255)
}

func loadHostCertIDsForSHA1DB(ctx context.Context, tx sqlx.QueryerContext, hostID uint, sha1s []string) (map[string]uint, error) {
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
		hc.sha1_sum IN (?) AND hc.host_id = ?`

	var certs []*fleet.HostCertificateRecord
	stmt, args, err := sqlx.In(stmt, binarySHA1s, hostID)
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
	const fromWhereClause = `
FROM
	host_certificates hc
	INNER JOIN host_certificate_sources hcs ON hc.id = hcs.host_certificate_id
WHERE
	hc.host_id = ?
	AND hc.deleted_at IS NULL
	`

	stmt := fmt.Sprintf(`
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
	%s`, fromWhereClause)

	countStmt := fmt.Sprintf(`
    	SELECT COUNT(*) %s
    	`, fromWhereClause)

	baseArgs := []interface{}{hostID}
	stmtPaged, args, err := appendListOptionsWithCursorToSQLSecure(stmt, baseArgs, &opts, hostCertificateAllowedOrderKeys)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "apply list options")
	}

	var certs []*fleet.HostCertificateRecord
	if err := sqlx.SelectContext(ctx, tx, &certs, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting host certificates")
	}

	var metaData *fleet.PaginationMetadata
	if opts.IncludeMetadata {
		var count uint
		if err := sqlx.GetContext(ctx, tx, &count, countStmt, baseArgs...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "counting host certificates")
		}
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opts.Page > 0, TotalResults: count}
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

	// FIXME: It is entirely possible for the caller to pass duplicates in the toReplaceSources slice
	// (e.g. multiple elements with the same source and username for the same certificate ID).
	// Although this function checks against duplicates in the database, it does not deduplicate the
	// slice itself. This can lead to unique constraint violations when ths function inserts new sources.
	//
	// For Apple, it was implicitly assumed that there would be no incoming duplicates (likely based
	// on Apple KeyChain behavior). But for Windows, duplicates are commonly reported by osquery. We
	// should consider the best pattern for ensuring deduplication happens here or up the call
	// stack. For now, we are deduping Windows certs in the upstream osqquery directIngest function,
	// but that may not be the best approach if we want to guard against other potential issues.

	// Sort by host_certificate_id to ensure consistent lock ordering and prevent deadlocks
	slices.SortFunc(toReplaceSources, func(a, b *fleet.HostCertificateRecord) int {
		if a.ID != b.ID {
			if a.ID < b.ID {
				return -1
			}
			return 1
		}
		// Secondary sort by source/username for determinism
		if a.Source != b.Source {
			return strings.Compare(string(a.Source), string(b.Source))
		}
		return strings.Compare(a.Username, b.Username)
	})

	// Build unique certificate IDs for deletion (already sorted from above)
	certIDs := make([]uint, 0, len(toReplaceSources))
	var lastID uint
	for i, source := range toReplaceSources {
		// Deduplicate: only add if this ID is different from the last one
		if i == 0 || source.ID != lastID {
			certIDs = append(certIDs, source.ID)
			lastID = source.ID
		}
	}

	// Check if any sources exist before deleting to avoid unnecessary gap locks
	stmtCheck := `SELECT EXISTS(SELECT 1 FROM host_certificate_sources WHERE host_certificate_id IN (?))`
	stmtCheck, args, err := sqlx.In(stmtCheck, certIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building check host cert sources query")
	}
	var exists bool
	if err := sqlx.GetContext(ctx, tx, &exists, stmtCheck, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host cert sources exist")
	}

	// Only delete if sources exist
	if exists {
		stmtDelete := `DELETE FROM host_certificate_sources WHERE host_certificate_id IN (?)`
		stmtDelete, args, err := sqlx.In(stmtDelete, certIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building delete host cert sources query")
		}
		if _, err := tx.ExecContext(ctx, stmtDelete, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting host cert sources")
		}
	}

	// Insert new sources
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

