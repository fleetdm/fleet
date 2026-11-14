package apnsmock

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// serve starts handler on 127.0.0.1 and returns its port.
func serve(t *testing.T, handler http.HandlerFunc) uint16 {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		t.Fatal(err)
	}
	return uint16(port)
}

func TestProbeReadsNodeStats(t *testing.T) {
	port := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stats" {
			http.NotFound(w, r)
			return
		}
		// Shape of cmd/apple-apns-mock's statsResponse; cluster is the sum
		// across instances and must not be what we report for one mock.
		_, _ = w.Write([]byte(`{"node_id":"n","nodes":1,
			"node":{"active_connections":12,"total_pushes":34},
			"cluster":{"active_connections":99,"total_pushes":99}}`))
	})
	got := Probe(port)
	if !got.Up || got.ActiveConnections != 12 || got.TotalPushes != 34 {
		t.Errorf("Probe = %+v, want up with 12 connections and 34 pushes", got)
	}
	if got.URL != URL(port) {
		t.Errorf("URL = %q, want %q", got.URL, URL(port))
	}
}

func TestProbeDown(t *testing.T) {
	// Grab a free port, then close it so nothing is listening.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint16(l.Addr().(*net.TCPAddr).Port)
	l.Close()

	if got := Probe(port); got.Up {
		t.Errorf("Probe on a closed port = %+v, want down", got)
	}
}

func TestProbeNotTheMock(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"error status": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
		},
		"not JSON": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html>some other server</html>"))
		},
		"other JSON": func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := Probe(serve(t, handler)); got.Up {
				t.Errorf("Probe = %+v, want down", got)
			}
		})
	}
}
