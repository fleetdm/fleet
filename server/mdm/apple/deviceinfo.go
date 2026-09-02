// The contents of this file have been copied and modified pursuant to the following
// license from the original source:
// https://github.com/korylprince/dep-webview-oidc/blob/2dd846a54fed04c16dd227b8c6c31665b4d0ebd8/header/header.go
//
// MIT License
//
// Copyright (c) 2023 Kory Prince
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package apple_mdm

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	_ "crypto/sha1" //nolint:gosec // registers SHA-1 for crypto.SHA1.New; Apple device-identity chains still use SHA-1
	_ "crypto/sha256"
	_ "crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	_ "embed"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/rootcert"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
)

const DeviceInfoHeader = "x-apple-aspen-deviceinfo"

// machineInfoVerify controls enforcement of MachineInfo (deviceinfo) signature
// verification during enrollment. It defaults to enforce and is configured at
// server startup from config.MDM.AppleMachineInfoVerify via
// SetMachineInfoVerification. When disabled, verification runs in audit mode:
// failures are logged but enrollment is allowed to proceed.
var machineInfoVerify atomic.Bool

func init() {
	machineInfoVerify.Store(true)
}

// SetMachineInfoVerification configures whether MachineInfo signature
// verification is enforced. Called once at server startup from
// config.MDM.AppleMachineInfoVerify.
func SetMachineInfoVerification(enabled bool) {
	machineInfoVerify.Store(enabled)
}

// MachineInfoVerificationEnabled reports whether MachineInfo signature
// verification is enforced. When false, verification runs in audit mode.
func MachineInfoVerificationEnabled() bool {
	return machineInfoVerify.Load()
}

// tbCleanup is the subset of testing.TB that SetMachineInfoVerificationForTest
// needs, kept as a local interface so the testing package stays out of this
// file's production import graph.
type tbCleanup interface {
	Cleanup(func())
}

// SetMachineInfoVerificationForTest sets the enforcement flag for the duration
// of a test, restoring the previous value on cleanup.
func SetMachineInfoVerificationForTest(t tbCleanup, enabled bool) {
	prev := machineInfoVerify.Load()
	machineInfoVerify.Store(enabled)
	t.Cleanup(func() {
		machineInfoVerify.Store(prev)
	})
}

// appleIphoneDeviceCA is the PEM data defined here converted to DER:
// https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/profile-service/profile-service.html#//apple_ref/doc/uid/TP40009505-CH2-SW24
//
//go:embed AppleIphoneDeviceCA.cer
var appleIphoneDeviceCACert []byte

// appleIphoneDeviceCA is Apple's Iphone Device CA parsed to an *x509.Certificate
var appleIphoneDeviceCA = rootcert.NewAppleCert(appleIphoneDeviceCACert)

// VerifyMachineInfoSignature verifies that the given PKCS7/CMS payload was
// signed by an Apple device identity: the signer certificate must be included
// in the payload, its signature over the content (or CMS authenticated
// attributes, when present) must verify, and the certificate chain must be
// issued by a pinned Apple device-identity CA.
//
// Apple device-identity chains are SHA-1 signed and contain long-expired
// intermediates (e.g. "Apple iPhone Device CA", expired 2014, still issues
// current leaves), so stock x509.Verify cannot be used: path validation here
// is signature-only, ignoring certificate validity windows and tolerating
// SHA-1, per Apple's guidance on the expired chain.
func VerifyMachineInfoSignature(p7 *pkcs7.PKCS7) error {
	return verifyAppleSignedPKCS7(p7, appleDeviceIdentityCAs())
}

// appleDeviceIdentityCAs returns the pinned Apple CAs that issue per-device
// identity certificates. A genuine deviceinfo signature chains to one of these.
// Currently only this one sub-CA is known and defined in Apple's enrollment docs.
//
// NB: This is deliberately the device-identity issuing CA, NOT an Apple root. It
// will however chain up to Apple root
func appleDeviceIdentityCAs() []*x509.Certificate {
	return []*x509.Certificate{appleIphoneDeviceCA}
}

// authenticatedAttribute mirrors the CMS Attribute encoding used by the pkcs7
// library so the signed attributes can be re-marshaled for signature
// verification (re-tagged from IMPLICIT [0] to the SET OF the signature is
// computed over).
type authenticatedAttribute struct {
	Type  asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"set"`
}

func verifyAppleSignedPKCS7(p7 *pkcs7.PKCS7, deviceCAs []*x509.Certificate) error {
	if p7 == nil {
		return errors.New("no pkcs7 payload")
	}
	if len(p7.Signers) == 0 {
		return errors.New("payload is not signed")
	}
	if len(p7.Signers) != 1 {
		return fmt.Errorf("expected exactly one signer, got %d", len(p7.Signers))
	}
	signer := p7.Signers[0]

	var signerCert *x509.Certificate
	for _, cert := range p7.Certificates {
		if bytes.Equal(cert.RawIssuer, signer.IssuerAndSerialNumber.IssuerName.FullBytes) &&
			cert.SerialNumber.Cmp(signer.IssuerAndSerialNumber.SerialNumber) == 0 {
			signerCert = cert
			break
		}
	}
	if signerCert == nil {
		return errors.New("signer certificate not included in payload")
	}

	hash, err := hashForDigestOID(signer.DigestAlgorithm.Algorithm)
	if err != nil {
		return err
	}

	// When CMS authenticated attributes are present, the signature is computed
	// over their DER encoding, and the content is bound to the signature via
	// the messageDigest attribute (RFC 5652 §5.4). Otherwise the signature is
	// over the content itself.
	signedData := p7.Content
	if len(signer.AuthenticatedAttributes) > 0 {
		var messageDigest []byte
		var contentType asn1.ObjectIdentifier
		attrs := make([]authenticatedAttribute, 0, len(signer.AuthenticatedAttributes))
		for _, attr := range signer.AuthenticatedAttributes {
			switch {
			case attr.Type.Equal(pkcs7.OIDAttributeMessageDigest):
				if _, err := asn1.Unmarshal(attr.Value.Bytes, &messageDigest); err != nil {
					return fmt.Errorf("unmarshal messageDigest attribute: %w", err)
				}
			case attr.Type.Equal(pkcs7.OIDAttributeContentType):
				if _, err := asn1.Unmarshal(attr.Value.Bytes, &contentType); err != nil {
					return fmt.Errorf("unmarshal contentType attribute: %w", err)
				}
			}
			attrs = append(attrs, authenticatedAttribute{Type: attr.Type, Value: attr.Value})
		}

		// RFC 5652 §11: both attributes are required when authenticated
		// attributes are present.
		if len(messageDigest) == 0 {
			return errors.New("missing messageDigest authenticated attribute")
		}
		if contentType == nil {
			return errors.New("missing contentType authenticated attribute")
		}
		if !contentType.Equal(pkcs7.OIDData) {
			return fmt.Errorf("unexpected contentType authenticated attribute: %s", contentType)
		}

		h := hash.New()
		h.Write(p7.Content)
		if subtle.ConstantTimeCompare(messageDigest, h.Sum(nil)) != 1 {
			return errors.New("content digest does not match signed messageDigest attribute")
		}

		if signedData, err = marshalAuthenticatedAttributes(attrs); err != nil {
			return fmt.Errorf("marshal authenticated attributes: %w", err)
		}
	}

	sigAlg, err := signatureAlgorithmForOIDs(signer.DigestEncryptionAlgorithm.Algorithm, signer.DigestAlgorithm.Algorithm)
	if err != nil {
		return err
	}
	// x509.Certificate.CheckSignature (unlike chain verification) still accepts
	// SHA-1, which Apple device identities use.
	if err := signerCert.CheckSignature(sigAlg, signedData, signer.EncryptedDigest); err != nil {
		return fmt.Errorf("content signature verification failed: %w", err)
	}

	return verifyChainToDeviceCA(signerCert, p7.Certificates, deviceCAs)
}

// marshalAuthenticatedAttributes re-encodes the authenticated attributes as
// the DER SET OF the CMS signature is computed over.
func marshalAuthenticatedAttributes(attrs []authenticatedAttribute) ([]byte, error) {
	encoded, err := asn1.Marshal(struct {
		A []authenticatedAttribute `asn1:"set"`
	}{A: attrs})
	if err != nil {
		return nil, err
	}
	// strip the outer SEQUENCE, leaving the SET OF
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(encoded, &raw); err != nil {
		return nil, err
	}
	return raw.Bytes, nil
}

func hashForDigestOID(oid asn1.ObjectIdentifier) (crypto.Hash, error) {
	switch {
	case oid.Equal(pkcs7.OIDDigestAlgorithmSHA1),
		oid.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA1),
		oid.Equal(pkcs7.OIDDigestAlgorithmECDSASHA1):
		return crypto.SHA1, nil
	case oid.Equal(pkcs7.OIDDigestAlgorithmSHA256),
		oid.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA256),
		oid.Equal(pkcs7.OIDDigestAlgorithmECDSASHA256):
		return crypto.SHA256, nil
	case oid.Equal(pkcs7.OIDDigestAlgorithmSHA384),
		oid.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA384),
		oid.Equal(pkcs7.OIDDigestAlgorithmECDSASHA384):
		return crypto.SHA384, nil
	case oid.Equal(pkcs7.OIDDigestAlgorithmSHA512),
		oid.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA512),
		oid.Equal(pkcs7.OIDDigestAlgorithmECDSASHA512):
		return crypto.SHA512, nil
	}
	return 0, fmt.Errorf("unsupported digest algorithm: %s", oid)
}

func signatureAlgorithmForOIDs(encryptionOID, digestOID asn1.ObjectIdentifier) (x509.SignatureAlgorithm, error) {
	switch {
	case encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmRSA),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA1),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA256),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA384),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmRSASHA512):
		hash, err := hashForDigestOID(digestOID)
		if err != nil {
			return 0, err
		}
		switch hash {
		case crypto.SHA1:
			return x509.SHA1WithRSA, nil
		case crypto.SHA256:
			return x509.SHA256WithRSA, nil
		case crypto.SHA384:
			return x509.SHA384WithRSA, nil
		case crypto.SHA512:
			return x509.SHA512WithRSA, nil
		}
	case encryptionOID.Equal(pkcs7.OIDDigestAlgorithmECDSASHA1):
		return x509.ECDSAWithSHA1, nil
	case encryptionOID.Equal(pkcs7.OIDDigestAlgorithmECDSASHA256):
		return x509.ECDSAWithSHA256, nil
	case encryptionOID.Equal(pkcs7.OIDDigestAlgorithmECDSASHA384):
		return x509.ECDSAWithSHA384, nil
	case encryptionOID.Equal(pkcs7.OIDDigestAlgorithmECDSASHA512):
		return x509.ECDSAWithSHA512, nil
	case encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmECDSAP256),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmECDSAP384),
		encryptionOID.Equal(pkcs7.OIDEncryptionAlgorithmECDSAP521):
		hash, err := hashForDigestOID(digestOID)
		if err != nil {
			return 0, err
		}
		switch hash {
		case crypto.SHA1:
			return x509.ECDSAWithSHA1, nil
		case crypto.SHA256:
			return x509.ECDSAWithSHA256, nil
		case crypto.SHA384:
			return x509.ECDSAWithSHA384, nil
		case crypto.SHA512:
			return x509.ECDSAWithSHA512, nil
		}
	}
	return 0, fmt.Errorf("unsupported signature algorithm: %s with digest %s", encryptionOID, digestOID)
}

// verifyChainToDeviceCA walks the certificate chain from the leaf and requires
// that some certificate in the path was issued by one of the pinned Apple
// device-identity CAs. Only signatures are verified: validity windows are
// ignored (the device CA and its issuers are long expired) and SHA-1 is
// tolerated per Apple documentation.
//
// The device CA is trusted by identity: its signature is checked against our
// embedded copy's public key, never against a copy taken from the payload's
// certificate bag. A chain that reaches a pinned Apple *root* through some
// other Apple sub-CA (Developer ID, WWDR, …) but never through a device CA is
// rejected.
func verifyChainToDeviceCA(leaf *x509.Certificate, bag []*x509.Certificate, deviceCAs []*x509.Certificate) error {
	cert := leaf
	for range len(bag) + 1 {
		for _, ca := range deviceCAs {
			if bytes.Equal(cert.RawIssuer, ca.RawSubject) {
				if err := ca.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
					return fmt.Errorf("verifying signature of %q against pinned Apple device CA %q: %w",
						cert.Subject.CommonName, ca.Subject.CommonName, err)
				}
				return nil
			}
		}

		var next *x509.Certificate
		var lastErr error
		for _, candidate := range bag {
			if bytes.Equal(candidate.Raw, cert.Raw) || !bytes.Equal(cert.RawIssuer, candidate.RawSubject) {
				continue
			}
			if !candidate.BasicConstraintsValid || !candidate.IsCA {
				lastErr = fmt.Errorf("issuer %q of %q is not a CA certificate", candidate.Subject.CommonName, cert.Subject.CommonName)
				continue
			}
			if err := candidate.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature); err != nil {
				lastErr = fmt.Errorf("verifying signature of %q against %q: %w",
					cert.Subject.CommonName, candidate.Subject.CommonName, err)
				continue
			}
			next = candidate
			break
		}
		if next == nil {
			if lastErr != nil {
				return lastErr
			}
			return fmt.Errorf("certificate chain from signer %q is not issued by a pinned Apple device CA", leaf.Subject.CommonName)
		}
		cert = next
	}
	return errors.New("certificate chain contains a cycle")
}

// ParseDeviceinfo attempts to parse the provided string, assuming it to be the base64-encoded value
// of an x-apple-aspen-deviceinfo header. If successful, it returns the parsed
// *fleet.MDMAppleMachineInfo along with the containing PKCS7 payload, which callers on
// Apple-signed enrollment lanes must pass to VerifyMachineInfoSignature.
func ParseDeviceinfo(b64 string) (*fleet.MDMAppleMachineInfo, *pkcs7.PKCS7, error) {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		if strings.Contains(err.Error(), "illegal base64 data") {
			// try with url encoding
			buf, err = base64.URLEncoding.DecodeString(b64)
		}
		if err != nil {
			return nil, nil, fmt.Errorf("could not decode base64: %w", err)
		}
	}

	return ParseMachineInfoFromPKCS7(buf)
}

// ParseMachineInfoFromPKCS7 parses a MachineInfo plist from a PKCS7 payload without verifying
// its signature. Callers must pass the returned PKCS7 payload to VerifyMachineInfoSignature.
func ParseMachineInfoFromPKCS7(buf []byte) (*fleet.MDMAppleMachineInfo, *pkcs7.PKCS7, error) {
	if err := cryptoutil.ValidateBERDepth(buf, cryptoutil.MaxBERDepth); err != nil {
		return nil, nil, fmt.Errorf("invalid pkcs7: %w", err)
	}

	p7, err := pkcs7.Parse(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("could not decode pkcs7: %w", err)
	}

	info := new(fleet.MDMAppleMachineInfo)
	if err = plist.Unmarshal(p7.Content, info); err != nil {
		return nil, nil, fmt.Errorf("could not decode plist: %w", err)
	}

	return info, p7, nil
}

// VerifyFromAppleIphoneDeviceCA verifies a certificate was signed by Apple's iPhone Device CA.
// Manually verify the certificate since Go has deprecated verifying SHA1WithRSA x509 certificates.
//
// NOTE: most of this code was taken from micromdm.
func VerifyFromAppleIphoneDeviceCA(c *x509.Certificate) error {
	if dev_mode.Env("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY") == "1" {
		return nil
	}

	if c == nil {
		return errors.New("no certificate provided")
	}

	var hashType crypto.Hash

	switch c.SignatureAlgorithm {
	case x509.SHA1WithRSA:
		hashType = crypto.SHA1
	case x509.SHA256WithRSA:
		hashType = crypto.SHA256
	default:
		return fmt.Errorf("%w: %s", x509.ErrUnsupportedAlgorithm, c.SignatureAlgorithm)
	}

	hasher := hashType.New()
	hasher.Write(c.RawTBSCertificate)
	hashed := hasher.Sum(nil)

	key, ok := appleIphoneDeviceCA.PublicKey.(*rsa.PublicKey)
	if !ok {
		panic("appleIphoneDeviceCA: invalid key type")
	}

	if err := rsa.VerifyPKCS1v15(key, hashType, hashed, c.Signature); err != nil {
		return fmt.Errorf("verifying signature: %w", err)
	}

	return nil
}
