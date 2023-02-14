package fleet

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mozilla.org/pkcs7"

	"github.com/micromdm/scep/v2/depot"
)

func TestMDMAppleConfigProfile(t *testing.T) {
	cases := []struct {
		testName     string
		mobileconfig Mobileconfig
		shouldFail   bool
	}{
		{
			testName:     "TestParseConfigProfileOK",
			mobileconfig: mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString()),
			shouldFail:   false,
		},
		{
			testName:     "TestParseConfigProfileNoIdentifier",
			mobileconfig: mobileconfigForTest("ValidName", "", uuid.NewString()),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoName",
			mobileconfig: mobileconfigForTest("", "ValidIdentifier", uuid.NewString()),
			shouldFail:   true,
		},
		{
			testName:     "TestParseConfigProfileNoNameNoIdentifier",
			mobileconfig: mobileconfigForTest("", "", uuid.NewString()),
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
				signedData, err := pkcs7.NewSignedData(mobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString()))
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
			mc := c.mobileconfig
			cp := new(MDMAppleConfigProfile)
			cp.Mobileconfig = mc

			parsed, err := cp.Mobileconfig.ParseConfigProfile()
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

func mobileconfigForTest(name string, identifier string, uuid string) Mobileconfig {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
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
`, name, identifier, uuid))
}
