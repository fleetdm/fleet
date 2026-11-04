package service

// Certificate payload profile tests for the MDM suite.
//
// Belongs here: profiles carrying SCEP and ACME payloads — Windows device, user, and hardcoded SCEP
// profiles, accepted placements of certificate markers and variables at upload, and ACME
// certificates for user- and system-scoped profiles.
//
// Does not belong here: certificate authority integrations such as DigiCert or custom SCEP
// (integration_mdm_scep_test.go) and certificate authority CRUD
// (integration_certificate_authorities_test.go).

import (
	"bytes"
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/xml"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	scepserver "github.com/fleetdm/fleet/v4/server/mdm/scep/server"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/profiles/windows-device-scep.xml
var windowsDeviceSCEPProfileBytes []byte

func (s *integrationMDMTestSuite) TestWindowsDeviceSCEPProfile() {
	testWindowsSCEPProfile(s, windowsDeviceSCEPProfileBytes)
}

//go:embed testdata/profiles/windows-user-scep.xml
var windowsUserSCEPProfileBytes []byte

func (s *integrationMDMTestSuite) TestWindowsUserSCEPProfile() {
	testWindowsSCEPProfile(s, windowsUserSCEPProfileBytes)
}

func testWindowsSCEPProfile(s *integrationMDMTestSuite, windowsScepProfile []byte) {
	t := s.T()
	ctx := context.Background()
	scepServer := scep_server.StartTestSCEPServer(t)
	scepServerURL := scepServer.URL + "/scep"

	// Create windows host and enroll in MDM
	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// verifyCommands runs a management session, drives any SCEP CSPs to completion against Fleet's
	// SCEP proxy, ACKs the rest of the commands with the given status, and asserts the SCEP exchange
	// succeeded (cert returned, no error). Returns the number of SCEP exchanges performed.
	verifyCommands := func(wantProfiles int, status string) int {
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		// profile installs + 2 protocol commands acks
		require.Len(t, cmds, wantProfiles+2)
		msgID, err := mdmDevice.GetCurrentMsgID()
		require.NoError(t, err)

		scepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		handled, scepResults, _ := mdmDevice.AppendSCEPInstallResponses(scepCtx, cmds, msgID, nil)

		atomicCmds := 0
		for _, c := range cmds {
			if c.Verb == "Atomic" {
				atomicCmds++
			}
			if _, ok := handled[c.Cmd.CmdID.Value]; ok {
				// Already ACKed by AppendSCEPInstallResponses. The status arg is intentionally ignored
				// for SCEP CSPs - the helper always sends 200, matching real Windows behavior.
				continue
			}
			cmdID := c.Cmd.CmdID.Value
			verb := c.Verb
			mdmDevice.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &cmdID,
				Cmd:     &verb,
				Data:    &status,
				Items:   nil,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		require.Equal(t, wantProfiles, atomicCmds)

		// Drain SCEP results and assert success.
		scepCount := 0
		for res := range scepResults {
			scepCount++
			require.NoError(t, res.Err, "SCEP exchange for %q failed", res.UniqueID)
			require.NotNil(t, res.Cert, "SCEP exchange for %q returned no certificate", res.UniqueID)
		}

		cmds, err = mdmDevice.SendResponse()
		require.NoError(t, err)
		// the ack of the message should be the only returned command
		require.Len(t, cmds, 1)
		return scepCount
	}

	// Upload SCEP profile with missing CA
	resp := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "WindowsSCEPProfile", Contents: windowsScepProfile},
		}},
		http.StatusBadRequest)
	errMsg := extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "Fleet variable $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_INTEGRATION does not exist.")

	// Create Custom SCEP CA
	ca := &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("INTEGRATION"),
		Challenge: new("integration-test"),
		URL:       new(scepServerURL),
	}

	_, err := s.ds.NewCertificateAuthority(ctx, ca)
	require.NoError(t, err)

	// Fail on missing OU
	resp = s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "WindowsSCEPProfile", Contents: bytes.ReplaceAll(windowsScepProfile, []byte(fleet.FleetVarSCEPRenewalID.WithPrefix()), []byte("BOGUS"))},
		}},
		http.StatusBadRequest)
	errMsg = extractServerErrorText(resp.Body)
	require.Contains(t, errMsg, "SCEP profile for custom SCEP certificate authority requires: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>, $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>, and $FLEET_VAR_CERTIFICATE_RENEWAL_ID variables")

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "WindowsSCEPProfile", Contents: windowsScepProfile},
		}},
		http.StatusNoContent)

	// Verify host receives the profile
	s.awaitTriggerProfileSchedule(t)

	// Check that profile status is Pending
	profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	var foundProfile bool
	for _, p := range profiles {
		if p.Name == "WindowsSCEPProfile" {
			foundProfile = true
			require.NotNil(t, p.Status)
			assert.Equal(t, fleet.MDMDeliveryPending, *p.Status)
		}
	}
	require.True(t, foundProfile, "WindowsSCEPProfile not found for host")

	scepCount := verifyCommands(1, syncml.CmdStatusOK)
	require.Equal(t, 1, scepCount, "SCEP exchange should have run exactly once")

	// The device ACKed the SCEP <Exec>, but for a Fleet-proxied SCEP profile that only means the exchange was
	// accepted, not that a certificate landed on the host. The profile stays "verifying" until Fleet observes the
	// matching certificate.
	profiles, err = s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	foundProfile = false
	profileUUID := ""
	for _, p := range profiles {
		if p.Name == "WindowsSCEPProfile" {
			foundProfile = true
			profileUUID = p.ProfileUUID
			require.NotNil(t, p.Status)
			assert.Equal(t, fleet.MDMDeliveryVerifying, *p.Status)
		}
	}
	require.True(t, foundProfile, "WindowsSCEPProfile not found for host")

	// Simulate osquery reporting the issued certificate. It carries the profile's renewal-ID marker
	// (fleet-<profile_uuid>) in its OU, which Fleet matches back to the profile's managed-certificate row.
	sha1Sum := make([]byte, 20)
	copy(sha1Sum, profileUUID)
	require.NoError(t, s.ds.UpdateHostCertificates(ctx, host.ID, host.UUID, []*fleet.HostCertificateRecord{{
		HostID:                    host.ID,
		CommonName:                "windows-scep-cert",
		SubjectCommonName:         "windows-scep-cert",
		SubjectOrganizationalUnit: "fleet-" + profileUUID,
		SHA1Sum:                   sha1Sum,
		NotValidBefore:            time.Now().Add(-time.Hour),
		NotValidAfter:             time.Now().Add(365 * 24 * time.Hour),
		Source:                    fleet.SystemHostCertificate,
	}}, fleet.HostCertificateOriginOsquery, nil))

	// Now that Fleet has observed the matching certificate, the profile is verified.
	profiles, err = s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	foundProfile = false
	for _, p := range profiles {
		if p.Name == "WindowsSCEPProfile" {
			foundProfile = true
			require.NotNil(t, p.Status)
			assert.Equal(t, fleet.MDMDeliveryVerified, *p.Status)
		}
	}
	require.True(t, foundProfile, "WindowsSCEPProfile not found for host")

	// Attempt simple SCEP call with GetCACaps operation to verify SCEP server is reachable
	identifier := host.UUID + "," + profileUUID + "," + "INTEGRATION"
	scepRes := s.DoRawWithHeaders("GET", apple_mdm.SCEPProxyPath+identifier+"/pkiclient.exe", nil, http.StatusOK, nil, "operation", "GetCACaps")
	body, err := io.ReadAll(scepRes.Body)
	require.NoError(t, err)
	assert.Equal(t, scepserver.DefaultCACaps, string(body))
}

// TestWindowsHardcodedSCEPProfile covers a Windows MDM SCEP profile that uses a hardcoded SCEP
// server URL and challenge instead of Fleet's Custom SCEP proxy variables. The test client
// should still drive the SCEP exchange against the literal URL; Fleet's SCEP proxy is not
// involved on this path.
func (s *integrationMDMTestSuite) TestWindowsHardcodedSCEPProfile() {
	t := s.T()
	ctx := t.Context()
	scepServer := scep_server.StartTestSCEPServer(t)
	scepServerURL := scepServer.URL + "/scep"

	fixtures := map[string][]byte{
		"./Device": windowsDeviceSCEPProfileBytes,
		"./User":   windowsUserSCEPProfileBytes,
	}
	for _, locPrefix := range []string{"./Device", "./User"} {
		t.Run(locPrefix, func(t *testing.T) {
			profileName := "HardcodedSCEPProfile" + strings.ReplaceAll(locPrefix, "/", "_")
			profile := bytes.ReplaceAll(fixtures[locPrefix],
				[]byte("$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_INTEGRATION"), []byte(scepServerURL))
			profile = bytes.ReplaceAll(profile,
				[]byte("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_INTEGRATION"), []byte("any-challenge"))
			profile = bytes.ReplaceAll(profile,
				[]byte("$FLEET_VAR_SCEP_RENEWAL_ID"), []byte("hardcoded-renewal"))

			s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
				batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
					{Name: profileName, Contents: profile},
				}},
				http.StatusNoContent)

			host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

			s.awaitTriggerProfileSchedule(t)

			cmds, err := mdmDevice.StartManagementSession()
			require.NoError(t, err)
			msgID, err := mdmDevice.GetCurrentMsgID()
			require.NoError(t, err)

			scepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			handled, scepResults, _ := mdmDevice.AppendSCEPInstallResponses(scepCtx, cmds, msgID, nil)
			for _, c := range cmds {
				if c.Verb == fleet.CmdStatus {
					continue
				}
				if _, ok := handled[c.Cmd.CmdID.Value]; ok {
					continue
				}
				cmdID := c.Cmd.CmdID.Value
				verb := c.Verb
				status := syncml.CmdStatusOK
				mdmDevice.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &cmdID,
					Cmd:     &verb,
					Data:    &status,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
			}

			scepCount := 0
			for res := range scepResults {
				scepCount++
				require.NoError(t, res.Err, "SCEP exchange failed")
				require.NotNil(t, res.Cert)
			}
			require.Equal(t, 1, scepCount, "expected one SCEP exchange against the hardcoded URL")

			_, err = mdmDevice.SendResponse()
			require.NoError(t, err)

			profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
			require.NoError(t, err)
			var found bool
			for _, p := range profiles {
				if p.Name == profileName {
					found = true
					require.NotNil(t, p.Status)
					assert.Equal(t, fleet.MDMDeliveryVerified, *p.Status)
				}
			}
			require.True(t, found, "%s not found for host", profileName)
		})
	}
}

// TestWindowsConditionalAccessSCEPProfile covers a Windows SCEP profile that enrolls against Fleet's conditional access
// SCEP server. The admin writes the challenge, a global enroll secret, into the profile directly or through a secret
// variable. Fleet fills in the SCEP URL at delivery and tracks the certificate the way it tracks a proxied certificate
// authority's.
func (s *integrationMDMTestSuite) TestWindowsConditionalAccessSCEPProfile() {
	t := s.T()
	ctx := t.Context()

	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "first+secret="}}))
	_, _, err := s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{
		{Name: "CONDACCESS_CHALLENGE", Value: "first+secret="},
		{Name: "BAD_CHALLENGE", Value: "bad_secret"},
	})
	require.NoError(t, err)
	withChallenge := func(challenge string) []byte {
		profile := bytes.ReplaceAll(windowsUserSCEPProfileBytes,
			[]byte("$FLEET_VAR_CUSTOM_SCEP_PROXY_URL_INTEGRATION"), []byte(fleet.FleetVarConditionalAccessSCEPURL.WithPrefix()))
		return bytes.ReplaceAll(profile, []byte("$FLEET_VAR_CUSTOM_SCEP_CHALLENGE_INTEGRATION"), []byte(challenge))
	}

	// A challenge Windows can't send is rejected at upload, secret variables included.
	resp := s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "ConditionalAccessSCEP", Contents: withChallenge("$FLEET_SECRET_BAD_CHALLENGE")},
	}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(resp.Body), `"Challenge" field must contain a global enroll secret`)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "ConditionalAccessSCEP", Contents: withChallenge("$FLEET_SECRET_CONDACCESS_CHALLENGE")},
	}}, http.StatusNoContent)

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	scepURL, err := url.JoinPath(appConfig.ServerSettings.ServerURL, "/api/fleet/conditional_access/scep")
	require.NoError(t, err)

	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	hostProfile := func() *fleet.HostMDMWindowsProfile {
		profiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
		require.NoError(t, err)
		for _, p := range profiles {
			if p.Name == "ConditionalAccessSCEP" {
				return &p
			}
		}
		require.FailNow(t, "ConditionalAccessSCEP not found for host")
		return nil
	}
	rawCommand := func(cmdUUID string) string {
		var raw string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &raw, `SELECT raw_command FROM windows_mdm_commands WHERE command_uuid = ?`, cmdUUID)
		})
		return raw
	}
	// Acknowledges every pending command and returns the challenge in the one SCEP profile the device received.
	ackCommands := func() string {
		cmds, err := mdmDevice.StartManagementSession()
		require.NoError(t, err)
		sceps, incomplete := mdmDevice.ExtractSCEPCommands(cmds)
		require.Empty(t, incomplete)
		require.Len(t, sceps, 1)
		ackWindowsCmds(t, mdmDevice, cmds)
		return sceps[0].Challenge
	}
	// Queues the profile for another delivery, the way a resend or the renewal job does.
	queueRedelivery := func(profileUUID string) {
		require.NoError(t, s.ds.ResendHostMDMProfile(ctx, host.UUID, profileUUID))
	}

	s.awaitTriggerProfileSchedule(t)
	p := hostProfile()
	require.NotNil(t, p.Status)
	require.Equal(t, fleet.MDMDeliveryPending, *p.Status)
	raw := rawCommand(p.CommandUUID)
	require.Contains(t, raw, "<Data>"+scepURL+"</Data>")
	// A secret variable is only expanded when the command is sent, so the stored command never holds the secret.
	require.Contains(t, raw, "<Data>$FLEET_SECRET_CONDACCESS_CHALLENGE</Data>")
	require.NotContains(t, raw, "first+secret=")

	managedCert, err := s.ds.GetWindowsHostMDMCertificateProfile(ctx, host.UUID, p.ProfileUUID, fleet.ConditionalAccessManagedCertificateCAName)
	require.NoError(t, err)
	require.NotNil(t, managedCert)
	require.Equal(t, fleet.CAConfigConditionalAccess, managedCert.Type)
	require.NotNil(t, managedCert.ChallengeRetrievedAt)

	// Windows acknowledges the Exec before it contacts the SCEP server, so the ack alone doesn't verify the profile.
	require.Equal(t, "first+secret=", ackCommands())
	p = hostProfile()
	require.NotNil(t, p.Status)
	require.Equal(t, fleet.MDMDeliveryVerifying, *p.Status)

	// After the enroll secret and the secret variable are rotated, a redelivery carries the new secret without re-uploading
	// the profile.
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "second-secret"}}))
	_, _, err = s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{{Name: "CONDACCESS_CHALLENGE", Value: "second-secret"}})
	require.NoError(t, err)
	queueRedelivery(p.ProfileUUID)
	s.awaitTriggerProfileSchedule(t)
	p = hostProfile()
	raw = rawCommand(p.CommandUUID)
	require.NotContains(t, raw, "second-secret")
	require.Equal(t, "second-secret", ackCommands())
}

// TestWindowsConditionalAccessRenderedProfiles uploads both rendered Windows conditional access profiles unchanged and
// enrolls the host's certificate against Fleet's conditional access SCEP server.
func (s *integrationMDMTestSuite) TestWindowsConditionalAccessRenderedProfiles() {
	t := s.T()
	ctx := t.Context()

	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "conditional-access-secret"}}))
	var profiles fleet.ConditionalAccessWindowsProfiles
	s.DoJSON("GET", "/api/latest/fleet/conditional_access/idp/windows/profiles", nil, http.StatusOK, &profiles)

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	// As copied, the SCEP profile still has the challenge placeholder and is rejected.
	body, headers := generateNewProfileMultipartRequest(t, "OktaSCEP.xml", []byte(profiles.SCEPProfile), s.token,
		map[string][]string{"team_id": {fmt.Sprint(tm.ID)}})
	resp := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	require.Contains(t, extractServerErrorText(resp.Body), `"Challenge" field must contain a global enroll secret`)
	// The challenge can be a secret variable; it is expanded for validation and stored as the variable.
	_, _, err = s.ds.UpsertSecretVariables(ctx, []fleet.SecretVariable{{Name: "OKTA_SCEP_CHALLENGE", Value: "conditional-access-secret"}})
	require.NoError(t, err)
	scepProfile := strings.Replace(profiles.SCEPProfile, conditionalAccessWindowsChallengePlaceholder, "$FLEET_SECRET_OKTA_SCEP_CHALLENGE", 1)
	for fileName, contents := range map[string]string{
		"OktaSCEP.xml":     scepProfile,
		"OktaBrowsers.xml": profiles.BrowserProfile,
	} {
		body, headers := generateNewProfileMultipartRequest(t, fileName, []byte(contents), s.token,
			map[string][]string{"team_id": {fmt.Sprint(tm.ID)}})
		s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	}

	host, mdmDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{host.ID})))
	s.awaitTriggerProfileSchedule(t)

	cmds, err := mdmDevice.StartManagementSession()
	require.NoError(t, err)
	msgID, err := mdmDevice.GetCurrentMsgID()
	require.NoError(t, err)
	scepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	handled, scepResults, _ := mdmDevice.AppendSCEPInstallResponses(scepCtx, cmds, msgID, nil)
	for _, c := range cmds {
		if c.Verb == fleet.CmdStatus {
			continue
		}
		if _, ok := handled[c.Cmd.CmdID.Value]; ok {
			continue
		}
		cmdID, verb, status := c.Cmd.CmdID.Value, c.Verb, syncml.CmdStatusOK
		mdmDevice.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &cmdID,
			Cmd:     &verb,
			Data:    &status,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
	}
	var certs []*x509.Certificate
	for res := range scepResults {
		require.NoError(t, res.Err, "SCEP exchange failed")
		certs = append(certs, res.Cert)
	}
	require.Len(t, certs, 1)
	_, err = mdmDevice.SendResponse()
	require.NoError(t, err)

	hostProfiles, err := s.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	require.NoError(t, err)
	statuses := map[string]fleet.MDMDeliveryStatus{}
	var scepProfileUUID string
	for _, p := range hostProfiles {
		require.NotNil(t, p.Status, p.Name)
		statuses[p.Name] = *p.Status
		if p.Name == "OktaSCEP" {
			scepProfileUUID = p.ProfileUUID
		}
	}
	require.Equal(t, fleet.MDMDeliveryVerifying, statuses["OktaSCEP"])
	require.Equal(t, fleet.MDMDeliveryVerified, statuses["OktaBrowsers"])

	cert := certs[0]
	require.Equal(t, fleet.ConditionalAccessOktaCertificateCN, cert.Subject.CommonName)
	require.Equal(t, []string{"fleet-" + scepProfileUUID}, cert.Subject.OrganizationalUnit)
	require.Len(t, cert.URIs, 1)
	require.Equal(t, "urn:device:fleet:uuid:"+host.UUID, cert.URIs[0].String())
	hostID, err := s.ds.GetConditionalAccessCertHostIDBySerialNumber(ctx, uint64(cert.SerialNumber.Int64())) //nolint:gosec // G115
	require.NoError(t, err)
	require.Equal(t, host.ID, hostID)
}

// #40639 — integration coverage for non-proxied cert renewal
// profile uploads. The renewal-ID marker is opt-in; ACME and non-proxied
// SCEP profiles upload regardless of marker presence or placement. Auto-
// renewal activates when the marker is in OU; otherwise the profile
// uploads and works without renewal.

const acmeProfileTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key><string>com.apple.security.acme</string>
			<key>PayloadIdentifier</key><string>com.fleetdm.test.acme.payload</string>
			<key>PayloadUUID</key><string>11111111-2222-3333-4444-555555555555</string>
			<key>PayloadVersion</key><integer>1</integer>
			<key>PayloadDisplayName</key><string>ACME Cert</string>
			<key>DirectoryURL</key><string>https://acme.example.com/directory</string>
			<key>Subject</key>
			<array>
				<array><array><string>CN</string><string>%s</string></array></array>
				<array><array><string>OU</string><string>%s</string></array></array>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key><string>ACME Profile</string>
	<key>PayloadIdentifier</key><string>com.fleetdm.test.profile.acme</string>
	<key>PayloadType</key><string>Configuration</string>
	<key>PayloadUUID</key><string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadVersion</key><integer>1</integer>
</dict>
</plist>`

const rawSCEPProfileTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key><string>com.apple.security.scep</string>
			<key>PayloadIdentifier</key><string>com.fleetdm.test.scep.payload</string>
			<key>PayloadUUID</key><string>22222222-3333-4444-5555-666666666666</string>
			<key>PayloadVersion</key><integer>1</integer>
			<key>PayloadDisplayName</key><string>Raw SCEP Cert</string>
			<key>PayloadContent</key>
			<dict>
				<key>Challenge</key><string>static-challenge-value</string>
				<key>URL</key><string>https://scep.example.com/scep</string>
				<key>Subject</key>
				<array>
					<array><array><string>CN</string><string>%s</string></array></array>
					<array><array><string>OU</string><string>%s</string></array></array>
				</array>
			</dict>
		</dict>
	</array>
	<key>PayloadDisplayName</key><string>Raw SCEP Profile</string>
	<key>PayloadIdentifier</key><string>com.fleetdm.test.profile.rawscep</string>
	<key>PayloadType</key><string>Configuration</string>
	<key>PayloadUUID</key><string>bbbbbbbb-cccc-dddd-eeee-ffffffffffff</string>
	<key>PayloadVersion</key><integer>1</integer>
</dict>
</plist>`

// TestACMEProfileUploadAcceptsAllMarkerPlacements confirms upload succeeds
// across all combinations of variable name and Subject placement.
func (s *integrationMDMTestSuite) TestACMEProfileUploadAcceptsAllMarkerPlacements() {
	t := s.T()

	uploadACME := func(name, cn, ou string) {
		profile := fmt.Sprintf(acmeProfileTemplate, cn, ou)
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: name, Contents: []byte(profile)},
			}},
			http.StatusNoContent)
	}

	t.Run("preferred marker in OU (auto-renewal active)", func(t *testing.T) {
		uploadACME("acme-ou-preferred", "static-cn", "$FLEET_VAR_CERTIFICATE_RENEWAL_ID")
	})
	t.Run("legacy marker in OU", func(t *testing.T) {
		uploadACME("acme-ou-legacy", "static-cn", "$FLEET_VAR_SCEP_RENEWAL_ID")
	})
	t.Run("preferred marker in CN", func(t *testing.T) {
		uploadACME("acme-cn-preferred", "$FLEET_VAR_CERTIFICATE_RENEWAL_ID", "static-ou")
	})
	t.Run("no marker (opt-out; no auto-renewal but still uploads)", func(t *testing.T) {
		uploadACME("acme-no-marker", "static-cn", "static-ou")
	})
}

// TestRawSCEPProfileUploadAcceptsAllMarkerPlacements mirrors the ACME
// matrix for the com.apple.security.scep payload type.
func (s *integrationMDMTestSuite) TestRawSCEPProfileUploadAcceptsAllMarkerPlacements() {
	t := s.T()

	uploadRawSCEP := func(name, cn, ou string) {
		profile := fmt.Sprintf(rawSCEPProfileTemplate, cn, ou)
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
			batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
				{Name: name, Contents: []byte(profile)},
			}},
			http.StatusNoContent)
	}

	t.Run("preferred marker in OU (auto-renewal active)", func(t *testing.T) {
		uploadRawSCEP("rawscep-ou-preferred", "static-cn", "$FLEET_VAR_CERTIFICATE_RENEWAL_ID")
	})
	t.Run("legacy marker in OU", func(t *testing.T) {
		uploadRawSCEP("rawscep-ou-legacy", "static-cn", "$FLEET_VAR_SCEP_RENEWAL_ID")
	})
	t.Run("preferred marker in CN", func(t *testing.T) {
		uploadRawSCEP("rawscep-cn-preferred", "$FLEET_VAR_CERTIFICATE_RENEWAL_ID", "static-ou")
	})
	t.Run("no marker (opt-out; no auto-renewal but still uploads)", func(t *testing.T) {
		uploadRawSCEP("rawscep-no-marker", "static-cn", "static-ou")
	})
}

// TestConditionalAccessProfileUploadsCleanly verifies that the Fleet-
// generated Conditional Access SCEP profile uploads cleanly via the
// documented custom-OS-settings path. The template includes the renewal
// marker, so deploying from the Settings UI activates
// auto-renewal by default.
func (s *integrationMDMTestSuite) TestConditionalAccessProfileUploadsCleanly() {
	t := s.T()

	var buf bytes.Buffer
	require.NoError(t, conditionalAccessAppleProfileTemplateParsed.Execute(&buf, appleProfileTemplateData{
		CACertBase64:     "ZHVtbXkK",
		SCEPURL:          "https://example.com/api/fleet/conditional_access/scep",
		Challenge:        "test-challenge",
		CertificateCN:    "Fleet conditional access for Okta",
		MTLSURL:          "https://okta.example.com/api/fleet/conditional_access/idp/sso",
		CACertUUID:       "11111111-1111-1111-1111-111111111111",
		SCEPPayloadUUID:  "22222222-2222-2222-2222-222222222222",
		IdentityPrefUUID: "33333333-3333-3333-3333-333333333333",
		ChromeConfigUUID: "44444444-4444-4444-4444-444444444444",
		RootPayloadUUID:  "55555555-5555-5555-5555-555555555555",
	}))

	// Use the single-profile endpoint (POST /configuration_profiles) — that's
	// the route the custom-OS-settings UI uses when an admin uploads a
	// .mobileconfig copied out of Settings > Integrations > Conditional access.
	body, headers := generateNewProfileMultipartRequest(
		t, "ConditionalAccessProfile.mobileconfig", buf.Bytes(), s.token, nil,
	)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
}

// TestWindowsSCEPProfilePreferredVariableAccepted exercises the
// Windows validator on a pre-existing surface: NDES / Custom SCEP
// proxy validators accept both legacy $FLEET_VAR_SCEP_RENEWAL_ID and
// preferred $FLEET_VAR_CERTIFICATE_RENEWAL_ID. The existing
// TestWindowsDeviceSCEPProfile covers the legacy spelling.
func (s *integrationMDMTestSuite) TestWindowsSCEPProfilePreferredVariableAccepted() {
	t := s.T()
	ctx := context.Background()

	scepServer := scep_server.StartTestSCEPServer(t)
	_, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("INTEGRATION"),
		Challenge: new("integration-test"),
		URL:       new(scepServer.URL + "/scep"),
	})
	require.NoError(t, err)

	// Guard against the fixture drifting: if the legacy token vanishes from
	// the embedded XML, ReplaceAll silently no-ops and we'd test the legacy
	// path twice instead of the preferred one.
	require.True(t,
		bytes.Contains(windowsDeviceSCEPProfileBytes, []byte("$FLEET_VAR_SCEP_RENEWAL_ID")),
		"fixture must contain $FLEET_VAR_SCEP_RENEWAL_ID for the swap to test the preferred name",
	)
	preferred := bytes.ReplaceAll(
		windowsDeviceSCEPProfileBytes,
		[]byte("$FLEET_VAR_SCEP_RENEWAL_ID"),
		[]byte("$FLEET_VAR_CERTIFICATE_RENEWAL_ID"),
	)
	require.True(t,
		bytes.Contains(preferred, []byte("$FLEET_VAR_CERTIFICATE_RENEWAL_ID")),
		"preferred-name swap did not produce the expected token",
	)
	require.False(t,
		bytes.Contains(preferred, []byte("$FLEET_VAR_SCEP_RENEWAL_ID")),
		"legacy token must be fully replaced",
	)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "WindowsSCEPProfilePreferred", Contents: preferred},
		}},
		http.StatusNoContent)
}

// acmeProfileForScope builds a profile with a single ACME payload in the given scope.
func acmeProfileForScope(ident string, scope fleet.PayloadScope) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key><string>com.apple.security.acme</string>
			<key>PayloadIdentifier</key><string>%[1]s.payload</string>
			<key>PayloadUUID</key><string>%[2]s</string>
			<key>PayloadVersion</key><integer>1</integer>
			<key>PayloadDisplayName</key><string>ACME payload</string>
			<key>DirectoryURL</key><string>https://acme.example.com/directory</string>
			<key>Subject</key>
			<array>
				<array><array><string>CN</string><string>test-device</string></array></array>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key><string>%[1]s</string>
	<key>PayloadIdentifier</key><string>%[1]s</string>
	<key>PayloadType</key><string>Configuration</string>
	<key>PayloadUUID</key><string>%[3]s</string>
	<key>PayloadVersion</key><integer>1</integer>
	<key>PayloadScope</key><string>%[4]s</string>
</dict>
</plist>`, ident, uuid.NewString(), uuid.NewString(), scope))
}

// acmeCertTemplate returns a cert template identifiable by common name.
func acmeCertTemplate(commonName string) *x509.Certificate {
	tmpl := mdmtesting.NewTestMDMAppleCertTemplate()
	tmpl.SerialNumber = big.NewInt(int64(len(commonName)) + 1000)
	tmpl.Subject.CommonName = commonName
	return tmpl
}

// ackProfileInstall drains the channel until it acknowledges wantCmdUUID,
// returning the next command the server handed back.
func ackProfileInstall(t *testing.T, d *mdmtest.TestAppleMDMClient, userChannel bool, wantCmdUUID string) *mdm.Command {
	idle, ack := d.Idle, d.Acknowledge
	if userChannel {
		idle, ack = d.UserIdle, d.UserAcknowledge
	}
	cmd, err := idle()
	require.NoError(t, err)
	for cmd != nil {
		isTarget := cmd.CommandUUID == wantCmdUUID
		next, err := ack(cmd.CommandUUID)
		require.NoError(t, err)
		if isTarget {
			return next
		}
		cmd = next
	}
	require.FailNow(t, "profile command was never delivered", "command uuid %s", wantCmdUUID)
	return nil
}

// nanoCertificateListUUIDs returns the CertificateList commands queued against one
// nano enrollment, telling a device-channel refetch apart from a user-channel one.
func nanoCertificateListUUIDs(t *testing.T, ds *mysql.Datastore, enrollmentID string) []string {
	var uuids []string
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &uuids,
			`SELECT neq.command_uuid
			 FROM nano_enrollment_queue neq
			 JOIN nano_commands nc ON neq.command_uuid = nc.command_uuid
			 WHERE nc.request_type = 'CertificateList' AND neq.id = ?
			 ORDER BY neq.created_at`, enrollmentID)
	})
	return uuids
}

// hostCertificateScopes maps common name to reported scope, per the host details API.
func (s *integrationMDMTestSuite) hostCertificateScopes(t *testing.T, hostID uint) map[string]string {
	var resp listHostCertificatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", hostID), nil, http.StatusOK, &resp)
	scopes := make(map[string]string, len(resp.Certificates))
	for _, c := range resp.Certificates {
		scope := string(c.Source)
		if c.Username != "" {
			scope += ":" + c.Username
		}
		scopes[c.CommonName] = scope
	}
	return scopes
}

// TestACMECertForUserChannelProfile covers an ACME cert from a user-scoped
// profile on macOS: it lands in the user's login keychain, which the device
// channel can't see, so the refetch has to go out on the user channel and the
// cert has to be recorded under the user's scope.
func (s *integrationMDMTestSuite) TestACMECertForUserChannelProfile() {
	t := s.T()
	ctx := t.Context()

	// Enroll a macOS device and drain the initial (fleetd/CA) profiles.
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)

	// User-channel enrollment, as DEP-enrolled macOS devices do automatically.
	require.NoError(t, mdmDevice.UserEnroll())
	userEnr, err := s.ds.GetNanoMDMUserEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, userEnr, "user enrollment must exist after UserEnroll")

	// Upload a user-scoped ACME profile.
	const profIdent = "com.fleetdm.test.profile.acme.wifi"
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "ACME-WiFi", Contents: acmeProfileForScope(profIdent, fleet.PayloadScopeUser)},
	}}, http.StatusNoContent)

	// Trigger profile reconciliation so the cron picks up the new profile.
	s.awaitTriggerProfileSchedule(t)

	// Verify the profile row was created with scope=User.
	ok, cmdUUID := hostHasAppleProfileOp(t, s.ds, host.UUID, profIdent, fleet.MDMOperationTypeInstall)
	require.True(t, ok, "ACME profile should be pending install")
	require.NotEmpty(t, cmdUUID)

	// Verify the install command is queued to the user enrollment.
	var queuedEnrollmentID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &queuedEnrollmentID,
			`SELECT id FROM nano_enrollment_queue WHERE command_uuid = ?`, cmdUUID)
	})
	require.Equal(t, userEnr.ID, queuedEnrollmentID, "install command must be queued to user enrollment")

	// Acking the install on the user channel fires the certificate refetch.
	ackProfileInstall(t, mdmDevice, true, cmdUUID)

	// --- The refetch must target the user channel, not the device channel ---
	require.Empty(t, nanoCertificateListUUIDs(t, s.ds, host.UUID),
		"no CertificateList may be queued to the device channel: it only reports the system keychain")
	userCertListUUIDs := nanoCertificateListUUIDs(t, s.ds, userEnr.ID)
	require.Len(t, userCertListUUIDs, 1,
		"exactly one CertificateList must be queued to the user channel")

	// --- Answer on the user channel with the login keychain's contents ---
	cmd, err := mdmDevice.UserIdle()
	require.NoError(t, err)
	for cmd != nil && cmd.Command.RequestType != "CertificateList" {
		cmd, err = mdmDevice.UserAcknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, cmd, "CertificateList must be delivered on user channel")
	require.Equal(t, userCertListUUIDs[0], cmd.CommandUUID)

	const acmeCN = "acme-user-keychain-cert"
	_, err = mdmDevice.AcknowledgeUserCertificateList(cmd.CommandUUID,
		[]*x509.Certificate{acmeCertTemplate(acmeCN)})
	require.NoError(t, err)

	// --- The cert shows up, attributed to the enrolled user's keychain ---
	require.Equal(t, "user:"+mdmDevice.Username, s.hostCertificateScopes(t, host.ID)[acmeCN],
		"ACME cert must show under the enrolled user's scope, not the system keychain")

	// Cleanup must run against the answering enrollment, or the user channel's
	// refetch rows are never reclaimed.
	var trackedCommands []fleet.HostMDMCommand
	trackedCommands, err = s.ds.GetHostMDMCommands(ctx, host.ID)
	require.NoError(t, err)
	require.NotContains(t, trackedCommands,
		fleet.HostMDMCommand{HostID: host.ID, CommandType: fleet.RefetchCertsCommandUUIDPrefix},
		"refetch certs tracking row must be cleared once the user channel answers")
}

// TestACMECertUserAndSystemScopedProfilesCoexist covers a host with ACME profiles
// in both scopes: each channel reports only its own keychain, so neither refetch
// may clear the other's cert — removals included.
func (s *integrationMDMTestSuite) TestACMECertUserAndSystemScopedProfilesCoexist() {
	t := s.T()
	ctx := t.Context()

	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)
	require.NoError(t, mdmDevice.UserEnroll())
	userEnr, err := s.ds.GetNanoMDMUserEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, userEnr)

	const (
		userProfIdent = "com.fleetdm.test.profile.acme.user"
		sysProfIdent  = "com.fleetdm.test.profile.acme.system"
		userCN        = "acme-user-keychain-cert"
		sysCN         = "acme-system-keychain-cert"
	)
	userProfile := fleet.MDMProfileBatchPayload{Name: "ACME-User", Contents: acmeProfileForScope(userProfIdent, fleet.PayloadScopeUser)}
	sysProfile := fleet.MDMProfileBatchPayload{Name: "ACME-System", Contents: acmeProfileForScope(sysProfIdent, fleet.PayloadScopeSystem)}

	// --- Install the user-scoped profile and ingest its login keychain cert ---
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{userProfile}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	ok, userCmdUUID := hostHasAppleProfileOp(t, s.ds, host.UUID, userProfIdent, fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	ackProfileInstall(t, mdmDevice, true, userCmdUUID)

	cmd, err := mdmDevice.UserIdle()
	require.NoError(t, err)
	for cmd != nil && cmd.Command.RequestType != "CertificateList" {
		cmd, err = mdmDevice.UserAcknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, cmd, "CertificateList must be delivered on user channel")
	_, err = mdmDevice.AcknowledgeUserCertificateList(cmd.CommandUUID, []*x509.Certificate{acmeCertTemplate(userCN)})
	require.NoError(t, err)
	require.Equal(t, "user:"+mdmDevice.Username, s.hostCertificateScopes(t, host.ID)[userCN])

	// --- Install the system-scoped profile and ingest its system keychain cert ---
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{userProfile, sysProfile}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	ok, sysCmdUUID := hostHasAppleProfileOp(t, s.ds, host.UUID, sysProfIdent, fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	ackProfileInstall(t, mdmDevice, false, sysCmdUUID)

	deviceCertListUUIDs := nanoCertificateListUUIDs(t, s.ds, host.UUID)
	require.Len(t, deviceCertListUUIDs, 1, "system-scoped ACME profile must refetch on the device channel")

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil && cmd.Command.RequestType != "CertificateList" {
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, cmd, "CertificateList must be delivered on device channel")
	// The device channel reports the system keychain only.
	_, err = mdmDevice.AcknowledgeCertificateList(mdmDevice.UUID, cmd.CommandUUID, []*x509.Certificate{acmeCertTemplate(sysCN)})
	require.NoError(t, err)

	scopes := s.hostCertificateScopes(t, host.ID)
	require.Equal(t, "system", scopes[sysCN], "system keychain cert must be ingested")
	require.Equal(t, "user:"+mdmDevice.Username, scopes[userCN],
		"device-channel refetch must not clear the login keychain cert it cannot see")

	// --- Removing the user-scoped profile clears only its own cert ---
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{sysProfile}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	ok, removeCmdUUID := hostHasAppleProfileOp(t, s.ds, host.UUID, userProfIdent, fleet.MDMOperationTypeRemove)
	require.True(t, ok, "user-scoped profile should be pending removal")
	ackProfileInstall(t, mdmDevice, true, removeCmdUUID)

	cmd, err = mdmDevice.UserIdle()
	require.NoError(t, err)
	for cmd != nil && cmd.Command.RequestType != "CertificateList" {
		cmd, err = mdmDevice.UserAcknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, cmd, "removal must refetch on the user channel")
	// The login keychain no longer holds the cert.
	_, err = mdmDevice.AcknowledgeUserCertificateList(cmd.CommandUUID, nil)
	require.NoError(t, err)

	scopes = s.hostCertificateScopes(t, host.ID)
	require.NotContains(t, scopes, userCN, "removed profile's user keychain cert must be cleared")
	require.Equal(t, "system", scopes[sysCN],
		"user-channel refetch must not clear the system keychain cert it cannot see")
}
