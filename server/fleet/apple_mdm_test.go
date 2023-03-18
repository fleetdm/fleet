package fleet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mozilla.org/pkcs7"

	"github.com/micromdm/scep/v2/depot"
)

func TestMDMAppleConfigProfile(t *testing.T) {
	cases := []struct {
		testName     string
		mobileconfig mobileconfig.Mobileconfig
		shouldFail   bool
	}{
		{
			testName:     "TestParseConfigProfileOK",
			mobileconfig: mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), ""),
			shouldFail:   false,
		},
		{
			testName:     "TestParseConfigProfileNoIdentifier",
			mobileconfig: mobileconfigForTest("ValidName", "", uuid.NewString(), ""),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoName",
			mobileconfig: mobileconfigForTest("", "ValidIdentifier", uuid.NewString(), ""),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoNameNoIdentifier",
			mobileconfig: mobileconfigForTest("", "", uuid.NewString(), ""),
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
				signedData, err := pkcs7.NewSignedData(mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), ""))
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
			mc := mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), mcPayloadContentForTest(c.payloadTypes))
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
			mc := mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), mcPayloadContentForTest(c.payloadIdentifiers))
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

func mobileconfigForTest(name string, identifier string, uuid string, payloadContent string) mobileconfig.Mobileconfig {
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
