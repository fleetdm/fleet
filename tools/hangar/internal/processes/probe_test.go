package processes

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestServeTCPCheck(t *testing.T) {
	// A real TLS server (self-signed test cert) should pass — the probe
	// accepts any cert, like fleet's dev cert.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	if !ServeTCPCheck(host, uint16(port)) {
		t.Errorf("ServeTCPCheck should succeed against the TLS server at %s", u.Host)
	}

	// A closed port should fail fast (connection refused).
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().(*net.TCPAddr)
	l.Close() // free the port
	if ServeTCPCheck("127.0.0.1", uint16(addr.Port)) {
		t.Error("ServeTCPCheck should fail against a closed port")
	}
}
