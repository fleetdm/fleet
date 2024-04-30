package fleet

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mozilla.org/pkcs7"

	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
)

func TestMDMAppleConfigProfile(t *testing.T) {
	cases := []struct {
		testName     string
		mobileconfig mobileconfig.Mobileconfig
		shouldFail   bool
	}{
		{
			testName:     "TestParseConfigProfileOK",
			mobileconfig: MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), ""),
			shouldFail:   false,
		},
		{
			testName:     "TestParseConfigProfileLeadingSpace",
			mobileconfig: append([]byte{' '}, []byte(MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), ""))...),
			shouldFail:   false,
		},
		{
			testName:     "TestParseConfigProfileNoIdentifier",
			mobileconfig: MobileconfigForTest("ValidName", "", uuid.NewString(), ""),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoName",
			mobileconfig: MobileconfigForTest("", "ValidIdentifier", uuid.NewString(), ""),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoNameNoIdentifier",
			mobileconfig: MobileconfigForTest("", "", uuid.NewString(), ""),
			shouldFail:   true,
		},
		{
			testName: "TestParseConfigProfileInvalidEncoding",
			mobileconfig: func() []byte {
				b, err := json.Marshal(MDMAppleConfigProfile{Name: "ValidName", Identifier: "ValidIdentifier"})
				require.NoError(t, err)
				return b
			}(),
			shouldFail: true,
		},
		{
			testName: "TestParseConfigProfilePKCS7Encoding",
			mobileconfig: func() []byte {
				// generate certificate for signed data test
				key, err := rsa.GenerateKey(rand.Reader, 2048)
				require.NoError(t, err)
				crtBytes, err := depot.NewCACert().SelfSign(rand.Reader, key.Public(), key)
				require.NoError(t, err)
				crt, err := x509.ParseCertificate(crtBytes)
				require.NoError(t, err)

				// encode mobileconfig as PKCS7 signed data
				signedData, err := pkcs7.NewSignedData(MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), ""))
				require.NoError(t, err)
				err = signedData.AddSigner(crt, key, pkcs7.SignerInfoConfig{})
				require.NoError(t, err)
				signedBytes, err := signedData.Finish()
				require.NoError(t, err)
				p7, err := pkcs7.Parse(signedBytes)
				require.NoError(t, err)
				require.NoError(t, p7.Verify())

				return signedBytes
			}(),
			shouldFail: false,
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			parsed, err := NewMDMAppleConfigProfile(c.mobileconfig, nil)
			if c.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, "ValidName", parsed.Name)
				require.Equal(t, "ValidIdentifier", parsed.Identifier)
			}
		})
	}
}

func TestMDMAppleConfigProfileScreenPayloadContent(t *testing.T) {
	cases := []struct {
		testName     string
		payloadTypes []string
		shouldFail   []string
	}{
		{
			testName:     "AllFileVaultScreened",
			payloadTypes: []string{"com.apple.security.FDERecoveryKeyEscrow", "com.apple.MCX.FileVault2", "com.apple.security.FDERecoveryRedirect"},
			shouldFail:   []string{"com.apple.security.FDERecoveryKeyEscrow", "com.apple.MCX.FileVault2", "com.apple.security.FDERecoveryRedirect"},
		},
		{
			testName:     "FileVault2Screened",
			payloadTypes: []string{"com.apple.MCX.FileVault2"},
			shouldFail:   []string{"com.apple.MCX.FileVault2"},
		},
		{
			testName:     "FDERecoveryKeyEscrowScreened",
			payloadTypes: []string{"com.apple.security.FDERecoveryKeyEscrow"},
			shouldFail:   []string{"com.apple.security.FDERecoveryKeyEscrow"},
		},
		{
			testName:     "FDERecoveryRedirectScreened",
			payloadTypes: []string{"com.apple.security.FDERecoveryRedirect"},
			shouldFail:   []string{"com.apple.security.FDERecoveryRedirect"},
		},
		{
			testName:     "OtherPayloadTypesOK",
			payloadTypes: []string{"com.apple.security.firewall", "com.apple.MCX"},
			shouldFail:   nil,
		},
		{
			testName:     "FileVaultMixedWithOtherPayloadTypes",
			payloadTypes: []string{"com.apple.MCX.FileVault2", "com.apple.security.firewall", "com.apple.security.FDERecoveryKeyEscrow", "com.apple.MCX"},
			shouldFail:   []string{"com.apple.MCX.FileVault2", "com.apple.security.FDERecoveryKeyEscrow"},
		},
		{
			testName:     "NoPayloadContent",
			payloadTypes: nil,
			shouldFail:   nil,
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			mc := MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), mcPayloadContentForTest(c.payloadTypes))
			parsed, err := NewMDMAppleConfigProfile(mc, nil)
			require.NoError(t, err)
			require.Equal(t, "ValidName", parsed.Name)
			require.Equal(t, "ValidIdentifier", parsed.Identifier)

			err = parsed.ValidateUserProvided()
			for _, pt := range c.shouldFail {
				require.Error(t, err)
				require.ErrorContains(t, err, pt)
			}
		})
	}
}

func TestMDMAppleConfigProfileScreenPayloadIdentifiers(t *testing.T) {
	cases := []struct {
		testName           string
		payloadIdentifiers []string
		shouldFail         []string
	}{
		{
			testName:           "AllFleetProfilesScreened",
			payloadIdentifiers: []string{"com.fleetdm.fleet.mdm.filevault", "com.fleetdm.fleetd.config"},
			shouldFail:         []string{"com.fleetdm.fleet.mdm.filevault", "com.fleetdm.fleetd.config"},
		},
		{
			testName:           "FileVault",
			payloadIdentifiers: []string{"com.fleetdm.fleet.mdm.filevault"},
			shouldFail:         []string{"com.fleetdm.fleet.mdm.filevault"},
		},
		{
			testName:           "Fleetd config",
			payloadIdentifiers: []string{"com.fleetdm.fleetd.config"},
			shouldFail:         []string{"com.fleetdm.fleetd.config"},
		},
		{
			testName:           "OtherPayloadTypesOK",
			payloadIdentifiers: []string{"com.my.custom.profile", "com.test.example"},
			shouldFail:         nil,
		},
		{
			testName:           "Mixed",
			payloadIdentifiers: []string{"com.fleetdm.fleet.mdm.filevault", "com.my.custom.profile", "com.test.example"},
			shouldFail:         []string{"com.fleetdm.fleet.mdm.filevault"},
		},
		{
			testName:           "NoPayloadContent",
			payloadIdentifiers: nil,
			shouldFail:         nil,
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			mc := MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), mcPayloadContentForTest(c.payloadIdentifiers))
			parsed, err := NewMDMAppleConfigProfile(mc, nil)
			require.NoError(t, err)
			require.Equal(t, "ValidName", parsed.Name)
			require.Equal(t, "ValidIdentifier", parsed.Identifier)

			err = parsed.ValidateUserProvided()
			for _, pt := range c.shouldFail {
				require.Error(t, err)
				require.ErrorContains(t, err, pt)
			}
		})
	}
}

func TestMDMAppleConfigProfileScreenReservedNames(t *testing.T) {
	type testcase struct {
		toplevelName string
		contentName  string
		shouldFail   bool
	}
	cases := []testcase{
		{"unreserved name", "unreserved name", false},
	}
	fleetNames := mdm.FleetReservedProfileNames()
	for name := range fleetNames {
		cases = append(cases, testcase{name, "unreserved name", true})
		cases = append(cases, testcase{"unreserved name", name, true})
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s-%s", c.toplevelName, c.contentName), func(t *testing.T) {
			payloadContent := fmt.Sprintf(`
				<dict>
					<key>PayloadDisplayName</key>
					<string>%s</string>
					<key>PayloadIdentifier</key>
					<string>ValidIdentitifer</string>
					<key>PayloadType</key>
					<string>ValidType</string>
					<key>PayloadUUID</key>
					<string>%s</string>
					<key>PayloadVersion</key>
					<integer>1</integer>
				</dict>`, c.contentName, uuid.NewString())

			mc := MobileconfigForTest(c.toplevelName, "ValidIdentifier", uuid.NewString(), payloadContent)
			parsed, err := NewMDMAppleConfigProfile(mc, nil)
			require.NoError(t, err)
			require.Equal(t, c.toplevelName, parsed.Name)
			require.Equal(t, "ValidIdentifier", parsed.Identifier)

			err = parsed.ValidateUserProvided()
			if c.shouldFail {
				require.Error(t, err)
				if c.toplevelName == "unreserved name" {
					require.ErrorContains(t, err, c.contentName)
				} else {
					require.ErrorContains(t, err, c.toplevelName)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func MobileconfigForTest(name string, identifier string, uuid string, payloadContent string) mobileconfig.Mobileconfig {
	pc := "<array/>"
	if payloadContent != "" {
		pc = fmt.Sprintf(`<array>%s
	</array>`, payloadContent)
	}
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	%s
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`, pc, name, identifier, uuid))
}

func mcPayloadContentForTest(refs []string) string {
	formatted := ""
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		ss := strings.Split(ref, ".")
		uuid := uuid.New()
		formatted += fmt.Sprintf(`
		<dict>
			<key>PayloadDisplayName</key>
			<string>%s</string>
			<key>PayloadIdentifier</key>
			<string>%s</string>
			<key>PayloadType</key>
			<string>%s</string>
			<key>PayloadUUID</key>
			<string>%s</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>`, ss[len(ss)-1], ref, ref, uuid)
	}

	return formatted
}

func TestHostDEPAssignment(t *testing.T) {
	cases := []struct {
		testName string
		input    HostDEPAssignment
		expect   bool
	}{
		{
			testName: "assigned to Fleet",
			input: HostDEPAssignment{
				HostID:    1,
				AddedAt:   time.Now(),
				DeletedAt: nil,
			},
			expect: true,
		},
		{
			testName: "was assigned Fleet but now deleted",
			input: HostDEPAssignment{
				HostID:    1,
				AddedAt:   time.Now(),
				DeletedAt: ptr.Time(time.Now()),
			},
			expect: false,
		},
		{
			testName: "empty struct",
			input:    HostDEPAssignment{},
			expect:   false,
		},
		{
			testName: "empty added at",
			input: HostDEPAssignment{
				HostID: 1,
			},
			expect: false,
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			require.Equal(t, c.expect, c.input.IsDEPAssignedToFleet())
		})
	}
}

func TestMDMProfileIsWithinGracePeriod(t *testing.T) {
	// create a test profile
	var b bytes.Buffer
	params := mobileconfig.FleetdProfileOptions{
		EnrollSecret: t.Name(),
		ServerURL:    "https://example.com",
		PayloadType:  mobileconfig.FleetdConfigPayloadIdentifier,
		PayloadName:  mdm.FleetdConfigProfileName,
	}
	err := mobileconfig.FleetdProfileTemplate.Execute(&b, params)
	require.NoError(t, err)
	testProfile, err := NewMDMAppleConfigProfile(b.Bytes(), nil)
	require.NoError(t, err)

	// set profile updated at 2 hours ago
	testProfile.UploadedAt = time.Now().Truncate(time.Second).Add(-2 * time.Hour)
	// set profile created at 24 hours ago (irrelevant but included for completeness)
	testProfile.CreatedAt = testProfile.UploadedAt.Add(-24 * time.Hour)

	cases := []struct {
		testName            string
		hostDetailUpdatedAt time.Time
		expect              bool
	}{
		{
			testName:            "outside grace period",
			hostDetailUpdatedAt: testProfile.UploadedAt.Add(61 * time.Minute), // more than 1 hour grace period
			expect:              false,
		},
		{
			testName:            "online host within grace period",
			hostDetailUpdatedAt: testProfile.UploadedAt.Add(59 * time.Minute), // less than 1 hour grace period
			expect:              true,
		},
		{
			testName:            "offline host within grace period",
			hostDetailUpdatedAt: testProfile.UploadedAt.Add(-48 * time.Hour), // grace period doesn't start until host is online (i.e. host detail updated at is after profile updated at)
			expect:              true,
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			ep := ExpectedMDMProfile{Identifier: testProfile.Identifier, EarliestInstallDate: testProfile.UploadedAt}
			require.Equal(t, c.expect, ep.IsWithinGracePeriod(c.hostDetailUpdatedAt))
		})
	}
}
