package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// setupAMAPIEnvVars sets up the required environment variables for AMAPI calls and returns a cleanup function.
func setupAMAPIEnvVars(t *testing.T) {
	oldPackageValue := os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	oldSHA256Value := os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", "com.fleetdm.agent")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", "abc123def456")
	t.Cleanup(func() {
		os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", oldPackageValue)
		os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", oldSHA256Value)
	})
}

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
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, expectedStatus, expectedDetail, fmt.Sprintf("CN=%s", host.HardwareSerial))
}

// verifyCertificateStatusWithSubject is like verifyCertificateStatus but allows specifying the expected subject name.
func (s *integrationMDMTestSuite) verifyCertificateStatusWithSubject(
	t *testing.T,
	host *fleet.Host,
	orbitNodeKey string,
	certificateTemplateID uint,
	certTemplateName string,
	caID uint,
	expectedStatus fleet.CertificateTemplateStatus,
	expectedDetail string,
	expectedSubjectName string,
) {
	// Verify via host API
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.NotEmpty(t, *getHostResp.Host.MDM.Profiles)
	// Find the profile by name
	var profile *fleet.HostMDMProfile
	for _, p := range *getHostResp.Host.MDM.Profiles {
		if p.Name == certTemplateName {
			profile = &p
			break
		}
	}
	require.NotNil(t, profile, "Profile %s not found in host MDM profiles", certTemplateName)
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
	require.Equal(t, expectedSubjectName, getCertResp.Certificate.SubjectName)
	require.Equal(t, string(fleet.CATypeCustomSCEPProxy), getCertResp.Certificate.CertificateAuthorityType)
	require.Equal(t, expectedStatus, getCertResp.Certificate.Status)

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

// createTestCertificateAuthority creates a test certificate authority for use in tests.
func (s *integrationMDMTestSuite) createTestCertificateAuthority(t *testing.T, ctx context.Context) (uint, *fleet.CertificateAuthority) {
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(t.Name() + "-CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	return ca.ID, ca
}

// createEnrolledAndroidHost creates an enrolled Android host in a team and returns the host and orbit node key.
func (s *integrationMDMTestSuite) createEnrolledAndroidHost(t *testing.T, ctx context.Context, enterpriseID string, teamID *uint, suffix string) (*fleet.Host, string) {
	hostUUID := uuid.NewString()
	androidHostInput := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       t.Name() + "-host-" + suffix,
			ComputerName:   t.Name() + "-device-" + suffix,
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          "build1",
			Memory:         1024,
			TeamID:         teamID,
			HardwareSerial: uuid.NewString(),
			UUID:           hostUUID,
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: ptr.String(enterpriseID),
			AppliedPolicyID:      ptr.String("1"),
		},
	}
	androidHostInput.SetNodeKey(enterpriseID)
	createdAndroidHost, err := s.ds.NewAndroidHost(ctx, androidHostInput)
	require.NoError(t, err)

	host := createdAndroidHost.Host
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

	return host, orbitNodeKey
}

// TestCertificateTemplateLifecycle tests the full Android certificate template lifecycle.
func (s *integrationMDMTestSuite) TestCertificateTemplateLifecycle() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

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
	certTemplateName := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate"
	var createResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName,
		TeamID:                 teamID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certificateTemplateID := createResp.ID

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeAddedCertificate{}.ActivityName(),
		fmt.Sprintf(
			`{"team_id": %d, "team_name": %q, "name": %q}`,
			teamID,
			teamName,
			certTemplateName,
		),
		0)

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

	// Step: Host attempts to update the certificate status to 'failed' via fleetd API
	// This should be ignored since the current status is not 'delivered'
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

	// Step: Verify the status is still 'verified' with details
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateVerified, successDetail)

	// Delete the cert
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/certificates/%d", certificateTemplateID), nil, http.StatusOK)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedCertificate{}.ActivityName(),
		fmt.Sprintf(
			`{"team_id": %d, "team_name": %q, "name": %q}`,
			teamID,
			teamName,
			certTemplateName,
		),
		0)
}

// TestCertificateTemplateSpecEndpointAndAMAPIFailure tests:
// 1. Creating a certificate template via spec/certificates endpoint with $FLEET_VAR_HOST_UUID
// 2. Enrolling a new host to a team that already has certificate templates (pending records created automatically)
// 3. AMAPI failure reverts status from 'delivering' back to 'pending'
func (s *integrationMDMTestSuite) TestCertificateTemplateSpecEndpointAndAMAPIFailure() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

	enterpriseID := s.enableAndroidMDM(t)

	// Step: Create a test team
	teamName := t.Name() + "-team"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(teamName),
		},
	}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID

	// Step: Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(t.Name() + "-CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Step: Create certificate template via spec/certificates endpoint with $FLEET_VAR_HOST_UUID
	certTemplateName := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate"
	var applyResp applyCertificateTemplateSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   certTemplateName,
				Team:                   teamName,
				CertificateAuthorityId: caID,
				SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusOK, &applyResp)

	// Step: Get the certificate template ID
	var listResp listCertificateTemplatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates?team_id=%d", teamID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Certificates, 1)
	certificateTemplateID := listResp.Certificates[0].ID

	// Step: Enroll a new Android host to the team (should automatically get pending certificate template)
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
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: ptr.String(enterpriseID),
			AppliedPolicyID:      ptr.String("1"),
		},
	}
	androidHostInput.SetNodeKey(enterpriseID)
	createdAndroidHost, err := s.ds.NewAndroidHost(ctx, androidHostInput)
	require.NoError(t, err)

	host := createdAndroidHost.Host

	// Set OrbitNodeKey for API authentication
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

	// Create pending certificate templates for this host (simulating what pubsub handlers do during enrollment)
	_, err = s.ds.CreatePendingCertificateTemplatesForNewHost(ctx, host.UUID, teamID)
	require.NoError(t, err)

	// SubjectName should use HOST_UUID
	expectedSubjectName := fmt.Sprintf("CN=%s", host.UUID)

	// Step: Set up AMAPI mock to fail on first call, succeed on second
	// This must be set up BEFORE running the worker since the worker also calls BuildAndSendFleetAgentConfig
	amapiCallCount := 0
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, _ string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		amapiCallCount++
		if amapiCallCount == 1 {
			// First call: verify status is 'delivering', then return error to simulate AMAPI failure
			s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivering, "", expectedSubjectName)
			return nil, errors.New("simulated AMAPI failure")
		}
		// Second call: succeed
		return &androidmanagement.Policy{}, nil
	}

	// Step: Queue and run the Android setup experience worker job
	// Note: Pending certificate templates were created above (simulating pubsub). The worker will deliver them.
	enterpriseName := "enterprises/" + enterpriseID
	err = worker.QueueRunAndroidSetupExperience(ctx, s.ds, log.NewNopLogger(), host.UUID, &teamID, enterpriseName)
	require.NoError(t, err)
	s.runWorker()

	// Step: Verify AMAPI was called once (during worker) and status reverted to 'pending' after failure
	require.Equal(t, 1, amapiCallCount, "AMAPI should have been called once by worker")
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplatePending, "", expectedSubjectName)

	// Step: Trigger reconciliation again (second attempt - should succeed)
	s.awaitTriggerAndroidProfileSchedule(t)

	// Step: Verify AMAPI was called again and status is now 'delivered'
	require.Equal(t, 2, amapiCallCount, "AMAPI should have been called twice")
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivered, "", expectedSubjectName)
}

// TestCertificateTemplateNoTeamWithIDPVariable tests:
// 1. Creating a certificate template for "no team" (team_id = 0)
// 2. Using $FLEET_VAR_HOST_END_USER_IDP_USERNAME which fails when host has no IDP user
// 3. Verifying status becomes 'failed' when fetching the certificate via fleetd API
func (s *integrationMDMTestSuite) TestCertificateTemplateNoTeamWithIDPVariable() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

	enterpriseID := s.enableAndroidMDM(t)

	// Step: Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(t.Name() + "-CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Step: Create certificate template for "no team" (team_id = 0) with IDP_USERNAME variable
	certTemplateName := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate"
	var createResp createCertificateTemplateResponse
	subjectName := "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME"
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName,
		TeamID:                 0, // No team
		CertificateAuthorityId: caID,
		SubjectName:            subjectName,
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certificateTemplateID := createResp.ID

	// Step: Create an enrolled Android host with NO team (team_id = NULL)
	hostUUID := uuid.NewString()
	androidHostInput := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       t.Name() + "-host",
			ComputerName:   t.Name() + "-device",
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          "build1",
			Memory:         1024,
			TeamID:         nil, // No team - this maps to team_id = 0 in certificate_templates
			HardwareSerial: uuid.NewString(),
			UUID:           hostUUID,
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: ptr.String(enterpriseID),
			AppliedPolicyID:      ptr.String("1"),
		},
	}
	androidHostInput.SetNodeKey(enterpriseID)
	createdAndroidHost, err := s.ds.NewAndroidHost(ctx, androidHostInput)
	require.NoError(t, err)

	host := createdAndroidHost.Host

	// Set OrbitNodeKey for API authentication
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

	// Create pending certificate templates for this host (simulating what pubsub handlers do during enrollment)
	// Use teamID = 0 for "no team" hosts
	_, err = s.ds.CreatePendingCertificateTemplatesForNewHost(ctx, host.UUID, 0)
	require.NoError(t, err)

	// Step: Set up AMAPI mock to succeed
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, _ string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}

	// Step: Queue and run the Android setup experience worker job
	// Note: Pending certificate templates were created above (simulating pubsub). The worker will deliver them.
	enterpriseName := "enterprises/" + enterpriseID
	err = worker.QueueRunAndroidSetupExperience(ctx, s.ds, log.NewNopLogger(), host.UUID, nil, enterpriseName)
	require.NoError(t, err)
	s.runWorker()

	// Step: Verify status is 'delivered' via host API (after worker runs)
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 1)
	require.Equal(t, string(fleet.CertificateTemplateDelivered), *(*getHostResp.Host.MDM.Profiles)[0].Status)

	// Step: Fetch certificate via fleetd API - this should trigger failure due to missing IDP username
	// The host has no IDP user associated, so $FLEET_VAR_HOST_END_USER_IDP_USERNAME cannot be replaced
	resp := s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", certificateTemplateID), nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	var getCertResp getDeviceCertificateTemplateResponse
	err = json.NewDecoder(resp.Body).Decode(&getCertResp)
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Step: Verify the response shows 'failed' status due to missing IDP username
	require.NotNil(t, getCertResp.Certificate)
	require.Equal(t, fleet.CertificateTemplateFailed, getCertResp.Certificate.Status)

	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateFailed, "", subjectName)
}

// TestCertificateTemplateUnenrollReenroll tests:
// 1. Host with existing certificate templates is unenrolled
// 2. A new certificate template is added while host is unenrolled (should NOT be marked for this host)
// 3. Host re-enrolls and should automatically get pending records for all certificate templates
func (s *integrationMDMTestSuite) TestCertificateTemplateUnenrollReenroll() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

	enterpriseID := s.enableAndroidMDM(t)

	// Step: Create a test team
	teamName := t.Name() + "-team"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(teamName),
		},
	}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID

	// Step: Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String(t.Name() + "-CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Step: Create an enrolled Android host in the team
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
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: ptr.String(enterpriseID),
			AppliedPolicyID:      ptr.String("1"),
		},
	}
	androidHostInput.SetNodeKey(enterpriseID)
	createdAndroidHost, err := s.ds.NewAndroidHost(ctx, androidHostInput)
	require.NoError(t, err)

	host := createdAndroidHost.Host

	// Set OrbitNodeKey for API authentication
	orbitNodeKey := *host.NodeKey
	host.OrbitNodeKey = &orbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	// Step: Create the first certificate template (while host is enrolled)
	certTemplateName := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate1"
	var createResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName,
		TeamID:                 teamID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certTemplateID := createResp.ID

	// Step: Verify host has pending certificate template record
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certTemplateID, certTemplateName, caID,
		fleet.CertificateTemplatePending, "", "CN="+host.HardwareSerial)

	// Step: Unenroll the host (simulates pubsub DELETED message)
	unenrolled, err := s.ds.SetAndroidHostUnenrolled(ctx, host.ID)
	require.NoError(t, err)
	require.True(t, unenrolled)

	// Verify host is actually unenrolled
	var enrolledStatus int
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &enrolledStatus, `SELECT enrolled FROM host_mdm WHERE host_id = ?`, host.ID)
	})
	require.Equal(t, 0, enrolledStatus, "Host should be marked as unenrolled in host_mdm")

	// Step: Create a second certificate template while host is unenrolled
	certTemplateName2 := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate2"
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName2,
		TeamID:                 teamID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certTemplateID2 := createResp.ID

	// Step: Verify that the unenrolled host did NOT get a host_certificate_templates record for the second template.
	// The host API only returns profiles that have host_certificate_templates records, so the second
	// template should not appear in the profiles list.
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 1, "Only first template should appear (second was created while host was unenrolled)")
	profile := (*getHostResp.Host.MDM.Profiles)[0]
	require.Equal(t, certTemplateName, profile.Name, "First certificate template should be present")
	require.NotNil(t, profile.Status)
	require.Equal(t, string(fleet.CertificateTemplatePending), *profile.Status)

	// Step: Re-enroll the host (simulates pubsub status report triggering UpdateAndroidHost with fromEnroll=true)
	err = s.ds.UpdateAndroidHost(ctx, createdAndroidHost, true)
	require.NoError(t, err)

	// Verify host is re-enrolled
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &enrolledStatus, `SELECT enrolled FROM host_mdm WHERE host_id = ?`, host.ID)
	})
	require.Equal(t, 1, enrolledStatus, "Host should be marked as enrolled in host_mdm after re-enrollment")

	// Step: Verify that the re-enrolled host now has BOTH certificate templates via the host API.
	// The helper verifies both via the host API (MDM.Profiles) and the fleetd certificate API.
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certTemplateID, certTemplateName, caID,
		fleet.CertificateTemplatePending, "", "CN="+host.HardwareSerial)
	s.verifyCertificateStatusWithSubject(t, host, orbitNodeKey, certTemplateID2, certTemplateName2, caID,
		fleet.CertificateTemplatePending, "", "CN="+host.UUID)
}

// TestCertificateTemplateTeamTransfer tests certificate template behavior when Android hosts transfer between teams:
// 1. Host with certs in various statuses (pending, delivering, delivered, verified, failed, remove) transfers teams -> all certs marked for removal
// 2. Host without any certs transfers to a team with certs -> gets new pending certs
// 3. Host with certs transfers to another team with different certs -> old certs removed, new certs added
func (s *integrationMDMTestSuite) TestCertificateTemplateTeamTransfer() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

	enterpriseID := s.enableAndroidMDM(t)

	// Create two teams with different certificate templates
	teamAName := t.Name() + "-teamA"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(teamAName),
		},
	}, http.StatusOK, &createTeamResp)
	teamAID := createTeamResp.Team.ID

	teamBName := t.Name() + "-teamB"
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String(teamBName),
		},
	}, http.StatusOK, &createTeamResp)
	teamBID := createTeamResp.Team.ID

	// Create enroll secrets for both teams
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teamAID), modifyTeamEnrollSecretsRequest{
		Secrets: []fleet.EnrollSecret{{Secret: "teamA-secret"}},
	}, http.StatusOK, &teamEnrollSecretsResponse{})
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teamBID), modifyTeamEnrollSecretsRequest{
		Secrets: []fleet.EnrollSecret{{Secret: "teamB-secret"}},
	}, http.StatusOK, &teamEnrollSecretsResponse{})

	// Create a test certificate authority
	caID, _ := s.createTestCertificateAuthority(t, ctx)

	// Create certificate templates for Team A
	certTemplateA1Name := strings.ReplaceAll(t.Name(), "/", "-") + "-TeamA-Cert1"
	var createCertResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateA1Name,
		TeamID:                 teamAID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createCertResp)
	certTemplateA1ID := createCertResp.ID

	certTemplateA2Name := strings.ReplaceAll(t.Name(), "/", "-") + "-TeamA-Cert2"
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateA2Name,
		TeamID:                 teamAID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
	}, http.StatusOK, &createCertResp)
	certTemplateA2ID := createCertResp.ID

	// Create certificate templates for Team B
	certTemplateB1Name := strings.ReplaceAll(t.Name(), "/", "-") + "-TeamB-Cert1"
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateB1Name,
		TeamID:                 teamBID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createCertResp)
	certTemplateB1ID := createCertResp.ID

	// Helper to get certificate template statuses for a host from the database
	getCertTemplateStatuses := func(hostUUID string) map[uint]struct {
		Status        fleet.CertificateTemplateStatus
		OperationType fleet.MDMOperationType
	} {
		result := make(map[uint]struct {
			Status        fleet.CertificateTemplateStatus
			OperationType fleet.MDMOperationType
		})
		var rows []struct {
			CertificateTemplateID uint                            `db:"certificate_template_id"`
			Status                fleet.CertificateTemplateStatus `db:"status"`
			OperationType         fleet.MDMOperationType          `db:"operation_type"`
		}
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows, `
				SELECT certificate_template_id, status, operation_type
				FROM host_certificate_templates
				WHERE host_uuid = ?
			`, hostUUID)
		})
		for _, r := range rows {
			result[r.CertificateTemplateID] = struct {
				Status        fleet.CertificateTemplateStatus
				OperationType fleet.MDMOperationType
			}{r.Status, r.OperationType}
		}
		return result
	}

	// Set up AMAPI mock to succeed
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, _ string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}

	t.Run("host with certs in all status/operation combinations transfers to team without certs", func(t *testing.T) {
		// Create a team with 9 certificate templates to test all status/operation combinations
		teamEName := t.Name() + "-teamE"
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
			TeamPayload: fleet.TeamPayload{
				Name: ptr.String(teamEName),
			},
		}, http.StatusOK, &createTeamResp)
		teamEID := createTeamResp.Team.ID

		// Create enroll secret for team E
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teamEID), modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: "teamE-secret"}},
		}, http.StatusOK, &teamEnrollSecretsResponse{})

		// Create a team without certificate templates for transfer target
		teamFName := t.Name() + "-teamF"
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
			TeamPayload: fleet.TeamPayload{
				Name: ptr.String(teamFName),
			},
		}, http.StatusOK, &createTeamResp)
		teamFID := createTeamResp.Team.ID

		// Create enroll secret for team F
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teamFID), modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: "teamF-secret"}},
		}, http.StatusOK, &teamEnrollSecretsResponse{})

		// Define all status/operation combinations to test
		type certTestCase struct {
			status         fleet.CertificateTemplateStatus
			operation      fleet.MDMOperationType
			shouldDelete   bool                            // true if record should be deleted during transfer
			expectedStatus fleet.CertificateTemplateStatus // expected status after transfer (if not deleted)
			expectedOp     fleet.MDMOperationType          // expected operation after transfer (if not deleted)
			templateID     uint
			templateName   string
		}

		testCases := []certTestCase{
			// Install operations: pending/failed are deleted, others are marked for removal
			{fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall, true, "", "", 0, ""},
			{fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeInstall, false, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, false, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateVerified, fleet.MDMOperationTypeInstall, false, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateFailed, fleet.MDMOperationTypeInstall, true, "", "", 0, ""},
			// Remove operations: all stay unchanged (removal already in progress)
			{fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, false, fleet.CertificateTemplatePending, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeRemove, false, fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeRemove, false, fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeRemove, 0, ""},
			{fleet.CertificateTemplateFailed, fleet.MDMOperationTypeRemove, false, fleet.CertificateTemplateFailed, fleet.MDMOperationTypeRemove, 0, ""},
		}

		// Create certificate templates for team E
		for i := range testCases {
			name := fmt.Sprintf("%s-Cert-%s-%s", strings.ReplaceAll(t.Name(), "/", "-"), testCases[i].status, testCases[i].operation)
			s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
				Name:                   name,
				TeamID:                 teamEID,
				CertificateAuthorityId: caID,
				SubjectName:            fmt.Sprintf("CN=Test-%d", i),
			}, http.StatusOK, &createCertResp)
			testCases[i].templateID = createCertResp.ID
			testCases[i].templateName = name
		}

		// Create host in Team E
		host, _ := s.createEnrolledAndroidHost(t, ctx, enterpriseID, &teamEID, "all-statuses")

		// Insert certificate template records with all status/operation combinations
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			for _, tc := range testCases {
				challenge := "challenge"
				if tc.status == fleet.CertificateTemplatePending || tc.status == fleet.CertificateTemplateFailed {
					challenge = "" // No challenge for pending/failed
				}
				_, err := q.ExecContext(ctx, `
					INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name)
					VALUES (?, ?, ?, ?, NULLIF(?, ''), ?)
				`, host.UUID, tc.templateID, tc.status, tc.operation, challenge, tc.templateName)
				if err != nil {
					return err
				}
			}
			return nil
		})

		// Verify initial state - should have 9 certificate template records
		statuses := getCertTemplateStatuses(host.UUID)
		require.Len(t, statuses, 9, "Should have 9 certificate template records")

		// Transfer host to Team F (no certs)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &teamFID,
			HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})

		// Run the worker to process the transfer
		s.runWorker()

		// Verify results
		statuses = getCertTemplateStatuses(host.UUID)

		// Count expected remaining records (those not deleted)
		expectedRemaining := 0
		for _, tc := range testCases {
			if !tc.shouldDelete {
				expectedRemaining++
			}
		}
		require.Len(t, statuses, expectedRemaining, "Should have %d certificate template records after transfer", expectedRemaining)

		// Verify each test case
		for _, tc := range testCases {
			status, exists := statuses[tc.templateID]
			if tc.shouldDelete {
				require.False(t, exists, "Record for %s/%s should be deleted", tc.status, tc.operation)
			} else {
				require.True(t, exists, "Record for %s/%s should exist", tc.status, tc.operation)
				require.Equal(t, tc.expectedStatus, status.Status,
					"Record for %s/%s should have status=%s", tc.status, tc.operation, tc.expectedStatus)
				require.Equal(t, tc.expectedOp, status.OperationType,
					"Record for %s/%s should have operation_type=%s", tc.status, tc.operation, tc.expectedOp)
			}
		}
	})

	t.Run("host without certs transfers to team with certs", func(t *testing.T) {
		// Create a team without certificate templates
		teamDName := t.Name() + "-teamD"
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
			TeamPayload: fleet.TeamPayload{
				Name: ptr.String(teamDName),
			},
		}, http.StatusOK, &createTeamResp)
		teamDID := createTeamResp.Team.ID

		// Create enroll secret for team D
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", teamDID), modifyTeamEnrollSecretsRequest{
			Secrets: []fleet.EnrollSecret{{Secret: "teamD-secret"}},
		}, http.StatusOK, &teamEnrollSecretsResponse{})

		// Create host in Team D (no certs)
		host, _ := s.createEnrolledAndroidHost(t, ctx, enterpriseID, &teamDID, "no-certs")

		// Verify no certificate templates for this host
		statuses := getCertTemplateStatuses(host.UUID)
		require.Empty(t, statuses, "Host should have no certificate templates initially")

		// Transfer host to Team B (has certs)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &teamBID,
			HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})

		// Run the worker to process the transfer
		s.runWorker()

		// Verify host now has Team B's certificate template as pending install
		statuses = getCertTemplateStatuses(host.UUID)
		require.Len(t, statuses, 1, "Host should have Team B's certificate template")
		require.Equal(t, fleet.CertificateTemplatePending, statuses[certTemplateB1ID].Status)
		require.Equal(t, fleet.MDMOperationTypeInstall, statuses[certTemplateB1ID].OperationType)
	})

	t.Run("host with certs transfers to team with different certs", func(t *testing.T) {
		// Create host in Team A
		host, orbitNodeKey := s.createEnrolledAndroidHost(t, ctx, enterpriseID, &teamAID, "transfer-certs")

		// Create pending certificate templates for this host (Team A certs)
		_, err := s.ds.CreatePendingCertificateTemplatesForNewHost(ctx, host.UUID, teamAID)
		require.NoError(t, err)

		// Set both certs to verified status (simulating they were both installed on device)
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE host_certificate_templates
				SET status = ?, fleet_challenge = 'challenge'
				WHERE host_uuid = ?
			`, fleet.CertificateTemplateVerified, host.UUID)
			return err
		})

		// Verify initial state - host has Team A's certs (both verified)
		statuses := getCertTemplateStatuses(host.UUID)
		require.Len(t, statuses, 2)
		require.Contains(t, statuses, certTemplateA1ID)
		require.Contains(t, statuses, certTemplateA2ID)
		require.Equal(t, fleet.CertificateTemplateVerified, statuses[certTemplateA1ID].Status)
		require.Equal(t, fleet.CertificateTemplateVerified, statuses[certTemplateA2ID].Status)

		// Transfer host to Team B
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &teamBID,
			HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})

		// Run the worker to process the transfer
		s.runWorker()

		// Verify:
		// - Team A's certs (both verified/installed) are marked as pending remove
		// - Team B's cert is added as pending install
		statuses = getCertTemplateStatuses(host.UUID)
		require.Len(t, statuses, 3, "Should have 2 old certs (pending remove) + 1 new cert (pending install)")

		// Team A certs should be pending remove
		require.Equal(t, fleet.CertificateTemplatePending, statuses[certTemplateA1ID].Status)
		require.Equal(t, fleet.MDMOperationTypeRemove, statuses[certTemplateA1ID].OperationType)
		require.Equal(t, fleet.CertificateTemplatePending, statuses[certTemplateA2ID].Status)
		require.Equal(t, fleet.MDMOperationTypeRemove, statuses[certTemplateA2ID].OperationType)

		// Team B cert should be pending install
		require.Equal(t, fleet.CertificateTemplatePending, statuses[certTemplateB1ID].Status)
		require.Equal(t, fleet.MDMOperationTypeInstall, statuses[certTemplateB1ID].OperationType)

		// Test that device can report "verified" for a pending removal and the record gets deleted.
		// This handles race conditions where the device processes the removal before the server
		// transitions the status through the full state machine.
		updateReq, err := json.Marshal(updateCertificateStatusRequest{
			Status:        string(fleet.CertificateTemplateVerified),
			OperationType: ptr.String(string(fleet.MDMOperationTypeRemove)),
		})
		require.NoError(t, err)

		resp := s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", certTemplateA1ID), updateReq, http.StatusOK, map[string]string{
			"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
		})
		_ = resp.Body.Close()

		// Verify the record was deleted
		statuses = getCertTemplateStatuses(host.UUID)
		require.Len(t, statuses, 2, "Should have 1 pending remove + 1 pending install after removal confirmed")
		_, exists := statuses[certTemplateA1ID]
		require.False(t, exists, "certTemplateA1 should be deleted after verified removal")
	})
}

// TestCertificateTemplateRenewal tests the full certificate renewal flow:
// 1. Certificate is created and delivered to device
// 2. Device reports "verified" status with validity data (certificate expiring soon)
// 3. Server detects expiring certificate via GetAndroidCertificateTemplatesForRenewal
// 4. Server marks certificate for renewal via SetAndroidCertificateTemplatesForRenewal
// 5. Certificate UUID changes, signaling renewal to device
// 6. Device sees new UUID and re-enrolls
func (s *integrationMDMTestSuite) TestCertificateTemplateRenewal() {
	t := s.T()
	ctx := t.Context()
	setupAMAPIEnvVars(t)

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

	// Create a test certificate authority via API
	caID, _ := s.createTestCertificateAuthority(t, ctx)

	// Create an enrolled Android host
	host, orbitNodeKey := s.createEnrolledAndroidHost(t, ctx, enterpriseID, &teamID, "renewal")

	// Create a certificate template
	certTemplateName := strings.ReplaceAll(t.Name(), "/", "-") + "-CertTemplate"
	var createResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   certTemplateName,
		TeamID:                 teamID,
		CertificateAuthorityId: caID,
		SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.ID)
	certificateTemplateID := createResp.ID

	// Set up AMAPI mock
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, _ string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}

	// Trigger the Android profile reconciliation job to deliver the certificate
	s.awaitTriggerAndroidProfileSchedule(t)

	// Verify status is now 'delivered'
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivered, "")

	// Get the original UUID before renewal
	var originalUUID string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &originalUUID,
			`SELECT COALESCE(BIN_TO_UUID(uuid, true), '') FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?`,
			host.UUID, certificateTemplateID)
	})
	require.NotEmpty(t, originalUUID)

	// Device updates certificate status to 'verified' WITH validity data
	// Certificate validity: 1 year, expiring in 7 days (within 30-day renewal threshold)
	now := time.Now().UTC()
	notValidBefore := now.AddDate(-1, 0, 7) // Started almost a year ago
	notValidAfter := now.Add(7 * 24 * time.Hour)
	serial := "ABC123DEF456"

	updateReq, err := json.Marshal(updateCertificateStatusRequest{
		Status:         string(fleet.CertificateTemplateVerified),
		Detail:         ptr.String("Certificate installed successfully"),
		NotValidBefore: &notValidBefore,
		NotValidAfter:  &notValidAfter,
		Serial:         &serial,
	})
	require.NoError(t, err)

	resp := s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", certificateTemplateID), updateReq, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	_ = resp.Body.Close()

	// Verify the status is 'verified'
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateVerified, "Certificate installed successfully")

	// Verify validity data was stored
	var storedValidity struct {
		NotValidBefore *time.Time `db:"not_valid_before"`
		NotValidAfter  *time.Time `db:"not_valid_after"`
		Serial         *string    `db:"serial"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &storedValidity,
			`SELECT not_valid_before, not_valid_after, serial FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?`,
			host.UUID, certificateTemplateID)
	})
	require.NotNil(t, storedValidity.NotValidBefore, "not_valid_before should be stored")
	require.NotNil(t, storedValidity.NotValidAfter, "not_valid_after should be stored")
	require.NotNil(t, storedValidity.Serial, "serial should be stored")
	require.Equal(t, serial, *storedValidity.Serial)

	// Test renewal detection: GetAndroidCertificateTemplatesForRenewal should find this certificate
	templates, err := s.ds.GetAndroidCertificateTemplatesForRenewal(ctx, 100)
	require.NoError(t, err)
	require.Len(t, templates, 1, "Should find 1 certificate for renewal")
	require.Equal(t, host.UUID, templates[0].HostUUID)
	require.Equal(t, certificateTemplateID, templates[0].CertificateTemplateID)

	// Trigger renewal: SetAndroidCertificateTemplatesForRenewal marks for renewal
	err = s.ds.SetAndroidCertificateTemplatesForRenewal(ctx, templates)
	require.NoError(t, err)

	// Verify the certificate is now pending with a NEW UUID
	var newRecord struct {
		Status         string  `db:"status"`
		UUID           string  `db:"uuid"`
		NotValidBefore *string `db:"not_valid_before"`
		NotValidAfter  *string `db:"not_valid_after"`
		Serial         *string `db:"serial"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &newRecord,
			`SELECT status, COALESCE(BIN_TO_UUID(uuid, true), '') AS uuid, not_valid_before, not_valid_after, serial
			 FROM host_certificate_templates WHERE host_uuid = ? AND certificate_template_id = ?`,
			host.UUID, certificateTemplateID)
	})

	// Status should be 'pending'
	require.Equal(t, string(fleet.CertificateTemplatePending), newRecord.Status, "Status should be pending after renewal trigger")

	// UUID should be different (signals renewal to device)
	require.NotEmpty(t, newRecord.UUID)
	require.NotEqual(t, originalUUID, newRecord.UUID, "UUID should change to signal renewal to device")

	// Validity fields should be cleared (will be re-populated after re-enrollment)
	require.Nil(t, newRecord.NotValidBefore, "not_valid_before should be cleared")
	require.Nil(t, newRecord.NotValidAfter, "not_valid_after should be cleared")
	require.Nil(t, newRecord.Serial, "serial should be cleared")

	// Trigger profile reconciliation again - this simulates the device seeing the new config
	s.awaitTriggerAndroidProfileSchedule(t)

	// Certificate should now be in 'delivered' state again, ready for re-enrollment
	s.verifyCertificateStatus(t, host, orbitNodeKey, certificateTemplateID, certTemplateName, caID, fleet.CertificateTemplateDelivered, "")
}
