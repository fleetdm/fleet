package service

import (
	"fmt"
	"net/http"
	"strings"
)

// mockRoundTripper is a custom http.RoundTripper that redirects requests to a mock server.
// Used by integration tests in this package.
type mockRoundTripper struct {
	mockServer  string
	origBaseURL string
	next        http.RoundTripper
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), rt.origBaseURL) {
		path := strings.TrimPrefix(req.URL.Path, "/")
		newURL := fmt.Sprintf("%s/%s", rt.mockServer, path)
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return rt.next.RoundTrip(newReq)
	}
	return rt.next.RoundTrip(req)
}
