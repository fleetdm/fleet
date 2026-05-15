// Package sceptest provides test helpers (SCEP test server, NDES admin
// page emulator, dynamic challenge server) for tests that exercise the SCEP
// proxy.
//
// It imports the "testing" package and must therefore only ever be imported
// from test code; importing it from production code would pull "testing"
// into the resulting binary.
package sceptest

import (
	"crypto/x509"
	_ "embed"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unicode/utf16"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/testca/ca.key
var caKey []byte

//go:embed testdata/testca/ca.pem
var caPem []byte

//go:embed testdata/mscep_admin_cache_full.html
var mscepAdminCacheFull []byte

//go:embed testdata/mscep_admin_insufficient_permissions.html
var mscepAdminInsufficientPermissions []byte

//go:embed testdata/mscep_admin_password.html
var mscepAdminPassword []byte

// NewTestSCEPServer creates a new SCEP server for testing purposes, backed by
// an in-temp-dir CA built from embedded test certs.
func NewTestSCEPServer(t *testing.T) *httptest.Server {
	t.Helper()

	caDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(caDir, "ca.key"), caKey, 0o644); err != nil {
		t.Fatalf("failed to write ca.key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(caDir, "ca.pem"), caPem, 0o644); err != nil {
		t.Fatalf("failed to write ca.pem: %v", err)
	}

	var err error
	var certDepot depot.Depot // cert storage
	t.Cleanup(func() {
		_ = os.Remove(caDir)
	})
	certDepot, err = filedepot.NewFileDepot(caDir)
	if err != nil {
		t.Fatal(err)
	}
	certDepot = &noopDepot{certDepot}
	crt, key, err := certDepot.CA([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	var svc scepserver.Service // scep service
	svc, err = scepserver.NewService(crt[0], key, scepserver.NopCSRSigner())
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	e := scepserver.MakeServerEndpoints(svc)
	scepHandler := scepserver.MakeHTTPHandler(e, svc, logger)
	r := mux.NewRouter()
	r.Handle("/scep", scepHandler)
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return server
}

type noopDepot struct{ depot.Depot }

func (d *noopDepot) Put(_ string, _ *x509.Certificate) error {
	return nil
}

// NewTestNDESAdminServer creates an httptest.Server that emulates the NDES
// admin page. responseTemplate selects which canned response to serve:
// "mscep_admin_cache_full", "mscep_admin_insufficient_permissions",
// "mscep_admin_password", or anything else for an empty body.
func NewTestNDESAdminServer(t *testing.T, responseTemplate string, _ int) *httptest.Server {
	t.Helper()

	var returnPage func() []byte
	returnStatus := http.StatusOK
	ndesAdminServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(returnStatus)
		if returnStatus == http.StatusOK {
			if _, err := w.Write(returnPage()); err != nil {
				t.Errorf("write NDES admin response: %v", err)
			}
		}
	}))
	t.Cleanup(ndesAdminServer.Close)

	// We need to convert the HTML page to UTF-16 encoding, which is used by Windows servers
	convertHTML := func(html []byte) []byte {
		datUTF16, err := utf16FromString(string(html))
		require.NoError(t, err)
		byteData := make([]byte, len(datUTF16)*2)
		for i, v := range datUTF16 {
			binary.LittleEndian.PutUint16(byteData[i*2:], v)
		}
		return byteData
	}

	switch responseTemplate {
	case "mscep_admin_cache_full":
		returnPage = func() []byte {
			return convertHTML(mscepAdminCacheFull)
		}
	case "mscep_admin_insufficient_permissions":
		returnPage = func() []byte {
			return convertHTML(mscepAdminInsufficientPermissions)
		}
	case "mscep_admin_password":
		returnPage = func() []byte {
			return convertHTML(mscepAdminPassword)
		}
	default:
		returnPage = func() []byte {
			return []byte{}
		}
	}

	return ndesAdminServer
}

// NewTestDynamicChallengeServer creates an httptest.Server that emulates a
// dynamic SCEP challenge endpoint, always returning the string
// "dynamic challenge".
func NewTestDynamicChallengeServer(t *testing.T) *httptest.Server {
	t.Helper()

	dynamicChallengeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Println(r.URL.Path)
		if _, err := w.Write([]byte("dynamic challenge")); err != nil {
			t.Errorf("write dynamic challenge response: %v", err)
		}
	}))
	t.Cleanup(dynamicChallengeServer.Close)

	return dynamicChallengeServer
}

// utf16FromString returns the UTF-16 encoding of the UTF-8 string s, with a
// terminating NUL added. If s contains a NUL byte at any location, it
// returns (nil, syscall.EINVAL).
func utf16FromString(s string) ([]uint16, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return nil, syscall.EINVAL
		}
	}
	return utf16.Encode([]rune(s + "\x00")), nil
}
