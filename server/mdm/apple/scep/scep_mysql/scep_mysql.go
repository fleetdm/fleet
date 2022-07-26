// Package scep_mysql implements a MySQL SCEP certificate depot.
package scep_mysql

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/micromdm/scep/v2/depot"
)

// ErrNotFound is returned when a requested resource is not found.
//
// TODO(lucas): Define this at the Depot interface level.
var ErrNotFound = errors.New("resource not found")

// Schema holds the MySQL schema for SCEP depot storage.
//
//go:embed schema.sql
var Schema string

// MySQLDepot is a MySQL-backed SCEP certificate depot.
type MySQLDepot struct {
	db *sql.DB

	// caCrt holds the CA's certificate.
	caCrt *x509.Certificate
	// caKey holds the (decrypted) CA private key.
	caKey *rsa.PrivateKey
}

var _ depot.Depot = (*MySQLDepot)(nil)

// NewMySQLDepot creates and returns a MySQLDepot.
//
// CreateCA or LoadCA should be called before any other operation.
func NewMySQLDepot(db *sql.DB) (*MySQLDepot, error) {
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &MySQLDepot{
		db: db,
	}, nil
}

// CA returns the CA's certificate and private key.
func (d *MySQLDepot) CA(_ []byte) ([]*x509.Certificate, *rsa.PrivateKey, error) {
	return []*x509.Certificate{d.caCrt}, d.caKey, nil
}

// Serial allocates and returns a new (increasing) serial number.
func (d *MySQLDepot) Serial() (*big.Int, error) {
	result, err := d.db.Exec(`INSERT INTO scep_serials () VALUES ();`)
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
//
// TODO(lucas): Implement and use allowTime and revokeOldCertificate.
// - allowTime are the maximum days before expiration to allow clients to do certificate renewal.
// - revokeOldCertificate specifies whether to revoke the old certificate once renewed.
func (d *MySQLDepot) HasCN(cn string, allowTime int, cert *x509.Certificate, revokeOldCertificate bool) (bool, error) {
	var ct int
	row := d.db.QueryRow(`SELECT COUNT(*) FROM scep_certificates WHERE name = ?`, cn)
	if err := row.Scan(&ct); err != nil {
		return false, err
	}
	return ct >= 1, nil
}

// LoadCA loads the CA's certificate and private key into the MySQLDepot and returns them.
//
// Returns ErrNotFound if they do not exist.
func (d *MySQLDepot) LoadCA(pass []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	var pemCert, pemKey []byte
	err := d.db.QueryRow(`
SELECT
    certificate_pem, key_pem
FROM
    scep_certificates
INNER JOIN scep_ca_keys
	ON scep_certificates.serial = scep_ca_keys.serial
WHERE
    scep_certificates.serial = 1;`,
	).Scan(&pemCert, &pemKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	block, _ := pem.Decode(pemCert)
	if block.Type != "CERTIFICATE" {
		return nil, nil, errors.New("PEM block not a certificate")
	}
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	block, _ = pem.Decode(pemKey)
	if !x509.IsEncryptedPEMBlock(block) {
		return nil, nil, errors.New("PEM block not encrypted")
	}
	keyBytes, err := x509.DecryptPEMBlock(block, pass)
	if err != nil {
		return nil, nil, err
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return nil, nil, err
	}
	d.caCrt = crt
	d.caKey = key
	return crt, key, nil
}

// CreateCA creates a self-signed CA certificate and returns the certificate and its private key.
// It sets the created certificate and private key into the MySQLDepot.
func (d *MySQLDepot) CreateCA(pass []byte, years int, cn, org, orgUnit, country string) (*x509.Certificate, *rsa.PrivateKey, error) {
	_, err := d.db.Exec(`INSERT IGNORE INTO scep_serials (serial) VALUES (1)`)
	if err != nil {
		return nil, nil, err
	}
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	caCert := depot.NewCACert(
		depot.WithYears(years),
		depot.WithCommonName(cn),
		depot.WithOrganization(org),
		depot.WithOrganizationalUnit(orgUnit),
		depot.WithCountry(country),
	)
	crtBytes, err := caCert.SelfSign(rand.Reader, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, nil, err
	}
	crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return nil, nil, err
	}
	err = d.Put(crt.Subject.CommonName, crt)
	if err != nil {
		return nil, nil, err
	}
	encPemBlock, err := x509.EncryptPEMBlock(
		rand.Reader,
		"RSA PRIVATE KEY",
		x509.MarshalPKCS1PrivateKey(privKey),
		pass,
		x509.PEMCipher3DES,
	)
	if err != nil {
		return nil, nil, err
	}
	_, err = d.db.Exec(`
INSERT INTO scep_ca_keys
    (serial, key_pem)
VALUES
    (?, ?)`,
		crt.SerialNumber.Int64(), // caCert.SelfSign always sets serial to 1.
		pem.EncodeToMemory(encPemBlock),
	)
	if err != nil {
		return nil, nil, err
	}
	d.caCrt = crt
	d.caKey = privKey
	return crt, privKey, nil
}

// Put stores a certificate under the given name.
//
// If the provided certificate has empty crt.Subject.CommonName,
// then the hex sha256 of the crt.Raw is used as name.
func (d *MySQLDepot) Put(name string, crt *x509.Certificate) error {
	if crt.Subject.CommonName == "" {
		name = fmt.Sprintf("%x", sha256.Sum256(crt.Raw))
	}
	if !crt.SerialNumber.IsInt64() {
		return errors.New("cannot represent serial number as int64")
	}
	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	}
	_, err := d.db.Exec(`
INSERT INTO scep_certificates
    (serial, name, not_valid_before, not_valid_after, certificate_pem)
VALUES
    (?, ?, ?, ?, ?)`,
		crt.SerialNumber.Int64(),
		name,
		crt.NotBefore,
		crt.NotAfter,
		pem.EncodeToMemory(block),
	)
	return err
}
