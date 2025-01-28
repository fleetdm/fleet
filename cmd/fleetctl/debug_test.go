package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Generated using this command in `go env GOROOT`/src/crypto/tls:
	// go run generate_cert.go --rsa-bits 1024 --host example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
	// Certificate is only valid for example.com, and so should fail validation
	// with a localhost-running httptest.NewTLSServer.
	exampleDotComCertDotPem = `-----BEGIN CERTIFICATE-----
MIICGzCCAYSgAwIBAgIRAM596905ZjtK0p+hURZWO7IwDQYJKoZIhvcNAQELBQAw
EjEQMA4GA1UEChMHQWNtZSBDbzAgFw03MDAxMDEwMDAwMDBaGA8yMDg0MDEyOTE2
MDAwMFowEjEQMA4GA1UEChMHQWNtZSBDbzCBnzANBgkqhkiG9w0BAQEFAAOBjQAw
gYkCgYEA57PzoKfRgAYvOte5RVKEm4g6hD6jhxeg/lyvuidbuL9XzyvWesKGqxXh
LxMTrAeH1T3LbLlU0c/OdwcPQRLErqXee0YM3OeVhlZLnnOfyywE7WRFwAtS+uSm
m61Mrx8VHLqXiN8R3yQPiHmekuHIDMvIkC793d2YpaV02grWH7ECAwEAAaNvMG0w
DgYDVR0PAQH/BAQDAgKkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1UdEwEB/wQF
MAMBAf8wHQYDVR0OBBYEFI3hGM84qbH234gBQmbCShCq0430MBYGA1UdEQQPMA2C
C2V4YW1wbGUuY29tMA0GCSqGSIb3DQEBCwUAA4GBAHqLUn9kpHdAElEwAP/7Xoth
yWkBFCfkIy2ftaWJKTB1nDfxbdEuJ1BfMDYyM5anYd+d/Id7w3fe3Wn+VkOnxxtZ
oug6edBNpdhp8r2/4t6n3AouK0/zG2naAlmXV0JoFuEvy2bX0BbbbPg+v4WNZIsC
0cUq8IOA9g0kHJar8rAI
-----END CERTIFICATE-----`
)

func TestDebugConnectionCommand(t *testing.T) {
	t.Run("without certificate, plain http server", func(t *testing.T) {
		// Plain HTTP server
		_, ds := runServerWithMockedDS(t)

		ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return nil, errors.New("invalid")
		}

		output := runAppForTest(t, []string{"debug", "connection"})
		// 3 successes: resolve host, dial address, check api endpoint
		require.Equal(t, 3, strings.Count(output, "Success:"))
	})

	t.Run("invalid certificate flag without address", func(t *testing.T) {
		_, err := runAppNoChecks([]string{"debug", "connection", "--fleet-certificate", "cert.pem"})
		require.Contains(t, err.Error(), "--fleet-certificate")
	})

	t.Run("invalid context flag with address", func(t *testing.T) {
		_, err := runAppNoChecks([]string{"debug", "connection", "--context", "test", "localhost:8080"})
		require.Contains(t, err.Error(), "--context")
	})

	t.Run("invalid config flag with address", func(t *testing.T) {
		_, err := runAppNoChecks([]string{"debug", "connection", "--config", "/tmp/nosuchfile", "localhost:8080"})
		require.Contains(t, err.Error(), "--config")
	})

	t.Run("with valid certificate", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error": "error", "node_invalid": true}`)
		}))
		defer srv.Close()
		t.Setenv("FLEET_SERVER_ADDRESS", srv.URL)

		// get the certificate of the TLS server
		certPath := rawCertToPemFile(t, srv.Certificate().Raw)

		output := runAppForTest(t, []string{"debug", "connection", "--fleet-certificate", certPath, srv.URL})
		// 4 successes: resolve host, dial address, certificate, check api endpoint
		t.Log(output)
		require.Equal(t, 4, strings.Count(output, "Success:"))
	})

	t.Run("with invalid certificate", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error": "error", "node_invalid": true}`)
		}))
		defer srv.Close()
		t.Setenv("FLEET_SERVER_ADDRESS", srv.URL)

		// get the invalid certificate (for example.com)
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.pem")
		require.NoError(t, os.WriteFile(certPath, []byte(exampleDotComCertDotPem), 0o600))

		buf, err := runAppNoChecks([]string{"debug", "connection", "--fleet-certificate", certPath, srv.URL})
		// 2 successes: resolve host, dial address
		t.Log(buf.String())
		require.Equal(t, 2, strings.Count(buf.String(), "Success:"))
		// 1 failure: invalid certificate
		require.Error(t, err)
		require.Equal(t, 1, strings.Count(err.Error(), "Fail: certificate:"))
	})
}

// encodes raw certificate bytes to a PEM-encoded temp file, returns the path.
func rawCertToPemFile(t *testing.T, raw []byte) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, pem.Encode(&buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: raw,
	}))

	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	require.NoError(t, os.WriteFile(certPath, buf.Bytes(), 0o600))
	return certPath
}

func TestDebugCheckAPIEndpoint(t *testing.T) {
	const timeout = 100 * time.Millisecond
	cases := [...]struct {
		code        int // == 0 panics, negative value waits for timeout, sets status code to absolute value
		body        string
		errContains string // empty if checkAPIEndpoint should not return an error
	}{
		{401, `{"error": "fail", "node_invalid": true}`, ""},
		{-401, `{"error": "fail", "node_invalid": true}`, "deadline exceeded"},
		{200, `{"error": "", "node_invalid": false}`, "unexpected 200 response"},
		{0, `panic`, "EOF"},
	}

	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := cases[atomic.LoadInt32(&callCount)]

		switch {
		case res.code == 0:
			panic(res.body)
		case res.code < 0:
			time.Sleep(timeout + timeout/10)
			res.code = -res.code
		}
		w.WriteHeader(res.code)
		fmt.Fprint(w, res.body)
	}))
	t.Cleanup(func() {
		srv.Close()
	})

	t.Setenv("FLEET_SERVER_ADDRESS", srv.URL)
	cli, base, err := rawHTTPClientFromConfig(Context{Address: srv.URL, TLSSkipVerify: true})
	require.NoError(t, err)
	for i, c := range cases {
		atomic.StoreInt32(&callCount, int32(i)) //nolint:gosec // dismiss G115
		t.Run(fmt.Sprint(c.code), func(t *testing.T) {
			err := checkAPIEndpoint(context.Background(), timeout, base, cli)
			if c.errContains == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.errContains)
			}
		})
	}
}

func TestDebugResolveHostname(t *testing.T) {
	const timeout = 100 * time.Millisecond

	// resolves host name
	err := resolveHostname(context.Background(), timeout, "localhost")
	require.NoError(t, err)

	// resolves ip4 address
	err = resolveHostname(context.Background(), timeout, "127.0.0.1")
	require.NoError(t, err)

	// resolves ip6 address
	err = resolveHostname(context.Background(), timeout, "::1")
	require.NoError(t, err)

	// fails on invalid host
	randBytes := make([]byte, 8)
	_, err = rand.Read(randBytes)
	require.NoError(t, err)
	noSuchHost := "no_such_host" + hex.EncodeToString(randBytes)

	err = resolveHostname(context.Background(), timeout, noSuchHost)
	require.Error(t, err)
}

func TestFilenameFunctions(t *testing.T) {
	nowFn = func() time.Time {
		now, _ := time.Parse(time.RFC3339, "1969-06-19T21:44:05Z")
		return now
	}
	defer func() { nowFn = time.Now }()

	t.Run("outfileName builds a file name using the name provided + current time ", func(t *testing.T) {
		name := outfileName("test")
		assert.Equal(t, "fleet-test-19690619214405Z", name)
	})

	t.Run("outfileNameWithExt builds a file name using the name and extension provided + current time ", func(t *testing.T) {
		name := outfileNameWithExt("test", "go")
		assert.Equal(t, "fleet-test-19690619214405Z.go", name)
	})
}
