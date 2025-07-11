package service

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUrlGeneration(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		require.Equal(t, "https://test.com/test/path", bc.url("test/path", "").String())
		require.Equal(t, "https://test.com/test/path?raw=query", bc.url("test/path", "raw=query").String())
	})

	t.Run("with prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "prefix/", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		require.Equal(t, "https://test.com/prefix/test/path", bc.url("test/path", "").String())
		require.Equal(t, "https://test.com/prefix/test/path?raw=query", bc.url("test/path", "raw=query").String())
	})
}

func TestParseResponseKnownErrors(t *testing.T) {
	cases := []struct {
		message string
		code    int
		out     error
	}{
		{"not found errors", http.StatusNotFound, notFoundErr{}},
		{"unauthenticated errors", http.StatusUnauthorized, ErrUnauthenticated},
		{"license errors", http.StatusPaymentRequired, ErrMissingLicense},
	}

	for _, c := range cases {
		t.Run(c.message, func(t *testing.T) {
			bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
			require.NoError(t, err)
			response := &http.Response{
				StatusCode: c.code,
				Body:       io.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
			}
			err = bc.parseResponse("GET", "", response, &struct{}{})
			require.ErrorIs(t, err, c.out)
		})
	}
}

func TestParseResponseOK(t *testing.T) {
	bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
	}

	var resDest struct{ Test string }
	err = bc.parseResponse("", "", response, &resDest)
	require.NoError(t, err)
	require.Equal(t, "ok", resDest.Test)
}

func TestParseResponseOKNoContent(t *testing.T) {
	bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	response := &http.Response{
		StatusCode: http.StatusNoContent,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}

	var resDest struct{ Err error }
	err = bc.parseResponse("", "", response, &resDest)
	require.NoError(t, err)
	require.Nil(t, resDest.Err)
}

func TestParseResponseGeneralErrors(t *testing.T) {
	t.Run("general HTTP errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
		}
		err = bc.parseResponse("GET", "", response, &struct{}{})
		require.Error(t, err)
	})

	t.Run("parse errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.NoError(t, err)
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(`invalid json`)),
		}
		err = bc.parseResponse("GET", "", response, &struct{}{})
		require.Error(t, err)
	})
}

func TestNewBaseClient(t *testing.T) {
	t.Run("invalid addresses are an error", func(t *testing.T) {
		_, err := newBaseClient("http://foo\x7f.com/", true, "", "", nil, fleet.CapabilityMap{}, nil)
		require.Error(t, err)
	})

	t.Run("http is only valid in development", func(t *testing.T) {
		cases := []struct {
			name               string
			address            string
			insecureSkipVerify bool
			expectedErr        error
		}{
			{"http non-local URL without insecureSkipVerify", "http://test.com", false, errInvalidScheme},
			{"http non-local URL with insecureSkipVerify", "http://test.com", true, nil},
			{"https", "https://test.com", false, nil},
			{"http localhost with insecureSkipVerify", "http://localhost:8080", true, nil},
			{"http localhost without insecureSkipVerify", "http://localhost:8080", false, nil},
			{"http local ip with insecureSkipVerify", "http://127.0.0.1:8080", true, nil},
			{"http local ip without insecureSkipVerify", "http://127.0.0.1:8080", false, nil},
		}

		for _, c := range cases {
			_, err := newBaseClient(c.address, c.insecureSkipVerify, "", "", nil, fleet.CapabilityMap{}, nil)
			require.Equal(t, c.expectedErr, err, c.name)
		}
	})
}

func TestClientCapabilities(t *testing.T) {
	cases := []struct {
		name         string
		capabilities fleet.CapabilityMap
		expected     string
	}{
		{"no capabilities", fleet.CapabilityMap{}, ""},
		{"one capability", fleet.CapabilityMap{fleet.Capability("test_capability"): {}}, "test_capability"},
		{
			"multiple capabilities",
			fleet.CapabilityMap{
				fleet.Capability("test_capability"):   {},
				fleet.Capability("test_capability_2"): {},
			},
			"test_capability,test_capability_2",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bc, err := newBaseClient("https://test.com", true, "", "", nil, c.capabilities, nil)
			require.NoError(t, err)

			var req http.Request
			bc.setClientCapabilitiesHeader(&req)
			require.ElementsMatch(t, strings.Split(c.expected, ","), strings.Split(req.Header.Get(fleet.CapabilitiesHeader), ","))
		})
	}
}

func TestServerCapabilities(t *testing.T) {
	// initial response has a single capability
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{fleet.CapabilitiesHeader: []string{"test_capability"}},
	}
	bc, err := newBaseClient("https://test.com", true, "", "", nil, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	testCapability := fleet.Capability("test_capability")

	err = bc.parseResponse("", "", response, &struct{}{})
	require.NoError(t, err)
	require.True(t, bc.GetServerCapabilities().Has(testCapability))

	// later on, the server is downgraded and no longer has the capability
	response = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{},
	}
	err = bc.parseResponse("", "", response, &struct{}{})
	require.NoError(t, err)
	require.Equal(t, fleet.CapabilityMap{}, bc.serverCapabilities)
	require.False(t, bc.GetServerCapabilities().Has(testCapability))

	// after an upgrade, the server has many capabilities
	response = &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		Header:     http.Header{fleet.CapabilitiesHeader: []string{"test_capability,test_capability_2"}},
	}
	err = bc.parseResponse("", "", response, &struct{}{})
	require.NoError(t, err)
	require.Equal(t, fleet.CapabilityMap{
		testCapability:                        {},
		fleet.Capability("test_capability_2"): {},
	}, bc.serverCapabilities)
	require.True(t, bc.GetServerCapabilities().Has(testCapability))
	require.True(t, bc.GetServerCapabilities().Has(fleet.Capability("test_capability")))
}

func TestClientCertificateAuth(t *testing.T) {
	httpRequestReceived := false

	clientCAs, err := certificate.LoadPEM(filepath.Join("testdata", "client-ca.crt"))
	require.NoError(t, err)

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpRequestReceived = true
	}))
	ts.TLS = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAs,
	}

	ts.StartTLS()
	t.Cleanup(func() {
		ts.Close()
	})

	// Try connecting without setting TLS client certificates.
	bc, err := newBaseClient(ts.URL, true, "", "", nil, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	request, err := http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)
	_, err = bc.http.Do(request)
	require.Error(t, err)
	require.False(t, httpRequestReceived)

	// Now try connecting by setting the correct TLS client certificates.
	clientCrt, err := certificate.LoadClientCertificateFromFiles(filepath.Join("testdata", "client.crt"), filepath.Join("testdata", "client.key"))
	require.NoError(t, err)
	require.NotNil(t, clientCrt)
	bc, err = newBaseClient(ts.URL, true, "", "", &clientCrt.Crt, fleet.CapabilityMap{}, nil)
	require.NoError(t, err)
	request, err = http.NewRequest("GET", ts.URL, nil)
	require.NoError(t, err)
	_, err = bc.http.Do(request)
	require.NoError(t, err)
	require.True(t, httpRequestReceived)
}
