package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func GenerateRandomCertificateSerialNumber() (*big.Int, error) {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, limit)
}

func SimpleSelfSignedRSAKeypair(cn string, days int) (key *rsa.PrivateKey, cert *x509.Certificate, err error) {
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return key, cert, err
	}

	serialNumber, err := GenerateRandomCertificateSerialNumber()
	if err != nil {
		return key, cert, err
	}
	timeNow := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             timeNow,
		NotAfter:              timeNow.Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{cn},
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return key, cert, err
	}
	cert, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return key, cert, err
	}

	return key, cert, err
}

type NopService struct{}

func (s *NopService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	return nil
}

func (s *NopService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	return nil
}

func (s *NopService) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	return nil
}

func (s *NopService) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	return nil, nil
}

func (s *NopService) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	return nil
}

func (s *NopService) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	return nil, nil
}

func (s *NopService) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	return nil, nil
}

func (s *NopService) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return nil, nil
}

func (s *NopService) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	return nil, nil
}
