package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/micromdm/micromdm/pkg/crypto/mdmcertutil"
	"golang.org/x/exp/slices"
)

const (
	vendorCertEnvName = "VENDOR_CERT_PEM"
	vendorKeyEnvName  = "VENDOR_KEY_PEM"
	vendorPassEnvName = "VENDOR_KEY_PASSPHRASE" //nolint:gosec
	csrEnvName        = "CSR_BASE64"

	rsaPrivateKeyPEMBlockType = "RSA PRIVATE KEY"
	certificatePEMBlockType   = "CERTIFICATE"
	csrPEMBlockType           = "CERTIFICATE REQUEST"

	wwdrIntermediaryCert = `-----BEGIN CERTIFICATE-----
MIIEUTCCAzmgAwIBAgIQfK9pCiW3Of57m0R6wXjF7jANBgkqhkiG9w0BAQsFADBi
MQswCQYDVQQGEwJVUzETMBEGA1UEChMKQXBwbGUgSW5jLjEmMCQGA1UECxMdQXBw
bGUgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkxFjAUBgNVBAMTDUFwcGxlIFJvb3Qg
Q0EwHhcNMjAwMjE5MTgxMzQ3WhcNMzAwMjIwMDAwMDAwWjB1MUQwQgYDVQQDDDtB
cHBsZSBXb3JsZHdpZGUgRGV2ZWxvcGVyIFJlbGF0aW9ucyBDZXJ0aWZpY2F0aW9u
IEF1dGhvcml0eTELMAkGA1UECwwCRzMxEzARBgNVBAoMCkFwcGxlIEluYy4xCzAJ
BgNVBAYTAlVTMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2PWJ/KhZ
C4fHTJEuLVaQ03gdpDDppUjvC0O/LYT7JF1FG+XrWTYSXFRknmxiLbTGl8rMPPbW
BpH85QKmHGq0edVny6zpPwcR4YS8Rx1mjjmi6LRJ7TrS4RBgeo6TjMrA2gzAg9Dj
+ZHWp4zIwXPirkbRYp2SqJBgN31ols2N4Pyb+ni743uvLRfdW/6AWSN1F7gSwe0b
5TTO/iK1nkmw5VW/j4SiPKi6xYaVFuQAyZ8D0MyzOhZ71gVcnetHrg21LYwOaU1A
0EtMOwSejSGxrC5DVDDOwYqGlJhL32oNP/77HK6XF8J4CjDgXx9UO0m3JQAaN4LS
VpelUkl8YDib7wIDAQABo4HvMIHsMBIGA1UdEwEB/wQIMAYBAf8CAQAwHwYDVR0j
BBgwFoAUK9BpR5R2Cf70a40uQKb3R01/CF4wRAYIKwYBBQUHAQEEODA2MDQGCCsG
AQUFBzABhihodHRwOi8vb2NzcC5hcHBsZS5jb20vb2NzcDAzLWFwcGxlcm9vdGNh
MC4GA1UdHwQnMCUwI6AhoB+GHWh0dHA6Ly9jcmwuYXBwbGUuY29tL3Jvb3QuY3Js
MB0GA1UdDgQWBBQJ/sAVkPmvZAqSErkmKGMMl+ynsjAOBgNVHQ8BAf8EBAMCAQYw
EAYKKoZIhvdjZAYCAQQCBQAwDQYJKoZIhvcNAQELBQADggEBAK1lE+j24IF3RAJH
Qr5fpTkg6mKp/cWQyXMT1Z6b0KoPjY3L7QHPbChAW8dVJEH4/M/BtSPp3Ozxb8qA
HXfCxGFJJWevD8o5Ja3T43rMMygNDi6hV0Bz+uZcrgZRKe3jhQxPYdwyFot30ETK
XXIDMUacrptAGvr04NM++i+MZp+XxFRZ79JI9AeZSWBZGcfdlNHAwWx/eCHvDOs7
bJmCS1JgOLU5gm3sUjFTvg+RTElJdI+mUcuER04ddSduvfnSXPN/wmwLCTbiZOTC
NwMUGdXqapSqqdv+9poIZ4vvK7iqF0mDr8/LvOnP6pVxsLRFoszlh6oKw0E6eVza
UDSdlTs=
-----END CERTIFICATE-----`
	appleRootCert = `-----BEGIN CERTIFICATE-----
MIIEuzCCA6OgAwIBAgIBAjANBgkqhkiG9w0BAQUFADBiMQswCQYDVQQGEwJVUzET
MBEGA1UEChMKQXBwbGUgSW5jLjEmMCQGA1UECxMdQXBwbGUgQ2VydGlmaWNhdGlv
biBBdXRob3JpdHkxFjAUBgNVBAMTDUFwcGxlIFJvb3QgQ0EwHhcNMDYwNDI1MjE0
MDM2WhcNMzUwMjA5MjE0MDM2WjBiMQswCQYDVQQGEwJVUzETMBEGA1UEChMKQXBw
bGUgSW5jLjEmMCQGA1UECxMdQXBwbGUgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkx
FjAUBgNVBAMTDUFwcGxlIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDkkakJH5HbHkdQ6wXtXnmELes2oldMVeyLGYne+Uts9QerIjAC6Bg+
+FAJ039BqJj50cpmnCRrEdCju+QbKsMflZ56DKRHi1vUFjczy8QPTc4UadHJGXL1
XQ7Vf1+b8iUDulWPTV0N8WQ1IxVLFVkds5T39pyez1C6wVhQZ48ItCD3y6wsIG9w
tj8BMIy3Q88PnT3zK0koGsj+zrW5DtleHNbLPbU6rfQPDgCSC7EhFi501TwN22IW
q6NxkkdTVcGvL0Gz+PvjcM3mo0xFfh9Ma1CWQYnEdGILEINBhzOKgbEwWOxaBDKM
aLOPHd5lc/9nXmW8Sdh2nzMUZaF3lMktAgMBAAGjggF6MIIBdjAOBgNVHQ8BAf8E
BAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUK9BpR5R2Cf70a40uQKb3
R01/CF4wHwYDVR0jBBgwFoAUK9BpR5R2Cf70a40uQKb3R01/CF4wggERBgNVHSAE
ggEIMIIBBDCCAQAGCSqGSIb3Y2QFATCB8jAqBggrBgEFBQcCARYeaHR0cHM6Ly93
d3cuYXBwbGUuY29tL2FwcGxlY2EvMIHDBggrBgEFBQcCAjCBthqBs1JlbGlhbmNl
IG9uIHRoaXMgY2VydGlmaWNhdGUgYnkgYW55IHBhcnR5IGFzc3VtZXMgYWNjZXB0
YW5jZSBvZiB0aGUgdGhlbiBhcHBsaWNhYmxlIHN0YW5kYXJkIHRlcm1zIGFuZCBj
b25kaXRpb25zIG9mIHVzZSwgY2VydGlmaWNhdGUgcG9saWN5IGFuZCBjZXJ0aWZp
Y2F0aW9uIHByYWN0aWNlIHN0YXRlbWVudHMuMA0GCSqGSIb3DQEBBQUAA4IBAQBc
NplMLXi37Yyb3PN3m/J20ncwT8EfhYOFG5k9RzfyqZtAjizUsZAS2L70c5vu0mQP
y3lPNNiiPvl4/2vIB+x9OYOLUyDTOMSxv5pPCmv/K/xZpwUJfBdAVhEedNO3iyM7
R6PVbyTi69G3cN8PReEnyvFteO3ntRcXqNx+IjXKJdXZD9Zr1KIkIxH3oayPc4Fg
xhtbCS+SsvhESPBgOJ4V9T0mZyCKM2r3DYLP3uujL/lTaltkwGMzd/c6ByxW69oP
IQ7aunMZT7XZNn/Bh1XZp5m5MkL72NVxnn6hUrcbvZNCJBIqxw8dtk2cXmPIS4AX
UKqK1drk/NAJBzewdXUh
-----END CERTIFICATE-----`
)

var (
	// emailAddressOID defined by https://oidref.com/1.2.840.113549.1.9.1
	emailAddressOID = []int{1, 2, 840, 113549, 1, 9, 1}
	// organizationOID defined by https://oidref.com/2.5.4.10
	organizationOID = []int{2, 5, 4, 10}
)

// See
// https://developer.apple.com/documentation/devicemanagement/implementing_device_management/setting_up_push_notifications_for_your_mdm_customers
// for the expected CSR format.

type signResult struct {
	Email   string `json:"email"`
	Org     string `json:"org"`
	Request string `json:"request"`
}

func main() {
	vendorCertPEM := os.Getenv(vendorCertEnvName)
	if vendorCertPEM == "" {
		log.Fatalf("vendor cert must be set in %s", vendorCertEnvName)
	}
	vendorKeyPEM := os.Getenv(vendorKeyEnvName)
	if vendorKeyPEM == "" {
		log.Fatalf("vendor key must be set in %s", vendorKeyEnvName)
	}
	vendorKeyPassphrase := os.Getenv(vendorPassEnvName)
	if vendorKeyPassphrase == "" {
		log.Fatalf("vendor key passphrase must be set in %s", vendorPassEnvName)
	}
	// We accept the CSR via environment variable to mitigate against command injection attacks in
	// the fleetdm.com website code that will call this with untrusted user input.
	csrBase64 := os.Getenv(csrEnvName)
	if csrBase64 == "" {
		log.Fatalf("CSR must be set in %s", csrEnvName)
	}

	out, err := processRequest(vendorCertPEM, vendorKeyPEM, vendorKeyPassphrase, csrBase64)
	if err != nil {
		log.Fatalf("process request: %s", err)
	}

	// Write output as JSON
	outJSON, err := json.Marshal(out)
	if err != nil {
		log.Fatalf("encode request JSON: %s", err)
	}
	fmt.Println(string(outJSON))
}

func processRequest(vendorCertPEM, vendorKeyPEM, vendorKeyPassphrase, csrBase64 string) (*signResult, error) {
	// Load vendor keys and certs
	vendorCert, err := decodeVendorCert([]byte(vendorCertPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse vendor cert: %w", err)
	}
	vendorKey, err := loadKey([]byte(vendorKeyPEM), []byte(vendorKeyPassphrase))
	if err != nil {
		return nil, fmt.Errorf("failed to load vendor private key: %w", err)
	}

	// Decode CSR input
	csr, err := base64.StdEncoding.DecodeString(csrBase64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode csr: %w", err)
	}
	certReq, err := decodeCSR(csr)
	if err != nil {
		return nil, fmt.Errorf("decode csr: %w", err)
	}

	// Get email and org from CSR
	email, org, err := getEmailOrg(certReq)
	if err != nil {
		return nil, fmt.Errorf("get subjects: %w", err)
	}

	// Tie it all together
	req, err := createPushCertificateRequest(vendorKey, vendorCert, certReq)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	encodedReq, err := req.Encode()
	if err != nil {
		return nil, fmt.Errorf("encode csr: %w", err)
	}

	return &signResult{Email: email, Org: org, Request: string(encodedReq)}, nil
}

func getEmailOrg(req *x509.CertificateRequest) (email, org string, err error) {
	if len(req.Subject.Names) != 2 {
		return "", "", errors.New("request must have exactly 2 subjects (organization and email)")
	}

	for _, name := range req.Subject.Names {
		switch {
		case slices.Equal(name.Type, emailAddressOID):
			str, ok := name.Value.(string)
			if !ok {
				return "", "", fmt.Errorf("email subject (%T) is not string value", name.Value)
			}
			email = str

		case slices.Equal(name.Type, organizationOID):
			str, ok := name.Value.(string)
			if !ok {
				return "", "", fmt.Errorf("organization subject (%T) is not string value", name.Value)
			}
			org = str

		default:
			return "", "", fmt.Errorf("unexpected subject: %v", name.Type)
		}
	}

	return email, org, nil
}

// Below functions copied with modifications from MicroMDM (MIT license):
// https://github.com/micromdm/micromdm/blob/e0943fdcd7ea2c6f79edf3e3496e1218e968ba2a/pkg/crypto/mdmcertutil/certutil.go

// createPushCertificateRequest creates a request structure required by identity.apple.com.
// It requires a "MDM CSR" certificate (the vendor certificate), a push CSR (the customer specific CSR),
// and the vendor private key.
func createPushCertificateRequest(vendorKey *rsa.PrivateKey, vendorCert *x509.Certificate, csr *x509.CertificateRequest) (*mdmcertutil.PushCertificateRequest, error) {
	// csr signature
	signature, err := signPushCSR(csr.Raw, vendorKey)
	if err != nil {
		return nil, fmt.Errorf("sign push CSR with private key: %w", err)
	}

	// vendor cert
	mdmPEM := string(pemCert(vendorCert.Raw))

	csrB64 := base64.StdEncoding.EncodeToString(csr.Raw)
	sig64 := base64.StdEncoding.EncodeToString(signature)
	pushReq := &mdmcertutil.PushCertificateRequest{
		PushCertRequestCSR:       csrB64,
		PushCertCertificateChain: makeCertChain(mdmPEM, wwdrIntermediaryCert, appleRootCert),
		PushCertSignature:        sig64,
	}
	return pushReq, nil
}

func makeCertChain(mdmPEM, wwdrPEM, rootPEM string) string {
	return mdmPEM + wwdrPEM + rootPEM
}

func pemCert(derBytes []byte) []byte {
	pemBlock := &pem.Block{
		Type:    certificatePEMBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	out := pem.EncodeToMemory(pemBlock)
	return out
}

func loadKey(keyPem, password []byte) (*rsa.PrivateKey, error) {
	pemBlock, _ := pem.Decode(keyPem)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	if pemBlock.Type != rsaPrivateKeyPEMBlockType {
		return nil, fmt.Errorf("unmatched type: %s", pemBlock.Type)
	}

	b, err := x509.DecryptPEMBlock(pemBlock, password)
	if err != nil {
		return nil, err
	}

	return x509.ParsePKCS1PrivateKey(b)
}

func signPushCSR(csrData []byte, key *rsa.PrivateKey) ([]byte, error) {
	h := sha256.New()
	h.Write(csrData)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("sign push CSR: %w", err)
	}
	return signature, nil
}

func decodeVendorCert(pemData []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(pemData)
	if pemBlock == nil {
		return nil, errors.New("cannot find the next PEM formatted block")
	}
	if pemBlock.Type != certificatePEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("unmatched type or headers")
	}
	return x509.ParseCertificate(pemBlock.Bytes)
}

func decodeCSR(pemData []byte) (*x509.CertificateRequest, error) {
	pemBlock, _ := pem.Decode(pemData)
	if pemBlock == nil {
		return nil, errors.New("cannot find the next PEM formatted block")
	}
	if pemBlock.Type != csrPEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("unmatched type or headers")
	}
	return x509.ParseCertificateRequest(pemBlock.Bytes)
}
