package ejbca

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"software.sslmate.com/src/go-pkcs12"
)

const (
	msgP12BadPassword = "The PKCS#12 password is incorrect or the file is damaged." // nolint:gosec // G101: a message, not a credential
	msgP12BER         = "The PKCS#12 file uses BER encoding, which Fleet can't read. Re-export it with OpenSSL (openssl pkcs12 -export) " +
		"to get a DER encoded file."
	msgP12Invalid  = "Couldn't read the PKCS#12 file: %s"
	msgP12Mismatch = "The private key in the PKCS#12 file doesn't match its certificate."
)

// ErrP12Password is the error of a PKCS#12 password that doesn't open the file, so a caller can tell it from a problem with the file.
var ErrP12Password = errors.New(msgP12BadPassword)

// ClientCredentials is the TLS client certificate and key read from a PKCS#12 file.
type ClientCredentials struct {
	// CertPEM is the leaf certificate followed by any chain certificates.
	CertPEM string
	// KeyPEM is the private key, PKCS#8 encoded.
	KeyPEM string
	// Leaf is the client certificate, for the caller to check its validity.
	Leaf *x509.Certificate
}

// ParseClientP12 reads the client certificate chain and private key of a PKCS#12 file, such as the one EJBCA's RA Web exports.
func ParseClientP12(ctx context.Context, p12 []byte, password string) (*ClientCredentials, error) {
	key, leaf, chain, err := pkcs12.DecodeChain(p12, password)
	if err != nil {
		switch {
		case errors.Is(err, pkcs12.ErrIncorrectPassword), errors.Is(err, pkcs12.ErrDecryption):
			return nil, ErrP12Password
		case isBER(p12):
			return nil, ctxerr.New(ctx, msgP12BER)
		default:
			return nil, ctxerr.Errorf(ctx, msgP12Invalid, err)
		}
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, ctxerr.Errorf(ctx, msgP12Invalid, err)
	}
	// A PKCS#12 doesn't say which certificate is the leaf, and the library takes the first, so find the one that belongs to the key.
	certs := append([]*x509.Certificate{leaf}, chain...)
	leafIdx := slices.IndexFunc(certs, func(c *x509.Certificate) bool { return publicKeyMatches(key, c) })
	if leafIdx < 0 {
		return nil, ctxerr.New(ctx, msgP12Mismatch)
	}
	leaf = certs[leafIdx]
	certs = slices.Insert(slices.Delete(certs, leafIdx, leafIdx+1), 0, leaf)

	creds := &ClientCredentials{
		KeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
		Leaf:   leaf,
	}
	for _, cert := range certs {
		creds.CertPEM += string(certificate.EncodeCertPEM(cert))
	}
	return creds, nil
}

// publicKeyMatches reports whether the certificate is for the public half of the private key.
func publicKeyMatches(privateKey any, cert *x509.Certificate) bool {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return false
	}
	pub, ok := signer.Public().(interface{ Equal(crypto.PublicKey) bool })
	return ok && pub.Equal(cert.PublicKey)
}

// isBER reports whether the outermost element has an indefinite length, which only BER allows.
func isBER(data []byte) bool {
	return len(data) > 1 && data[0] == 0x30 && data[1] == 0x80
}
