package scep_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/scep"
)

func testParsePKIMessage(t *testing.T, data []byte) *scep.PKIMessage {
	msg, err := scep.ParsePKIMessage(data)
	if err != nil {
		t.Fatal(err)
	}
	validateParsedPKIMessage(t, msg)
	return msg
}

func validateParsedPKIMessage(t *testing.T, msg *scep.PKIMessage) {
	if msg.TransactionID == "" {
		t.Errorf("expected TransactionID attribute")
	}
	if msg.MessageType == "" {
		t.Errorf("expected MessageType attribute")
	}
	switch msg.MessageType {
	case scep.CertRep:
		if len(msg.RecipientNonce) == 0 {
			t.Errorf("expected RecipientNonce attribute")
		}
	case scep.PKCSReq, scep.UpdateReq, scep.RenewalReq:
		if len(msg.SenderNonce) == 0 {
			t.Errorf("expected SenderNonce attribute")
		}
	}
}

// Tests the case when servers reply with PKCS #7 signed-data that contains no
// certificates assuming that the client can request CA certificates using
// GetCaCert request.
func TestParsePKIEnvelopeCert_MissingCertificatesForSigners(t *testing.T) {
	certRepMissingCertificates := loadTestFile(t, "testdata/testca2/CertRep_NoCertificatesForSigners.der")
	caPEM := loadTestFile(t, "testdata/testca2/ca2.pem")

	// Try to parse the PKIMessage without providing certificates for signers.
	_, err := scep.ParsePKIMessage(certRepMissingCertificates)
	if err == nil {
		t.Fatal("parsed PKIMessage without providing signer certificates")
	}

	signerCert := decodePEMCert(t, caPEM)
	msg, err := scep.ParsePKIMessage(certRepMissingCertificates, scep.WithCACerts([]*x509.Certificate{signerCert}))
	if err != nil {
		t.Fatalf("failed to parse PKIMessage: %v", err)
	}
	validateParsedPKIMessage(t, msg)
}

func TestDecryptPKIEnvelopeCSR(t *testing.T) {
	pkcsReq := loadTestFile(t, "testdata/PKCSReq.der")
	msg := testParsePKIMessage(t, pkcsReq)
	cacert, cakey := loadCACredentials(t)
	err := msg.DecryptPKIEnvelope(cacert, cakey)
	if err != nil {
		t.Fatal(err)
	}
	if msg.CSRReqMessage.CSR == nil {
		t.Errorf("expected non-nil CSR field")
	}
}

func TestDecryptPKIEnvelopeCert(t *testing.T) {
	certRep := loadTestFile(t, "testdata/CertRep.der")
	testParsePKIMessage(t, certRep)
	// clientcert, clientkey := loadClientCredentials(t)
	// err = msg.DecryptPKIEnvelope(clientcert, clientkey)
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func TestSignCSR(t *testing.T) {
	pkcsReq := loadTestFile(t, "testdata/PKCSReq.der")
	msg := testParsePKIMessage(t, pkcsReq)
	cacert, cakey := loadCACredentials(t)
	err := msg.DecryptPKIEnvelope(cacert, cakey)
	if err != nil {
		t.Fatal(err)
	}
	csr := msg.CSRReqMessage.CSR
	id, err := cryptoutil.GenerateSubjectKeyID(csr.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(4),
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-600).UTC(),
		NotAfter:     time.Now().AddDate(1, 0, 0).UTC(),
		SubjectKeyId: id,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageAny,
			x509.ExtKeyUsageClientAuth,
		},
	}
	// sign the CSR creating a DER encoded cert
	crtBytes, err := x509.CreateCertificate(rand.Reader, tmpl, cacert, csr.PublicKey, cakey)
	if err != nil {
		t.Fatal(err)
	}
	crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		t.Fatal(err)
	}
	certRep, err := msg.Success(cacert, cakey, crt)
	if err != nil {
		t.Fatal(err)
	}
	testParsePKIMessage(t, certRep.Raw)
}

func TestNewCSRRequest(t *testing.T) {
	for _, test := range []struct {
		testName          string
		keyUsage          x509.KeyUsage
		certsSelectorFunc scep.CertsSelectorFunc
		shouldCreateCSR   bool
	}{
		{
			"KeyEncipherment not set with NOP certificates selector",
			x509.KeyUsageCertSign,
			scep.NopCertsSelector(),
			true,
		},
		{
			"KeyEncipherment is set with NOP certificates selector",
			x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
			scep.NopCertsSelector(),
			true,
		},
		{
			"KeyEncipherment not set with Encipherment certificates selector",
			x509.KeyUsageCertSign,
			scep.EnciphermentCertsSelector(),
			false,
		},
		{
			"KeyEncipherment is set with Encipherment certificates selector",
			x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
			scep.EnciphermentCertsSelector(),
			true,
		},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()
			key, err := newRSAKey(2048)
			if err != nil {
				t.Fatal(err)
			}
			derBytes, err := newCSR(key, "john.doe@example.com", "US", "com.apple.scep.2379B935-294B-4AF1-A213-9BD44A2C6688")
			if err != nil {
				t.Fatal(err)
			}
			csr, err := x509.ParseCertificateRequest(derBytes)
			if err != nil {
				t.Fatal(err)
			}
			clientcert, clientkey := loadClientCredentials(t)
			cacert, cakey := createCaCertWithKeyUsage(t, test.keyUsage)
			tmpl := &scep.PKIMessage{
				MessageType: scep.PKCSReq,
				Recipients:  []*x509.Certificate{cacert},
				SignerCert:  clientcert,
				SignerKey:   clientkey,
			}

			pkcsreq, err := scep.NewCSRRequest(csr, tmpl, scep.WithCertsSelector(test.certsSelectorFunc))
			if test.shouldCreateCSR && err != nil {
				t.Fatalf("keyUsage: %d, failed creating a CSR request: %v", test.keyUsage, err)
			}
			if !test.shouldCreateCSR && err == nil {
				t.Fatalf("keyUsage: %d, shouldn't have created a CSR: %v", test.keyUsage, err)
			}
			if !test.shouldCreateCSR {
				return
			}
			msg := testParsePKIMessage(t, pkcsreq.Raw)
			err = msg.DecryptPKIEnvelope(cacert, cakey)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// create a new RSA private key
func newRSAKey(bits int) (*rsa.PrivateKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return private, nil
}

// create a CSR using the same parameters as Keychain Access would produce
func newCSR(priv *rsa.PrivateKey, email, country, cname string) ([]byte, error) {
	subj := pkix.Name{
		Country:    []string{country},
		CommonName: cname,
		ExtraNames: []pkix.AttributeTypeAndValue{{
			Type:  []int{1, 2, 840, 113549, 1, 9, 1},
			Value: email,
		}},
	}
	template := &x509.CertificateRequest{
		Subject: subj,
	}
	return x509.CreateCertificateRequest(rand.Reader, template, priv)
}

func loadTestFile(t *testing.T, path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// createCaCertWithKeyUsage generates a CA key and certificate with keyUsage.
func createCaCertWithKeyUsage(t *testing.T, keyUsage x509.KeyUsage) (*x509.Certificate, *rsa.PrivateKey) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	caCert := depot.NewCACert(
		depot.WithCountry("US"),
		depot.WithOrganization("MICROMDM"),
		depot.WithCommonName("MICROMDM SCEP CA"),
		depot.WithKeyUsage(keyUsage),
	)
	crtBytes, err := caCert.SelfSign(rand.Reader, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

func loadCACredentials(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	cert, err := loadCertFromFile("testdata/testca/ca.crt")
	if err != nil {
		t.Fatal(err)
	}
	key, err := loadKeyFromFile("testdata/testca/ca.key")
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

func loadClientCredentials(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	cert, err := loadCertFromFile("testdata/testclient/client.pem")
	if err != nil {
		t.Fatal(err)
	}
	key, err := loadKeyFromFile("testdata/testclient/client.key")
	if err != nil {
		t.Fatal(err)
	}
	return cert, key
}

const (
	rsaPrivateKeyPEMBlockType = "RSA PRIVATE KEY"
	certificatePEMBlockType   = "CERTIFICATE"
)

func loadCertFromFile(path string) (*x509.Certificate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	if pemBlock.Type != certificatePEMBlockType {
		return nil, errors.New("unmatched type or headers")
	}
	return x509.ParseCertificate(pemBlock.Bytes)
}

// load an encrypted private key from disk
func loadKeyFromFile(path string) (*rsa.PrivateKey, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("PEM decode failed")
	}
	if pemBlock.Type != rsaPrivateKeyPEMBlockType {
		return nil, errors.New("unmatched type or headers")
	}

	// testca key has a password
	if len(pemBlock.Headers) > 0 {
		password := []byte("")
		b, err := x509.DecryptPEMBlock(pemBlock, password)
		if err != nil {
			return nil, err
		}
		return x509.ParsePKCS1PrivateKey(b)
	}

	return x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
}

func decodePEMCert(t *testing.T, data []byte) *x509.Certificate {
	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		t.Fatal("PEM decode failed")
	}
	if pemBlock.Type != certificatePEMBlockType {
		t.Fatal("unmatched type or headers")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
