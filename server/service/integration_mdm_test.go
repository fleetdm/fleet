package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	mathrand "math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/tokenpki"
	scepclient "github.com/micromdm/scep/v2/client"
	"github.com/micromdm/scep/v2/cryptoutil/x509util"
	"github.com/micromdm/scep/v2/scep"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.mozilla.org/pkcs7"
)

func TestIntegrationsMDM(t *testing.T) {
	testingSuite := new(integrationMDMTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationMDMTestSuite struct {
	withServer
	suite.Suite
	fleetCfg config.FleetConfig
}

func (s *integrationMDMTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationMDMTestSuite")

	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(s.T(), err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)

	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, testCertPEM, testKeyPEM, testBMToken)

	mdmStorage, err := s.ds.NewMDMAppleMDMStorage(testCertPEM, testKeyPEM)
	require.NoError(s.T(), err)
	depStorage, err := s.ds.NewMDMAppleDEPStorage(*testBMToken)
	require.NoError(s.T(), err)
	scepStorage, err := s.ds.NewMDMAppleSCEPDepot(testCertPEM, testKeyPEM)
	require.NoError(s.T(), err)

	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		FleetConfig: &fleetCfg,
		MDMStorage:  mdmStorage,
		DEPStorage:  depStorage,
		SCEPStorage: scepStorage,
		MDMPusher:   dummyMDMPusher{},
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedAdminToken = s.token
	s.fleetCfg = fleetCfg
}

func (s *integrationMDMTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

func (s *integrationMDMTestSuite) TestAppleGetAppleMDM() {
	t := s.T()

	var mdmResp getAppleMDMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple", nil, http.StatusOK, &mdmResp)
	// returned values are dummy, this is a test certificate
	require.Equal(t, "FleetDM", mdmResp.Issuer)
	require.NotZero(t, mdmResp.SerialNumber)
	require.Equal(t, "FleetDM", mdmResp.CommonName)
	require.NotZero(t, mdmResp.RenewDate)

	// GET /api/latest/fleet/mdm/apple_bm is not tested because it makes a call
	// to an Apple API that would a) fail because we use dummy token/certs and b)
	// could get us in trouble with many invalid requests.
	// TODO: eventually add a way to mock the apple API, maybe with a test http
	// server running and a way to use its URL instead of Apple's. (#8948)
}

func (s *integrationMDMTestSuite) TestDeviceMDMManualEnroll() {
	t := s.T()

	token := "token_test_manual_enroll"
	createHostAndDeviceToken(t, s.ds, token)

	// invalid token fails
	s.DoRaw("GET", "/api/latest/fleet/device/invalid_token/mdm/apple/manual_enrollment_profile", nil, http.StatusUnauthorized)

	// valid token downloads the profile
	resp := s.DoRaw("GET", "/api/latest/fleet/device/"+token+"/mdm/apple/manual_enrollment_profile", nil, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	require.Contains(t, resp.Header, "Content-Disposition")
	require.Contains(t, resp.Header, "Content-Type")
	require.Contains(t, resp.Header, "X-Content-Type-Options")
	require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
	require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")
	headerLen, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	require.NoError(t, err)
	require.Equal(t, len(body), headerLen)

	var profile struct {
		PayloadIdentifier string `plist:"PayloadIdentifier"`
	}
	require.NoError(t, plist.Unmarshal(body, &profile))
	require.Equal(t, apple_mdm.FleetPayloadIdentifier, profile.PayloadIdentifier)
}

func (s *integrationMDMTestSuite) TestDeviceEnrollment() {
	t := s.T()

	// Enroll two devices into MDM
	deviceA := newMDMEnrolledDevice(s)
	deviceB := newMDMEnrolledDevice(s)

	// Find the ID of Fleet's MDM solution
	var mdmID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ?`, fleet.WellKnownMDMFleet)
	})

	// Check that both devices are returned by the /hosts endpoint
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, listHostsRes.Hosts, 2)
	require.EqualValues(
		t,
		[]string{deviceA.uuid, deviceB.uuid},
		[]string{listHostsRes.Hosts[0].UUID, listHostsRes.Hosts[1].UUID},
	)

	var targetHostID uint
	var lastEnroll time.Time
	for _, host := range listHostsRes.Hosts {
		if host.UUID == deviceA.uuid {
			targetHostID = host.ID
			lastEnroll = host.LastEnrolledAt
			break
		}
	}

	// Activities are generated for each device
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities, "order_key", "created_at")
	require.GreaterOrEqual(t, len(activities.Activities), 2)

	details := []*json.RawMessage{}
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_enrolled" {
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			details = append(details, activity.Details)
		}
	}
	require.Len(t, details, 2)
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceA.serial, deviceA.model, deviceA.serial), string(*details[0]))
	require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceB.serial, deviceB.model, deviceB.serial), string(*details[1]))

	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: deviceA.uuid,
	})
	require.NoError(t, err)
	var enrollResp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	// query all hosts
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// we still have only two hosts
	require.Len(t, listHostsRes.Hosts, 2)

	// LastEnrolledAt should have been updated
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", targetHostID), nil, http.StatusOK, &getHostResp)
	require.Greater(t, getHostResp.Host.LastEnrolledAt, lastEnroll)

	// Unenroll a device
	deviceA.checkout()

	// An activity is created
	activities = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_unenrolled" {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			details = append(details, activity.Details)
			require.JSONEq(t, fmt.Sprintf(`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false}`, deviceA.serial, deviceA.model, deviceA.serial), string(*activity.Details))
		}
	}
	require.True(t, found)
}

func (s *integrationMDMTestSuite) TestDeviceMultipleAuthMessages() {
	d := newMDMEnrolledDevice(s)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(s.T(), listHostsRes.Hosts, 1)

	// send the auth message again, we still have only one host
	d.authenticate()
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(s.T(), listHostsRes.Hosts, 1)
}

type device struct {
	uuid   string
	serial string
	model  string

	s        *integrationMDMTestSuite
	scepCert *x509.Certificate
	scepKey  *rsa.PrivateKey
}

func newDevice(s *integrationMDMTestSuite) *device {
	return &device{
		uuid:   strings.ToUpper(uuid.New().String()),
		serial: randSerial(),
		model:  "MacBookPro16,1",
		s:      s,
	}
}

func newMDMEnrolledDevice(s *integrationMDMTestSuite) *device {
	d := newDevice(s)
	d.scepEnroll()
	d.authenticate()
	return d
}

func (d *device) authenticate() {
	payload := map[string]string{
		"MessageType":  "Authenticate",
		"UDID":         d.uuid,
		"Model":        d.model,
		"DeviceName":   "testdevice" + d.serial,
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
		"SerialNumber": d.serial,
	}
	d.request("application/x-apple-aspen-mdm-checkin", payload)
}

func (d *device) checkout() {
	payload := map[string]string{
		"MessageType":  "CheckOut",
		"Topic":        "com.apple.mgmt.External." + d.uuid,
		"UDID":         d.uuid,
		"EnrollmentID": "testenrollmentid-" + d.uuid,
	}
	d.request("application/x-apple-aspen-mdm-checkin", payload)
}

func (d *device) request(reqType string, payload map[string]string) {
	body, err := plist.Marshal(payload)
	require.NoError(d.s.T(), err)

	signedData, err := pkcs7.NewSignedData(body)
	require.NoError(d.s.T(), err)
	err = signedData.AddSigner(d.scepCert, d.scepKey, pkcs7.SignerInfoConfig{})
	require.NoError(d.s.T(), err)
	sig, err := signedData.Finish()
	require.NoError(d.s.T(), err)

	d.s.DoRawWithHeaders(
		"POST",
		"/mdm/apple/mdm",
		body,
		200,
		map[string]string{
			"Content-Type":  reqType,
			"Mdm-Signature": base64.StdEncoding.EncodeToString(sig),
		},
	)
}

func (d *device) scepEnroll() {
	t := d.s.T()
	ctx := context.Background()
	logger := kitlog.NewJSONLogger(os.Stdout)
	logger = level.NewFilter(logger, level.AllowDebug())
	client, err := scepclient.New(d.s.server.URL+apple_mdm.SCEPPath, logger)
	require.NoError(t, err)

	resp, _, err := client.GetCACert(ctx, "")
	require.NoError(t, err)

	certs, err := x509.ParseCertificates(resp)
	require.NoError(t, err)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	csrTemplate := x509util.CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "fleet-test",
			},
			SignatureAlgorithm: x509.SHA256WithRSA,
		},
		ChallengePassword: d.s.fleetCfg.MDMApple.SCEP.Challenge,
	}
	csrDerBytes, err := x509util.CreateCertificateRequest(rand.Reader, &csrTemplate, key)
	require.NoError(t, err)
	csr, err := x509.ParseCertificateRequest(csrDerBytes)
	require.NoError(t, err)

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour)

	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "SCEP SIGNER",
			Organization: csr.Subject.Organization,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDerBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certDerBytes)
	require.NoError(t, err)

	tmpl := &scep.PKIMessage{
		MessageType: scep.PKCSReq,
		Recipients:  certs,
		SignerKey:   key,
		SignerCert:  cert,
		CSRReqMessage: &scep.CSRReqMessage{
			ChallengePassword: d.s.fleetCfg.MDMApple.SCEP.Challenge,
		},
	}

	msg, err := scep.NewCSRRequest(csr, tmpl, scep.WithLogger(logger))
	require.NoError(t, err)

	respBytes, err := client.PKIOperation(ctx, msg.Raw)
	require.NoError(t, err)

	respMsg, err := scep.ParsePKIMessage(respBytes, scep.WithLogger(logger), scep.WithCACerts(msg.Recipients))
	require.NoError(t, err)
	require.Equal(t, scep.SUCCESS, respMsg.PKIStatus)

	err = respMsg.DecryptPKIEnvelope(cert, key)
	require.NoError(t, err)

	d.scepCert = respMsg.CertRepMessage.Certificate
	d.scepKey = key
}

// numbers plus capital letters without I, L, O for readability
const serialLetters = "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"

func randSerial() string {
	b := make([]byte, 12)
	for i := range b {
		//nolint:gosec // not used for crypto, only to generate random serial for testing
		b[i] = serialLetters[mathrand.Intn(len(serialLetters))]
	}
	return string(b)
}

var testBMToken = &nanodep_client.OAuth1Tokens{
	ConsumerKey:       "test_consumer",
	ConsumerSecret:    "test_secret",
	AccessToken:       "test_access_token",
	AccessSecret:      "test_access_secret",
	AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
}
