package service

// Host certificate tests for the core (no-license) suite.
//
// Belongs here: listing a host's certificates, and creating, updating and deleting
// certificate templates including via the spec endpoint.
//
// Does not belong here: certificate authority configuration, which belongs to the
// MDM suite (integration_certificate_authorities_test.go).

import (
	"context"
	"crypto/sha1" // nolint: gosec
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestHostCertificates() {
	t := s.T()
	ctx := context.Background()

	token := "good_token"
	host := createOrbitEnrolledHost(t, "linux", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// no certificate at the moment
	var certResp listHostCertificatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp)
	require.Empty(t, certResp.Certificates)

	certResp = listHostCertificatesResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK)
	err := json.NewDecoder(res.Body).Decode(&certResp)
	require.NoError(t, err)
	require.Empty(t, certResp.Certificates)

	// create some certs for that host
	certNames := []string{"a", "b", "c", "d", "e"}
	now := time.Now()
	// sorting by not_valid_after should get us "d", "c", "e", "a", "b"
	notValidAfterTimes := []time.Time{
		now.Add(time.Minute), now.Add(time.Hour),
		now.Add(time.Second), now.Add(time.Millisecond),
		now.Add(2 * time.Second),
	}
	certs := make([]*fleet.HostCertificateRecord, 0, len(certNames))
	for i, name := range certNames {
		sha1Sum := sha1.Sum([]byte(name)) // nolint:gosec
		certs = append(certs, &fleet.HostCertificateRecord{
			HostID:         host.ID,
			CommonName:     name,
			SHA1Sum:        sha1Sum[:],
			SubjectCountry: "s" + name,
			IssuerCountry:  "i" + name,
			NotValidBefore: now.Add(-24 * time.Hour), // 1 day ago
			NotValidAfter:  notValidAfterTimes[i],
			Source:         fleet.SystemHostCertificate,
		})
	}
	require.NoError(t, s.ds.UpdateHostCertificates(ctx, host.ID, host.UUID, certs, fleet.HostCertificateOriginOsquery, nil))

	// list all certs
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp)
	require.Len(t, certResp.Certificates, len(certNames))
	for i, cert := range certResp.Certificates {
		want := certNames[i]
		require.Equal(t, want, cert.CommonName)
		require.NotNil(t, cert.Subject)
		require.Equal(t, "s"+want, cert.Subject.Country)
		require.NotNil(t, cert.Issuer)
		require.Equal(t, "i"+want, cert.Issuer.Country)
		require.Equal(t, fleet.SystemHostCertificate, cert.Source)
	}

	certResp = listHostCertificatesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&certResp)
	require.NoError(t, err)
	require.Len(t, certResp.Certificates, len(certNames))
	for i, cert := range certResp.Certificates {
		want := certNames[i]
		require.Equal(t, want, cert.CommonName)
		require.NotNil(t, cert.Subject)
		require.Equal(t, "s"+want, cert.Subject.Country)
		require.NotNil(t, cert.Issuer)
		require.Equal(t, "i"+want, cert.Issuer.Country)
		require.Equal(t, fleet.SystemHostCertificate, cert.Source)
	}

	// non-existing host
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID+1000), nil, http.StatusNotFound, &certResp)
	// for the device endpoint, the token is the authentication so if it doesn't
	// exist, the endpoint is unauthorized.
	certResp = listHostCertificatesResponse{}
	s.DoRawNoAuth("GET", "/api/latest/fleet/device/NO-SUCH-TOKEN/certificates", nil, http.StatusUnauthorized)

	pluckCertNames := func(certs []*fleet.HostCertificatePayload) []string {
		names := make([]string, 0, len(certs))
		for _, cert := range certs {
			names = append(names, cert.CommonName)
		}
		return names
	}

	// fails if order_key  is invalid
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusBadRequest, &certResp, "order_key", "no-such-column")

	certResp = listHostCertificatesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusBadRequest, "order_key", "no-such-column")
	require.Contains(t, extractServerErrorText(res.Body), "invalid order key")

	// test the pagination options
	cases := []struct {
		queryParams []string
		wantNames   []string
		wantMeta    fleet.PaginationMetadata
	}{
		{queryParams: []string{"page", "0", "per_page", "2"}, wantNames: []string{"a", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "2"}, wantNames: []string{"c", "d"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true}},
		{queryParams: []string{"page", "2", "per_page", "2"}, wantNames: []string{"e"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "3", "per_page", "2"}, wantNames: []string{}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "0", "per_page", "4", "order_direction", "desc"}, wantNames: []string{"e", "d", "c", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "4", "order_direction", "desc"}, wantNames: []string{"a"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "0", "per_page", "3", "order_key", "not_valid_after"}, wantNames: []string{"d", "c", "e"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "3", "order_key", "not_valid_after"}, wantNames: []string{"a", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.queryParams, "_"), func(t *testing.T) {
			certResp = listHostCertificatesResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp, c.queryParams...)
			require.Len(t, certResp.Certificates, len(c.wantNames))
			require.Equal(t, c.wantNames, pluckCertNames(certResp.Certificates))
			require.Equal(t, c.wantMeta, *certResp.Meta)

			certResp = listHostCertificatesResponse{}
			res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK, c.queryParams...)
			err = json.NewDecoder(res.Body).Decode(&certResp)
			require.NoError(t, err)
			require.Len(t, certResp.Certificates, len(c.wantNames))
			require.Equal(t, c.wantNames, pluckCertNames(certResp.Certificates))
			require.Equal(t, c.wantMeta, *certResp.Meta)
		})
	}
}

func (s *integrationTestSuite) TestUpdateHostCertificateTemplate() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestUpdateHostCertificateTemplate SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	certTemplate := &fleet.CertificateTemplate{
		Name:                   "TestUpdateHostCertificateTemplate-Cert",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	}
	savedTemplate, err := s.ds.CreateCertificateTemplate(ctx, certTemplate)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate)

	orbitNodeKey := uuid.New().String()
	uuid := uuid.New().String()
	hostName := "test-update-host-certificate-template"

	// Create a host
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		NodeKey:  &orbitNodeKey,
		UUID:     uuid,
		Hostname: hostName,
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	host.OrbitNodeKey = &orbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	certificateTemplateID := savedTemplate.ID

	// Delete the certificate after the test is done, so the team can be deleted.
	defer func() {
		// Clean up
		err = s.ds.DeleteCertificateTemplate(ctx, certificateTemplateID)
		require.NoError(t, err)
	}()

	// Create a record in host_certificate_templates using ad hoc SQL
	sql := `
INSERT INTO host_certificate_templates (
	host_uuid,
	certificate_template_id,
	status,
	fleet_challenge,
	operation_type,
	name
) VALUES (?, ?, ?, ?, ?, ?);
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, sql, host.UUID, certificateTemplateID, "pending", "some_challenge_value", "install", savedTemplate.Name)
		require.NoError(t, err)
		return nil
	})

	// Enable Android MDM and verify GetHost returns operation_type for certificate templates
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origAndroidEnabled := appCfg.MDM.AndroidEnabledAndConfigured
	appCfg.MDM.AndroidEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	err = s.ds.SetAndroidEnabledAndConfigured(ctx, true)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.AndroidEnabledAndConfigured = origAndroidEnabled
		_ = s.ds.SaveAppConfig(ctx, appCfg)
		_ = s.ds.SetAndroidEnabledAndConfigured(ctx, origAndroidEnabled)
	}()

	// Verify GetHost returns operation_type for certificate templates
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 1)
	profile := (*getHostResp.Host.MDM.Profiles)[0]
	require.Equal(t, savedTemplate.Name, profile.Name)
	require.Equal(t, fleet.AndroidCertificateTemplateProfileID, profile.ProfileUUID)
	require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "operation_type should be populated for certificate templates")

	// Test cases
	cases := []struct {
		name                    string
		templateID              uint
		newStatus               string
		newOperationType        *string
		detail                  *string
		expectedResponseStatus  int
		expectedResponseMessage string
		headers                 map[string]string
	}{
		{
			name:                   "Valid Update",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			detail:                 new("Certificate Verified"),
			expectedResponseStatus: http.StatusOK,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "Invalid Status",
			templateID:              certificateTemplateID,
			newStatus:               "invalid_status",
			expectedResponseStatus:  http.StatusUnprocessableEntity,
			expectedResponseMessage: "invalid status value",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "Wrong node key",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnauthorized,
			expectedResponseMessage: "host certificate template not found",
			headers: map[string]string{
				"Authorization": "Node key wrong-node-key",
			},
		},
		{
			name:                    "With no auth headers",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnauthorized,
			expectedResponseMessage: "host certificate template not found",
		},
		{
			name:                    "Wrong Template ID",
			templateID:              9999,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusNotFound,
			expectedResponseMessage: "host certificate template not found",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type install",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new("install"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type remove",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new("remove"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type empty string",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new(""),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "with invalid operation_type",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnprocessableEntity,
			expectedResponseMessage: "must be 'install' or 'remove'",
			newOperationType:        new("invalid_operation"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("TestUpdateHostCertificateTemplate:%s", tc.name), func(t *testing.T) {
			req, err := json.Marshal(updateCertificateStatusRequest{
				Status:        tc.newStatus,
				Detail:        tc.detail,
				OperationType: tc.newOperationType,
			})
			require.NoError(t, err)

			resp := s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", tc.templateID), req, tc.expectedResponseStatus, tc.headers)
			require.NoError(t, resp.Body.Close())
		})
	}
}

func (s *integrationTestSuite) TestDeleteCertificateTemplate() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestDeleteCertificateTemplate Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestDeleteCertificateTemplate SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	certTemplate := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplate-Cert",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject",
	}
	savedTemplate, err := s.ds.CreateCertificateTemplate(ctx, certTemplate)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate)
	certificateTemplateID := savedTemplate.ID
	certTemplateName := savedTemplate.Name

	// Create hosts with different certificate template statuses
	hostPending, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-pending",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostDelivered, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-delivered",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostVerified, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-verified",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostFailed, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-failed",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	// Insert host_certificate_templates with various statuses
	insertSQL := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// Pending status - should be deleted
		_, err := q.ExecContext(ctx, insertSQL, hostPending.UUID, certificateTemplateID, "pending", "install", nil, certTemplateName)
		require.NoError(t, err)
		// Delivered status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostDelivered.UUID, certificateTemplateID, "delivered", "install", "challenge1", certTemplateName)
		require.NoError(t, err)
		// Verified status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostVerified.UUID, certificateTemplateID, "verified", "install", "challenge2", certTemplateName)
		require.NoError(t, err)
		// Failed status - should be deleted (never successfully installed)
		_, err = q.ExecContext(ctx, insertSQL, hostFailed.UUID, certificateTemplateID, "failed", "install", "challenge3", certTemplateName)
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

	// Verify the records exist before deletion via GetHost API
	var getHostResp getHostResponse
	for _, tc := range []struct {
		host           *fleet.Host
		hostName       string
		expectedStatus string
	}{
		{hostPending, "hostPending", string(fleet.CertificateTemplatePending)},
		{hostDelivered, "hostDelivered", string(fleet.CertificateTemplateDelivered)},
		{hostVerified, "hostVerified", string(fleet.CertificateTemplateVerified)},
		{hostFailed, "hostFailed", string(fleet.CertificateTemplateFailed)},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		require.NotNil(t, getHostResp.Host.MDM.Profiles, "%s should have MDM profiles before deletion", tc.hostName)

		profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
		require.NotNil(t, profile, "%s should have certificate template profile %s before deletion", tc.hostName, certTemplateName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, tc.expectedStatus, *profile.Status, "%s profile status should be %s before deletion", tc.hostName, tc.expectedStatus)
		require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "%s profile operation_type should be install before deletion", tc.hostName)
	}

	// Delete the certificate template via API
	var deleteResp deleteCertificateTemplateResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/certificates/%d", certificateTemplateID), nil, http.StatusOK, &deleteResp)

	// After deletion:
	// - hostPending (pending/install) should have NO profile (record was deleted - never installed)
	// - hostFailed (failed/install) should have NO profile (record was deleted - never successfully installed)
	// - hostDelivered, hostVerified should have pending/remove profiles
	//   (kept for cron job to process removal from devices)

	// Verify hostPending has no profile after deletion (was pending/install, never installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostPending.ID), nil, http.StatusOK, &getHostResp)
	profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
	require.Nil(t, profile, "hostPending should not have certificate template profile after deletion")

	// Verify hostFailed has no profile after deletion (was failed/install, never successfully installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostFailed.ID), nil, http.StatusOK, &getHostResp)
	profile = findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
	require.Nil(t, profile, "hostFailed should not have certificate template profile after deletion")

	// Verify hosts that had delivered/verified status now have pending/remove profiles
	for _, tc := range []struct {
		host     *fleet.Host
		hostName string
	}{
		{hostDelivered, "hostDelivered"},
		{hostVerified, "hostVerified"},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
		require.NotNil(t, profile, "%s should have pending remove profile after deletion", tc.hostName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, string(fleet.CertificateTemplatePending), *profile.Status, "%s profile status should be pending after deletion", tc.hostName)
		require.Equal(t, fleet.MDMOperationTypeRemove, profile.OperationType, "%s profile operation_type should be remove after deletion", tc.hostName)
	}
}

func (s *integrationTestSuite) TestDeleteCertificateTemplateSpec() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestDeleteCertificateTemplateSpec Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestDeleteCertificateTemplateSpec SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Create two certificate templates
	certTemplate1 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplateSpec-Cert1",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	}
	savedTemplate1, err := s.ds.CreateCertificateTemplate(ctx, certTemplate1)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate1)
	certTemplateID1 := savedTemplate1.ID
	certTemplateName1 := savedTemplate1.Name

	certTemplate2 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplateSpec-Cert2",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 2",
	}
	savedTemplate2, err := s.ds.CreateCertificateTemplate(ctx, certTemplate2)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate2)
	certTemplateID2 := savedTemplate2.ID
	certTemplateName2 := savedTemplate2.Name

	// Create hosts with different certificate template statuses
	hostPending, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-pending",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostDelivered, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-delivered",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostVerified, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-verified",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostFailed, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-failed",
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
		// Template 1 - hosts with pending and delivered status
		_, err := q.ExecContext(ctx, insertSQL, hostPending.UUID, certTemplateID1, "pending", "install", nil, certTemplateName1)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, insertSQL, hostDelivered.UUID, certTemplateID1, "delivered", "install", "challenge1", certTemplateName1)
		require.NoError(t, err)

		// Template 2 - hosts with verified and failed status
		_, err = q.ExecContext(ctx, insertSQL, hostVerified.UUID, certTemplateID2, "verified", "install", "challenge2", certTemplateName2)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, insertSQL, hostFailed.UUID, certTemplateID2, "failed", "install", "challenge3", certTemplateName2)
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

	// Verify the records exist before deletion via GetHost API
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

	// Delete both certificate templates via spec endpoint (batch delete)
	var delBatchResp deleteCertificateTemplateSpecsResponse
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids":     []uint{certTemplateID1, certTemplateID2},
		"team_id": teamID,
	}, http.StatusOK, &delBatchResp)

	// Verify certificate templates were deleted
	_, err = s.ds.GetCertificateTemplateById(ctx, certTemplateID1)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 1 should be deleted")

	_, err = s.ds.GetCertificateTemplateById(ctx, certTemplateID2)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 2 should be deleted")

	// After deletion:
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
