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
	"crypto/sha1" // nolint:gosec // See comments regarding Apple's Root CA below
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/groob/plist"
	"github.com/smallstep/pkcs7"
)

const DeviceInfoHeader = "x-apple-aspen-deviceinfo"

// appleRootCert is https://www.apple.com/appleca/AppleIncRootCertificate.cer
//
//go:embed AppleIncRootCertificate.cer
var appleRootCert []byte

// appleRootCA is Apple's Root CA parsed to an *x509.Certificate
var appleRootCA = newAppleCert(appleRootCert)

// appleIphoneDeviceCA is the PEM data defined here converted to DER:
// https://developer.apple.com/library/archive/documentation/NetworkingInternet/Conceptual/iPhoneOTAConfiguration/profile-service/profile-service.html#//apple_ref/doc/uid/TP40009505-CH2-SW24
//
//go:embed AppleIphoneDeviceCA.cer
var appleIphoneDeviceCACert []byte

// appleIphoneDeviceCA is Apple's Iphone Device CA parsed to an *x509.Certificate
var appleIphoneDeviceCA = newAppleCert(appleIphoneDeviceCACert)

func newAppleCert(crt []byte) *x509.Certificate {
	cert, err := x509.ParseCertificate(crt)
	if err != nil {
		panic(fmt.Errorf("could not parse cert: %w", err))
	}
	return cert
}

// verifyPKCS7SHA1RSA performs a manual SHA1withRSA verification, since it's deprecated in Go 1.18.
// If verifyChain is true, the signer certificate and its chain of certificates is verified against Apple's Root CA.
// Also note that the certificate validity time window of the signing cert is not checked, since the cert is expired.
// This follows guidance from Apple on the expired certificate.
func verifyPKCS7SHA1RSA(p7 *pkcs7.PKCS7, verifyChain bool) error {
	if len(p7.Signers) == 0 {
		return errors.New("not signed")
	}

	// get signing cert
	issuer := p7.Signers[0].IssuerAndSerialNumber
	var signer *x509.Certificate
	for _, cert := range p7.Certificates {
		if bytes.Equal(cert.RawIssuer, issuer.IssuerName.FullBytes) && cert.SerialNumber.Cmp(issuer.SerialNumber) == 0 {
			signer = cert
		}
	}

	// get sha1 hash of content
	hashed := sha1.Sum(p7.Content) // nolint:gosec

	// verify content signature
	signature := p7.Signers[0].EncryptedDigest
	if err := rsa.VerifyPKCS1v15(signer.PublicKey.(*rsa.PublicKey), crypto.SHA1, hashed[:], signature); err != nil {
		return fmt.Errorf("signature could not be verified: %w", err)
	}

	if !verifyChain {
		return nil
	}

	// verify chain from signer to root
	cert := signer
outer:
	for {
		// check if cert is signed by root
		if bytes.Equal(cert.RawIssuer, appleRootCA.RawSubject) {
			hashed := sha1.Sum(cert.RawTBSCertificate) // nolint:gosec
			// check signature
			if err := rsa.VerifyPKCS1v15(appleRootCA.PublicKey.(*rsa.PublicKey), crypto.SHA1, hashed[:], cert.Signature); err != nil {
				return fmt.Errorf("could not verify root CA signature: %w", err)
			}
			return nil
		}
		for _, c := range p7.Certificates {
			if cert == c {
				continue
			}
			// check if cert is signed by intermediate cert in chain
			if bytes.Equal(cert.RawIssuer, c.RawSubject) {
				// check signature
				hashed := sha1.Sum(cert.RawTBSCertificate) // nolint:gosec
				if err := rsa.VerifyPKCS1v15(c.PublicKey.(*rsa.PublicKey), crypto.SHA1, hashed[:], cert.Signature); err != nil {
					return fmt.Errorf("could not verify chained certificate signature: %w", err)
				}
				cert = c
				continue outer
			}
		}
		return errors.New("certificate root not found")
	}
}

// ParseDeviceinfo attempts to parse the provided string, assuming it to be the base64-encoded value
// of an x-apple-aspen-deviceinfo header. If successful, it returns the parsed *fleet.MDMAppleMachineInfo. If the
// verify parameter is specified as true, the signature is also verified against Apple's Root CA and
// an error will be returned if the signature is invalid.
//
// Warning: The information in this header, despite being signed by Apple PKI, shouldn't be trusted
// for device attestation or other security purposes. See the related [documentation] and referenced
// [article] for more information.
//
// [documentation]: https://github.com/korylprince/dep-webview-oidc/blob/2dd846a54fed04c16dd227b8c6c31665b4d0ebd8/docs/Architecture.md#x-apple-aspen-deviceinfo-header
// [article]: https://duo.com/labs/research/mdm-me-maybe
func ParseDeviceinfo(b64 string, verify bool) (*fleet.MDMAppleMachineInfo, error) {
	buf, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("could not decode base64: %w", err)
	}

	p7, err := pkcs7.Parse(buf)
	if err != nil {
		return nil, fmt.Errorf("could not decode pkcs7: %w", err)
	}

	// verify signature and certificate chain
	if verify {
		if err = verifyPKCS7SHA1RSA(p7, verify); err != nil {
			return nil, fmt.Errorf("could not verify signature: %w", err)
		}
	}

	info := new(fleet.MDMAppleMachineInfo)
	if err = plist.Unmarshal(p7.Content, info); err != nil {
		return nil, fmt.Errorf("could not decode plist: %w", err)
	}

	return info, nil
}

// VerifyFromAppleIphoneDeviceCA verifies a certificate was signed by Apple's iPhone Device CA.
// Manually verify the certificate since Go has deprecated verifying SHA1WithRSA x509 certificates.
//
// NOTE: most of this code was taken from micromdm.
func VerifyFromAppleIphoneDeviceCA(c *x509.Certificate) error {
	if os.Getenv("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY") == "1" {
		return nil
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
