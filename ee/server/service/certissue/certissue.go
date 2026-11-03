// Package certissue has the pieces the certificate authority clients share: building a CSR with a new key, bundling the issued
// certificate and key for Apple profiles, reading and writing Microsoft UPNs in a subjectAltName, and telling which failures are worth retrying.
package certissue

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"golang.org/x/crypto/cryptobyte"
	cryptobyteasn1 "golang.org/x/crypto/cryptobyte/asn1"
	"software.sslmate.com/src/go-pkcs12"
)

var (
	// oidMicrosoftUPN is the otherName type for a User Principal Name in a subjectAltName.
	oidMicrosoftUPN = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2, 3}
	// OIDSubjectAltName is the subjectAltName extension.
	OIDSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}
)

// NewCSR generates an RSA 2048 key and a PEM CSR for it with the common name and, when there are non-blank UPNs, a UPN subjectAltName.
func NewCSR(ctx context.Context, commonName string, upns []string) (*rsa.PrivateKey, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "generating RSA private key")
	}
	template := &x509.CertificateRequest{Subject: pkix.Name{CommonName: commonName}}
	if ext, ok := UPNExtension(upns); ok {
		template.ExtraExtensions = []pkix.Extension{ext}
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "creating CSR")
	}
	csr := strings.TrimSpace(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})))
	return privateKey, csr, nil
}

// EncodePKCS12 bundles the key and certificate under a new random password, in the legacy format Apple devices can import.
func EncodePKCS12(ctx context.Context, privateKey *rsa.PrivateKey, cert *x509.Certificate) (data []byte, password string, err error) {
	password, err = server.GenerateRandomText(10)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "generating password for PKCS12 bundle")
	}
	data, err = pkcs12.Legacy.Encode(privateKey, cert, nil, password)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "creating PKCS12 bundle")
	}
	return data, password, nil
}

// NonBlank returns the values that are not empty or whitespace.
func NonBlank(values []string) []string {
	return slices.DeleteFunc(slices.Clone(values), func(v string) bool { return strings.TrimSpace(v) == "" })
}

// UPNExtension builds a subjectAltName extension with a Microsoft UPN otherName for each non-blank name:
//
//	GeneralName ::= CHOICE { otherName [0] OtherName, ... }
//	OtherName   ::= SEQUENCE { type-id OBJECT IDENTIFIER, value [0] EXPLICIT ANY DEFINED BY type-id }
//
// where the value is a UTF8String.
func UPNExtension(upns []string) (pkix.Extension, bool) {
	names := NonBlank(upns)
	if len(names) == 0 {
		return pkix.Extension{}, false
	}
	contextConstructed := cryptobyteasn1.Tag(0).ContextSpecific().Constructed()
	var b cryptobyte.Builder
	b.AddASN1(cryptobyteasn1.SEQUENCE, func(b *cryptobyte.Builder) {
		for _, upn := range names {
			b.AddASN1(contextConstructed, func(b *cryptobyte.Builder) {
				b.AddASN1ObjectIdentifier(oidMicrosoftUPN)
				b.AddASN1(contextConstructed, func(b *cryptobyte.Builder) {
					b.AddASN1(cryptobyteasn1.UTF8String, func(b *cryptobyte.Builder) { b.AddBytes([]byte(upn)) })
				})
			})
		}
	})
	value, err := b.Bytes()
	if err != nil {
		return pkix.Extension{}, false
	}
	return pkix.Extension{Id: OIDSubjectAltName, Value: value}, true
}

// UPNs returns the Microsoft UPNs in the subjectAltName of the certificate.
func UPNs(cert *x509.Certificate) []string {
	otherName := cryptobyteasn1.Tag(0).ContextSpecific().Constructed()
	var upns []string
	for _, ext := range cert.Extensions {
		names := cryptobyte.String(ext.Value)
		if !ext.Id.Equal(OIDSubjectAltName) || !names.ReadASN1(&names, cryptobyteasn1.SEQUENCE) {
			continue
		}
		for !names.Empty() {
			var name cryptobyte.String
			var tag cryptobyteasn1.Tag
			if !names.ReadAnyASN1(&name, &tag) {
				break
			}
			var oid asn1.ObjectIdentifier
			var value, upn cryptobyte.String
			if tag == otherName && name.ReadASN1ObjectIdentifier(&oid) && oid.Equal(oidMicrosoftUPN) &&
				name.ReadASN1(&value, otherName) && value.ReadASN1(&upn, cryptobyteasn1.UTF8String) {
				upns = append(upns, string(upn))
			}
		}
	}
	return upns
}
