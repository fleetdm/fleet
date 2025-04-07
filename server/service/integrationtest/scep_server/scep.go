package scep_server

import (
	"crypto/x509"
	_ "embed"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
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
		var certDepot depot.Depot // cert storage
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
		svc, err = scepserver.NewService(crt[0], key, scepserver.NopCSRSigner())
		if err != nil {
			t.Fatal(err)
		}
		logger := kitlog.NewNopLogger()
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

type noopCertDepot struct{ depot.Depot }

func (d *noopCertDepot) Put(_ string, _ *x509.Certificate) error {
	return nil
}
