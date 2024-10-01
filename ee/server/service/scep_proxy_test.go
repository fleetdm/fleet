package service

import (
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"unicode/utf16"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNDESSCEPAdminURL(t *testing.T) {
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

	proxy := &fleet.NDESSCEPProxyIntegration{
		AdminURL: ndesAdminServer.URL,
		Username: "admin",
		Password: "password",
	}

	returnStatus = http.StatusNotFound
	err := ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "unexpected status code")

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
	returnStatus = http.StatusOK
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_cache_full.html")
	}
	err = ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "the password cache is full")

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
