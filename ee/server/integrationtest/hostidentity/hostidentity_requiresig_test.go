//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"bytes"
	"crypto/elliptic"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/stretchr/testify/require"
)

func TestHostIdentityRequireSignature(t *testing.T) {
	// Set up suite with requireSignature = true
	s := SetUpSuite(t, "integrationtest.HostIdentityRequireSignature", true)

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"OrbitEnrollAndConfig", testOrbitEnrollAndConfigWithRequiredSignature},
		{"OsqueryEnrollFailsWithoutSignature", testOsqueryEnrollFailsWithoutSignature},
		{"OrbitEnrollFailsWithoutSignature", testOrbitEnrollFailsWithoutSignature},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.BaseSuite.DS, []string{
				"host_identity_scep_serials", "host_identity_scep_certificates",
			}...)
			c.fn(t, s)
		})
	}
}

func testOrbitEnrollAndConfigWithRequiredSignature(t *testing.T, s *Suite) {
	// Get certificate using shared function from hostidentity_test.go
	cert, eccPrivateKey := testGetCertWithCurve(t, s, elliptic.P384())

	// Test enrollment first WITHOUT signature (should fail)
	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + cert.Subject.CommonName,
		HardwareSerial:    "test-serial-" + cert.Subject.CommonName,
		Hostname:          "test-hostname-" + cert.Subject.CommonName,
		OsqueryIdentifier: cert.Subject.CommonName,
	}

	// Test without signature first (should fail)
	s.Do(t, "POST", "/api/fleet/orbit/enroll", enrollRequest, http.StatusUnauthorized)

	// Now test with signature (should succeed)
	reqBody, err := json.Marshal(enrollRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/enroll", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Create signer using the shared helper from hostidentity_test.go
	signer := createHTTPSigner(t, eccPrivateKey, cert)

	// Sign the request
	err = signer.Sign(req)
	require.NoError(t, err)

	// Send the signed request
	client := fleethttp.NewClient()
	httpResp, err := client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	// The request with a valid HTTP signature should succeed
	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Orbit enrollment with HTTP signature should succeed")

	// Parse the response
	var enrollResp enrollOrbitResponse
	err = json.NewDecoder(httpResp.Body).Decode(&enrollResp)
	require.NoError(t, err)
	require.NotEmpty(t, enrollResp.OrbitNodeKey, "Should receive orbit node key")
	require.NoError(t, enrollResp.Err)

	// Test config endpoint without signature (should fail)
	configReq := orbitConfigRequest{OrbitNodeKey: enrollResp.OrbitNodeKey}
	s.Do(t, "POST", "/api/fleet/orbit/config", configReq, http.StatusUnauthorized)

	// Test config endpoint with signature (should succeed)
	reqBody, err = json.Marshal(configReq)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", s.Server.URL+"/api/fleet/orbit/config", bytes.NewReader(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	err = signer.Sign(req)
	require.NoError(t, err)

	httpResp, err = client.Do(req)
	require.NoError(t, err)
	defer httpResp.Body.Close()

	require.Equal(t, http.StatusOK, httpResp.StatusCode, "Config request with HTTP signature should succeed")
}

func testOsqueryEnrollFailsWithoutSignature(t *testing.T, s *Suite) {
	// Test osquery enrollment without signature (should fail)
	enrollRequest := contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   testEnrollmentSecret,
		HostIdentifier: "osquery-enroll-without-signature-test",
		HostDetails: map[string]map[string]string{
			"osquery_info": {
				"version": "5.0.0",
			},
		},
	}

	// Send request without HTTP signature (should fail)
	// Use Do instead of DoJSON since the server returns HTML error pages
	s.Do(t, "POST", "/api/v1/osquery/enroll", enrollRequest, http.StatusUnauthorized)

	// Also test the alternative osquery enroll endpoint
	s.Do(t, "POST", "/api/osquery/enroll", enrollRequest, http.StatusUnauthorized)
}

func testOrbitEnrollFailsWithoutSignature(t *testing.T, s *Suite) {
	identifier := "orbit-enroll-without-signature-test"
	// Test orbit enrollment without signature (should fail)
	enrollRequest := contract.EnrollOrbitRequest{
		EnrollSecret:      testEnrollmentSecret,
		HardwareUUID:      "test-uuid-" + identifier,
		HardwareSerial:    "test-serial-" + identifier,
		Hostname:          "test-hostname-" + identifier,
		OsqueryIdentifier: identifier,
	}

	// Send request without HTTP signature (should fail)
	// Use Do instead of DoJSON since the server returns HTML error pages
	s.Do(t, "POST", "/api/fleet/orbit/enroll", enrollRequest, http.StatusUnauthorized)
}
