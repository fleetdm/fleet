package service

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/integrationtest/scep_server"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/profiles/windows-device-scep.xml
var windowsDeviceSCEPProfileForRenewalTest []byte

// Phase 2 (#40639) — integration coverage for non-proxied cert renewal
// profile uploads. Datastore + unit tests live in the respective _test.go
// files; this exercises the public API surface end-to-end.
//
// Under Decision 2.6 (marker is opt-in), ACME and non-proxied SCEP profiles
// upload regardless of marker presence or placement. Auto-renewal activates
// when the marker is in OU; otherwise the profile uploads and works without
// renewal (4.85-style manual redeploy).

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

// TestACMEProfileUploadAcceptsAllMarkerPlacements exercises the public API
// surface end-to-end. The marker is opt-in; the matrix confirms upload
// succeeds across all combinations of variable name and Subject placement.
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
// documented path (custom OS settings). With PR 2.3e (#45580) the
// template includes the renewal marker, so deploying the profile from
// the Settings UI activates auto-renewal by default — no manual edit.
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

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "ConditionalAccessProfile", Contents: buf.Bytes()},
		}},
		http.StatusNoContent)
}

// TestWindowsSCEPProfilePreferredVariableAccepted exercises PR #45237's
// Windows validator change on a pre-existing surface: NDES / Custom SCEP
// proxy validators now accept both legacy $FLEET_VAR_SCEP_RENEWAL_ID and
// preferred $FLEET_VAR_CERTIFICATE_RENEWAL_ID. The existing
// TestWindowsDeviceSCEPProfile covers the legacy spelling.
func (s *integrationMDMTestSuite) TestWindowsSCEPProfilePreferredVariableAccepted() {
	t := s.T()
	ctx := context.Background()

	scepServer := scep_server.StartTestSCEPServer(t)
	_, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("INTEGRATION"),
		Challenge: ptr.String("integration-test"),
		URL:       ptr.String(scepServer.URL + "/scep"),
	})
	require.NoError(t, err)

	preferred := bytes.ReplaceAll(
		windowsDeviceSCEPProfileForRenewalTest,
		[]byte("$FLEET_VAR_SCEP_RENEWAL_ID"),
		[]byte("$FLEET_VAR_CERTIFICATE_RENEWAL_ID"),
	)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "WindowsSCEPProfilePreferred", Contents: preferred},
		}},
		http.StatusNoContent)
}
