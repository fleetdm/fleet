package scep

import (
	"crypto/x509"
	_ "embed"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
)

//go:embed testdata/testca/ca.key
var caKey []byte

//go:embed testdata/testca/ca.pem
var caPem []byte

// NewTestSCEPServer creates a new SCEP server for testing purposes. The depotPath should be the
// relative path to the directory where the test CA files are stored (e.g., "./testdata/testca")
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
	logger := kitlog.NewNopLogger()
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

//go:embed testdata/mscep_admin_cache_full.html
var mscepAdminCacheFull []byte

//go:embed testdata/mscep_admin_insufficient_permissions.html
var mscepAdminInsufficientPermissions []byte

//go:embed testdata/mscep_admin_password.html
var mscepAdminPassword []byte

func NewTestNDESAdminServer(t *testing.T, responseTemplate string, responseStatus int) *httptest.Server {
	t.Helper()

	var returnPage func() []byte
	returnStatus := http.StatusOK
	ndesAdminServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(returnStatus)
		if returnStatus == http.StatusOK {
			_, err := w.Write(returnPage())
			require.NoError(t, err)
		}
	}))
	t.Cleanup(ndesAdminServer.Close)

	// We need to convert the HTML page to UTF-16 encoding, which is used by Windows servers
	convertHTML := func(html []byte) []byte {
		datUTF16, err := testutils.UTF16FromString(string(html))
		require.NoError(t, err)
		byteData := make([]byte, len(datUTF16)*2)
		for i, v := range datUTF16 {
			binary.LittleEndian.PutUint16(byteData[i*2:], v)
		}
		return byteData
	}

	switch responseTemplate {
	case "mscep_admin_cache_full":
		// Catch ths issue when NDES password cache is full
		returnPage = func() []byte {
			return convertHTML(mscepAdminCacheFull)
		}
	case "mscep_admin_insufficient_permissions":
		// Catch this issue when account has insufficient permissions
		returnPage = func() []byte {
			return convertHTML(mscepAdminInsufficientPermissions)
		}
	case "mscep_admin_password":
		// All good, NDES admin page loads correctly
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

func NewTestDynamicChallengeServer(t *testing.T) *httptest.Server {
	t.Helper()

	dynamicChallengeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Println(r.URL.Path)
		_, err := w.Write([]byte("dynamic challenge"))
		require.NoError(t, err)
	}))
	t.Cleanup(dynamicChallengeServer.Close)

	return dynamicChallengeServer
}
