package scepserver_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/stretchr/testify/require"
)

type identifierRecordingService struct {
	scepserver.ServiceWithIdentifier
	identifier string
}

func (s *identifierRecordingService) GetCACaps(_ context.Context, identifier string) ([]byte, error) {
	s.identifier = identifier
	return []byte("POSTPKIOperation"), nil
}

func TestHTTPHandlerWithIdentifierRouting(t *testing.T) {
	svc := &identifierRecordingService{}
	handler := scepserver.MakeHTTPHandlerWithIdentifier(scepserver.MakeServerEndpointsWithIdentifier(svc), "/mdm/scep/proxy/",
		slog.New(slog.DiscardHandler))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cases := []struct {
		name           string
		method         string
		path           string
		wantStatus     int
		wantIdentifier string
	}{
		{"get", http.MethodGet, "/mdm/scep/proxy/host%2Cprofile%2CNDES", http.StatusOK, "host,profile,NDES"},
		{"get with pkiclient.exe suffix", http.MethodGet, "/mdm/scep/proxy/host%2Cprofile%2CNDES/pkiclient.exe", http.StatusOK,
			"host,profile,NDES"},
		{"escaped slash stays in the identifier", http.MethodGet, "/mdm/scep/proxy/host%2Cprofile%2Cca%2Fname", http.StatusOK,
			"host,profile,ca/name"},
		{"head", http.MethodHead, "/mdm/scep/proxy/host%2Cprofile%2CNDES", http.StatusOK, "host,profile,NDES"},
		{"missing identifier", http.MethodGet, "/mdm/scep/proxy/", http.StatusNotFound, ""},
		{"unknown suffix", http.MethodGet, "/mdm/scep/proxy/host%2Cprofile%2CNDES/other", http.StatusNotFound, ""},
		{"unsupported method", http.MethodPut, "/mdm/scep/proxy/host%2Cprofile%2CNDES", http.StatusMethodNotAllowed, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc.identifier = ""
			req, err := http.NewRequestWithContext(t.Context(), c.method, server.URL+c.path+"?operation=GetCACaps", nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			require.Equal(t, c.wantStatus, resp.StatusCode)
			require.Equal(t, c.wantIdentifier, svc.identifier)
		})
	}
}
