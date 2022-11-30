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
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/micromdm/micromdm/pkg/crypto/mdmcertutil"
	"golang.org/x/exp/slices"
)

const (
	vendorCertEnvName = "VENDOR_CERT_PEM"
	vendorKeyEnvName  = "VENDOR_KEY_PEM"
	vendorPassEnvName = "VENDOR_KEY_PASSPHRASE"
	csrEnvName        = "CSR_BASE64"

	rsaPrivateKeyPEMBlockType = "RSA PRIVATE KEY"
	certificatePEMBlockType   = "CERTIFICATE"
	csrPEMBlockType           = "CERTIFICATE REQUEST"

	wwdrIntermediaryURL = "https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer"
	appleRootCAURL      = "https://www.apple.com/appleca/AppleIncRootCertificate.cer"
)

// emailAddressOID defined by https://oidref.com/1.2.840.113549.1.9.1
var emailAddressOID = []int{1, 2, 840, 113549, 1, 9, 1}

// TEST CSR
// LS0tLS1CRUdJTiBDRVJUSUZJQ0FURSBSRVFVRVNULS0tLS0KTUlJQ2R6Q0NBVjRDQVFBd01URU9NQXdHQTFVRUNoTUZSbXhsWlhReEh6QWRCZ2txaGtpRzl3MEJDUUVNRUhwaApZMmhBWm14bFpYUmtiUzVqYjIwd2dnRWlNQTBHQ1NxR1NJYjNEUUVCQVFVQUE0SUJEd0F3Z2dFS0FvSUJBUUdJCkdCQWlHdzM1WGxLNHd5TUVCMGloZmwxY3lLUXBhR0d2cUh0UUJIcGpRUk9tS055RmF5WkRNcnJERG9DWEg4V0gKblJUYVdad1BUZDhtUkIyMnhvRkh4em83Ym5McDJ3UG1Ld3U3d3pEeWpsSk9tcWxZTW9QbnFnays5TXRjLzRMNgpNdVB5Q0NaWDZNdWhNdEhLZWc4TDMrWGtHYnpZRGE1alYzV0VUYWhzenh2UEFFWVF5end3UGdBTFJwdDRzcGJWCmdiZG44bThuUDRFMGk2R0tkemZiQUloaWppMEpNTnhSakNQejlRNWEyME5XYkkvMkk2dWF2bW1ZWCtkUHk4T3IKbUh0RVJ2M2pid2kwMHRWdTlaT3ZBalZrZU5ZSkFSUk9nTnArdnpHUXVoQW9lbDFGL1N4eHg5bjRzbnNoREV0TQpBa2pxL3d5YzlNYUxWSk5jMnZLZkFnTUJBQUdnQURBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFJQUFVeC9jOE1KCkNvczBUb3FrV0lLOGYxbktvbTNXemxtVW9SRHRSWHRwUHQwelJHNkRrcFRhcnhvb0JOd0ZKTGpqdFh1WFFsUEcKb3p1VlplY2w2TE03V1Y3aXVpQzJUb2t3TUx6bDVhUWJpZkRoelpCb3FGRTFGbDRXQ0pOaUdJeXgwN2lBZlFGaApYZ3gwMWtmN0w4WWpEVjVTTzhnd0dOeUV2S1kwRGNFanBkamtrTEpvc09GOXJPeURiSkppYk1hYWVlWDV0eUpoCnpYVHVialN6K1pRd0dqNCtVYi96Zmxub2g1eWRxVkptUXNqZkk1UDZIWHJvWWdnYVpTWXVRNFUxT2JPRDJscVAKZVRhTVQxSlZNMUZaMkhteG9NRTdCUWpsUWdwWElOSjBLdEVRRDRySnZEa3ZYRXVpTzcvUGVvdzZjWUFwMmN0UQpKak9ET3A0TmxkbEVkOTA9Ci0tLS0tRU5EIENFUlRJRklDQVRFIFJFUVVFU1QtLS0tLQo=

// See
// https://developer.apple.com/documentation/devicemanagement/implementing_device_management/setting_up_push_notifications_for_your_mdm_customers
// for the expected CSR format.

func main() {
	// Load vendor keys and certs
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
	vendorCert, err := decodeVendorCert([]byte(vendorCertPEM))
	if err != nil {
		log.Fatalf("failed to parse vendor cert: %s", err.Error())
	}
	vendorKey, err := loadKey([]byte(vendorKeyPEM), []byte(vendorKeyPassphrase))
	if err != nil {
		log.Fatalf("failed to load vendor private key: %s", err.Error())
	}

	// Decode CSR input
	// We accept the CSR via environment variable to mitigate against command injection attacks in
	// the fleetdm.com website code that will call this with untrusted user input.
	csrBase64 := os.Getenv(csrEnvName)
	if csrBase64 == "" {
		log.Fatalf("CSR must be set in %s", csrEnvName)
	}
	csr, err := base64.StdEncoding.DecodeString(string(csrBase64))
	if err != nil {
		log.Fatalf("base64 decode csr: %s", err.Error())
	}
	certReq, err := decodeCSR(csr)
	if err != nil {
		log.Fatalf("decode pem: %s", err.Error())
	}

	// Get email from CSR
	email, err := getEmail(certReq)
	if err != nil {
		log.Fatalf("get email: %s", err.Error())
	}

	// Tie it all together
	req, err := createPushCertificateRequest(vendorKey, vendorCert, certReq)
	if err != nil {
		log.Fatalf("create request: %s", err.Error())
	}
	encodedReq, err := req.Encode()
	if err != nil {
		log.Fatalf("encode csr: %v", err)
	}

	// Write output as JSON
	out := struct {
		Email   string `json:"email"`
		Request string `json:"request"`
	}{
		Email:   email,
		Request: string(encodedReq),
	}
	outJSON, err := json.Marshal(out)
	if err != nil {
		log.Fatalf("encode request JSON: %s", err.Error())
	}
	fmt.Println(string(outJSON))
}

func getEmail(req *x509.CertificateRequest) (string, error) {
	for _, name := range req.Subject.Names {
		if slices.Equal(name.Type, emailAddressOID) {
			str, ok := name.Value.(string)
			if !ok {
				return "", errors.New("email subject is not string value")
			}
			return str, nil
		}
	}
	return "", errors.New("missing email subject")
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
	mdmPEM := pemCert(vendorCert.Raw)

	// wwdr cert
	wwdrCertBytes, err := loadCertfromHTTP(wwdrIntermediaryURL)
	if err != nil {
		return nil, fmt.Errorf("load WWDR certificate from %s: %w", wwdrIntermediaryURL, err)
	}
	wwdrPEM := pemCert(wwdrCertBytes)

	// apple root certificate
	rootCertBytes, err := loadCertfromHTTP(appleRootCAURL)
	if err != nil {
		return nil, fmt.Errorf("load root certificate from %s: %w", appleRootCAURL, err)
	}
	rootPEM := pemCert(rootCertBytes)

	csrB64 := base64.StdEncoding.EncodeToString(csr.Raw)
	sig64 := base64.StdEncoding.EncodeToString(signature)
	pushReq := &mdmcertutil.PushCertificateRequest{
		PushCertRequestCSR:       csrB64,
		PushCertCertificateChain: makeCertChain(mdmPEM, wwdrPEM, rootPEM),
		PushCertSignature:        sig64,
	}
	return pushReq, nil
}

func makeCertChain(mdmPEM, wwdrPEM, rootPEM []byte) string {
	return string(mdmPEM) + string(wwdrPEM) + string(rootPEM)
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
		return nil, errors.New("unmatched type or headers")
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

func loadCertfromHTTP(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create GET request for %s: %w", url, err)
	}
	req.Header.Set("Accept", "*/*") // required by Apple at some point.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got %s when trying to http.Get %s", resp.Status, url)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read get certificate request response body: %w", err)
	}

	crt, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("parse wwdr intermediate certificate: %w", err)
	}
	return crt.Raw, nil
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
