package certificate

import (
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchPEMInvalidHostname(t *testing.T) {
	t.Parallel()

	_, err := FetchPEM("foobar")
	require.Error(t, err)
}

func TestFetchPEM(t *testing.T) {
	t.Parallel()

	certPath := filepath.Join("testdata", "test.crt")
	keyPath := filepath.Join("testdata", "test.key")
	expectedCert, err := ioutil.ReadFile(certPath)
	require.NoError(t, err)

	var port int
	go func() {
		// Assign any available port
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		port = listener.Addr().(*net.TCPAddr).Port
		defer listener.Close()

		err = http.ServeTLS(listener, nil, certPath, keyPath)
		require.NoError(t, err)
	}()
	// Sleep to allow the goroutine to run and start the server.
	time.Sleep(10 * time.Millisecond)

	pem, err := FetchPEM("localhost:" + strconv.Itoa(port))
	require.NoError(t, err)
	assert.Equal(t, expectedCert, pem)
}
