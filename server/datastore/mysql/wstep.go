package mysql

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
)

// CertStore implements storage tasks associated with MS-WSTEP messages in the MS-MDE2
// protocol. It is implemented by fleet.Datastore.
var _ microsoft_mdm.CertStore = (*Datastore)(nil)

// WSTEPStoreCertificate stores a certificate under the given name.
//
// If the provided certificate has empty crt.Subject.CommonName,
// then the hex sha256 of the crt.Raw is used as name.
func (ds *Datastore) WSTEPStoreCertificate(ctx context.Context, name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" {
		name = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}
	certPEM := certificate.EncodeCertPEM(crt)
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO wstep_certificates
    (serial, name, not_valid_before, not_valid_after, certificate_pem)
VALUES
    (?, ?, ?, ?, ?)`,
		crt.SerialNumber.Int64(),
		name,
		crt.NotBefore,
		crt.NotAfter,
		certPEM,
	)
	return err
}

// WSTEPNewSerial allocates and returns a new (increasing) serial number.
func (ds *Datastore) WSTEPNewSerial(ctx context.Context) (*big.Int, error) {
	result, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO wstep_serials () VALUES ();`)
	if err != nil {
		return nil, err
	}
	lid, err := result.LastInsertId() // TODO: ok if sequential and not random?
	if err != nil {
		return nil, err
	}
	// TODO: check maxSerialNumber?
	return big.NewInt(lid), nil
}

func (ds *Datastore) WSTEPAssociateCertHash(ctx context.Context, deviceUUID string, hash string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO wstep_cert_auth_associations (id, sha256) VALUES (?, ?) AS new
ON DUPLICATE KEY
UPDATE sha256 = new.sha256;`,
		deviceUUID,
		strings.ToUpper(hash), // TODO: confirm if this is necessary
	)
	return err
}
