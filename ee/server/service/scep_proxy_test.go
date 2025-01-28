package service

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	filedepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/file"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNDESSCEPAdminURL(t *testing.T) {
	// t.Parallel() // This test is not parallel because it changes the global NDESTimeout

	var returnPage func() []byte
	returnStatus := http.StatusOK
	wait := false
	ndesAdminServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wait {
			time.Sleep(1 * time.Second)
		}
		w.WriteHeader(returnStatus)
		if returnStatus == http.StatusOK {
			_, err := w.Write(returnPage())
			require.NoError(t, err)
		}
	}))
	t.Cleanup(ndesAdminServer.Close)

	proxy := fleet.NDESSCEPProxyIntegration{
		AdminURL: ndesAdminServer.URL,
		Username: "admin",
		Password: "password",
	}

	returnStatus = http.StatusNotFound
	err := ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "unexpected status code")
	returnStatus = http.StatusOK

	// Catch timeout issue
	origNDESTimeout := NDESTimeout
	NDESTimeout = ptr.Duration(1 * time.Microsecond)
	t.Cleanup(func() {
		NDESTimeout = origNDESTimeout
	})
	wait = true
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	wait = false
	NDESTimeout = origNDESTimeout

	// We need to convert the HTML page to UTF-16 encoding, which is used by Windows servers
	returnPageFromFile := func(path string) []byte {
		dat, err := os.ReadFile(path)
		require.NoError(t, err)
		datUTF16, err := utf16FromString(string(dat))
		require.NoError(t, err)
		byteData := make([]byte, len(datUTF16)*2)
		for i, v := range datUTF16 {
			binary.BigEndian.PutUint16(byteData[i*2:], v)
		}
		return byteData
	}

	// Catch ths issue when NDES password cache is full
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_cache_full.html")
	}
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "the password cache is full")

	// Catch ths issue when account has insufficient permissions
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_insufficient_permissions.html")
	}
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "does not have sufficient permissions")

	// Nothing returned
	returnPage = func() []byte {
		return []byte{}
	}
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "could not retrieve the enrollment challenge password")

	// All good
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_password.html")
	}
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.NoError(t, err)
}

func TestValidateNDESSCEPURL(t *testing.T) {
	t.Parallel()
	srv := newSCEPServer(t)

	proxy := fleet.NDESSCEPProxyIntegration{
		URL: srv.URL + "/scep",
	}
	err := ValidateNDESSCEPURL(context.Background(), proxy, kitlog.NewNopLogger())
	assert.NoError(t, err)

	proxy.URL = srv.URL + "/bozo"
	err = ValidateNDESSCEPURL(context.Background(), proxy, kitlog.NewNopLogger())
	assert.ErrorContains(t, err, "could not retrieve CA certificate")

}

// utf16FromString returns the UTF-16 encoding of the UTF-8 string s, with a terminating NUL added.
// If s contains a NUL byte at any location, it returns (nil, syscall.EINVAL).
func utf16FromString(s string) ([]uint16, error) {
	for i := 0; i < len(s); i++ {
		if s[i] == 0 {
			return nil, syscall.EINVAL
		}
	}
	return utf16.Encode([]rune(s + "\x00")), nil
}

func newSCEPServer(t *testing.T) *httptest.Server {
	var err error
	var certDepot depot.Depot // cert storage
	depotPath := "./testdata/testca"
	t.Cleanup(func() {
		_ = os.Remove("./testdata/testca/serial")
		_ = os.Remove("./testdata/testca/index.txt")
	})
	certDepot, err = filedepot.NewFileDepot(depotPath)
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
