package depot

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

const maxCommonNameLength = 64

// ConditionalAccessSCEPDepot is a MySQL-backed SCEP certificate depot for conditional access.
type ConditionalAccessSCEPDepot struct {
	db     *sqlx.DB
	ds     fleet.Datastore
	logger log.Logger
	config *config.FleetConfig
}

var _ depot.Depot = (*ConditionalAccessSCEPDepot)(nil)

// NewConditionalAccessSCEPDepot creates and returns a *ConditionalAccessSCEPDepot.
func NewConditionalAccessSCEPDepot(db *sqlx.DB, ds fleet.Datastore, logger log.Logger, cfg *config.FleetConfig) (*ConditionalAccessSCEPDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &ConditionalAccessSCEPDepot{
		db:     db,
		ds:     ds,
		logger: logger,
		config: cfg,
	}, nil
}

// CA returns the CA's certificate and private key.
func (d *ConditionalAccessSCEPDepot) CA(_ []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := assets.KeyPair(context.Background(), d.ds,
		fleet.MDMAssetConditionalAccessCACert,
		fleet.MDMAssetConditionalAccessCAKey)
	if err != nil {
		return nil, nil, fmt.Errorf("getting assets: %w", err)
	}

	pk, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("private key not in RSA format")
	}

	return []*x509.Certificate{cert.Leaf}, pk, nil
}

// Serial allocates and returns a new (increasing) serial number.
func (d *ConditionalAccessSCEPDepot) Serial() (*big.Int, error) {
	// Insert an empty row to generate a new auto-incremented serial number
	result, err := d.db.Exec(`INSERT INTO conditional_access_scep_serials () VALUES ();`)
	if err != nil {
		return nil, err
	}
	lid, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	return big.NewInt(lid), nil
}

// HasCN returns whether the given certificate exists in the depot.
// For conditional access SCEP, renewal is not supported, so this always returns false.
func (d *ConditionalAccessSCEPDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	// Not used - no renewal support for conditional access
	return false, nil
}

// Put stores a certificate under the given name.
// The certificate must contain a SAN URI with the device UUID in the format:
// urn:device:apple:uuid:<uuid>
// The UUID is used to look up the host in Fleet, and the certificate is only issued
// if the host exists. Old certificates for the same host are automatically revoked.
func (d *ConditionalAccessSCEPDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" || len(crt.Subject.CommonName) > maxCommonNameLength {
		return errors.New("common name empty or too long")
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}

	// Extract UUID from SAN URI
	// Expected format: urn:device:apple:uuid:<device-uuid>
	uuid := extractUUIDFromCert(crt)
	if uuid == "" {
		return errors.New("no device UUID found in certificate SAN URI")
	}

	// Look up host BEFORE storing certificate
	host, err := d.ds.HostByIdentifier(context.Background(), uuid)
	if err != nil {
		return fmt.Errorf("host not found for UUID %s: %w", uuid, err)
	}

	// Apply rate limiting if configured
	cooldown := d.config.Osquery.EnrollCooldown
	if cooldown > 0 {
		existingCertCreatedAt, err := d.ds.GetConditionalAccessCertCreatedAtByHostID(context.Background(), host.ID)
		switch {
		case err != nil && !fleet.IsNotFound(err):
			return fmt.Errorf("checking existing certificate: %w", err)
		case err == nil:
			// Certificate exists, check if rate limit applies
			if time.Since(*existingCertCreatedAt) < cooldown {
				return backoff.Permanent(ctxerr.Errorf(context.Background(), "host %s requesting certificates too often", uuid))
			}
		}
		// If certificate doesn't exist or rate limit doesn't apply, continue
	}

	certPEM := certificate.EncodeCertPEM(crt)

	// Insert new certificate - following industry best practice of issuing new cert
	// before revoking old ones. For zero-downtime rotation, old certificates are NOT
	// immediately revoked, allowing a grace period where both are valid.
	//
	// This prevents authentication failures when:
	// - Network delays in delivering the new certificate to the client
	// - Client is offline during certificate rotation (client will request new cert when it comes back online)
	_, err = d.db.ExecContext(context.Background(), `
		INSERT INTO conditional_access_scep_certificates
			(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem)
		VALUES
			(?, ?, ?, ?, ?, ?)`,
		crt.SerialNumber.Int64(),
		host.ID,
		name,
		crt.NotBefore,
		crt.NotAfter,
		certPEM,
	)
	if err != nil {
		return err
	}

	level.Info(d.logger).Log(
		"msg", "stored conditional access certificate",
		"cn", name,
		"serial", crt.SerialNumber.Int64(),
		"host_id", host.ID,
		"uuid", uuid,
	)

	return nil
}

// extractUUIDFromCert extracts the device UUID from the SAN URI field.
// Expected format: urn:device:apple:uuid:<uuid>
// Returns the UUID portion or empty string if not found.
func extractUUIDFromCert(crt *x509.Certificate) string {
	const prefix = "urn:device:apple:uuid:"
	for _, uri := range crt.URIs {
		// Check if this is a device URI (urn:device:apple:uuid:...)
		uriStr := uri.String()
		if strings.HasPrefix(uriStr, prefix) {
			return strings.TrimPrefix(uriStr, prefix)
		}
	}
	return ""
}
