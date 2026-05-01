package scep_server

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"log/slog"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	scepdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/gorilla/mux"
	"github.com/smallstep/scep"
)

//go:embed testdata/ca.crt
var caCert []byte

//go:embed testdata/ca.key
var caKey []byte

//go:embed testdata/ca.pem
var caPem []byte

func StartTestSCEPServer(t *testing.T) *httptest.Server {

	caDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(caDir, "ca.crt"), caCert, 0644); err != nil {
		t.Fatalf("failed to write ca.crt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.key"), caKey, 0644); err != nil {
		t.Fatalf("failed to write ca.key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), caPem, 0644); err != nil {
		t.Fatalf("failed to write ca.pem: %v", err)
	}

	// Spin up an "external" SCEP server, which Fleet server will proxy
	newSCEPServer := func(t *testing.T) *httptest.Server {
		var server *httptest.Server
		t.Cleanup(func() {
			if server != nil {
				server.Close()
			}
		})

		var err error
		var certDepot scepdepot.Depot // cert storage
		certDepot, err = filedepot.NewFileDepot(caDir)
		if err != nil {
			t.Fatal(err)
		}
		certDepot = &noopCertDepot{certDepot}
		crt, key, err := certDepot.CA([]byte{})
		if err != nil {
			t.Fatal(err)
		}

		var svc scepserver.Service // scep service
		svc, err = scepserver.NewService(crt[0], key, newInMemorySigner(crt[0], key))
		if err != nil {
			t.Fatal(err)
		}
		logger := slog.New(slog.DiscardHandler)
		e := scepserver.MakeServerEndpoints(svc)
		scepHandler := scepserver.MakeHTTPHandler(e, svc, logger)
		r := mux.NewRouter()
		r.Handle("/scep", scepHandler)
		server = httptest.NewServer(r)
		return server
	}
	scepServer := newSCEPServer(t)
	return scepServer
}

type noopCertDepot struct{ scepdepot.Depot }

func (d *noopCertDepot) Put(_ string, _ *x509.Certificate) error {
	return nil
}

// newInMemorySigner returns a SCEP signer that issues certs directly from the test CA without
// touching any on-disk depot. Each issued cert gets a unique serial derived from a counter so
// the signer is safe under concurrent calls.
//
// The signer does not validate the SCEP challenge: tests that rely on Fleet's SCEP proxy do
// challenge enforcement at the proxy layer (via ConsumeChallenge); tests that hit this server
// directly use whatever challenge the profile carried.
func newInMemorySigner(caCert *x509.Certificate, caKey *rsa.PrivateKey) scepserver.CSRSignerContextFunc {
	var counter atomic.Int64
	return func(_ context.Context, m *scep.CSRReqMessage) (*x509.Certificate, error) {
		serial := big.NewInt(time.Now().UnixNano() + counter.Add(1))
		now := time.Now()
		tpl := &x509.Certificate{
			SerialNumber:          serial,
			Subject:               m.CSR.Subject,
			NotBefore:             now.Add(-1 * time.Minute),
			NotAfter:              now.Add(365 * 24 * time.Hour),
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: true,
			SignatureAlgorithm:    m.CSR.SignatureAlgorithm,
		}
		der, err := x509.CreateCertificate(cryptorand.Reader, tpl, caCert, m.CSR.PublicKey, caKey)
		if err != nil {
			return nil, err
		}
		return x509.ParseCertificate(der)
	}
}
