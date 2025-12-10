package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// verifyCertificateStatus is a helper function that verifies the certificate template status
// via both the host API and the fleetd certificate API.
func (s *integrationMDMTestSuite) verifyCertificateStatus(
	t *testing.T,
	host *fleet.Host,
	orbitNodeKey string,
	certificateTemplateID uint,
	certTemplateName string,
	caID uint,
	expectedStatus fleet.CertificateTemplateStatus,
	expectedDetail string,
) {
	// Verify via host API
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 1)
	profile := (*getHostResp.Host.MDM.Profiles)[0]
	require.Equal(t, certTemplateName, profile.Name)
	require.NotNil(t, profile.Status)
	require.Equal(t, string(expectedStatus), *profile.Status)
	if expectedDetail != "" {
		require.Equal(t, expectedDetail, profile.Detail)
	}

	// Verify via fleetd certificate API
	resp := s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", certificateTemplateID), nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	var getCertResp getDeviceCertificateTemplateResponse
	err := json.NewDecoder(resp.Body).Decode(&getCertResp)
	require.NoError(t, err)
	_ = resp.Body.Close()

	require.NotNil(t, getCertResp.Certificate)
	// Verify all fields in the response
	require.Equal(t, certificateTemplateID, getCertResp.Certificate.ID)
	require.Equal(t, certTemplateName, getCertResp.Certificate.Name)
	require.Equal(t, caID, getCertResp.Certificate.CertificateAuthorityId)
	require.NotEmpty(t, getCertResp.Certificate.CertificateAuthorityName)
	require.NotEmpty(t, getCertResp.Certificate.CreatedAt)
	// SubjectName has Fleet variables replaced with host-specific values
	require.Equal(t, fmt.Sprintf("CN=%s", host.HardwareSerial), getCertResp.Certificate.SubjectName)
	require.Equal(t, string(fleet.CATypeCustomSCEPProxy), getCertResp.Certificate.CertificateAuthorityType)
	require.NotNil(t, getCertResp.Certificate.Status)
	require.Equal(t, expectedStatus, *getCertResp.Certificate.Status)

	// Verify challenges based on status
	if expectedStatus == fleet.CertificateTemplateDelivered {
		// Challenges should be returned when status is 'delivered'
		require.NotNil(t, getCertResp.Certificate.SCEPChallenge)
		require.NotEmpty(t, *getCertResp.Certificate.SCEPChallenge)
		require.NotNil(t, getCertResp.Certificate.FleetChallenge)
		require.NotEmpty(t, *getCertResp.Certificate.FleetChallenge)
	} else {
		// Challenges should be nil for other statuses
		require.Nil(t, getCertResp.Certificate.SCEPChallenge)
		require.Nil(t, getCertResp.Certificate.FleetChallenge)
	}
}

// TestCertificateTemplateLifecycle tests the full Android certificate template lifecycle.
func (s *integrationMDMTestSuite) TestCertificateTemplateLifecycle() {
	t := s.T()
	ctx := t.Context()

	// Set up environment variables required for AMAPI calls
	oldPackageValue := os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	oldSHA256Value := os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", "com.fleetdm.agent")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", "abc123def456")
	t.Cleanup(func() {
		os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", oldPackageValue)
		os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", oldSHA256Value)
	})

	enterpriseID := s.enableAndroidMDM(t)

	// Create a test team
	teamName := t.Name() + "-team"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(teamName),
		},
	}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID

	// Create a test certificate authority (using Datastore directly to bypass SCEP URL validation)
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(t.Name() + "-CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Create an enrolled Android host in the team
	hostUUID := uuid.NewString()
	androidHostInput := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       t.Name() + "-host",
			ComputerName:   t.Name() + "-device",
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          "build1",
			Memory:         1024,
			TeamID:         &teamID,
			HardwareSerial: uuid.NewString(),
			UUID:           hostUUID,
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""), // Remove dashes to fit in VARCHAR(37)
			EnterpriseSpecificID: ptr.String(enterpriseID),
			AppliedPolicyID:      ptr.String("1"),
		},
	}
	androidHostInput.SetNodeKey(enterpriseID)
	createdAndroidHost, err := s.ds.NewAndroidHost(ctx, androidHostInput)
	require.NoError(t, err)

	host := createdAndroidHost.Host

	// Set OrbitNodeKey for API authentication (same as NodeKey for Android hosts)
	orbitNodeKey := *host.NodeKey
	host.OrbitNodeKey = &orbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	// Mark host as enrolled in host_mdm
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
			INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server)
			VALUES (?, 1, 'https://example.com', 0, 0)
			ON DUPLICATE KEY UPDATE enrolled = 1
		`, host.ID)
		return err
	})

	// Step: Create a certificate template
	certTemplateName := t.Name() + "-CertTemplate"
	var createResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName,
		TeamID:                 teamID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certificateTemplateID := createResp.ID

	// Step: Verify status is 'pending'
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplatePending, "")

	// Step: Set up AMAPI mock to verify 'delivering' status during the call
	deliveringStatusVerified := false
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, _ string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivering, "")
		deliveringStatusVerified = true
		return &androidmanagement.Policy{}, nil
	}

	// Step: Trigger the Android profile reconciliation job and wait for completion
	// This transitions: pending → delivering → delivered (with fleet_challenge)
	s.awaitTriggerAndroidProfileSchedule(t)

	// Step: Verify the AMAPI callback was invoked and 'delivering' status was verified
	require.True(t, deliveringStatusVerified, "AMAPI callback should have been invoked")

	// Step: Verify status is now 'delivered'
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivered, "")

	// Step: Host updates the certificate status to 'verified' via fleetd API
	successDetail := "Certificate installed successfully"
	updateReq, err := json.Marshal(updateCertificateStatusRequest{
		Status: string(fleet.CertificateTemplateVerified),
		Detail: ptr.String(successDetail),
	})
	require.NoError(t, err)

	resp := s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", certificateTemplateID), updateReq, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	_ = resp.Body.Close()

	// Step: Verify the status is 'verified'
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateVerified, successDetail)

	// Step: Host updates the certificate status to 'failed' via fleetd API
	failedDetail := "Certificate installation failed: invalid challenge"
	updateReq, err = json.Marshal(updateCertificateStatusRequest{
		Status: string(fleet.CertificateTemplateFailed),
		Detail: ptr.String(failedDetail),
	})
	require.NoError(t, err)

	resp = s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", certificateTemplateID), updateReq, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	_ = resp.Body.Close()

	// Step: Verify the status is 'failed' with details
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateFailed, failedDetail)
}
