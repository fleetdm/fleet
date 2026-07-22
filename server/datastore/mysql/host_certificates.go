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

func isVerifiedStatus(s *fleet.MDMDeliveryStatus) bool {
	if s == nil {
		return false
	}
	return *s == fleet.MDMDeliveryVerified
}

// hmmcBackfillGrace separates an in-flight renewal (recently NULL'd by
// reconcile, may still be matched by the pre-renewal cert in
// host_certificates) from a stuck row that needs wide-pool recovery. See
// issue #44111.
const hmmcBackfillGrace = 4 * time.Hour

func (ds *Datastore) ListHostCertificates(ctx context.Context, hostID uint, opts fleet.ListOptions) ([]*fleet.HostCertificateRecord, *fleet.PaginationMetadata, error) {
	return listHostCertsDB(ctx, ds.reader(ctx), hostID, opts)
}

func (ds *Datastore) UpdateHostCertificates(ctx context.Context, hostID uint, hostUUID string, certs []*fleet.HostCertificateRecord, origin fleet.HostCertificateOrigin, observedScopes []fleet.HostCertificateScope) error {
	type certSourceToSet struct {
		Source   fleet.HostCertificateSource
		Username string
	}

	// observedScopes restricts which (source, username) scopes reconciliation may soft-delete. A nil slice means every
	// scope was observed this run (the macOS keychain model, where all keychains are always readable, and the MDM path). A non-nil slice (the Windows path) preserves certificates whose scope
	// is not listed, because osquery can only enumerate a user's certificates while that user is logged in.
	var observedSet map[certSourceToSet]struct{}
	if observedScopes != nil {
		observedSet = make(map[certSourceToSet]struct{}, len(observedScopes))
		for _, s := range observedScopes {
			observedSet[certSourceToSet{Source: s.Source, Username: s.Username}] = struct{}{}
		}
	}
	isObserved := func(s certSourceToSet) bool {
		if observedScopes == nil {
			return true
		}
		_, ok := observedSet[s]
		return ok
	}
	// desiredSources returns the source set to persist for a certificate: every source reported in the incoming batch,
	// plus any existing source whose scope was NOT observed this run. For the macOS/MDM path (nil observedScopes) every
	// scope is observed.
	desiredSources := func(incoming, existing []certSourceToSet) []certSourceToSet {
		result := append([]certSourceToSet(nil), incoming...)
		have := make(map[certSourceToSet]struct{}, len(incoming))
		for _, s := range incoming {
			have[s] = struct{}{}
		}
		for _, s := range existing {
			if isObserved(s) {
				continue
			}
			if _, ok := have[s]; ok {
				continue
			}
			result = append(result, s)
		}
		return result
	}

	incomingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(certs))
	incomingSourcesBySHA1 := make(map[string][]certSourceToSet, len(certs))
	for _, cert := range certs {
		// Tag every incoming cert with the calling ingestion source. We trust the
		// caller for this — origin scopes deletion semantics, not data integrity.
		cert.Origin = origin
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
		// Dedupe (source, username) tuples per certificate: osquery can report the same certificate scope more than once.
		srcToSet := certSourceToSet{Source: cert.Source, Username: cert.Username}
		if !slices.Contains(incomingSourcesBySHA1[normalizedSHA1], srcToSet) {
			incomingSourcesBySHA1[normalizedSHA1] = append(incomingSourcesBySHA1[normalizedSHA1], srcToSet)
		}
		incomingBySHA1[normalizedSHA1] = cert
	}

	// get existing certs for this host; we'll use the reader because we expect certs to change
	// infrequently and they will be eventually consistent
	existingCerts, _, err := listHostCertsDB(ctx, ds.reader(ctx), hostID, fleet.ListOptions{}) // requesting unpaginated results with default limit of 1 million
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host certificates for update")
	}

	// listHostCertsDB returns one row per (certificate row x source row). host_certificates has no unique index on (host_id,
	// sha1_sum), so treat the newest row as canonical: diff sources against it alone, and soft-delete the duplicates in the
	// transaction below. Newest wins.
	existingBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(existingCerts))
	for _, ec := range existingCerts {
		normalizedSHA1 := strings.ToUpper(hex.EncodeToString(ec.SHA1Sum))
		if cur, ok := existingBySHA1[normalizedSHA1]; !ok || ec.ID > cur.ID {
			existingBySHA1[normalizedSHA1] = ec
		}
	}
	existingSourcesBySHA1 := make(map[string][]certSourceToSet, len(existingBySHA1))
	existingSourceRowIDsBySHA1 := make(map[string]map[certSourceToSet]uint, len(existingBySHA1))
	var certIDsToRetire []uint      // duplicate (and below, replaced) host_certificates rows, soft-deleted in the tx (self-heal)
	var sourceRowIDsToRetire []uint // their host_certificate_sources rows, deleted
	seenRetiredCertIDs := make(map[uint]struct{})
	for _, ec := range existingCerts {
		normalizedSHA1 := strings.ToUpper(hex.EncodeToString(ec.SHA1Sum))
		winner := existingBySHA1[normalizedSHA1]
		if ec.ID != winner.ID {
			if _, ok := seenRetiredCertIDs[ec.ID]; !ok {
				seenRetiredCertIDs[ec.ID] = struct{}{}
				certIDsToRetire = append(certIDsToRetire, ec.ID)
			}
			sourceRowIDsToRetire = append(sourceRowIDsToRetire, ec.SourceID)
			continue
		}
		srcToSet := certSourceToSet{Source: ec.Source, Username: ec.Username}
		existingSourcesBySHA1[normalizedSHA1] = append(existingSourcesBySHA1[normalizedSHA1], srcToSet)
		if existingSourceRowIDsBySHA1[normalizedSHA1] == nil {
			existingSourceRowIDsBySHA1[normalizedSHA1] = make(map[certSourceToSet]uint)
		}
		existingSourceRowIDsBySHA1[normalizedSHA1][srcToSet] = ec.SourceID
	}

	toInsert := make([]*fleet.HostCertificateRecord, 0, len(incomingBySHA1))
	toInsertBySHA1 := make(map[string]*fleet.HostCertificateRecord, len(incomingBySHA1))
	toSetSourcesBySHA1 := make(map[string][]certSourceToSet, len(incomingBySHA1))
	// Existing mdm-origin rows that osquery is also reporting. One-way
	// downgrade: osquery sees a strict superset of the keychain, so dual
	// observation isn't evidence of MDM delivery.
	var toDowngrade []uint
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
		// Persist the reported sources plus any existing source in a scope we did not observe (preserving a logged-off
		// user's source). For the macOS/MDM path this is exactly incomingSources.
		newSources := desiredSources(incomingSources, existingSources)
		slices.SortFunc(newSources, sliceSortFunc)
		slices.SortFunc(existingSources, sliceSortFunc)

		if !slices.Equal(newSources, existingSources) {
			toSetSourcesBySHA1[sha1] = newSources
		}

		existing, hasExisting := existingBySHA1[sha1]
		if hasExisting && existing.Origin == fleet.HostCertificateOriginMDM && incoming.Origin == fleet.HostCertificateOriginOsquery {
			toDowngrade = append(toDowngrade, existing.ID)
		}

		// Check by SHA but also validity dates, as certs with dynamic SCEP challenges, the profile contents does not change other than validity dates.
		if hasExisting && existing.NotValidBefore.Equal(incoming.NotValidBefore) && existing.NotValidAfter.Equal(incoming.NotValidAfter) {
			// TODO: should we always update existing records? skipping updates reduces db load but
			// osquery is using sha1 so we consider subtleties
			ds.logger.DebugContext(ctx, fmt.Sprintf("host certificates: already exists: %s", sha1), "host_id", hostID) // TODO: silence this log after initial rollout period
		} else {
			toInsert = append(toInsert, incoming)
			toInsertBySHA1[sha1] = incoming
		}
	}

	// Update host_mdm_managed_certificates from the host's reported certs.
	// Runs on every UpdateHostCertificates call so a stuck row (renewal cert
	// already in host_certificates but never matched) recovers even when this
	// call has no toInsert. Per hmmc row: pool = incomingBySHA1 when stuck
	// (NULL past hmmcBackfillGrace AND profile 'verified'), else toInsertBySHA1.
	// See issue #44111.
	hostMDMManagedCertsToUpdate := make([]*fleet.MDMManagedCertificate, 0, len(toInsert))
	// JOINs to the per-platform profile tables surface delivery status so we
	// don't widen the pool while a renewal is genuinely in flight.
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
		// Skip CA types that don't carry a renewal-ID marker — today only
		// DigiCert, which is server-issued and managed without matching
		// against ingested certs. Empty/NULL `Type` (rows created by the
		// non-proxied insert path below) IS eligible.
		if hostMDMManagedCert.Type != "" && !hostMDMManagedCert.Type.SupportsRenewalID() {
			continue
		}

		verified := isVerifiedStatus(row.AppleStatus) || isVerifiedStatus(row.WindowsStatus)
		stuck := hostMDMManagedCert.NotValidAfter == nil &&
			now.Sub(hostMDMManagedCert.UpdatedAt) > hmmcBackfillGrace &&
			verified

		var pool map[string]*fleet.HostCertificateRecord
		switch {
		case stuck:
			pool = incomingBySHA1
		case len(toInsertBySHA1) > 0:
			pool = toInsertBySHA1
		default:
			continue
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

	// Non-proxied insert path: for each profile installed on this host
	// without an existing host_mdm_managed_certificates row, see if any
	// incoming cert's Subject carries the `fleet-<profile_uuid>` marker.
	// If so, create the row from the cert's metadata. This activates
	// renewal for ACME / non-proxied SCEP flows where Fleet isn't in the
	// issuance path so no row gets created at issuance time.
	hostMDMManagedCertsToInsert := make([]*fleet.MDMManagedCertificate, 0, len(incomingBySHA1))
	if len(incomingBySHA1) > 0 {
		existingProfileUUIDs := make(map[string]struct{}, len(hostMDMManagedCerts))
		for _, row := range hostMDMManagedCerts {
			existingProfileUUIDs[row.ProfileUUID] = struct{}{}
		}
		var candidateProfileUUIDs []string
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &candidateProfileUUIDs, `
			SELECT profile_uuid FROM host_mdm_apple_profiles
			WHERE host_uuid = ? AND operation_type = ?
			UNION
			SELECT profile_uuid FROM host_mdm_windows_profiles
			WHERE host_uuid = ? AND operation_type = ?`,
			hostUUID, fleet.MDMOperationTypeInstall,
			hostUUID, fleet.MDMOperationTypeInstall,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "list candidate profile UUIDs for managed cert insert")
		}
		for _, profileUUID := range candidateProfileUUIDs {
			if _, exists := existingProfileUUIDs[profileUUID]; exists {
				continue
			}
			renewalIDString := "fleet-" + profileUUID
			var bestMatch *fleet.HostCertificateRecord
			for _, cert := range incomingBySHA1 {
				if !strings.Contains(cert.SubjectCommonName, renewalIDString) &&
					!strings.Contains(cert.SubjectOrganizationalUnit, renewalIDString) {
					continue
				}
				// Skip certs outside their validity window: a device may
				// still be reporting a just-expired cert alongside its
				// renewal, and latching onto it would seed the row with
				// backward-pointing dates.
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
			// Use a fixed sentinel for ca_name on non-proxied rows.
			// Proxied flows set ca_name from Fleet-controlled CA
			// registration (stable across renewals); deriving it from
			// the cert's Issuer CN would drift if the upstream CA ever
			// renames. The cert's actual issuer is available in
			// host_certificates for support visibility.
			// Type is written as NULL by insertHostMDMManagedCertDB —
			// Fleet wasn't in the issuance path so it doesn't know the
			// CA type. The struct's Type field is left unset.
			hostMDMManagedCertsToInsert = append(hostMDMManagedCertsToInsert, &fleet.MDMManagedCertificate{
				HostUUID:       hostUUID,
				ProfileUUID:    profileUUID,
				NotValidBefore: &bestMatch.NotValidBefore,
				NotValidAfter:  &bestMatch.NotValidAfter,
				CAName:         "non_proxied",
				Serial:         ptr.String(fmt.Sprintf("%040s", bestMatch.Serial)),
			})
		}
	}

	toDelete := make([]uint, 0, len(existingBySHA1))
	for sha1, existing := range existingBySHA1 {
		if _, ok := incomingBySHA1[sha1]; ok {
			// present in the incoming batch, reconciled above
			continue
		}
		// Source-scoped delete: only remove rows whose origin matches the calling ingestion source.
		if existing.Origin != origin {
			continue
		}
		// Preserve sources in scopes we did not observe this run (e.g. a logged-off Windows user) and drop the
		// observed-but-no-longer-reported ones.
		existingSources := existingSourcesBySHA1[sha1]
		preserved := desiredSources(nil, existingSources)
		switch {
		case len(preserved) == 0:
			toDelete = append(toDelete, existing.ID)
		case len(preserved) != len(existingSources):
			toSetSourcesBySHA1[sha1] = preserved
		}
	}

	// Whether osquery could read at least one user's certificate store in this report. SINGLE-USER ASSUMPTION: we treat
	// "any user cert observed" as "the target user's store was readable", which holds when the device has one primary
	// user.
	anyUserCertObserved := slices.ContainsFunc(certs, func(c *fleet.HostCertificateRecord) bool {
		return c.Source == fleet.UserHostCertificate
	})

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := insertHostCertsDB(ctx, tx, toInsert); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host certs")
		}

		// Self-heal: retire duplicate and replaced cert rows (and their source rows) before resolving canonical ids,
		// so a previously poisoned host converges back to one active row per certificate.
		if err := softDeleteHostCertsDB(ctx, tx, hostID, certIDsToRetire); err != nil {
			return ctxerr.Wrap(ctx, err, "soft delete duplicate host certs")
		}

		// Compute the precise source-row changes: delete only rows that are stale (by primary key) and insert only
		// tuples that are missing. The previous implementation deleted by host_certificate_id ranges and re-inserted
		// the full set; under REPEATABLE READ such a range DELETE takes next-key/gap locks on the unique index
		// (including the supremum record when the list contains a just-inserted cert id), which serialized concurrent
		// hosts' inserts during ingestion waves (#49705).
		staleSourceRowIDs := append([]uint(nil), sourceRowIDsToRetire...)
		var sourceRowsToInsert []*fleet.HostCertificateRecord
		if len(toSetSourcesBySHA1) > 0 {
			// must reload the DB IDs to insert the host_certificate_sources rows
			certIDsBySHA1, err := loadHostCertIDsForSHA1DB(ctx, tx, hostID, slices.Collect(maps.Keys(toSetSourcesBySHA1)))
			if err != nil {
				return ctxerr.Wrap(ctx, err, "load host certs ids")
			}

			for sha1, desired := range toSetSourcesBySHA1 {
				certID, ok := certIDsBySHA1[sha1]
				if !ok {
					// cert row not found on the writer (e.g. deleted concurrently); nothing to attach sources to
					continue
				}
				existingRowIDs := existingSourceRowIDsBySHA1[sha1]
				desiredSet := make(map[certSourceToSet]struct{}, len(desired))
				for _, s := range desired {
					desiredSet[s] = struct{}{}
				}
				for s, rowID := range existingRowIDs {
					if _, ok := desiredSet[s]; !ok {
						staleSourceRowIDs = append(staleSourceRowIDs, rowID)
					}
				}
				for _, s := range desired {
					if _, ok := existingRowIDs[s]; ok {
						continue
					}
					sourceRowsToInsert = append(sourceRowsToInsert, &fleet.HostCertificateRecord{
						ID:       certID,
						Source:   s.Source,
						Username: s.Username,
					})
				}
			}
		}
		if err := deleteHostCertSourceRowsDB(ctx, tx, staleSourceRowIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "delete stale host cert sources")
		}
		if err := insertHostCertSourceRowsDB(ctx, tx, sourceRowsToInsert); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host cert sources")
		}

		if err := softDeleteHostCertsDB(ctx, tx, hostID, toDelete); err != nil {
			return ctxerr.Wrap(ctx, err, "soft delete host certs")
		}

		if err := downgradeHostCertsOriginToOsqueryDB(ctx, tx, hostID, toDowngrade); err != nil {
			return ctxerr.Wrap(ctx, err, "downgrade host certs origin")
		}

		if err := updateHostMDMManagedCertDetailsDB(ctx, tx, hostMDMManagedCertsToUpdate); err != nil {
			return ctxerr.Wrap(ctx, err, "update host mdm managed cert details")
		}

		if err := insertHostMDMManagedCertDB(ctx, tx, hostMDMManagedCertsToInsert); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host mdm managed cert rows")
		}

		// A proxied Windows SCEP profile sits in "verifying" until the certificate it requested is observed on the host.
		// The managed-cert updates above set not_valid_after/serial when a reported cert matched the profile's
		// renewal-ID marker, so a matched-and-valid managed-cert row is the signal that the certificate landed. Flip
		// those profiles to "verified" (self-healing any that were "failed" from a proxy-observed error).
		if err := verifyWindowsSCEPProfilesFromObservedCertsDB(ctx, tx, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "verify windows scep profiles from observed certs")
		}

		// Backstop: a proxied Windows SCEP profile that never gets its certificate would otherwise sit in
		// "verifying" forever. Once the grace period has elapsed and this report proves we could read the store
		// where the certificate belongs but it isn't there, fail it. Runs after the flip above, so anything still
		// "verifying" here has no observed certificate.
		if err := failStuckWindowsSCEPProfilesDB(ctx, tx, hostUUID, anyUserCertObserved); err != nil {
			return ctxerr.Wrap(ctx, err, "fail stuck windows scep profiles")
		}
		return nil
	})
}

// windowsSCEPVerificationGracePeriod is how long a proxied Windows SCEP profile may stay in "verifying" (measured from
// the host's ACK of the profile, i.e. host_mdm_windows_profiles.updated_at) before an ingest that proves the relevant
// certificate store was readable, yet lacks the certificate, is treated as a failure.
const windowsSCEPVerificationGracePeriod = time.Hour

// windowsSCEPCertNotFoundDetail is the failure detail recorded when the verification backstop fires.
const windowsSCEPCertNotFoundDetail = "Fleet did not detect the SCEP certificate on the host after profile was delivered."

// failStuckWindowsSCEPProfilesDB is the verification backstop for proxied Windows SCEP profiles. It runs on certificate
// ingestion (so an offline host, or one whose agent can't enumerate certificates, never ingests and is never failed)
// and marks a profile "failed" only when we have positive evidence the certificate is missing:
//
//   - Device-scoped profiles (SyncML uses the ./Device SCEP node): the LocalMachine store is always readable when
//     osquery reports, so any ingest past the grace period with the certificate still absent is a genuine failure.
//   - User-scoped profiles (SyncML uses the ./User SCEP node): the certificate lives in a user's store, which osquery
//     can read only while that user is logged in. We fail only when this report includes at least one user
//     certificate, proving a user store was readable. SINGLE-USER ASSUMPTION: Fleet does not track which user a
//     ./User Windows profile targets, so we assume the device has one primary user.
func failStuckWindowsSCEPProfilesDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string, anyUserCertObserved bool) error {
	caTypes := fleet.ListCATypesWithRenewalIDSupport()
	caTypeStrs := make([]string, 0, len(caTypes))
	for _, t := range caTypes {
		caTypeStrs = append(caTypeStrs, string(t))
	}
	graceSeconds := int(windowsSCEPVerificationGracePeriod.Seconds())

	var query string
	var args []any
	if anyUserCertObserved {
		// System and user scope observed. No need to inspect the profile's SyncML scope.
		query = `
			UPDATE host_mdm_windows_profiles hwmp
			JOIN host_mdm_managed_certificates hmmc
				ON hmmc.host_uuid = hwmp.host_uuid AND hmmc.profile_uuid = hwmp.profile_uuid
			SET hwmp.status = ?, hwmp.detail = ?
			WHERE hwmp.host_uuid = ?
				AND hwmp.operation_type = ?
				AND hwmp.status = ?
				AND hmmc.type IN (?)
				AND hwmp.updated_at < DATE_SUB(NOW(), INTERVAL ? SECOND)`
		args = []any{
			fleet.MDMDeliveryFailed, windowsSCEPCertNotFoundDetail, hostUUID, fleet.MDMOperationTypeInstall,
			fleet.MDMDeliveryVerifying, caTypeStrs, graceSeconds,
		}
	} else {
		// Only the LocalMachine store is provably readable this run. Restrict to device-scoped profiles (SyncML
		// without a ./User SCEP node); a user-scoped certificate may just be waiting for its user to log in.
		query = `
			UPDATE host_mdm_windows_profiles hwmp
			JOIN host_mdm_managed_certificates hmmc
				ON hmmc.host_uuid = hwmp.host_uuid AND hmmc.profile_uuid = hwmp.profile_uuid
			JOIN mdm_windows_configuration_profiles cp
				ON cp.profile_uuid = hwmp.profile_uuid
			SET hwmp.status = ?, hwmp.detail = ?
			WHERE hwmp.host_uuid = ?
				AND hwmp.operation_type = ?
				AND hwmp.status = ?
				AND hmmc.type IN (?)
				AND hwmp.updated_at < DATE_SUB(NOW(), INTERVAL ? SECOND)
				AND cp.syncml NOT LIKE ?`
		args = []any{
			fleet.MDMDeliveryFailed, windowsSCEPCertNotFoundDetail, hostUUID, fleet.MDMOperationTypeInstall,
			fleet.MDMDeliveryVerifying, caTypeStrs, graceSeconds, "%/User/Vendor/MSFT/ClientCertificateInstall/SCEP%",
		}
	}

	stmt, inArgs, err := sqlx.In(query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building windows scep backstop query")
	}
	if _, err := tx.ExecContext(ctx, stmt, inArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "failing stuck windows scep profiles")
	}
	return nil
}

// verifyWindowsSCEPProfilesFromObservedCertsDB flips a host's proxied Windows SCEP install profiles from "verifying" or
// "failed" to "verified" once their managed-certificate row shows a certificate was observed (its serial/validity dates
// were populated by the renewal-ID matcher in UpdateHostCertificates). Current validity is intentionally NOT required:
// observing that the CA issued a certificate matching this profile's renewal-ID proves the enrollment succeeded, so we
// mark it verified regardless of the certificate's lifetime. A short-lived certificate that has since expired is a
// renewal concern (handled by RenewMDMManagedCertificates), not a verification failure.
func verifyWindowsSCEPProfilesFromObservedCertsDB(ctx context.Context, tx sqlx.ExtContext, hostUUID string) error {
	caTypes := fleet.ListCATypesWithRenewalIDSupport()
	caTypeStrs := make([]string, 0, len(caTypes))
	for _, t := range caTypes {
		caTypeStrs = append(caTypeStrs, string(t))
	}
	stmt, args, err := sqlx.In(`
		UPDATE host_mdm_windows_profiles hwmp
		JOIN host_mdm_managed_certificates hmmc
			ON hmmc.host_uuid = hwmp.host_uuid AND hmmc.profile_uuid = hwmp.profile_uuid
		SET hwmp.status = ?, hwmp.detail = ''
		WHERE hwmp.host_uuid = ?
			AND hwmp.operation_type = ?
			AND hwmp.status IN (?, ?)
			AND hmmc.type IN (?)
			AND hmmc.not_valid_after IS NOT NULL`,
		fleet.MDMDeliveryVerified, hostUUID, fleet.MDMOperationTypeInstall,
		fleet.MDMDeliveryVerifying, fleet.MDMDeliveryFailed, caTypeStrs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building windows scep verify query")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "flipping windows scep profiles to verified")
	}
	return nil
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
		hc.sha1_sum IN (?) AND hc.host_id = ? AND hc.deleted_at IS NULL`

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
		// Keep the newest row when duplicates exist (matching the canonical-row selection in UpdateHostCertificates)
		// so sources attach to the row that survives duplicate healing instead of an arbitrary duplicate.
		if curID, ok := certIDsBySHA1[normalizedSHA1]; !ok || cert.ID > curID {
			certIDsBySHA1[normalizedSHA1] = cert.ID
		}
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
	hc.origin,
	hcs.id AS source_id,
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

// deleteHostCertSourceRowsDB deletes host_certificate_sources rows by primary key. Primary-key deletes of known-existing
// rows take record locks only, unlike deletes over host_certificate_id ranges, whose next-key/gap locks on the unique
// index blocked concurrent hosts' inserts (#49705).
func deleteHostCertSourceRowsDB(ctx context.Context, tx sqlx.ExtContext, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	// Sort for deterministic lock ordering across concurrent transactions.
	slices.Sort(ids)
	stmt, args, err := sqlx.In(`DELETE FROM host_certificate_sources WHERE id IN (?)`, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building delete host cert sources query")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host cert sources")
	}
	return nil
}

// insertHostCertSourceRowsDB inserts the given (host_certificate_id, source, username) tuples. The caller passes only
// tuples it believes are missing; ON DUPLICATE KEY UPDATE is a no-op guard for concurrent ingestions of the same report
// for one host (osquery resends results when a write times out), where failing the whole transaction with a
// duplicate-key error would 500 the endpoint and trigger yet another resend.
func insertHostCertSourceRowsDB(ctx context.Context, tx sqlx.ExtContext, rows []*fleet.HostCertificateRecord) error {
	if len(rows) == 0 {
		return nil
	}
	// Sort by (host_certificate_id, source, username) for deterministic lock ordering across concurrent transactions.
	slices.SortFunc(rows, func(a, b *fleet.HostCertificateRecord) int {
		if a.ID != b.ID {
			if a.ID < b.ID {
				return -1
			}
			return 1
		}
		if a.Source != b.Source {
			return strings.Compare(string(a.Source), string(b.Source))
		}
		return strings.Compare(a.Username, b.Username)
	})

	const singleRowPlaceholderCount = 3
	placeholders := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*singleRowPlaceholderCount)
	for _, row := range rows {
		placeholders = append(placeholders, "("+strings.Repeat("?,", singleRowPlaceholderCount-1)+"?)")
		args = append(args, row.ID, row.Source, row.Username)
	}
	stmt := fmt.Sprintf(`
	INSERT INTO host_certificate_sources (
		host_certificate_id,
		source,
		username
	) VALUES %s
	ON DUPLICATE KEY UPDATE host_certificate_id = host_certificate_sources.host_certificate_id`,
		strings.Join(placeholders, ","))
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
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
	issuer_common_name,
	origin
) VALUES %s`

	placeholders := make([]string, 0, len(certs))
	const singleRowPlaceholderCount = 20
	args := make([]interface{}, 0, len(certs)*singleRowPlaceholderCount)
	for _, cert := range certs {
		placeholders = append(placeholders, "("+strings.Repeat("?,", singleRowPlaceholderCount-1)+"?)")
		origin := cert.Origin
		if origin == "" {
			origin = fleet.HostCertificateOriginOsquery
		}
		args = append(args,
			cert.HostID, cert.SHA1Sum, cert.NotValidBefore, cert.NotValidAfter, cert.CertificateAuthority, cert.CommonName,
			cert.KeyAlgorithm, cert.KeyStrength, cert.KeyUsage, cert.Serial, cert.SigningAlgorithm,
			cert.SubjectCountry, cert.SubjectOrganization, cert.SubjectOrganizationalUnit, cert.SubjectCommonName,
			cert.IssuerCountry, cert.IssuerOrganization, cert.IssuerOrganizationalUnit, cert.IssuerCommonName,
			origin)
	}

	stmt = fmt.Sprintf(stmt, strings.Join(placeholders, ","))

	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting host certificates")
	}
	return nil
}

func downgradeHostCertsOriginToOsqueryDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	stmt := `UPDATE host_certificates SET origin = ? WHERE host_id = ? AND id IN (?) AND origin = ?`
	stmt, args, err := sqlx.In(stmt, fleet.HostCertificateOriginOsquery, hostID, ids, fleet.HostCertificateOriginMDM)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building downgrade origin query")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "downgrading host certificate origin")
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

// softDeleteMDMHostCertsDB clears all MDM-origin cert rows on unenroll, where
// nothing else will: no more CertificateList, and osquery can't see
// hardware-bound ACME certs.
func softDeleteMDMHostCertsDB(ctx context.Context, tx sqlx.ExtContext, hostID uint) error {
	const stmt = `UPDATE host_certificates SET deleted_at = NOW(6) WHERE host_id = ? AND origin = ? AND deleted_at IS NULL`
	if _, err := tx.ExecContext(ctx, stmt, hostID, fleet.HostCertificateOriginMDM); err != nil {
		return ctxerr.Wrap(ctx, err, "soft deleting mdm host certificates")
	}
	return nil
}

// See the Datastore interface for contract. Batched to bound lock scope: a
// host can have many MDM certs, so a mass unenroll could otherwise lock a
// large set in one statement.
func (ds *Datastore) SoftDeleteMDMHostCertificatesForUnenrolledHosts(ctx context.Context) (int64, error) {
	const batchSize = 1000
	// Derived-table wrap is required: MySQL forbids referencing the updated
	// table directly in the subquery's FROM.
	const stmt = `
		UPDATE host_certificates
		SET deleted_at = NOW(6)
		WHERE deleted_at IS NULL AND id IN (
			SELECT id FROM (
				SELECT hc.id
				FROM host_certificates hc
				JOIN host_mdm hm ON hm.host_id = hc.host_id AND hm.enrolled = 0
				WHERE hc.origin = ? AND hc.deleted_at IS NULL
				ORDER BY hc.id
				LIMIT ?
			) AS batch
		)`
	var total int64
	for {
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, fleet.HostCertificateOriginMDM, batchSize)
		if err != nil {
			return total, ctxerr.Wrap(ctx, err, "soft-delete mdm host certificates for unenrolled hosts")
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, ctxerr.Wrap(ctx, err, "rows affected for mdm host certificates sweep")
		}
		total += n
		if n < batchSize {
			break
		}
	}
	return total, nil
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

// insertHostMDMManagedCertDB creates host_mdm_managed_certificates rows for
// non-proxied SCEP/ACME flows discovered via cert ingestion. type is always
// written as NULL because Fleet wasn't in the issuance path and doesn't know
// the CA type. Uses INSERT IGNORE so a row created concurrently by another
// transaction (e.g., a SCEP proxy issuance) doesn't cause a duplicate-key
// error here — the matcher's UPDATE pass picks up that row on the next
// ingestion call.
func insertHostMDMManagedCertDB(ctx context.Context, tx sqlx.ExtContext, certs []*fleet.MDMManagedCertificate) error {
	if len(certs) == 0 {
		return nil
	}
	for _, c := range certs {
		_, err := tx.ExecContext(ctx, `
			INSERT IGNORE INTO host_mdm_managed_certificates
				(host_uuid, profile_uuid, ca_name, type,
				 not_valid_before, not_valid_after, serial)
			VALUES (?, ?, ?, NULL, ?, ?, ?)`,
			c.HostUUID, c.ProfileUUID, c.CAName,
			c.NotValidBefore, c.NotValidAfter, c.Serial)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert host mdm managed certificate")
		}
	}
	return nil
}
