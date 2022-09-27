package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUrlGeneration(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
		require.NoError(t, err)
		require.Equal(t, "https://test.com/test/path", bc.url("test/path", "").String())
		require.Equal(t, "https://test.com/test/path?raw=query", bc.url("test/path", "raw=query").String())
	})

	t.Run("with prefix", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "prefix/", fleet.CapabilityMap{})
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
			bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
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
	bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
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

func TestParseResponseGeneralErrors(t *testing.T) {
	t.Run("general HTTP errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
		require.NoError(t, err)
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewBufferString(`{"test": "ok"}`)),
		}
		err = bc.parseResponse("GET", "", response, &struct{}{})
		require.Error(t, err)
	})

	t.Run("parse errors", func(t *testing.T) {
		bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
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
		_, err := newBaseClient("invalid", true, "", "", fleet.CapabilityMap{})
		require.Error(t, err)
	})

	t.Run("http is only valid in development", func(t *testing.T) {
		_, err := newBaseClient("http://test.com", true, "", "", fleet.CapabilityMap{})
		require.Error(t, err)

		_, err = newBaseClient("http://localhost:8080", true, "", "", fleet.CapabilityMap{})
		require.NoError(t, err)

		_, err = newBaseClient("http://127.0.0.1:8080", true, "", "", fleet.CapabilityMap{})
		require.NoError(t, err)
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
			"test_capability,test_capability_2"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bc, err := newBaseClient("https://test.com", true, "", "", c.capabilities)
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
	bc, err := newBaseClient("https://test.com", true, "", "", fleet.CapabilityMap{})
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
