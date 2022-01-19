package service

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCertificateChain(t *testing.T) {
	server, teardown := setupCertificateChain(t)
	defer teardown()

	certFile := "testdata/server.pem"
	cert, err := tls.LoadX509KeyPair(certFile, "testdata/server.key")
	require.Nil(t, err)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server.StartTLS()

	u, err := url.Parse(server.URL)
	require.Nil(t, err)

	conn, err := connectTLS(context.Background(), u)
	require.Nil(t, err)

	have, want := len(conn.ConnectionState().PeerCertificates), len(cert.Certificate)
	require.Equal(t, have, want)

	original, _ := ioutil.ReadFile(certFile)
	returned, err := chain(context.Background(), conn.ConnectionState(), "")
	require.Nil(t, err)
	require.Equal(t, returned, original)
}

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(dump)
	})
}

func setupCertificateChain(t *testing.T) (server *httptest.Server, teardown func()) {
	server = httptest.NewUnstartedServer(echoHandler())
	return server, server.Close
}
