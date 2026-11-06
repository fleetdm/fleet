package service

// Certificate tests for the enterprise (premium) suite.
//
// Belongs here: certificate template specs and deleting a fleet's certificate templates.
//
// Does not belong here: certificate authorities (integration_certificate_authorities_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestDeleteTeamCertificateTemplates() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestDeleteTeamCertificateTemplates Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestDeleteTeamCertificateTemplates SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Create two certificate templates on the team
	certTemplate1 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteTeamCertificateTemplates-Cert1",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	}
	savedTemplate1, err := s.ds.CreateCertificateTemplate(ctx, certTemplate1)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate1)
	certificateTemplateID1 := savedTemplate1.ID
	certTemplateName1 := savedTemplate1.Name

	certTemplate2 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteTeamCertificateTemplates-Cert2",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 2",
	}
	savedTemplate2, err := s.ds.CreateCertificateTemplate(ctx, certTemplate2)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate2)
	certificateTemplateID2 := savedTemplate2.ID
	certTemplateName2 := savedTemplate2.Name

	// Create hosts with different certificate template statuses
	hostPending, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-team-cert-template-host-pending",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostDelivered, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-team-cert-template-host-delivered",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostVerified, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-team-cert-template-host-verified",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostFailed, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-team-cert-template-host-failed",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	// Insert host_certificate_templates with various statuses for both templates
	insertSQL := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// Template 1 - Various statuses
		// Pending status - should be deleted
		_, err := q.ExecContext(ctx, insertSQL, hostPending.UUID, certificateTemplateID1, "pending", "install", nil, certTemplateName1)
		require.NoError(t, err)
		// Delivered status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostDelivered.UUID, certificateTemplateID1, "delivered", "install", "challenge1", certTemplateName1)
		require.NoError(t, err)

		// Template 2 - Various statuses
		// Verified status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostVerified.UUID, certificateTemplateID2, "verified", "install", "challenge2", certTemplateName2)
		require.NoError(t, err)
		// Failed status - should be deleted (never successfully installed)
		_, err = q.ExecContext(ctx, insertSQL, hostFailed.UUID, certificateTemplateID2, "failed", "install", nil, certTemplateName2)
		require.NoError(t, err)
		return nil
	})

	// Enable Android MDM so GetHost returns certificate template profiles
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origAndroidEnabled := appCfg.MDM.AndroidEnabledAndConfigured
	appCfg.MDM.AndroidEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	err = s.ds.SetAndroidEnabledAndConfigured(ctx, true)
	require.NoError(t, err)
	// Wait for cache to expire (default 1 second)
	time.Sleep(2 * time.Second)
	defer func() {
		appCfg.MDM.AndroidEnabledAndConfigured = origAndroidEnabled
		_ = s.ds.SaveAppConfig(ctx, appCfg)
		_ = s.ds.SetAndroidEnabledAndConfigured(ctx, origAndroidEnabled)
	}()

	// Helper to find the certificate template profile by name
	findProfile := func(profiles *[]fleet.HostMDMProfile, name string) *fleet.HostMDMProfile {
		if profiles == nil {
			return nil
		}
		for _, p := range *profiles {
			if p.Name == name {
				return &p
			}
		}
		return nil
	}

	// Verify the records exist before team deletion via GetHost API
	var getHostResp getHostResponse
	for _, tc := range []struct {
		host           *fleet.Host
		hostName       string
		expectedStatus string
		templateName   string
	}{
		{hostPending, "hostPending", string(fleet.CertificateTemplatePending), certTemplateName1},
		{hostDelivered, "hostDelivered", string(fleet.CertificateTemplateDelivered), certTemplateName1},
		{hostVerified, "hostVerified", string(fleet.CertificateTemplateVerified), certTemplateName2},
		{hostFailed, "hostFailed", string(fleet.CertificateTemplateFailed), certTemplateName2},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		require.NotNil(t, getHostResp.Host.MDM.Profiles, "%s should have MDM profiles before deletion", tc.hostName)

		profile := findProfile(getHostResp.Host.MDM.Profiles, tc.templateName)
		require.NotNil(t, profile, "%s should have certificate template profile %s before deletion", tc.hostName, tc.templateName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, tc.expectedStatus, *profile.Status, "%s profile status should be %s before deletion", tc.hostName, tc.expectedStatus)
		require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "%s profile operation_type should be install before deletion", tc.hostName)
	}

	// Delete the team via API (this should delete all certificate templates on the team)
	var deleteResp deleteTeamResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d", teamID), nil, http.StatusOK, &deleteResp)

	// Verify the certificate templates were deleted
	_, err = s.ds.GetCertificateTemplateById(ctx, certificateTemplateID1)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 1 should be deleted")

	_, err = s.ds.GetCertificateTemplateById(ctx, certificateTemplateID2)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 2 should be deleted")

	// After team deletion:
	// - hostPending (pending/install) should have NO profile (record was deleted - never installed)
	// - hostFailed (failed/install) should have NO profile (record was deleted - never successfully installed)
	// - hostDelivered, hostVerified should have pending/remove profiles
	//   (kept for cron job to process removal from devices)

	// Verify hostPending has no profile after deletion (was pending/install, never installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostPending.ID), nil, http.StatusOK, &getHostResp)
	profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName1)
	require.Nil(t, profile, "hostPending should not have certificate template profile after deletion")

	// Verify hostFailed has no profile after deletion (was failed/install, never successfully installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostFailed.ID), nil, http.StatusOK, &getHostResp)
	profile = findProfile(getHostResp.Host.MDM.Profiles, certTemplateName2)
	require.Nil(t, profile, "hostFailed should not have certificate template profile after deletion")

	// Verify hosts that had delivered/verified status now have pending/remove profiles
	for _, tc := range []struct {
		host         *fleet.Host
		hostName     string
		templateName string
	}{
		{hostDelivered, "hostDelivered", certTemplateName1},
		{hostVerified, "hostVerified", certTemplateName2},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		profile := findProfile(getHostResp.Host.MDM.Profiles, tc.templateName)
		require.NotNil(t, profile, "%s should have pending remove profile after deletion", tc.hostName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, string(fleet.CertificateTemplatePending), *profile.Status, "%s profile status should be pending after deletion", tc.hostName)
		require.Equal(t, fleet.MDMOperationTypeRemove, profile.OperationType, "%s profile operation_type should be remove after deletion", tc.hostName)
	}
}

func (s *integrationEnterpriseTestSuite) TestCertificatesSpecs() {
	t := s.T()
	ctx := context.Background()

	// findCertID returns the ID of the cert with the given name, or fails the test. Using
	// name-based lookup instead of list-position keeps the test robust against any order
	// changes in GetCertificateTemplatesByTeamID / the list endpoint.
	findCertID := func(certs []*fleet.CertificateTemplateResponseSummary, name string) uint {
		t.Helper()
		for _, c := range certs {
			if c.Name == name {
				return c.ID
			}
		}
		require.FailNowf(t, "certificate not found", "expected certificate %q in list of %d", name, len(certs))
		return 0
	}

	// create team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
	require.NoError(t, err)

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("Test SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)

	// invalid Fleet variable in subject name
	var applyResp applyCertificateTemplateSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   "Invalid Template",
				Team:                   team.Name,
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_NOT_VALID/OU=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusBadRequest, &applyResp)

	// test with non-existent team name
	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   "Invalid Team Template",
				Team:                   "NonExistentTeam",
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusNotFound, &applyResp)

	activitiesBeforeInsert := s.listActivities()

	// valid templates - test team name (not team ID)
	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 1",
				Team:                   team.Name,
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID/ST=$FLEET_VAR_HOST_HARDWARE_SERIAL",
			},
			{
				Name:                   "Template 2",
				Team:                   team.Name,
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusOK, &applyResp)

	// Only one activity per team
	activitiesAfterInsert := s.listActivities()
	require.Len(t, activitiesAfterInsert, len(activitiesBeforeInsert)+1, "expected exactly one new activity for the team")
	s.lastActivityMatches(
		fleet.ActivityTypeEditedAndroidCertificate{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "team_id": %d, "team_name": %q}`, team.ID, team.Name, team.ID, team.Name),
		0,
	)

	// list specs
	var listCertifcatesResp listCertificateTemplatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates?team_id=%d", team.ID), nil, http.StatusOK, &listCertifcatesResp)
	require.Len(t, listCertifcatesResp.Certificates, 2)
	assert.ElementsMatch(t, []string{"Template 1", "Template 2"}, []string{listCertifcatesResp.Certificates[0].Name, listCertifcatesResp.Certificates[1].Name})

	lastActivityID := s.lastActivityMatches("", "", 0)

	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   "Template 1",
				Team:                   team.Name,
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID/ST=$FLEET_VAR_HOST_HARDWARE_SERIAL",
			},
			{
				Name:                   "Template 2",
				Team:                   team.Name,
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusOK, &applyResp)

	// No new activities created
	currentActivityID := s.lastActivityMatches("", "", 0)
	assert.Equal(t, lastActivityID, currentActivityID, "no new activity should be created when re-applying same certificates")

	// Create a host to get certificate get by id endpoint
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("test-cert-node-key"),
		UUID:            "test-uuid-12345",
		Hostname:        "test-cert-host.local",
		HardwareSerial:  "TEST-SERIAL-67890",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	orbitNodeKey := uuid.New().String()
	host.OrbitNodeKey = &orbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	savedCertificateTemplates, _, err := s.ds.GetCertificateTemplatesByTeamID(ctx, team.ID, fleet.ListOptions{Page: 0, PerPage: 10})
	require.NoError(t, err)
	certID := findCertID(savedCertificateTemplates, "Template 1")

	// Create a host_certificate_templates record for this host (simulating what happens during Android enrollment)
	// The endpoint /api/fleetd/certificates/{id} requires a host_certificate_templates record to exist
	err = s.ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{
		{
			HostUUID:              host.UUID,
			CertificateTemplateID: certID,
			Status:                fleet.CertificateTemplateDelivered,
			OperationType:         fleet.MDMOperationTypeInstall,
		},
	})
	require.NoError(t, err)

	var getCertResp getDeviceCertificateTemplateResponse

	resp := s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", certID), nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getCertResp))
	require.NoError(t, resp.Body.Close())
	require.Equal(t, getCertResp.Certificate.Status, fleet.CertificateTemplateFailed) // failed because no IDP user to replace variables

	// Add an IDP user for the host
	err = s.ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "test.user@example.com",
			Source: fleet.DeviceMappingMDMIdpAccounts,
		},
	}, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)

	// Get certificate without node_key
	resp = s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", certID), nil, http.StatusUnauthorized, nil)
	require.NoError(t, resp.Body.Close())

	// Get certificate with node_key (should return replaced variables)
	resp = s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", certID), nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getCertResp))
	require.NoError(t, resp.Body.Close())
	require.NotNil(t, getCertResp.Certificate)

	assert.Contains(t, getCertResp.Certificate.SubjectName, "test.user@example.com")
	assert.Contains(t, getCertResp.Certificate.SubjectName, "test-uuid-12345")
	assert.Contains(t, getCertResp.Certificate.SubjectName, "TEST-SERIAL-67890")

	// Get certificate without host_uuid (should return subject with variables)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates/%d", certID), nil, http.StatusOK, &getCertResp)
	require.NotNil(t, getCertResp.Certificate)

	assert.Contains(t, getCertResp.Certificate.SubjectName, "$FLEET_VAR_HOST_END_USER_IDP_USERNAME")
	assert.Contains(t, getCertResp.Certificate.SubjectName, "$FLEET_VAR_HOST_UUID")
	assert.Contains(t, getCertResp.Certificate.SubjectName, "$FLEET_VAR_HOST_HARDWARE_SERIAL")

	// batch delete certificate templates
	var delBatchResp deleteCertificateTemplateSpecsResponse
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids":     []uint{listCertifcatesResp.Certificates[0].ID, listCertifcatesResp.Certificates[1].ID},
		"team_id": team.ID,
	}, http.StatusOK, &delBatchResp)

	// Verify activity was created for deleting certificates
	s.lastActivityMatches(
		fleet.ActivityTypeEditedAndroidCertificate{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "team_id": %d, "team_name": %q}`, team.ID, team.Name, team.ID, team.Name),
		0,
	)

	// list specs
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates?team_id=%d", team.ID), nil, http.StatusOK, &listCertifcatesResp)
	require.Empty(t, listCertifcatesResp.Certificates)

	activitiesBeforeNoTeam := s.listActivities()

	// certificate templates for "No team"
	s.DoJSON("POST", "/api/latest/fleet/spec/certificates", applyCertificateTemplateSpecsRequest{
		Specs: []*fleet.CertificateRequestSpec{
			{
				Name:                   "No Team Template 1",
				Team:                   "No team",
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/OU=$FLEET_VAR_HOST_UUID",
			},
			{
				Name:                   "No Team Template 2",
				Team:                   "",
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_HARDWARE_SERIAL",
			},
			{
				Name: "No Team Template 3",
				// No Team field, should default to empty string
				CertificateAuthorityId: ca.ID,
				SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
			},
		},
	}, http.StatusOK, &applyResp)

	// Only one activity was created for "No team"
	activitiesAfterNoTeam := s.listActivities()
	require.Len(t, activitiesAfterNoTeam, len(activitiesBeforeNoTeam)+1, "expected exactly one new activity for no team")
	s.lastActivityMatches(
		fleet.ActivityTypeEditedAndroidCertificate{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "team_id": null, "team_name": null}`,
		0,
	)

	// list specs for "no team" (team_id 0)
	var noTeamCertificatesResp listCertificateTemplatesResponse
	s.DoJSON("GET", "/api/latest/fleet/certificates", nil, http.StatusOK, &noTeamCertificatesResp)
	require.Len(t, noTeamCertificatesResp.Certificates, 3)
	certNames := []string{noTeamCertificatesResp.Certificates[0].Name, noTeamCertificatesResp.Certificates[1].Name, noTeamCertificatesResp.Certificates[2].Name}
	assert.ElementsMatch(t, []string{"No Team Template 1", "No Team Template 2", "No Team Template 3"}, certNames)

	// Create a host
	noTeamHost, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("test-cert-no-team-node-key"),
		UUID:            "test-no-team-uuid-12345",
		Hostname:        "test-cert-no-team-host.local",
		HardwareSerial:  "TEST-NO-TEAM-SERIAL",
		TeamID:          nil, // No team
	})
	require.NoError(t, err)

	noTeamOrbitNodeKey := uuid.New().String()
	noTeamHost.OrbitNodeKey = &noTeamOrbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, noTeamHost))

	// Add an IDP user for host
	err = s.ds.ReplaceHostDeviceMapping(ctx, noTeamHost.ID, []*fleet.HostDeviceMapping{
		{
			HostID: noTeamHost.ID,
			Email:  "no.team.user@example.com",
			Source: fleet.DeviceMappingMDMIdpAccounts,
		},
	}, fleet.DeviceMappingMDMIdpAccounts)
	require.NoError(t, err)

	savedNoTeamCertTemplates, _, err := s.ds.GetCertificateTemplatesByTeamID(ctx, 0, fleet.ListOptions{Page: 0, PerPage: 10})
	require.NoError(t, err)
	require.Len(t, savedNoTeamCertTemplates, 3)
	noTeamCertID := findCertID(savedNoTeamCertTemplates, "No Team Template 1")

	// Create a host_certificate_templates record for this no-team host
	err = s.ds.BulkInsertHostCertificateTemplates(ctx, []fleet.HostCertificateTemplate{
		{
			HostUUID:              noTeamHost.UUID,
			CertificateTemplateID: noTeamCertID,
			Status:                fleet.CertificateTemplateDelivered,
			OperationType:         fleet.MDMOperationTypeInstall,
		},
	})
	require.NoError(t, err)

	// Get certificate with orbit node_key (should return replaced variables)
	var getNoTeamCertResp getDeviceCertificateTemplateResponse
	resp = s.DoRawWithHeaders("GET", fmt.Sprintf("/api/fleetd/certificates/%d", noTeamCertID), nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Node key %s", noTeamOrbitNodeKey),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&getNoTeamCertResp))
	require.NoError(t, resp.Body.Close())
	require.NotNil(t, getNoTeamCertResp.Certificate)
	assert.Contains(t, getNoTeamCertResp.Certificate.SubjectName, "no.team.user@example.com")
	assert.Contains(t, getNoTeamCertResp.Certificate.SubjectName, "test-no-team-uuid-12345")

	var profilesSummaryResp getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", nil, http.StatusOK, &profilesSummaryResp)
	require.NotNil(t, profilesSummaryResp)
	require.Equal(t, uint(0), profilesSummaryResp.Verified)
	require.Equal(t, uint(0), profilesSummaryResp.Verifying)
	require.Equal(t, uint(0), profilesSummaryResp.Failed)
	require.Equal(t, uint(0), profilesSummaryResp.Pending)

	// creating a certificate
	var createCertResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", map[string]any{
		"name":                     "POST No Team Cert",
		"certificate_authority_id": ca.ID,
		"subject_name":             "CN=$FLEET_VAR_HOST_UUID",
		// team_id intentionally omitted - should default to 0
	}, http.StatusOK, &createCertResp)
	require.NotZero(t, createCertResp.ID)
	require.Equal(t, "POST No Team Cert", createCertResp.Name)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates/%d", createCertResp.ID), nil, http.StatusOK, &getCertResp)
	require.NotNil(t, getCertResp.Certificate)

	// Delete is authorized properly
	observerEmail := "observer-cert-test@fleetdm.com"
	observerPwd := test.GoodPassword
	observerUser := &fleet.User{
		Name:       "Observer User",
		Email:      observerEmail,
		GlobalRole: new(fleet.RoleObserver),
	}
	require.NoError(t, observerUser.SetPassword(observerPwd, 10, 10))
	_, err = s.ds.NewUser(ctx, observerUser)
	require.NoError(t, err)

	// Switch to observer user
	s.token = s.getCachedUserToken(observerEmail, observerPwd)
	// just in case test fails, restore to admin
	defer func() { s.token = s.getTestAdminToken() }()

	observerTargetCertID := findCertID(savedNoTeamCertTemplates, "No Team Template 1")
	// Delete with observer
	resp = s.Do("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids":     []uint{observerTargetCertID},
		"team_id": uint(0), // "No team"
	}, http.StatusForbidden)
	resp.Body.Close()

	// Switch back to admin
	s.token = s.getTestAdminToken()

	// Verify the certificate still exists (wasn't deleted by observer)
	s.DoJSON("GET", "/api/latest/fleet/certificates", nil, http.StatusOK, &noTeamCertificatesResp)
	found := false
	for _, cert := range noTeamCertificatesResp.Certificates {
		if cert.ID == observerTargetCertID {
			found = true
			break
		}
	}
	require.True(t, found, "Certificate should not be deleted by observer user")

	// Cannot delete certificates from a different team
	// Create team 2
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Test Team 2"})
	require.NoError(t, err)
	team2ID := team2.ID
	// Create a certificate in team2
	var team2CertResp createCertificateTemplateResponse
	s.DoJSON("POST", "/api/latest/fleet/certificates", createCertificateTemplateRequest{
		Name:                   "Team 2 Cert",
		TeamID:                 team2ID,
		CertificateAuthorityId: ca.ID,
		SubjectName:            "CN=$FLEET_VAR_HOST_UUID",
	}, http.StatusOK, &team2CertResp)
	var forbiddenDelResp deleteCertificateTemplateSpecsResponse
	// Delete with team 1 id and certificate from team 2
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids":     []uint{team2CertResp.ID},
		"team_id": team.ID,
	}, http.StatusForbidden, &forbiddenDelResp)
	var listTeam2Resp listCertificateTemplatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificates?team_id=%d", team2ID),
		nil, http.StatusOK, &listTeam2Resp)
	require.Len(t, listTeam2Resp.Certificates, 1)
	require.Equal(t, "Team 2 Cert", listTeam2Resp.Certificates[0].Name)

	var delResp deleteCertificateTemplateResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/certificates/%d", createCertResp.ID), nil, http.StatusOK, &delResp)

	// Refresh the no-team list so we don't pick a stale entry from a snapshot taken before the
	// preceding single-cert delete. Then pick a known cert by name.
	s.DoJSON("GET", "/api/latest/fleet/certificates", nil, http.StatusOK, &noTeamCertificatesResp)
	deletedNoTeamCertID := findCertID(noTeamCertificatesResp.Certificates, "No Team Template 1")
	var delBatchResp2 deleteCertificateTemplateSpecsResponse
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids": []uint{deletedNoTeamCertID},
		// team_id intentionally omitted - should default to 0
	}, http.StatusOK, &delBatchResp2)

	s.DoJSON("GET", "/api/latest/fleet/certificates", nil, http.StatusOK, &noTeamCertificatesResp)
	require.Len(t, noTeamCertificatesResp.Certificates, 2)
	for _, cert := range noTeamCertificatesResp.Certificates {
		require.NotEqual(t, deletedNoTeamCertID, cert.ID, "deleted certificate should not appear in list")
	}

	// Delete the two remaining no-team templates by name lookup, not list-position.
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids": []uint{
			findCertID(noTeamCertificatesResp.Certificates, "No Team Template 2"),
			findCertID(noTeamCertificatesResp.Certificates, "No Team Template 3"),
		},
		"team_id": uint(0),
	}, http.StatusOK, &delBatchResp)

	s.DoJSON("GET", "/api/latest/fleet/certificates", nil, http.StatusOK, &noTeamCertificatesResp)
	require.Empty(t, noTeamCertificatesResp.Certificates)
}
