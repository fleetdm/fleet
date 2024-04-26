package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/cryptoutil/x509util"
)

const (
	csrPEMBlockType = "CERTIFICATE REQUEST"
)

type csrOptions struct {
	cn, org, country, ou, locality, province, challenge string
	key                                                 *rsa.PrivateKey
}

func loadOrMakeCSR(path string, opts *csrOptions) (*x509.CertificateRequest, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		if os.IsExist(err) {
			return loadCSRfromFile(path)
		}
		return nil, err
	}
	defer file.Close()

	subject := pkix.Name{
		CommonName:         opts.cn,
		Organization:       subjOrNil(opts.org),
		OrganizationalUnit: subjOrNil(opts.ou),
		Province:           subjOrNil(opts.province),
		Locality:           subjOrNil(opts.locality),
		Country:            subjOrNil(opts.country),
	}
	template := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject:            subject,
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
	}
	if opts.challenge != "" {
		template.ChallengePassword = opts.challenge
	}

	derBytes, _ := x509util.CreateCertificateRequest(rand.Reader, &template, opts.key)
	pemBlock := &pem.Block{
		Type:  csrPEMBlockType,
		Bytes: derBytes,
	}
	if err := pem.Encode(file, pemBlock); err != nil {
		return nil, err
	}
	return x509.ParseCertificateRequest(derBytes)
}

// returns nil or []string{input} to populate pkix.Name.Subject
func subjOrNil(input string) []string {
	if input == "" {
		return nil
	}
	return []string{input}
}

// load PEM encoded CSR from file
func loadCSRfromFile(path string) (*x509.CertificateRequest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(data)
	if pemBlock == nil {
		return nil, errors.New("cannot find the next PEM formatted block")
	}
	if pemBlock.Type != csrPEMBlockType || len(pemBlock.Headers) != 0 {
		return nil, errors.New("unmatched type or headers")
	}
	return x509.ParseCertificateRequest(pemBlock.Bytes)
}
