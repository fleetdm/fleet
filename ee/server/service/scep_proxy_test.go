package service

import (
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNDESSCEPAdminURL(t *testing.T) {
	t.Parallel()

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

	proxy := fleet.NDESSCEPProxyCA{
		AdminURL: ndesAdminServer.URL,
		Username: "admin",
		Password: "password",
	}

	returnStatus = http.StatusNotFound
	logger := kitlog.NewNopLogger()
	svc := NewSCEPConfigService(logger, nil)
	err := svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "unexpected status code")
	returnStatus = http.StatusOK

	// Catch timeout issue
	svc = NewSCEPConfigService(logger, ptr.Duration(1*time.Microsecond))
	wait = true
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	wait = false
	svc = NewSCEPConfigService(logger, nil)

	// We need to convert the HTML page to UTF-16 encoding, which is used by Windows servers
	returnPageFromFile := func(path string) []byte {
		dat, err := os.ReadFile(path)
		require.NoError(t, err)
		datUTF16, err := utf16FromString(string(dat))
		require.NoError(t, err)
		byteData := make([]byte, len(datUTF16)*2)
		for i, v := range datUTF16 {
			binary.LittleEndian.PutUint16(byteData[i*2:], v)
		}
		return byteData
	}

	// Catch ths issue when NDES password cache is full
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_cache_full.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "the password cache is full")

	// Catch ths issue when account has insufficient permissions
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_insufficient_permissions.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "does not have sufficient permissions")

	// Nothing returned
	returnPage = func() []byte {
		return []byte{}
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "could not retrieve the enrollment challenge password")

	// All good
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_password.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.NoError(t, err)
}

func TestValidateNDESSCEPURL(t *testing.T) {
	t.Parallel()
	srv := NewTestSCEPServer(t)

	proxy := fleet.NDESSCEPProxyCA{
		URL: srv.URL + "/scep",
	}
	logger := kitlog.NewNopLogger()
	svc := NewSCEPConfigService(logger, nil)
	err := svc.ValidateSCEPURL(context.Background(), proxy.URL)
	assert.NoError(t, err)

	proxy.URL = srv.URL + "/bozo"
	err = svc.ValidateSCEPURL(context.Background(), proxy.URL)
	assert.ErrorContains(t, err, "could not retrieve CA certificate")
}
