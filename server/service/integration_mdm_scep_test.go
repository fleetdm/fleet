package service

// SCEP and certificate authority integration tests for the MDM suite.
//
// Belongs here: the NDES and Smallstep SCEP proxies, and DigiCert and custom SCEP certificate
// authorities (configuration and end-to-end certificate delivery through profiles).
//
// Does not belong here: certificate authority CRUD (integration_certificate_authorities_test.go),
// SCEP challenges in enrollment profiles (integration_mdm_apple_enrollment_test.go), and SCEP
// certificate renewal (integration_mdm_lifecycle_test.go).

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/certissue"
	"github.com/fleetdm/fleet/v4/ee/server/service/ejbca/ejbcatest"
	svc_scep "github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/smallstep/scep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"software.sslmate.com/src/go-pkcs12"
)

func (s *integrationMDMTestSuite) TestSCEPProxy() {
	s.runSCEPProxyTestWithOptionalSuffix("")
}

func (s *integrationMDMTestSuite) TestSCEPProxyWithMicrosoftSuffix() {
	// Windows SCEP client appends pkiclient.exe to the URL for all requests
	s.runSCEPProxyTestWithOptionalSuffix("/pkiclient.exe")
}

// suffix is optional but will be tested with all endpoints. The idea of this is to ensure
// we get the exact same behavior whether it is used or not.
func (s *integrationMDMTestSuite) runSCEPProxyTestWithOptionalSuffix(suffix string) {
	t := s.T()
	ctx := context.Background()

	// Add an MDM profile
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "Good1"),
		mobileconfigForTest("N2", "Bad1"),
	}
	// add global profile
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// trigger a profile sync
	s.awaitRunAppleMDMWorkerSchedule()
	s.awaitTriggerProfileSchedule(t)
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID))
	profiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 2)
	var profileUUID string
	var goodProfile, badProfile fleet.HostMDMAppleProfile
	for _, profile := range profiles {
		switch profile.Identifier {
		case "Good1":
			profileUUID = profile.ProfileUUID
			goodProfile = profile
		case "Bad1":
			profile.Status = &fleet.MDMDeliveryFailed
			badProfile = profile
		}
	}
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       badProfile.ProfileUUID,
		ProfileIdentifier: badProfile.Identifier,
		ProfileName:       badProfile.Name,
		HostUUID:          host.UUID,
		CommandUUID:       badProfile.CommandUUID,
		OperationType:     badProfile.OperationType,
		Status:            badProfile.Status,
		Detail:            badProfile.Detail,
		Checksum:          []byte("checksum"),
		Scope:             fleet.PayloadScopeSystem,
	}, {
		ProfileUUID:       goodProfile.ProfileUUID,
		ProfileIdentifier: goodProfile.Identifier,
		ProfileName:       goodProfile.Name,
		HostUUID:          host.UUID,
		CommandUUID:       goodProfile.CommandUUID,
		OperationType:     goodProfile.OperationType,
		Status:            goodProfile.Status,
		Detail:            goodProfile.Detail,
		Checksum:          []byte("checksum"),
		Scope:             fleet.PayloadScopeSystem,
	}})
	require.NoError(t, err)
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:    host.UUID,
			ProfileUUID: profileUUID,
			Type:        fleet.CAConfigNDES,
			CAName:      "NDES",
		},
		{
			HostUUID:    host.UUID,
			ProfileUUID: badProfile.ProfileUUID,
			Type:        fleet.CAConfigNDES,
			CAName:      "NDES",
		},
	})
	require.NoError(t, err)

	identifier := url.PathEscape(host.UUID + "," + profileUUID)
	badIdentifier := url.PathEscape(host.UUID + "," + badProfile.ProfileUUID)

	data, err := os.ReadFile("./testdata/PKCSReq.der")
	require.NoError(t, err)
	message := base64.StdEncoding.EncodeToString(data)

	// NDES not configured
	res := s.DoRawNoAuth("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest)
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "missing operation")
	// Provide SCEP operation (GetCACaps)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetCACerts)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (PKIOperation)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "PKIOperation",
		"message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetNextCACert)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetNextCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "not implemented")

	// Configure a bad SCEP URL using local test server
	testServer, cleanup := createTestServerWithStatusCode(http.StatusGone)
	defer cleanup()

	// Bypass service validation to insert a bad url
	adminUrl := "bad-admin-url"
	username := "user"
	password := "pass"
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Name:     new("bad-url"),
		URL:      &testServer.URL,
		AdminURL: &adminUrl,
		Username: &username,
		Password: &password,
	})
	require.NoError(t, err)

	// The proxy route is unauthenticated, so upstream failures must not disclose the CA URL to the caller.
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACaps from SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACert from SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not do PKIOperation on SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)

	// Test timeout error
	ndesTimeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	*s.scepConfig.Timeout = time.Microsecond
	t.Cleanup(func() {
		*s.scepConfig.Timeout = 10 * time.Second
		ndesTimeoutServer.Close()
	})

	// Bypass service validation to verify timeout behaviour
	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	ca, err = s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Name:     new("bad-url"),
		URL:      &ndesTimeoutServer.URL,
		AdminURL: &adminUrl,
		Username: &username,
		Password: &password,
	})
	require.NoError(t, err)

	// GetCACaps
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation", "GetCACaps")
	// GetCACert
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation", "GetCACert")
	// PKIOperation
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation",
		"PKIOperation", "message", message)
	*s.scepConfig.Timeout = 10 * time.Second

	scepServer := scep_server.StartTestSCEPServer(t)

	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	_, err = s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:     string(fleet.CATypeNDESSCEPProxy),
		Name:     new("bad-url"),
		URL:      new(scepServer.URL + "/scep"),
		AdminURL: &adminUrl,
		Username: &username,
		Password: &password,
	})
	require.NoError(t, err)

	// GetCACaps
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation", "GetCACaps")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, scepserver.DefaultCACaps, string(body))

	// GetCACert
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation", "GetCACert")
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	certs, err := x509.ParseCertificates(body)
	require.NoError(t, err)
	assert.Len(t, certs, 1)

	// PKIOperation
	// Invalid identifier format
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozo"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid identifier")
	// Non-Apple config profile (missing leading 'a')
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CbozoProfile"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid profile UUID")
	// Unknown host/profile
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CabozoProfile"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "unknown identifier")
	// Profile which is not pending
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// "Good" request without one-time challenge.
	// Note, a cert is not returned here because the message is not a fully valid SCEP request. However, building a valid SCEP request is a bit involved,
	// and this request is sufficient for testing our proxy functionality.
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err := scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)

	// Expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: new(time.Now().Add(-svc_scep.NDESChallengeInvalidAfter)),
			Type:                 fleet.CAConfigNDES,
			CAName:               "NDES",
		},
	})
	require.NoError(t, err)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "challenge password")

	// Non-expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: new(time.Now().Add(-svc_scep.NDESChallengeInvalidAfter + time.Minute)),
			Type:                 fleet.CAConfigNDES,
			CAName:               "NDES",
		},
	})
	require.NoError(t, err)

	// Profile status is not yet "pending" (should be null) until profile sync
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	// Good request with non-expired challenge
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err = scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)
}

func (s *integrationMDMTestSuite) TestSmallstepSCEPProxy() {
	s.runSmallstepSCEPProxyTestWithOptionalSuffix("")
}

func (s *integrationMDMTestSuite) TestSmallstepSCEPProxyWithMicrosoftSuffix() {
	// Windows SCEP client appends pkiclient.exe to the URL for all requests
	s.runSmallstepSCEPProxyTestWithOptionalSuffix("/pkiclient.exe")
}

func (s *integrationMDMTestSuite) runSmallstepSCEPProxyTestWithOptionalSuffix(suffix string) {
	t := s.T()
	ctx := context.Background()
	caName := "Smallstep" + strings.ReplaceAll(strings.ReplaceAll(suffix, "/", ""), ".", "")

	// Add an MDM profile
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "Good1"),
		mobileconfigForTest("N2", "Bad1"),
	}
	// add global profile
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID))
	profiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 2)
	var profileUUID string
	var goodProfile, badProfile fleet.HostMDMAppleProfile
	for _, profile := range profiles {
		switch profile.Identifier {
		case "Good1":
			profileUUID = profile.ProfileUUID
			goodProfile = profile
		case "Bad1":
			profile.Status = &fleet.MDMDeliveryFailed
			badProfile = profile
		}
	}
	err = s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
		ProfileUUID:       badProfile.ProfileUUID,
		ProfileIdentifier: badProfile.Identifier,
		ProfileName:       badProfile.Name,
		HostUUID:          host.UUID,
		CommandUUID:       badProfile.CommandUUID,
		OperationType:     badProfile.OperationType,
		Status:            badProfile.Status,
		Detail:            badProfile.Detail,
		Checksum:          []byte("checksum"),
		Scope:             fleet.PayloadScopeSystem,
	}, {
		ProfileUUID:       goodProfile.ProfileUUID,
		ProfileIdentifier: goodProfile.Identifier,
		ProfileName:       goodProfile.Name,
		HostUUID:          host.UUID,
		CommandUUID:       goodProfile.CommandUUID,
		OperationType:     goodProfile.OperationType,
		Status:            goodProfile.Status,
		Detail:            goodProfile.Detail,
		Checksum:          []byte("checksum"),
		Scope:             fleet.PayloadScopeSystem,
	}})
	require.NoError(t, err)
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:    host.UUID,
			ProfileUUID: profileUUID,
			Type:        fleet.CAConfigSmallstep,
			CAName:      caName,
		},
		{
			HostUUID:    host.UUID,
			ProfileUUID: badProfile.ProfileUUID,
			Type:        fleet.CAConfigSmallstep,
			CAName:      caName,
		},
	})
	require.NoError(t, err)

	identifier := url.PathEscape(host.UUID + "," + profileUUID + "," + caName)
	badIdentifier := url.PathEscape(host.UUID + "," + badProfile.ProfileUUID + "," + caName)

	data, err := os.ReadFile("./testdata/PKCSReq.der")
	require.NoError(t, err)
	message := base64.StdEncoding.EncodeToString(data)

	// NDES not configured
	res := s.DoRawNoAuth("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest)
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "missing operation")
	// Provide SCEP operation (GetCACaps)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetCACerts)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (PKIOperation)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "PKIOperation",
		"message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), svc_scep.MessageSCEPProxyNotConfigured)
	// Provide SCEP operation (GetNextCACert)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation", "GetNextCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "not implemented")

	// Configure a bad SCEP URL using local test server
	testServer, cleanup := createTestServerWithStatusCode(http.StatusGone)
	defer cleanup()

	// Bypass service validation to insert a bad url
	challengeUrl := "bad-admin-url"
	username := "user"
	password := "pass"
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:         string(fleet.CATypeSmallstep),
		Name:         new(caName),
		URL:          &testServer.URL,
		ChallengeURL: &challengeUrl,
		Username:     &username,
		Password:     &password,
	})
	require.NoError(t, err)

	// The proxy route is unauthenticated, so upstream failures must not disclose the CA URL to the caller.
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation", "GetCACaps")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACaps from SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation", "GetCACert")
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not GetCACert from SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusInternalServerError, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "Could not do PKIOperation on SCEP server")
	assert.NotContains(t, string(errBody), testServer.URL)

	// Test timeout error
	ndesTimeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	*s.scepConfig.Timeout = time.Microsecond
	t.Cleanup(func() {
		*s.scepConfig.Timeout = 10 * time.Second
		ndesTimeoutServer.Close()
	})

	// Bypass service validation to verify timeout behaviour
	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	ca, err = s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:         string(fleet.CATypeSmallstep),
		Name:         new(caName),
		URL:          &ndesTimeoutServer.URL,
		ChallengeURL: &challengeUrl,
		Username:     &username,
		Password:     &password,
	})
	require.NoError(t, err)

	// GetCACaps
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation", "GetCACaps")
	// GetCACert
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation", "GetCACert")
	// PKIOperation
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusRequestTimeout, nil, "operation",
		"PKIOperation", "message", message)
	*s.scepConfig.Timeout = 10 * time.Second

	scepServer := scep_server.StartTestSCEPServer(t)

	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	_, err = s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:         string(fleet.CATypeSmallstep),
		Name:         new(caName),
		URL:          new(scepServer.URL + "/scep"),
		ChallengeURL: &challengeUrl,
		Username:     &username,
		Password:     &password,
	})
	require.NoError(t, err)

	// GetCACaps
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation", "GetCACaps")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, scepserver.DefaultCACaps, string(body))

	// GetCACert
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation", "GetCACert")
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	certs, err := x509.ParseCertificates(body)
	require.NoError(t, err)
	assert.Len(t, certs, 1)

	// PKIOperation
	// Invalid identifier format
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozo"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid identifier")
	// Non-Apple config profile (missing leading 'a')
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CbozoProfile"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "invalid profile UUID")
	// Unknown host/profile
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"bozoHost%2CabozoProfile"+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "unknown identifier")
	// Profile which is not pending
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// "Good" request without one-time challenge.
	// Note, a cert is not returned here because the message is not a fully valid SCEP request. However, building a valid SCEP request is a bit involved,
	// and this request is sufficient for testing our proxy functionality.
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err := scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)

	// Expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: new(time.Now().Add(-svc_scep.SmallstepChallengeInvalidAfter)),
			Type:                 fleet.CAConfigSmallstep,
			CAName:               caName,
		},
	})
	require.NoError(t, err)
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "challenge password")

	// Non-expired challenge
	err = s.ds.BulkUpsertMDMManagedCertificates(ctx, []*fleet.MDMManagedCertificate{
		{
			HostUUID:             host.UUID,
			ProfileUUID:          profileUUID,
			ChallengeRetrievedAt: new(time.Now().Add(-svc_scep.SmallstepChallengeInvalidAfter + time.Minute)),
			Type:                 fleet.CAConfigSmallstep,
			CAName:               caName,
		},
	})
	require.NoError(t, err)

	// Profile status is not yet "pending" (should be null) until profile sync
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+badIdentifier+suffix, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	errBody, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Contains(t, string(errBody), "profile status")

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	// Good request with non-expired challenge
	res = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+suffix, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	pkiMessage, err = scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)
}

func (s *integrationMDMTestSuite) TestDigiCertConfig() {
	t := s.T()
	ctx := context.Background()
	digiCertServer := createMockDigiCertServer(t)

	// Add DigiCert integration with bad URL using local test server
	testServer, cleanup := createTestServerWithStatusCode(http.StatusGone)
	defer cleanup()

	caBad := newMockDigicertCA(testServer.URL, "ca")
	certPayload := createCertificateAuthorityRequest{
		DigiCert: &caBad,
	}
	res := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusBadRequest, &res)

	// Add 3 DigiCert integrations
	ca0 := newMockDigicertCA(digiCertServer.server.URL, "ca0")
	ca0.APIToken = "api_token0"
	ca1 := newMockDigicertCA(digiCertServer.server.URL, "ca1")
	ca1.APIToken = "api_token1"
	ca2 := newMockDigicertCA(digiCertServer.server.URL, "ca2")
	ca2.APIToken = "api_token2"
	certPayload.CertificateAuthorityPayload.DigiCert = &ca0
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)
	certPayload.CertificateAuthorityPayload.DigiCert = &ca1
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)
	certPayload.CertificateAuthorityPayload.DigiCert = &ca2
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)

	listRes := listCertificateAuthoritiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/certificate_authorities", &listCertificateAuthoritiesRequest{}, http.StatusOK, &listRes)

	digiCerts := []*fleet.CertificateAuthority{}
	digiCertsWithSecrets := []*fleet.CertificateAuthority{}
	for _, ca := range listRes.CertificateAuthorities {
		if ca.Type == string(fleet.CATypeDigiCert) {
			getRes := getCertificateAuthorityResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), &getCertificateAuthorityRequest{}, http.StatusOK, &getRes)
			digiCerts = append(digiCerts, getRes.CertificateAuthority)

			caWithSecret, err := s.ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
			require.NoError(t, err)
			digiCertsWithSecrets = append(digiCertsWithSecrets, caWithSecret)
		}
	}
	assert.Len(t, digiCerts, 3)

	for _, ca := range digiCerts {
		assert.Equal(t, fleet.MaskedPassword, *ca.APIToken)
	}

	require.Len(t, digiCertsWithSecrets, 3)
	assert.Equal(t, "api_token0", *digiCertsWithSecrets[0].APIToken)
	assert.Equal(t, "api_token1", *digiCertsWithSecrets[1].APIToken)
	assert.Equal(t, "api_token2", *digiCertsWithSecrets[2].APIToken)

	// Check 3 added activities are present
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
	require.NotEmpty(t, listActivities.Activities)
	activity := fleet.ActivityAddedDigiCert{}
	caNames := make([]string, 0, 3)
	for _, act := range listActivities.Activities {
		if act.Type == activity.ActivityName() {
			err := json.Unmarshal(*act.Details, &activity)
			require.NoError(t, err)
			caNames = append(caNames, activity.Name)
			if len(caNames) == 3 {
				break
			}
		}
	}
	slices.Sort(caNames)
	assert.Equal(t, []string{"ca0", "ca1", "ca2"}, caNames)

	// TODO(HCA): Re enable and switch out with endpoints once update endpoint is implemented.
	/*
		// Add 1, modify 1, delete 1, keep 1 the same (DigiCert integrations)
		ca1.URL += "//"
		ca3 := getDigiCertIntegration(digiCertServer.server.URL, "ca3")
		ca3.APIToken = "api_token3"
		appConfig = map[string]interface{}{
			"integrations": map[string]interface{}{
				"digicert": []fleet.DigiCertCA{ca3, ca2, ca1},
			},
		}
		raw, err = json.Marshal(appConfig)
		require.NoError(t, err)
		req.RawMessage = raw
		res = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", &req, http.StatusOK, &res)
		require.Len(t, res.Integrations.DigiCert.Value, 3)
		assert.NotEqual(t, res.Integrations.DigiCert.Value[0].Name, res.Integrations.DigiCert.Value[1].Name)
		assert.NotEqual(t, res.Integrations.DigiCert.Value[1].Name, res.Integrations.DigiCert.Value[2].Name)
		for _, ca := range res.Integrations.DigiCert.Value {
			switch ca.Name {
			case "ca1":
				assert.True(t, ca.Equals(&ca1))
			case "ca2":
				assert.True(t, ca.Equals(&ca2))
			case "ca3":
				assert.True(t, ca.Equals(&ca3))
			default:
				t.Fatalf("unexpected ca name: %s", ca.Name)
			}
			assert.Equal(t, ca.APIToken, fleet.MaskedPassword)
		}
		assets, err = s.ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
		require.NoError(t, err)
		require.Len(t, assets, 3)
		assert.EqualValues(t, "api_token1", assets["ca1"].Value)
		assert.EqualValues(t, "api_token2", assets["ca2"].Value)
		assert.EqualValues(t, "api_token3", assets["ca3"].Value)

		listActivities = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
			&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
		require.True(t, len(listActivities.Activities) > 0)
		activity = fleet.ActivityAddedDigiCert{}
		editActivity := fleet.ActivityEditedDigiCert{}
		delActivity := fleet.ActivityDeletedDigiCert{}
		var numFound int
		for _, act := range listActivities.Activities {
			switch act.Type {
			case activity.ActivityName():
				err := json.Unmarshal(*act.Details, &activity)
				require.NoError(t, err)
				assert.Equal(t, activity.Name, ca3.Name)
				numFound++
			case editActivity.ActivityName():
				err := json.Unmarshal(*act.Details, &editActivity)
				require.NoError(t, err)
				assert.Equal(t, editActivity.Name, ca1.Name)
				numFound++
			case delActivity.ActivityName():
				err := json.Unmarshal(*act.Details, &delActivity)
				require.NoError(t, err)
				assert.Equal(t, delActivity.Name, ca0.Name)
				numFound++
			}
			if numFound == 3 {
				break
			}
		}
		assert.Equal(t, 3, numFound)
	*/

	// Clear DigiCert integrations
	for _, ca := range digiCerts {
		s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	}

	// TODO(HCA): Re enable and switch out with endpoints once update endpoint is implemented.
	/*
		// Check that 3 deleted activities are present
		listActivities = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
			&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
		require.True(t, len(listActivities.Activities) > 0)
		delActivity := fleet.ActivityDeletedDigiCert{}
		caNames = make([]string, 0, 3)
		for _, act := range listActivities.Activities {
			if act.Type == delActivity.ActivityName() {
				err := json.Unmarshal(*act.Details, &delActivity)
				require.NoError(t, err)
				caNames = append(caNames, delActivity.Name)
				if len(caNames) == 3 {
					break
				}
			}
		}
		slices.Sort(caNames)
		assert.EqualValues(t, caNames, []string{"ca1", "ca2", "ca3"})
	*/
}

func (s *integrationMDMTestSuite) TestDigiCertIntegration() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)
	digiCertServer := createMockDigiCertServer(t)

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup = fleet.MacOSSetup{
		EnableEndUserAuthentication: false,
	}
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Add DigiCert config
	ca := newMockDigicertCA(digiCertServer.server.URL, "my_CA")
	ca.APIToken = "api_token0"
	certPayload := createCertificateAuthorityRequest{
		DigiCert: &ca,
	}
	res := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	_, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)

	// Receive enrollment profiles (we are not checking/testing these here)
	for {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		if cmd == nil {
			break
		}
		_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	// Add a profile with a bad CA
	profile := digiCertForTest("N1", "BadCA", "badName")
	rawRes := s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: profile},
	}}, http.StatusBadRequest)
	errMsg := extractServerErrorText(rawRes.Body)
	require.Contains(t, errMsg, "_badName does not exist")

	// ////////////////////////////////
	// Test a good profile -- happy path
	profile = digiCertForTest("N2", "I2", "my_CA")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N2", Contents: profile},
	}}, http.StatusNoContent)
	profiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 1)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	p := s.assertConfigProfilesByIdentifier(nil, "I2", true)
	require.Contains(t, string(p.Mobileconfig), "com.fleetdm.pkcs12")

	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting PKCS12 certificate")
	var fullCmd micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)
	require.NotNil(t, fullCmd.Command)
	require.NotNil(t, fullCmd.Command.InstallProfile)
	rawProfile := fullCmd.Command.InstallProfile.Payload
	if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
		p7, err := pkcs7.Parse(rawProfile)
		require.NoError(t, err)
		require.NoError(t, p7.Verify())
		rawProfile = p7.Content
	}

	type pkcs12Profile struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	var pkcs12Prof pkcs12Profile
	require.NoError(t, plist.Unmarshal(rawProfile, &pkcs12Prof))
	assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[0]["PayloadType"].(string))
	password := pkcs12Prof.PayloadContent[0]["Password"].(string)
	data := pkcs12Prof.PayloadContent[0]["PayloadContent"].([]byte)
	_, certificate, err := pkcs12.Decode(data, password)
	require.NoError(t, err)
	assert.Equal(t, ca.CertificateCommonName, certificate.Subject.CommonName)

	// Check request to DigiCert
	digiCertServer.certReqMu.Lock()
	assert.Equal(t, ca.ProfileID, digiCertServer.certReq.Profile["id"])
	assert.Equal(t, ca.CertificateSeatID, digiCertServer.certReq.Seat["seat_id"])
	assert.Equal(t, "x509", digiCertServer.certReq.DeliveryFormat)
	assert.Equal(t, ca.CertificateCommonName, digiCertServer.certReq.Attributes["subject"].(map[string]any)["common_name"])

	// Need to convert UPNs to string slice since assert.Equal cannot compare string slice and interface slice
	upnsReceived := digiCertServer.certReq.Attributes["extensions"].(map[string]any)["san"].(map[string]any)["user_principal_names"].([]any)
	var stringSlice []string
	for _, item := range upnsReceived {
		if str, ok := item.(string); ok {
			stringSlice = append(stringSlice, str)
		}
	}
	assert.Equal(t, ca.CertificateUserPrincipalNames, stringSlice)
	digiCertServer.certReqMu.Unlock()
	certProf, err := s.ds.GetAppleHostMDMCertificateProfile(ctx, host.UUID, p.ProfileUUID, "my_CA")
	require.NoError(t, err)
	require.NotNil(t, certProf)
	require.NotNil(t, certProf.NotValidAfter)
	assert.Equal(t, digiCertServer.notAfter, *certProf.NotValidAfter)

	// ////////////////////////////////
	// Try a DigiCert CA that will fail
	caFail := newMockDigicertCA(digiCertServer.server.URL, "fail_CA")
	caFail.CertificateCommonName = "Fail"
	certPayload = createCertificateAuthorityRequest{
		DigiCert: &caFail,
	}
	resFail := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &resFail)

	profileFail := digiCertForTest("N3", "I3", "fail_CA")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N2", Contents: profile}, // resend the previous profile so it doesn't get removed
		{Name: "N3", Contents: profileFail},
	}}, http.StatusNoContent)
	profiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 2)

	s.awaitTriggerProfileSchedule(t)
	p = s.assertConfigProfilesByIdentifier(nil, "I3", true)
	require.Contains(t, string(p.Mobileconfig), "com.fleetdm.pkcs12")

	getHostResp := getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	var foundGood, foundFailed bool
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	for _, prof := range *getHostResp.Host.MDM.Profiles {
		if prof.Name == "N2" {
			foundGood = true
			assert.EqualValues(t, fleet.MDMDeliveryVerifying, *prof.Status)
			assert.Equal(t, fleet.MDMOperationTypeInstall, prof.OperationType)
			continue
		}
		if prof.Name == "N3" {
			foundFailed = true
			assert.EqualValues(t, fleet.MDMDeliveryFailed, *prof.Status)
			assert.Equal(t, fleet.MDMOperationTypeInstall, prof.OperationType)
			assert.Contains(t, prof.Detail, "Expected Fail")
			continue
		}
	}
	assert.True(t, foundGood)
	assert.True(t, foundFailed)

	// ////////////////////////////////
	// Test DigiCert CA with Fleet variables
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
			host.ID, "idp@example.com", "mdm_idp_accounts",
		)

		return err
	})
	caFleetVars := fleet.DigiCertCA{
		Name:                          "FleetVars",
		URL:                           digiCertServer.server.URL,
		APIToken:                      "api_token",
		ProfileID:                     "profile_id",
		CertificateCommonName:         "$FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + " $FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP),
		CertificateUserPrincipalNames: []string{"_${FLEET_VAR_" + string(fleet.FleetVarHostEndUserEmailIDP) + "}_"},
		CertificateSeatID:             "_${FLEET_VAR_" + string(fleet.FleetVarHostHardwareSerial) + "}_",
	}
	certPayload = createCertificateAuthorityRequest{
		DigiCert: &caFleetVars,
	}
	resFleetVars := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &resFleetVars)

	profileFleetVars := digiCertForTest("N4", "I4", "FleetVars")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N2", Contents: profile}, // resend the previous profile so it doesn't get removed
		{Name: "N4", Contents: profileFleetVars},
	}}, http.StatusNoContent)
	profiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 2)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	p = s.assertConfigProfilesByIdentifier(nil, "I4", true)
	require.Contains(t, string(p.Mobileconfig), "com.fleetdm.pkcs12")

	// Check that status is pending (and not failed)
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	var found bool
	for _, prof := range *getHostResp.Host.MDM.Profiles {
		if prof.Name == "N4" {
			found = true
			assert.EqualValues(t, fleet.MDMDeliveryPending, *prof.Status)
			assert.Equal(t, fleet.MDMOperationTypeInstall, prof.OperationType)
			assert.Empty(t, prof.Detail)
			continue
		}
	}
	assert.True(t, found)

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	// first command is the RemoveProfile of N3 (which was install failed)
	require.NotNil(t, cmd, "Expecting removal of failed N3 install")
	require.Equal(t, "RemoveProfile", cmd.Command.RequestType)

	// second command is the InstallProfile of N4
	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting PKCS12 certificate")
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	fullCmd = micromdm.CommandPayload{}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	require.NotNil(t, fullCmd.Command)
	require.NotNil(t, fullCmd.Command.InstallProfile)
	rawProfile = fullCmd.Command.InstallProfile.Payload
	if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
		p7, err := pkcs7.Parse(rawProfile)
		require.NoError(t, err)
		require.NoError(t, p7.Verify())
		rawProfile = p7.Content
	}

	pkcs12Prof = pkcs12Profile{}
	require.NoError(t, plist.Unmarshal(rawProfile, &pkcs12Prof))
	assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[0]["PayloadType"].(string))
	password = pkcs12Prof.PayloadContent[0]["Password"].(string)
	data = pkcs12Prof.PayloadContent[0]["PayloadContent"].([]byte)
	_, certificate, err = pkcs12.Decode(data, password)
	require.NoError(t, err)
	assert.Equal(t, host.HardwareSerial+" idp@example.com", certificate.Subject.CommonName)

	// Check request to DigiCert
	digiCertServer.certReqMu.Lock()
	assert.Equal(t, caFleetVars.ProfileID, digiCertServer.certReq.Profile["id"])
	assert.Equal(t, "_"+host.HardwareSerial+"_", digiCertServer.certReq.Seat["seat_id"])
	assert.Equal(t, "x509", digiCertServer.certReq.DeliveryFormat)
	assert.Equal(t, host.HardwareSerial+" idp@example.com", digiCertServer.certReq.Attributes["subject"].(map[string]any)["common_name"])

	// Need to convert UPNs to string slice since assert.Equal cannot compare string slice and interface slice
	upnsReceived = digiCertServer.certReq.Attributes["extensions"].(map[string]any)["san"].(map[string]any)["user_principal_names"].([]any)
	stringSlice = nil
	for _, item := range upnsReceived {
		if str, ok := item.(string); ok {
			stringSlice = append(stringSlice, str)
		}
	}
	assert.Equal(t, []string{"_idp@example.com_"}, stringSlice)
	digiCertServer.certReqMu.Unlock()

	// ////////////////////////////////
	// Test a good profile with 2 DigiCert payloads -- happy path
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N2", Contents: profile},          // resend the previous profile so it doesn't get removed
		{Name: "N4", Contents: profileFleetVars}, // resend the previous profile so it doesn't get removed
		{Name: "DigiCert2", Contents: digiCert2Mobileconfig},
	}}, http.StatusNoContent)
	profiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 3)
	p = s.assertConfigProfilesByIdentifier(nil, "DigiCert2", true)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting PKCS12 certificate")
	fullCmd = micromdm.CommandPayload{}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)
	require.NotNil(t, fullCmd.Command)
	require.NotNil(t, fullCmd.Command.InstallProfile)
	rawProfile = fullCmd.Command.InstallProfile.Payload
	if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
		p7, err := pkcs7.Parse(rawProfile)
		require.NoError(t, err)
		require.NoError(t, p7.Verify())
		rawProfile = p7.Content
	}

	pkcs12Prof = pkcs12Profile{}
	require.NoError(t, plist.Unmarshal(rawProfile, &pkcs12Prof))
	require.Len(t, pkcs12Prof.PayloadContent, 2)
	assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[0]["PayloadType"].(string))
	assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[1]["PayloadType"].(string))
	password = pkcs12Prof.PayloadContent[0]["Password"].(string)
	data = pkcs12Prof.PayloadContent[0]["PayloadContent"].([]byte)
	_, certificate, err = pkcs12.Decode(data, password)
	require.NoError(t, err)
	assert.Equal(t, ca.CertificateCommonName, certificate.Subject.CommonName)
	password = pkcs12Prof.PayloadContent[1]["Password"].(string)
	data = pkcs12Prof.PayloadContent[1]["PayloadContent"].([]byte)
	_, certificate, err = pkcs12.Decode(data, password)
	require.NoError(t, err)
	assert.Equal(t, host.HardwareSerial+" idp@example.com", certificate.Subject.CommonName)

	prof, err := s.ds.GetAppleHostMDMCertificateProfile(ctx, host.UUID, p.ProfileUUID, "my_CA")
	require.NoError(t, err)
	require.NotNil(t, prof)
	assert.NotNil(t, prof.NotValidAfter)
	assert.Equal(t, fleet.CAConfigDigiCert, prof.Type)
	assert.Equal(t, fleet.MDMDeliveryVerifying, *prof.Status)
	prof, err = s.ds.GetAppleHostMDMCertificateProfile(ctx, host.UUID, p.ProfileUUID, "FleetVars")
	require.NoError(t, err)
	require.NotNil(t, prof)
	assert.NotNil(t, prof.NotValidAfter)
	assert.Equal(t, fleet.CAConfigDigiCert, prof.Type)
	assert.Equal(t, fleet.MDMDeliveryVerifying, *prof.Status)

	// ////////////////////////////////
	// Modify the CN/UPN/Seat ID of the profile -- DigiCert verification should not happen
	digiCertServer.server.Close() // so that verify fails if attempted
	caFleetVars.CertificateCommonName = "common_name"
	caFleetVars.CertificateUserPrincipalNames = nil
	caFleetVars.CertificateSeatID = "seat_id"
	// TODO(HCA): Call update endpoint to follow logic below once implemented
	/* appConfig = map[string]interface{}{
		"integrations": map[string]interface{}{
			"digicert": []fleet.DigiCertCA{ca, caFail, caFleetVars},
		},
	}
	raw, err = json.Marshal(appConfig)
	require.NoError(t, err)
	req = modifyAppConfigRequest{}
	req.RawMessage = raw
	res = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", &req, http.StatusOK, &res)
	assert.Len(t, res.Integrations.DigiCert.Value, 3)*/
}

func (s *integrationMDMTestSuite) TestDigiCertIntegrationWithHostPlatform() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)
	digiCertServer := createMockDigiCertServer(t)

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup = fleet.MacOSSetup{
		EnableEndUserAuthentication: false,
	}
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Add DigiCert config
	ca := newMockDigicertCA(digiCertServer.server.URL, "my_CA")
	ca.APIToken = "api_token0"
	certPayload := createCertificateAuthorityRequest{
		DigiCert: &ca,
	}
	res := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	_, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)

	// Receive enrollment profiles (we are not checking/testing these here)
	for {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		if cmd == nil {
			break
		}
		_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	// ////////////////////////////////
	// Test DigiCert CA with HOST_PLATFORM Fleet variable
	caPlatform := fleet.DigiCertCA{
		Name:                          "PlatformVar",
		URL:                           digiCertServer.server.URL,
		APIToken:                      "api_token",
		ProfileID:                     "profile_id",
		CertificateCommonName:         "$FLEET_VAR_" + string(fleet.FleetVarHostPlatform) + "-device",
		CertificateUserPrincipalNames: []string{"${FLEET_VAR_" + string(fleet.FleetVarHostPlatform) + "}@example.com"},
		CertificateSeatID:             "seat_${FLEET_VAR_" + string(fleet.FleetVarHostPlatform) + "}",
	}
	certPayload = createCertificateAuthorityRequest{
		DigiCert: &caPlatform,
	}
	resPlatform := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &resPlatform)

	profilePlatform := digiCertForTest("N5", "I5", "PlatformVar")
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N5", Contents: profilePlatform},
	}}, http.StatusNoContent)
	profiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(profiles), 1)

	// trigger a profile sync
	s.awaitTriggerProfileSchedule(t)
	s.assertConfigProfilesByIdentifier(nil, "I5", true)

	// Check that status is pending (and not failed)
	getHostResp := getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	found := false
	for _, prof := range *getHostResp.Host.MDM.Profiles {
		if prof.Name == "N5" {
			found = true
			assert.EqualValues(t, fleet.MDMDeliveryPending, *prof.Status)
			assert.Equal(t, fleet.MDMOperationTypeInstall, prof.OperationType)
			assert.Empty(t, prof.Detail)
			continue
		}
	}
	assert.True(t, found)

	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting PKCS12 certificate for platform variable")
	require.Equal(t, "InstallProfile", cmd.Command.RequestType)
	fullCmd := micromdm.CommandPayload{}
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	require.NotNil(t, fullCmd.Command)
	require.NotNil(t, fullCmd.Command.InstallProfile)
	rawProfile := fullCmd.Command.InstallProfile.Payload
	if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
		p7, err := pkcs7.Parse(rawProfile)
		require.NoError(t, err)
		require.NoError(t, p7.Verify())
		rawProfile = p7.Content
	}

	type pkcs12Profile struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}

	pkcs12Prof := pkcs12Profile{}
	require.NoError(t, plist.Unmarshal(rawProfile, &pkcs12Prof))
	assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[0]["PayloadType"].(string))
	password := pkcs12Prof.PayloadContent[0]["Password"].(string)
	data := pkcs12Prof.PayloadContent[0]["PayloadContent"].([]byte)
	_, certificate, err := pkcs12.Decode(data, password)
	require.NoError(t, err)
	// darwin platform should be converted to "macos"
	assert.Equal(t, "macos-device", certificate.Subject.CommonName)

	// Check request to DigiCert for platform variable replacement
	digiCertServer.certReqMu.Lock()
	assert.Equal(t, caPlatform.ProfileID, digiCertServer.certReq.Profile["id"])
	assert.Equal(t, "seat_macos", digiCertServer.certReq.Seat["seat_id"])
	assert.Equal(t, "x509", digiCertServer.certReq.DeliveryFormat)
	assert.Equal(t, "macos-device", digiCertServer.certReq.Attributes["subject"].(map[string]any)["common_name"])

	// Need to convert UPNs to string slice since assert.Equal cannot compare string slice and interface slice
	upnsReceived := digiCertServer.certReq.Attributes["extensions"].(map[string]any)["san"].(map[string]any)["user_principal_names"].([]any)
	var stringSlice []string
	for _, item := range upnsReceived {
		if str, ok := item.(string); ok {
			stringSlice = append(stringSlice, str)
		}
	}
	assert.Equal(t, []string{"macos@example.com"}, stringSlice)
	digiCertServer.certReqMu.Unlock()

	// ////////////////////////////////
	// Modify the CN/UPN/Seat ID of the profile -- DigiCert verification should not happen
	digiCertServer.server.Close() // so that verify fails if attempted
}

//go:embed testdata/profiles/digicert.mobileconfig
var digiCertMobileconfig string

//go:embed testdata/profiles/digicert2.mobileconfig
var digiCert2Mobileconfig []byte

//go:embed testdata/profiles/ejbca.mobileconfig
var ejbcaMobileconfig string

func ejbcaForTest(name, identifier, caName string) []byte {
	return []byte(fmt.Sprintf(ejbcaMobileconfig, caName, caName, name, identifier, uuid.New().String()))
}

func digiCertForTest(name, identifier, caName string) []byte {
	return []byte(fmt.Sprintf(digiCertMobileconfig, caName, caName, name, identifier, uuid.New().String()))
}

//go:embed testdata/profiles/custom-scep.mobileconfig
var customSCEPMobileconfig string

//go:embed testdata/profiles/custom-scep2.mobileconfig
var customSCEPMobileconfig2 []byte

func customSCEPForTest(name, identifier, caName string) []byte {
	return []byte(fmt.Sprintf(customSCEPMobileconfig, caName, caName, name, identifier, uuid.New().String()))
}

func createMockDigiCertServer(t *testing.T) *mockDigiCertServer {
	profileRegex := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`)
	certRegex := regexp.MustCompile(`^/mpki/api/v1/certificate`)
	mockServer := &mockDigiCertServer{
		certReqMu: &sync.Mutex{},
		notAfter:  time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Second),
	}
	mockServer.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			matches := profileRegex.FindStringSubmatch(r.URL.Path)
			if len(matches) != 2 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			profileID := matches[1]

			resp := map[string]string{
				"id":     profileID,
				"name":   "Test CA",
				"status": "Active",
			}
			err := json.NewEncoder(w).Encode(resp)
			assert.NoError(t, err)
		case http.MethodPost:
			if len(certRegex.FindStringSubmatch(r.URL.Path)) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			mockServer.certReqMu.Lock()
			defer mockServer.certReqMu.Unlock()
			var req certificateReq
			err := json.NewDecoder(r.Body).Decode(&req)
			assert.NoError(t, err)
			mockServer.certReq = req

			// Decode the PEM format CSR into DER
			block, _ := pem.Decode([]byte(req.CSR))
			if !assert.NotNil(t, block, "failed to decode PEM block containing CSR") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.Equal(t, "CERTIFICATE REQUEST", block.Type, "unexpected PEM block type")

			// Parse the CSR
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			if !assert.NoError(t, err) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			assert.NoError(t, csr.CheckSignature())

			// Setting CertificateCommonName to "Fail" allows us to test a failure getting a DigiCert certificate
			if csr.Subject.CommonName == "Fail" {
				w.WriteHeader(http.StatusBadRequest)
				type errorResponse struct {
					Errors any `json:"errors"`
				}
				err = json.NewEncoder(w).Encode(errorResponse{
					Errors: []struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					}{
						{
							Message: "Expected Fail",
						},
					},
				})
				assert.NoError(t, err)
				return
			}

			// Generate a self-signed certificate based on the CSR
			serialNumber := big.NewInt(time.Now().Unix())
			certTemplate := &x509.Certificate{
				SerialNumber: serialNumber,
				Subject:      csr.Subject,
				NotBefore:    time.Now(),
				NotAfter:     mockServer.notAfter,
				KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
				ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			}

			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)

			certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, &privateKey.PublicKey, privateKey)
			assert.NoError(t, err)

			// Encode the certificate to PEM
			certPEM := new(bytes.Buffer)
			err = pem.Encode(certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
			assert.NoError(t, err)

			w.WriteHeader(http.StatusCreated)
			resp := map[string]string{
				"serial_number":   serialNumber.String(),
				"delivery_format": "x509",
				"certificate":     certPEM.String(),
			}
			err = json.NewEncoder(w).Encode(resp)
			assert.NoError(t, err)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(mockServer.server.Close)
	return mockServer
}

func (s *integrationMDMTestSuite) TestCustomSCEPConfig() {
	t := s.T()
	ctx := context.Background()
	scepServer := scep_server.StartTestSCEPServer(t)
	scepServerURL := scepServer.URL + "/scep"

	// TODO(hca): Update test to use apply group CA endpoint?

	caBad := newMockCustomSCEPProxyCA(scepServer.URL, "ca")
	certPayload := createCertificateAuthorityRequest{
		CustomSCEPProxy: &caBad,
	}
	res := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusBadRequest, &res)

	// Add 3 CustomSCEPProxy integrations
	ca0 := newMockCustomSCEPProxyCA(scepServerURL, "ca0")
	ca0.Challenge = "challenge0"
	ca1 := newMockCustomSCEPProxyCA(scepServerURL, "ca1")
	ca1.Challenge = "challenge1"
	ca2 := newMockCustomSCEPProxyCA(scepServerURL, "ca2")
	ca2.Challenge = "challenge2"
	certPayload.CertificateAuthorityPayload.CustomSCEPProxy = &ca0
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)
	certPayload.CertificateAuthorityPayload.CustomSCEPProxy = &ca1
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)
	certPayload.CertificateAuthorityPayload.CustomSCEPProxy = &ca2
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res)

	listRes := listCertificateAuthoritiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/certificate_authorities", &listCertificateAuthoritiesRequest{}, http.StatusOK, &listRes)

	customScepProxies := []*fleet.CertificateAuthority{}
	customScepProxiesWithSecrets := []*fleet.CertificateAuthority{}
	for _, ca := range listRes.CertificateAuthorities {
		if ca.Type == string(fleet.CATypeCustomSCEPProxy) {
			getRes := getCertificateAuthorityResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), &getCertificateAuthorityRequest{}, http.StatusOK, &getRes)
			customScepProxies = append(customScepProxies, getRes.CertificateAuthority)

			caWithSecret, err := s.ds.GetCertificateAuthorityByID(ctx, ca.ID, true)
			require.NoError(t, err)
			customScepProxiesWithSecrets = append(customScepProxiesWithSecrets, caWithSecret)
		} else {
			t.Fatalf("found ca that is not a scep proxy")
		}
	}
	assert.Len(t, customScepProxies, 3)

	for _, ca := range customScepProxies {
		assert.Equal(t, fleet.MaskedPassword, *ca.Challenge)
	}

	require.Len(t, customScepProxiesWithSecrets, 3)
	assert.Equal(t, "challenge0", *customScepProxiesWithSecrets[0].Challenge)
	assert.Equal(t, "challenge1", *customScepProxiesWithSecrets[1].Challenge)
	assert.Equal(t, "challenge2", *customScepProxiesWithSecrets[2].Challenge)

	// Check 3 added activities are present
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
	require.NotEmpty(t, listActivities.Activities)
	activity := fleet.ActivityAddedCustomSCEPProxy{}
	caNames := make([]string, 0, 3)
	for _, act := range listActivities.Activities {
		if act.Type == activity.ActivityName() {
			err := json.Unmarshal(*act.Details, &activity)
			require.NoError(t, err)
			caNames = append(caNames, activity.Name)
			if len(caNames) == 3 {
				break
			}
		}
	}
	slices.Sort(caNames)
	assert.Equal(t, []string{"ca0", "ca1", "ca2"}, caNames)

	// TODO(HCA): Re enable and switch out with endpoints once update endpoint is implemented.
	/* // Add 1, modify 1, delete 1, keep 1 the same (CustomSCEPProxy integrations)
	ca1.Challenge = "challenge1-modified"
	ca3 := newMockCustomSCEPProxyCA(scepServerURL, "ca3")
	ca3.Challenge = "challenge3"
	appConfig = map[string]interface{}{
		"integrations": map[string]interface{}{
			"custom_scep_proxy": []fleet.CustomSCEPProxyCA{ca3, ca2, ca1},
		},
	}
	raw, err = json.Marshal(appConfig)
	require.NoError(t, err)
	req.RawMessage = raw
	res = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", &req, http.StatusOK, &res)
	require.Len(t, res.Integrations.CustomSCEPProxy.Value, 3)
	assert.NotEqual(t, res.Integrations.CustomSCEPProxy.Value[0].Name, res.Integrations.CustomSCEPProxy.Value[1].Name)
	assert.NotEqual(t, res.Integrations.CustomSCEPProxy.Value[1].Name, res.Integrations.CustomSCEPProxy.Value[2].Name)
	for _, ca := range res.Integrations.CustomSCEPProxy.Value {
		switch ca.Name {
		case "ca1":
			assert.True(t, ca.Equals(&ca1))
		case "ca2":
			assert.True(t, ca.Equals(&ca2))
		case "ca3":
			assert.True(t, ca.Equals(&ca3))
		default:
			t.Fatalf("unexpected ca name: %s", ca.Name)
		}
		assert.Equal(t, ca.Challenge, fleet.MaskedPassword)
	}
	assets, err = s.ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	require.NoError(t, err)
	require.Len(t, assets, 3)
	assert.EqualValues(t, "challenge1-modified", assets["ca1"].Value)
	assert.EqualValues(t, "challenge2", assets["ca2"].Value)
	assert.EqualValues(t, "challenge3", assets["ca3"].Value)

	listActivities = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
	require.True(t, len(listActivities.Activities) > 0)
	activity = fleet.ActivityAddedCustomSCEPProxy{}
	editActivity := fleet.ActivityEditedCustomSCEPProxy{}
	delActivity := fleet.ActivityDeletedCustomSCEPProxy{}
	var numFound int
	for _, act := range listActivities.Activities {
		switch act.Type {
		case activity.ActivityName():
			err := json.Unmarshal(*act.Details, &activity)
			require.NoError(t, err)
			assert.Equal(t, activity.Name, ca3.Name)
			numFound++
		case editActivity.ActivityName():
			err := json.Unmarshal(*act.Details, &editActivity)
			require.NoError(t, err)
			assert.Equal(t, editActivity.Name, ca1.Name)
			numFound++
		case delActivity.ActivityName():
			err := json.Unmarshal(*act.Details, &delActivity)
			require.NoError(t, err)
			assert.Equal(t, delActivity.Name, ca0.Name)
			numFound++
		}
		if numFound == 3 {
			break
		}
	}
	assert.Equal(t, 3, numFound) */

	// Clear CustomSCEPProxy integrations
	for _, ca := range customScepProxies {
		s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", ca.ID), nil, http.StatusNoContent)
	}

	// TODO(HCA): Re enable and switch out with endpoints once update endpoint is implemented.
	/*
		// Check that 3 deleted activities are present
		listActivities = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
			&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
		require.True(t, len(listActivities.Activities) > 0)
		delActivity = fleet.ActivityDeletedCustomSCEPProxy{}
		caNames = make([]string, 0, 3)
		for _, act := range listActivities.Activities {
			if act.Type == delActivity.ActivityName() {
				err := json.Unmarshal(*act.Details, &delActivity)
				require.NoError(t, err)
				caNames = append(caNames, delActivity.Name)
				if len(caNames) == 3 {
					break
				}
			}
		}
		slices.Sort(caNames)
		assert.EqualValues(t, caNames, []string{"ca1", "ca2", "ca3"})
	*/
}

func (s *integrationMDMTestSuite) TestCustomSCEPIntegration() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)
	scepServer := scep_server.StartTestSCEPServer(t)
	scepServerURL := scepServer.URL + "/scep"

	// parseSCEPProfile returns the parsed SCEP profile along with the identifier from the profile URL
	parseSCEPProfile := func(t *testing.T, raw []byte, scepConfig fleet.CustomSCEPProxyCA) (SCEPProfileContent, string) {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(raw, &fullCmd))
		require.NotNil(t, fullCmd.Command)
		require.NotNil(t, fullCmd.Command.InstallProfile)
		rawProfile := fullCmd.Command.InstallProfile.Payload
		if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
			p7, err := pkcs7.Parse(rawProfile)
			require.NoError(t, err)
			require.NoError(t, p7.Verify())
			rawProfile = p7.Content
		}
		var scepProfile SCEPProfileContent
		require.NoError(t, plist.Unmarshal(rawProfile, &scepProfile))

		require.Equal(t, "com.apple.security.scep", scepProfile.PayloadContent[0].PayloadType)
		require.Equal(t, scepConfig.Challenge, scepProfile.PayloadContent[0].PayloadContent.Challenge)
		expectBaseURL := s.server.URL + apple_mdm.SCEPProxyPath
		require.True(t, strings.HasPrefix(scepProfile.PayloadContent[0].PayloadContent.URL, expectBaseURL))
		identifier := strings.TrimPrefix(scepProfile.PayloadContent[0].PayloadContent.URL, expectBaseURL)

		return scepProfile, identifier
	}

	// parseIdentifier is a helper function to parse the identifier from the SCEP profile and check
	// components of the identifier against expected values. It returns the SCEP challenge.
	parseIdentifier := func(t *testing.T, identifier, wantHostUUID, wantProfUUID, wantSCEPName string) string {
		parts := strings.Split(identifier, url.PathEscape(","))
		require.Len(t, parts, 4)
		require.Equal(t, wantHostUUID, parts[0])
		require.Equal(t, wantProfUUID, parts[1])
		require.Equal(t, wantSCEPName, parts[2])
		gotChallenge := parts[3]
		require.NotEmpty(t, gotChallenge, "Challenge should not be empty in the identifier")
		return gotChallenge
	}

	// checkChallenge is a helper function to check if the challenge exists in the database.
	checkChallenge := func(t *testing.T, challenge string, expectFound bool) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var foundChallenge string
			stmt := "SELECT challenge FROM challenges where challenge = ?"
			err := sqlx.GetContext(context.Background(), q, &foundChallenge, stmt, challenge)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				require.False(t, expectFound)
				require.Empty(t, foundChallenge)
				return nil
			default:
				require.NoError(t, err, "Failed to check challenge existence in the database")
				require.True(t, expectFound, "found challenge %s, but expected not to find it", foundChallenge)
				require.NotEmpty(t, foundChallenge, "challenge should not be empty in the database")
				return nil
			}
		})
	}

	// checkProfileInstallStatus is a helper function to check the status of a profile in the host
	// response. It asserts that the profile with the given name exists, checks its status,
	// operation type, and detail. It returns the profile UUID for further checks.
	checkProfileInstallStatus := func(t *testing.T, hostResp getDeviceHostResponse, profileName string, wantStatus fleet.MDMDeliveryStatus, wantDetail string) string {
		var found bool
		require.NotNil(t, hostResp.Host.MDM.Profiles)
		var profileUUID string
		for _, prof := range *hostResp.Host.MDM.Profiles {
			if prof.Name == profileName {
				found = true
				require.EqualValues(t, wantStatus, *prof.Status)
				require.Equal(t, fleet.MDMOperationTypeInstall, prof.OperationType)
				require.Contains(t, prof.Detail, wantDetail)
				profileUUID = prof.ProfileUUID
				break
			}
		}
		require.True(t, found)
		return profileUUID
	}

	// verifySCEPProfile is a helper function to verify the SCEP profile in the host profiles.
	verifySCEPProfile := func(t *testing.T, host *fleet.Host, hostProf fleet.HostMacOSProfile, wantProfUUID string, wantCAName string) {
		hostProfs := map[string]*fleet.HostMacOSProfile{
			hostProf.Identifier: &hostProf,
		}
		require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, hostProfs))
		prof, err := s.ds.GetAppleHostMDMCertificateProfile(context.Background(), host.UUID, wantProfUUID, wantCAName)
		require.NoError(t, err)
		require.NotNil(t, prof)
		require.Equal(t, wantCAName, prof.CAName)
		require.Equal(t, fleet.CAConfigCustomSCEPProxy, prof.Type)
		require.Equal(t, fleet.MDMDeliveryVerified, *prof.Status)
	}

	s.awaitTriggerProfileSchedule(t)

	// Create a host and then enroll to MDM.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// trigger a profile sync
	s.awaitRunAppleMDMWorkerSchedule()
	// Receive enrollment profiles (we are not checking/testing these here)
	for {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		if cmd == nil {
			break
		}
		_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	err := s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	// /////////////////////////////////////////
	// Upload a profile without defining the CA
	resp := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N0", Contents: customSCEPForTest("N0", "I0", "scepName")},
	}}, http.StatusBadRequest)
	errMsg := extractServerErrorText(resp.Body)
	assert.Contains(t, errMsg, "FLEET_VAR_CUSTOM_SCEP_")
	assert.Contains(t, errMsg, "_scepName does not exist")

	// /////////////////////////////////////////
	// Add custom SCEP config
	ca0 := newMockCustomSCEPProxyCA(scepServerURL, "scepName")
	ca1 := newMockCustomSCEPProxyCA(scepServerURL, "scepName2")
	t.Logf("scepName2 challenge:%s", ca1.Challenge)
	certPayload := createCertificateAuthorityRequest{
		CustomSCEPProxy: &ca0,
	}
	res1 := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res1)
	certPayload.CertificateAuthorityPayload.CustomSCEPProxy = &ca1
	res2 := createCertificateAuthorityResponse{}
	s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", &certPayload, http.StatusOK, &res2)

	// //////////////////////////////////////////////
	// Upload a profile with two valid SCEP payloads.
	resp = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "CustomSCEP2", Contents: customSCEPMobileconfig2},
	}}, http.StatusBadRequest)
	errMsg = extractServerErrorText(resp.Body)
	assert.Contains(t, errMsg, fleet.MultipleSCEPPayloadsErrMsg)

	// /////////////////////////////////////////
	// Happy path: upload the profile containing a single valid SCEP config
	profileContents := customSCEPForTest("N0", "I0", "scepName")
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N0", Contents: profileContents},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	getHostResp := getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUID := checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryPending, "")

	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting SCEP profile")

	_, identifier := parseSCEPProfile(t, cmd.Raw, ca0)
	gotChallenge := parseIdentifier(t, identifier, host.UUID, profileUUID, "scepName")
	checkChallenge(t, gotChallenge, true)

	// /////////////////////////////////////
	// Test SCEP traffic being sent by host

	// Invalid profile identifier
	scepRes := s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"invalid_identifier,p1234-uuid", nil, http.StatusBadRequest, nil, "operation", "GetCACaps")
	scepResErr := extractServerErrorText(scepRes.Body)
	require.Contains(t, scepResErr, "invalid profile UUID (only Apple, Windows, and Android")

	// Verify Windows profiles is allowed (with dummy values that will fail lookup)
	scepRes = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+"invalid_identifier,w1234-uuid", nil, http.StatusBadRequest, nil, "operation", "GetCACaps")
	scepResErr = extractServerErrorText(scepRes.Body)
	require.Contains(t, scepResErr, "unknown identifier in URL path") // Both invalid host and profile, but it allows it to lookup

	// GetCACaps
	scepRes = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation", "GetCACaps")
	body, err := io.ReadAll(scepRes.Body)
	require.NoError(t, err)
	assert.Equal(t, scepserver.DefaultCACaps, string(body))
	// Check that the challenge is still in the database (only deleted after PKIOperation)
	checkChallenge(t, gotChallenge, true)

	// GetCACert
	scepRes = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation", "GetCACert")
	body, err = io.ReadAll(scepRes.Body)
	require.NoError(t, err)
	certs, err := x509.ParseCertificates(body)
	require.NoError(t, err)
	assert.Len(t, certs, 1)
	// Check that the challenge is still in the database (only deleted after PKIOperation)
	checkChallenge(t, gotChallenge, true)

	// PKIOperation
	data, err := os.ReadFile("./testdata/PKCSReq.der")
	require.NoError(t, err)
	message := base64.StdEncoding.EncodeToString(data)

	// Note, a cert is not returned here because the message is not a fully valid SCEP request. However, building a valid SCEP request is a bit involved,
	// and this request is sufficient for testing our proxy functionality.
	scepRes = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(scepRes.Body)
	require.NoError(t, err)
	pkiMessage, err := scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)
	// Check that the challenge is removed from the database after PKIOperation
	checkChallenge(t, gotChallenge, false)

	// Host acknowledges the profile and it is marked as verifying
	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)

	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUIDTest := checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryVerifying, "")
	require.Equal(t, profileUUID, profileUUIDTest, "Expected the same profile UUID after re-sending the SCEP profile")

	// Try again, it should fail because the profile is not pending
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	// Failed challenge doesn't resend the SCEP profile unless it is pending
	s.awaitTriggerProfileSchedule(t)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd, "Not expecting SCEP profile")

	// Mark the profile as pending so that it can be resent
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := "UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ? AND profile_uuid = ?"
		r, err := q.ExecContext(context.Background(), stmt, nil, host.UUID, profileUUID)
		require.NoError(t, err, "Failed to update profile status in the database")
		rowsAffected, _ := r.RowsAffected()
		require.Equal(t, int64(1), rowsAffected, "Expected to update 1 row for the profile status")
		return nil
	})

	// Try to do SCEP with deleted challenge, it will fail because challenge is no longer in the datastore
	// but a new challenge will be generated and the profile resent
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	// Check profile status, raw status will be nil (awaiting the reconcile job)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '') FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Empty(t, status, "Expected profile status to be nil after re-sending the SCEP profile")
		return nil
	})
	// Reconcile profiles, raw status becomes pending
	s.awaitTriggerProfileSchedule(t)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '') FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.MDMDeliveryPending), status, "Expected profile status to be pending after re-sending the SCEP profile")
		return nil
	})

	// Device checks in again and should receive the SCEP profile
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting SCEP profile")
	// Status should still be pending because we haven't acknowledged the profile command yet
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '')  FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.MDMDeliveryPending), status, "Expected profile status to be pending after re-sending the SCEP profile")
		return nil
	})

	// Identifier should be different, and the challenge should be different
	_, identifier2 := parseSCEPProfile(t, cmd.Raw, ca0)
	require.NotEqual(t, identifier, identifier2, "Expected a different identifier after re-sending the SCEP profile")
	gotChallenge2 := parseIdentifier(t, identifier2, host.UUID, profileUUID, "scepName")
	require.NotEqual(t, gotChallenge, gotChallenge2, "Expected a new challenge after re-sending the SCEP profile")
	checkChallenge(t, gotChallenge, false) // old challenge deleted
	checkChallenge(t, gotChallenge2, true) // new challenge added

	// Check the host details
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUID2 := checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryPending, "")
	require.Equal(t, profileUUID, profileUUID2, "Expected the same profile UUID after re-sending the SCEP profile")

	// Expire the challenge so that the next PKIOperation fails
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := "UPDATE challenges SET created_at = ? WHERE challenge = ?"
		res, err := q.ExecContext(context.Background(), stmt, time.Now().Add(-2*time.Hour), gotChallenge2)
		require.NoError(t, err, "Failed to expire the challenge in the database")
		rowsAffected, _ := res.RowsAffected()
		require.Equal(t, int64(1), rowsAffected, "Expected to update 1 row for the challenge")
		return nil
	})

	// Do SCEP again, it should fail because the challenge is expired but a new challenge should be
	// generated and the profile resent
	_ = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier2, nil, http.StatusBadRequest, nil, "operation",
		"PKIOperation", "message", message)
	// Check profile status, raw status will be empty (awaiting the reconcile job)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '') FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Empty(t, status, "Expected profile status to be empty after re-sending the SCEP profile")
		return nil
	})

	// Check hosts details, derived status should still be pending
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUID2 = checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryPending, "")
	require.Equal(t, profileUUID, profileUUID2, "Expected the same profile UUID after re-sending the SCEP profile")

	// Reconcile profiles, raw status becomes pending
	s.awaitTriggerProfileSchedule(t)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '') FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.MDMDeliveryPending), status, "Expected profile status to be pending after re-sending the SCEP profile")
		return nil
	})

	// Device checks in again and should receive the SCEP profile
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd, "Expecting SCEP profile")
	// Status should still be pending because we haven't acknowledged the profile command yet
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status string
		stmt := "SELECT coalesce(status, '') FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?"
		err := sqlx.GetContext(context.Background(), q, &status, stmt, host.UUID, profileUUID)
		require.NoError(t, err)
		require.Equal(t, string(fleet.MDMDeliveryPending), status, "Expected profile status to be pending after re-sending the SCEP profile")
		return nil
	})

	// Identifier should be different, and the challenge should be different
	_, identifier3 := parseSCEPProfile(t, cmd.Raw, ca0)
	require.NotEqual(t, identifier2, identifier3, "Expected a different identifier after re-sending the SCEP profile")
	gotChallenge3 := parseIdentifier(t, identifier3, host.UUID, profileUUID, "scepName")
	require.NotEqual(t, gotChallenge2, gotChallenge3, "Expected a new challenge after re-sending the SCEP profile")
	checkChallenge(t, gotChallenge, false)  // old challenge deleted
	checkChallenge(t, gotChallenge2, false) // old challenge deleted
	checkChallenge(t, gotChallenge3, true)  // new challenge added

	scepRes = s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier3, nil, http.StatusOK, nil, "operation",
		"PKIOperation", "message", message)
	body, err = io.ReadAll(scepRes.Body)
	require.NoError(t, err)
	pkiMessage, err = scep.ParsePKIMessage(body, scep.WithCACerts(certs))
	require.NoError(t, err)
	assert.Equal(t, scep.CertRep, pkiMessage.MessageType)

	cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)
	assert.Nil(t, cmd)
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUID3 := checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryVerifying, "")
	require.Equal(t, profileUUID2, profileUUID3, "Expected the same profile UUID after acknowledging the SCEP profile")

	verifySCEPProfile(t, host, fleet.HostMacOSProfile{
		Identifier:  "I0",
		DisplayName: "N0",
		InstallDate: time.Now(),
	}, profileUUID, "scepName")

	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	profileUUID4 := checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryVerified, "")
	require.Equal(t, profileUUID3, profileUUID4, "Expected the same profile UUID after verifying the SCEP profile")

	// No more commands pending
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// ////////////////////////////////////////////
	// Remove the CAs and try to re-send the profile
	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", res1.ID), nil, http.StatusNoContent)
	s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", res2.ID), nil, http.StatusNoContent)
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", host.ID, profileUUID), nil, http.StatusAccepted)
	s.awaitTriggerProfileSchedule(t)
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	checkProfileInstallStatus(t, getHostResp, "N0", fleet.MDMDeliveryFailed,
		"scepName certificate authority doesn't exist")
}

// createTestServerWithStatusCode creates a local HTTP test server that always returns the specified status code.
// The server should be closed after use by calling the returned cleanup function.
func createTestServerWithStatusCode(statusCode int) (*httptest.Server, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
	}))

	cleanup := func() {
		server.Close()
	}

	return server, cleanup
}

func (s *integrationMDMTestSuite) TestEJBCAIntegration() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)
	fake := ejbcatest.New(t, nil)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM certificate_authorities")
		return err
	})

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup = fleet.MacOSSetup{EnableEndUserAuthentication: false}
	require.NoError(t, s.ds.SaveAppConfig(ctx, appConfig))

	const p12Password = "p12-password"
	createCA := func(fake *ejbcatest.Server, name, endEntityProfile string) uint {
		var res createCertificateAuthorityResponse
		s.DoJSON("POST", "/api/latest/fleet/certificate_authorities", map[string]any{"ejbca": map[string]any{
			"name":                             name,
			"url":                              fake.HTTP.URL,
			"client_p12":                       fake.ClientP12(t, time.Now().Add(365*24*time.Hour), p12Password),
			"password":                         p12Password,
			"trust_ca_bundle":                  fake.ServerPEM,
			"certificate_authority_name":       ejbcatest.CAName,
			"certificate_profile_name":         ejbcatest.CertProfile,
			"end_entity_profile_name":          endEntityProfile,
			"certificate_common_name":          "$FLEET_VAR_HOST_HARDWARE_SERIAL cn",
			"username_template":                "$FLEET_VAR_HOST_UUID",
			"certificate_user_principal_names": []string{"upn@example.com"},
		}}, http.StatusOK, &res)
		return res.ID
	}
	caID := createCA(fake, "ejbca_CA", ejbcatest.EndEntityProfile)

	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	// Receive the enrollment profiles, which this test doesn't check.
	for {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		if cmd == nil {
			break
		}
		_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID))

	// installedCertificate receives the next profile the host is sent, and returns the certificate in its PKCS#12 payload.
	installedCertificate := func() *x509.Certificate {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expecting the PKCS#12 profile")
		require.Equal(t, "InstallProfile", cmd.Command.RequestType)
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		rawProfile := fullCmd.Command.InstallProfile.Payload
		if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
			p7, err := pkcs7.Parse(rawProfile)
			require.NoError(t, err)
			require.NoError(t, p7.Verify())
			rawProfile = p7.Content
		}
		var pkcs12Prof struct {
			PayloadContent []map[string]any `plist:"PayloadContent"`
		}
		require.NoError(t, plist.Unmarshal(rawProfile, &pkcs12Prof))
		require.Len(t, pkcs12Prof.PayloadContent, 1)
		assert.Equal(t, "com.apple.security.pkcs12", pkcs12Prof.PayloadContent[0]["PayloadType"])
		_, certificate, err := pkcs12.Decode(pkcs12Prof.PayloadContent[0]["PayloadContent"].([]byte), pkcs12Prof.PayloadContent[0]["Password"].(string))
		require.NoError(t, err)
		return certificate
	}
	setProfiles := func(profiles ...fleet.MDMProfileBatchPayload) {
		s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles}, http.StatusNoContent)
	}

	// A profile that uses a CA that doesn't exist is rejected.
	rawRes := s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: ejbcaForTest("N1", "I1", "badName")},
	}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(rawRes.Body), "_badName does not exist")
	s.assertConfigProfilesByIdentifier(nil, "I1", false)

	// The host gets a certificate, with the common name and UPN of the CA filled in, and under the username of the CA.
	profile := ejbcaForTest("N2", "I2", "ejbca_CA")
	setProfiles(fleet.MDMProfileBatchPayload{Name: "N2", Contents: profile})
	s.awaitTriggerProfileSchedule(t)
	p := s.assertConfigProfilesByIdentifier(nil, "I2", true)
	certificate := installedCertificate()
	assert.Equal(t, host.HardwareSerial+" cn", certificate.Subject.CommonName)
	assert.Equal(t, []string{"upn@example.com"}, certissue.UPNs(certificate))
	_, enrolls := fake.Snapshot()
	require.Len(t, enrolls, 1)
	assert.Equal(t, host.UUID, enrolls[0]["username"])

	managed, err := s.ds.GetAppleHostMDMCertificateProfile(ctx, host.UUID, p.ProfileUUID, "ejbca_CA")
	require.NoError(t, err)
	require.NotNil(t, managed)
	assert.Equal(t, fleet.CAConfigEJBCA, managed.Type)
	assert.Equal(t, fleet.MDMDeliveryVerifying, *managed.Status)
	require.NotNil(t, managed.NotValidAfter)
	assert.True(t, managed.NotValidAfter.Equal(certificate.NotAfter))
	require.NotNil(t, managed.Serial)
	assert.Equal(t, fmt.Sprintf("%040s", certificate.SerialNumber.Text(16)), *managed.Serial)

	// Changing the connection settings doesn't issue new certificates.
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", caID),
		map[string]any{"ejbca": map[string]any{"trust_ca_bundle": fake.ServerPEM}}, http.StatusOK)
	s.awaitTriggerProfileSchedule(t)
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd, "a connection change doesn't resend the profile")

	// Changing a certificate field does, and the host gets a new certificate.
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/certificate_authorities/%d", caID),
		map[string]any{"ejbca": map[string]any{"certificate_common_name": "new $FLEET_VAR_HOST_HARDWARE_SERIAL"}}, http.StatusOK)
	s.awaitTriggerProfileSchedule(t)
	certificate = installedCertificate()
	assert.Equal(t, "new "+host.HardwareSerial, certificate.Subject.CommonName)
	_, enrolls = fake.Snapshot()
	assert.Len(t, enrolls, 2)

	// An enrollment EJBCA rejects fails the profile with the reason.
	createCA(fake, "fail_CA", "noSuchEndEntityProfile")
	setProfiles(
		fleet.MDMProfileBatchPayload{Name: "N2", Contents: profile},
		fleet.MDMProfileBatchPayload{Name: "N3", Contents: ejbcaForTest("N3", "I3", "fail_CA")},
	)
	s.awaitTriggerProfileSchedule(t)
	getHostResp := getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	var foundFailed bool
	for _, prof := range *getHostResp.Host.MDM.Profiles {
		if prof.Name == "N3" {
			foundFailed = true
			assert.EqualValues(t, fleet.MDMDeliveryFailed, *prof.Status)
			assert.Contains(t, prof.Detail, "Couldn't get certificate from EJBCA for fail_CA.")
			assert.Contains(t, prof.Detail, `End Entity Profile of name "noSuchEndEntityProfile" was not found`)
		}
	}
	assert.True(t, foundFailed)

	// An EJBCA that is temporarily unavailable keeps the profile queued for the retries a profile gets, and then fails it.
	unavailable := ejbcatest.New(t, func(f *ejbcatest.Server) { f.EnrollStatus = http.StatusServiceUnavailable })
	createCA(unavailable, "down_CA", ejbcatest.EndEntityProfile)
	setProfiles(
		fleet.MDMProfileBatchPayload{Name: "N2", Contents: profile},
		fleet.MDMProfileBatchPayload{Name: "N3", Contents: ejbcaForTest("N3", "I3", "fail_CA")},
		fleet.MDMProfileBatchPayload{Name: "N4", Contents: ejbcaForTest("N4", "I4", "down_CA")},
	)
	hostProfile := func(name string) fleet.HostMDMProfile {
		var resp getDeviceHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &resp)
		for _, prof := range *resp.Host.MDM.Profiles {
			if prof.Name == name {
				return prof
			}
		}
		require.Failf(t, "profile not found", "profile %s", name)
		return fleet.HostMDMProfile{}
	}
	statusOf := func(prof fleet.HostMDMProfile) string {
		if prof.Status == nil {
			return ""
		}
		return *prof.Status
	}
	retries := func() uint {
		counts, err := s.ds.GetHostMDMProfilesRetryCounts(ctx, &fleet.Host{UUID: host.UUID, Platform: "darwin"})
		require.NoError(t, err)
		for _, count := range counts {
			if count.ProfileIdentifier == "I4" {
				return count.Retries
			}
		}
		require.Fail(t, "no retry count for the profile")
		return 0
	}
	for attempt := uint(1); attempt <= mdm.MaxAppleProfileRetries; attempt++ {
		s.awaitTriggerProfileSchedule(t)
		assert.Equal(t, attempt, retries())
		assert.NotEqual(t, string(fleet.MDMDeliveryFailed), statusOf(hostProfile("N4")), "attempt %d", attempt)
	}
	s.awaitTriggerProfileSchedule(t)
	failedProfile := hostProfile("N4")
	assert.Equal(t, string(fleet.MDMDeliveryFailed), statusOf(failedProfile))
	assert.Contains(t, failedProfile.Detail, "Couldn't get certificate from EJBCA for down_CA.")
	var enrollAttempts int
	for _, request := range unavailable.Requests() {
		if strings.HasPrefix(request, "POST ") && strings.HasSuffix(request, "/v1/certificate/pkcs10enroll") {
			enrollAttempts++
		}
	}
	assert.Equal(t, int(mdm.MaxAppleProfileRetries)+1, enrollAttempts)
}
