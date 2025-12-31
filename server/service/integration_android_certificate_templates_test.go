package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

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

// TestCertificateTemplateLifecycle tests the full Android certificate template lifecycle.
func (s *integrationMDMTestSuite) TestCertificateTemplateLifecycle() {
	t := s.T()
	ctx := t.Context()
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
	certTemplateName := t.Name() + "-CertTemplate"
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
	certTemplateName := t.Name() + "-CertTemplate"
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
	certTemplateName := t.Name() + "-CertTemplate1"
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
	certTemplateName2 := t.Name() + "-CertTemplate2"
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
