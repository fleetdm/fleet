package scepserver_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
	"time"

	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	boltdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/bolt"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"

	"github.com/smallstep/scep"
	bolt "go.etcd.io/bbolt"
)

func TestCaCert(t *testing.T) {
	// init bolt depot CA
	boltDepot := createDB(0o666, nil)
	key, err := boltDepot.CreateOrLoadKey(2048)
	if err != nil {
		t.Fatal(err)
	}
	_, err = boltDepot.CreateOrLoadCA(key, 5, "MicroMDM", "US")
	if err != nil {
		t.Fatal(err)
	}

	// use exported interface
	depot := scepdepot.Depot(boltDepot)

	// load CA & key again
	certs, key, err := depot.CA([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	caCert := certs[0]

	// SCEP service
	svc, err := scepserver.NewService(caCert, key, scepserver.SignCSRAdapter(scepdepot.NewSigner(depot)))
	if err != nil {
		t.Fatal(err)
	}

	// generate scep "client" keys, csr, cert
	selfKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	csrBytes, err := newCSR(selfKey, "ou", "loc", "province", "country", "cname", "org")
	if err != nil {
		t.Fatal(err)
	}
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		t.Fatal(err)
	}
	signerCert, err := selfSign(selfKey, csr)
	if err != nil {
		t.Fatal(err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	var serCollector []*big.Int

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		// check CA
		caBytes, num, err := svc.GetCACert(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if have, want := num, 1; have != want {
			t.Errorf("i=%d, have %d, want %d", i, have, want)
		}

		if have, want := caBytes, caCert.Raw; !bytes.Equal(have, want) {
			t.Errorf("i=%d, have %v, want %v", i, have, want)
		}

		// create scep "client" request
		tmpl := &scep.PKIMessage{
			MessageType: scep.PKCSReq,
			Recipients:  []*x509.Certificate{caCert},
			SignerKey:   selfKey,
			SignerCert:  signerCert,
		}
		msg, err := scep.NewCSRRequest(csr, tmpl)
		if err != nil {
			t.Fatal(err)
		}

		// submit to service
		respMsgBytes, err := svc.PKIOperation(ctx, msg.Raw)
		if err != nil {
			t.Fatal(err)
		}

		// read and decrypt reply
		respMsg, err := scep.ParsePKIMessage(respMsgBytes)
		if err != nil {
			t.Fatal(err)
		}

		err = respMsg.DecryptPKIEnvelope(signerCert, selfKey)
		if err != nil {
			t.Fatal(err)
		}

		// verify issued certificate is from the CA
		respCert := respMsg.CertRepMessage.Certificate
		opts := x509.VerifyOptions{
			Roots:     roots,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		chains, err := respCert.Verify(opts)
		if err != nil {
			t.Error(err)
		}
		if len(chains) < 1 {
			t.Error("no established chain between issued cert and CA")
		}

		if csr.SignatureAlgorithm != respCert.SignatureAlgorithm {
			t.Fatal(fmt.Errorf("cert signature algo %s different from csr signature algo %s",
				csr.SignatureAlgorithm.String(),
				respCert.SignatureAlgorithm.String()))
		}

		// verify unique certificate serials
		for _, ser := range serCollector {
			if respCert.SerialNumber.Cmp(ser) == 0 {
				t.Error("seen serial number before!")
			}
		}
		serCollector = append(serCollector, respCert.SerialNumber)
	}

}

func createDB(mode os.FileMode, options *bolt.Options) *boltdepot.Depot {
	// Create temporary path.
	f, _ := ioutil.TempFile("", "bolt-")
	f.Close()
	os.Remove(f.Name())

	db, err := bolt.Open(f.Name(), mode, options)
	if err != nil {
		panic(err.Error())
	}
	d, err := boltdepot.NewBoltDepot(db)
	if err != nil {
		panic(err.Error())
	}
	return d
}

func newCSR(priv *rsa.PrivateKey, ou string, locality string, province string, country string, cname, org string) ([]byte, error) {
	subj := pkix.Name{
		CommonName: cname,
	}
	if len(org) > 0 {
		subj.Organization = []string{org}
	}
	if len(ou) > 0 {
		subj.OrganizationalUnit = []string{ou}
	}
	if len(province) > 0 {
		subj.Province = []string{province}
	}
	if len(locality) > 0 {
		subj.Locality = []string{locality}
	}
	if len(country) > 0 {
		subj.Country = []string{country}
	}
	template := &x509.CertificateRequest{
		Subject: subj,
	}
	return x509.CreateCertificateRequest(rand.Reader, template, priv)
}

func selfSign(priv *rsa.PrivateKey, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 1)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "SCEP SIGNER",
			Organization: csr.Subject.Organization,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(derBytes)
}
