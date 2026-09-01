package fleet

import (
	"bytes"
	"crypto/md5" //nolint:gosec // matches EffectiveDDMToken's DDM token hashing
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMDMAppleConfigProfile(t *testing.T) {
	cases := []struct {
		testName     string
		mobileconfig mobileconfig.Mobileconfig
		shouldFail   bool
		errString    *string
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
			shouldFail: true,
		},
		{
			testName:     "TestParseConfigProfileUnescapedCharsInPayload",
			mobileconfig: MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), `<string>Unescaped & < > ' "</string>`),
			shouldFail:   true,
			errString:    new("The configuration profile contains special characters (&, <, >, ', \") that must be XML-escaped. Please escape them (e.g. & → &amp;, < → &lt;) and try again."),
		},
		{
			testName:     "TestParseConfigProfileUnescapedCharsInIdentifier",
			mobileconfig: MobileconfigForTest("ValidName", "Valid<Identifier", uuid.NewString(), `<string>Valid</string>`),
			shouldFail:   true,
			errString:    new("The configuration profile contains special characters (&, <, >, ', \") that must be XML-escaped. Please escape them (e.g. & → &amp;, < → &lt;) and try again."),
		},
		{
			testName:     "TestParseConfigProfileUnescapedCharsInName",
			mobileconfig: MobileconfigForTest("Valid<Name", "ValidIdentifier", uuid.NewString(), `<string>Valid</string>`),
			shouldFail:   true,
			errString:    new("The configuration profile contains special characters (&, <, >, ', \") that must be XML-escaped. Please escape them (e.g. & → &amp;, < → &lt;) and try again."),
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			parsed, err := NewMDMAppleConfigProfile(c.mobileconfig, nil)
			if c.shouldFail {
				require.Error(t, err)
				if c.errString != nil {
					require.ErrorContains(t, err, *c.errString)
				}
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
			shouldFail:   []string{mobileconfig.DiskEncryptionProfileRestrictionErrMsg},
		},
		{
			testName:     "FileVault2Screened",
			payloadTypes: []string{"com.apple.security.firewall", "com.apple.MCX.FileVault2"},
			shouldFail:   []string{mobileconfig.DiskEncryptionProfileRestrictionErrMsg},
		},
		{
			testName:     "FDERecoveryKeyEscrowScreened",
			payloadTypes: []string{"com.apple.security.FDERecoveryKeyEscrow"},
			shouldFail:   []string{mobileconfig.DiskEncryptionProfileRestrictionErrMsg},
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
			shouldFail:   []string{mobileconfig.DiskEncryptionProfileRestrictionErrMsg},
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

			// Test with allowCustomOSUpdatesAndFileVault = false (default behavior)
			err = parsed.ValidateUserProvided(false)
			for _, pt := range c.shouldFail {
				require.Error(t, err)
				require.ErrorContains(t, err, pt)
			}
			if len(c.shouldFail) == 0 {
				require.NoError(t, err)
			}
		})
	}
}

func TestMDMAppleConfigProfileAllowCustomFileVault(t *testing.T) {
	cases := []struct {
		testName     string
		payloadTypes []string
	}{
		{
			testName:     "FileVault2Allowed",
			payloadTypes: []string{"com.apple.MCX.FileVault2"},
		},
		{
			testName:     "FDERecoveryKeyEscrowAllowed",
			payloadTypes: []string{"com.apple.security.FDERecoveryKeyEscrow"},
		},
		{
			testName:     "AllFileVaultTypesAllowed",
			payloadTypes: []string{"com.apple.security.FDERecoveryKeyEscrow", "com.apple.MCX.FileVault2"},
		},
		{
			testName:     "FileVaultMixedWithOtherPayloadTypes",
			payloadTypes: []string{"com.apple.MCX.FileVault2", "com.apple.security.firewall", "com.apple.security.FDERecoveryKeyEscrow"},
		},
	}

	for _, c := range cases {
		t.Run(c.testName, func(t *testing.T) {
			mc := MobileconfigForTest("ValidName", "ValidIdentifier", uuid.NewString(), mcPayloadContentForTest(c.payloadTypes))
			parsed, err := NewMDMAppleConfigProfile(mc, nil)
			require.NoError(t, err)
			require.Equal(t, "ValidName", parsed.Name)
			require.Equal(t, "ValidIdentifier", parsed.Identifier)

			// When allowCustomFileVault = true, these profiles should be allowed
			err = parsed.ValidateUserProvided(true)
			require.NoError(t, err)
		})
	}
}

func TestMDMAppleRawDeclarationValidateUserProvided(t *testing.T) {
	cases := []struct {
		name        string
		declType    string
		identifier  string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid configuration declaration",
			declType: "com.apple.configuration.passcode.settings",
			wantErr:  false,
		},
		{
			// Regression test: software update enforcement used to be blocked
			// unless a special flag was set; it is now allowed by default.
			name:     "software update enforcement allowed by default",
			declType: "com.apple.configuration.softwareupdate.enforcement.specific",
			wantErr:  false,
		},
		{
			name:        "forbidden declaration type",
			declType:    "com.apple.configuration.watch.enrollment",
			wantErr:     true,
			errContains: "com.apple.configuration.watch.enrollment is a forbidden declaration type.",
		},
		{
			name:        "status subscriptions not allowed",
			declType:    "com.apple.configuration.management.status-subscriptions",
			wantErr:     true,
			errContains: "Declaration profile can't include status subscription type.",
		},
		{
			name:        "managed app configuration not allowed",
			declType:    "com.apple.configuration.app.managed",
			wantErr:     true,
			errContains: "Declaration profile can't include software management types. To manage software, please use the Software tab.",
		},
		{
			name:        "managed app configuration not allowed",
			declType:    "com.apple.configuration.package",
			wantErr:     true,
			errContains: "Declaration profile can't include software management types. To manage software, please use the Software tab.",
		},
		{
			name:        "non-configuration declaration not allowed",
			declType:    "com.apple.activation.simple",
			wantErr:     true,
			errContains: "Only configuration declarations (com.apple.configuration.) and management declarations (com.apple.management.) are supported.",
		},
		{
			name:     "management declaration allowed",
			declType: "com.apple.management.organization-info",
			wantErr:  false,
		},
		{
			name:     "management properties declaration allowed",
			declType: "com.apple.management.properties",
			wantErr:  false,
		},
		{
			name:        "identifier over Apple's 64 octet limit",
			declType:    "com.apple.configuration.passcode.settings",
			identifier:  strings.Repeat("a", 65),
			wantErr:     true,
			errContains: "Identifier must be 64 bytes or fewer.",
		},
		{
			// octets, not characters: 22 three-byte runes exceed 64
			name:        "multibyte identifier counted in octets",
			declType:    "com.apple.configuration.passcode.settings",
			identifier:  strings.Repeat("日", 22),
			wantErr:     true,
			errContains: "Identifier must be 64 bytes or fewer.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			identifier := c.identifier
			if identifier == "" {
				identifier = "test-identifier"
			}
			decl := &MDMAppleRawDeclaration{
				Type:       c.declType,
				Identifier: identifier,
			}

			err := decl.ValidateUserProvided()
			if c.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, c.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMDMAppleDeclarationPayloadScope(t *testing.T) {
	t.Parallel()

	t.Run("parse and default", func(t *testing.T) {
		cases := []struct {
			name  string
			raw   string
			want  PayloadScope
			valid bool
		}{
			{name: "absent defaults to System", raw: `{"Type":"com.apple.configuration.passcode.settings","Identifier":"x"}`, want: PayloadScopeSystem, valid: true},
			{name: "explicit System", raw: `{"Type":"x","Identifier":"y","PayloadScope":"System"}`, want: PayloadScopeSystem, valid: true},
			{name: "explicit User", raw: `{"Type":"x","Identifier":"y","PayloadScope":"User"}`, want: PayloadScopeUser, valid: true},
			{name: "invalid value", raw: `{"Type":"x","Identifier":"y","PayloadScope":"Nope"}`, valid: false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				decl, err := GetRawDeclarationValues([]byte(c.raw))
				require.NoError(t, err)
				if c.valid {
					require.NoError(t, decl.ValidateScope())
					require.Equal(t, c.want, decl.ScopeOrDefault())
				} else {
					require.Error(t, decl.ValidateScope())
					require.ErrorContains(t, decl.ValidateScope(), "Invalid PayloadScope")
				}
			})
		}
	})
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

			err = parsed.ValidateUserProvided(false)
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

			err = parsed.ValidateUserProvided(false)
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

func TestDEPDeviceErrorTypeMessage(t *testing.T) {
	cases := []struct {
		errType DEPDeviceErrorType
		expect  string
	}{
		{DEPDeviceErrorTokenInvalid, "Fleet can't connect to Apple Business. An admin needs to renew the AB token."},
		{DEPDeviceErrorTermsExpired, "Apple Business terms/conditions have changed. An admin must accept them."},
		{DEPDeviceErrorNotFound, "Fleet can't find this host in Apple Business. It may have been removed or assigned to a different MDM server."},
		{DEPDeviceErrorServerError, "Apple's servers are temporarily unavailable. Please try again later."},
		{DEPDeviceErrorUnavailable, "Fleet can't retrieve data from Apple right now. Please try again later."},
		{DEPDeviceErrorType("unknown"), "Fleet can't retrieve data from Apple right now. Please try again later."},
	}

	for _, c := range cases {
		t.Run(string(c.errType), func(t *testing.T) {
			require.Equal(t, c.expect, c.errType.Message())
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

func TestMDMAppleHostDeclarationEqual(t *testing.T) {
	t.Parallel()

	// This test is intended to ensure that the Equal method on MDMAppleHostDeclaration is updated when new fields are added.
	// The Equal method is used to identify whether database update is needed.

	items := [...]MDMAppleHostDeclaration{{}, {}}

	numberOfFields := 0
	for i := 0; i < len(items); i++ {
		rValue := reflect.ValueOf(&items[i]).Elem()
		numberOfFields = rValue.NumField()
		for j := 0; j < numberOfFields; j++ {
			field := rValue.Field(j)
			switch field.Kind() {
			case reflect.String:
				valueToSet := fmt.Sprintf("test %d", i)
				field.SetString(valueToSet)
			case reflect.Int:
				field.SetInt(int64(i))
			case reflect.Bool:
				field.SetBool(i%2 == 0)
			case reflect.Pointer:
				field.Set(reflect.New(field.Type().Elem()))
			default:
				t.Fatalf("unhandled field type %s", field.Kind())
			}
		}
	}

	status0 := MDMDeliveryStatus("status")
	status1 := MDMDeliveryStatus("status")
	items[0].Status = &status0
	assert.False(t, items[0].Equal(items[1]))

	// Set known fields to be equal
	fieldsInEqualMethod := 0
	items[1].HostUUID = items[0].HostUUID
	fieldsInEqualMethod++
	items[1].DeclarationUUID = items[0].DeclarationUUID
	fieldsInEqualMethod++
	items[1].Name = items[0].Name
	fieldsInEqualMethod++
	items[1].Identifier = items[0].Identifier
	fieldsInEqualMethod++
	items[1].OperationType = items[0].OperationType
	fieldsInEqualMethod++
	items[1].Detail = items[0].Detail
	fieldsInEqualMethod++
	items[1].Token = items[0].Token
	fieldsInEqualMethod++
	items[1].Status = &status1
	fieldsInEqualMethod++
	items[1].SecretsUpdatedAt = items[0].SecretsUpdatedAt
	fieldsInEqualMethod++
	items[1].VariablesUpdatedAt = items[0].VariablesUpdatedAt
	fieldsInEqualMethod++
	items[1].AssetsUpdatedAt = items[0].AssetsUpdatedAt
	fieldsInEqualMethod++
	items[1].ActivationUpdatedAt = items[0].ActivationUpdatedAt
	fieldsInEqualMethod++
	items[1].Scope = items[0].Scope
	fieldsInEqualMethod++
	assert.Equal(t, fieldsInEqualMethod, numberOfFields, "MDMAppleHostDeclaration.Equal needs to be updated for new/updated field(s)")
	assert.True(t, items[0].Equal(items[1]))

	// Set pointers to nil
	items[0].Status = nil
	items[1].Status = nil
	assert.True(t, items[0].Equal(items[1]))
}

func TestEffectiveDDMToken(t *testing.T) {
	t.Parallel()

	const staticToken = "abc123"
	vars := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	assets := time.Date(2026, 7, 10, 8, 30, 0, 0, time.UTC)

	// md5Hex mirrors the hashing done by EffectiveDDMToken so we can assert the
	// exact concatenation order, which MUST match the SQL token computation in
	// MDMAppleDDMDeclarationsToken: HEX(token) + variables_updated_at + assets_updated_at.
	md5Hex := func(parts ...string) string {
		h := md5.New() //nolint:gosec // matches EffectiveDDMToken
		for _, p := range parts {
			_, _ = h.Write([]byte(p))
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	const layout = "2006-01-02 15:04:05.000000"

	t.Run("no vars, no assets returns static token unchanged", func(t *testing.T) {
		require.Equal(t, staticToken, EffectiveDDMToken(staticToken, nil, nil, nil))
	})

	t.Run("only variables", func(t *testing.T) {
		require.Equal(t, md5Hex(staticToken, vars.Format(layout)), EffectiveDDMToken(staticToken, &vars, nil, nil))
	})

	t.Run("only assets", func(t *testing.T) {
		got := EffectiveDDMToken(staticToken, nil, &assets, nil)
		require.Equal(t, md5Hex(staticToken, assets.Format(layout)), got)
		// An asset update must change the effective token away from the static one.
		require.NotEqual(t, staticToken, got)
	})

	t.Run("both variables and assets, order is static+vars+assets", func(t *testing.T) {
		require.Equal(t, md5Hex(staticToken, vars.Format(layout), assets.Format(layout)), EffectiveDDMToken(staticToken, &vars, &assets, nil))
	})

	t.Run("different asset timestamps yield different tokens", func(t *testing.T) {
		later := assets.Add(time.Second)
		require.NotEqual(t, EffectiveDDMToken(staticToken, nil, &assets, nil), EffectiveDDMToken(staticToken, nil, &later, nil))
	})
}

func TestMDMManagedCertificateEqual(t *testing.T) {
	t.Parallel()

	// Create two different time values for testing
	now := time.Now().Truncate(time.Second)
	later := now.Add(1 * time.Hour)

	// Create a serial string for testing
	serial1 := "serial1"
	serial2 := "serial2"

	// Create two instances with different values for all fields
	cert1 := MDMManagedCertificate{
		ProfileUUID:          "profile1",
		HostUUID:             "host1",
		ChallengeRetrievedAt: &now,
		NotValidBefore:       &now,
		NotValidAfter:        &later,
		Type:                 "type1",
		CAName:               "ca1",
		Serial:               &serial1,
	}

	cert2 := MDMManagedCertificate{
		ProfileUUID:          "profile2",
		HostUUID:             "host2",
		ChallengeRetrievedAt: &later,
		NotValidBefore:       &later,
		NotValidAfter:        &now,
		Type:                 "type2",
		CAName:               "ca2",
		Serial:               &serial2,
	}

	// Initial assertion - should not be equal
	assert.False(t, cert1.Equal(cert2))

	// Make fields equal one by one and test
	cert2.ProfileUUID = cert1.ProfileUUID
	assert.False(t, cert1.Equal(cert2))

	cert2.HostUUID = cert1.HostUUID
	assert.False(t, cert1.Equal(cert2))

	cert2.Type = cert1.Type
	assert.False(t, cert1.Equal(cert2))

	cert2.CAName = cert1.CAName
	assert.False(t, cert1.Equal(cert2))

	// Make time pointers equal
	cert2.ChallengeRetrievedAt = cert1.ChallengeRetrievedAt
	assert.False(t, cert1.Equal(cert2))

	cert2.NotValidBefore = cert1.NotValidBefore
	assert.False(t, cert1.Equal(cert2))

	cert2.NotValidAfter = cert1.NotValidAfter
	assert.False(t, cert1.Equal(cert2))

	// Make serial equal
	cert2.Serial = cert1.Serial
	assert.True(t, cert1.Equal(cert2))

	// Test nil pointer scenarios
	cert1.ChallengeRetrievedAt = nil
	assert.False(t, cert1.Equal(cert2))
	cert2.ChallengeRetrievedAt = nil
	assert.True(t, cert1.Equal(cert2))

	cert1.NotValidBefore = nil
	assert.False(t, cert1.Equal(cert2))
	cert2.NotValidBefore = nil
	assert.True(t, cert1.Equal(cert2))

	cert1.NotValidAfter = nil
	assert.False(t, cert1.Equal(cert2))
	cert2.NotValidAfter = nil
	assert.True(t, cert1.Equal(cert2))

	cert1.Serial = nil
	assert.False(t, cert1.Equal(cert2))
	cert2.Serial = nil
	assert.True(t, cert1.Equal(cert2))

	// Test time fields with same value but different memory addresses
	time1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	time2 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	// Verify these are different objects with the same value
	assert.NotSame(t, &time1, &time2)
	assert.True(t, time1.Equal(time2))

	cert1.ChallengeRetrievedAt = &time1
	cert2.ChallengeRetrievedAt = &time2
	assert.True(t, cert1.Equal(cert2))

	cert1.NotValidBefore = &time1
	cert2.NotValidBefore = &time2
	assert.True(t, cert1.Equal(cert2))

	cert1.NotValidAfter = &time1
	cert2.NotValidAfter = &time2
	assert.True(t, cert1.Equal(cert2))

	// Test serial with same value but different memory addresses
	serialStr1 := "same-serial"
	serialStr2 := "same-serial"
	assert.NotSame(t, &serialStr1, &serialStr2)

	cert1.Serial = &serialStr1
	cert2.Serial = &serialStr2
	assert.True(t, cert1.Equal(cert2))
}

func TestConfigurationProfileLabelEqual(t *testing.T) {
	t.Parallel()

	// This test is intended to ensure that the cmp.Equal method on ConfigurationProfileLabel is updated when new fields are added.
	// The cmp.Equal method is used to identify whether database update is needed.

	items := [...]ConfigurationProfileLabel{{}, {}}

	numberOfFields := 0
	for i := 0; i < len(items); i++ {
		rValue := reflect.ValueOf(&items[i]).Elem()
		numberOfFields = rValue.NumField()
		for j := 0; j < numberOfFields; j++ {
			field := rValue.Field(j)
			switch field.Kind() {
			case reflect.String:
				valueToSet := fmt.Sprintf("test %d", i)
				field.SetString(valueToSet)
			case reflect.Int:
				field.SetInt(int64(i))
			case reflect.Uint:
				field.SetUint(uint64(i))
			case reflect.Bool:
				field.SetBool(i%2 == 0)
			case reflect.Pointer:
				field.Set(reflect.New(field.Type().Elem()))
			default:
				t.Fatalf("unhandled field type %s", field.Kind())
			}
		}
	}

	assert.False(t, cmp.Equal(items[0], items[1]))

	// Set known fields to be equal
	fieldsInEqualMethod := 0
	items[1].ProfileUUID = items[0].ProfileUUID
	fieldsInEqualMethod++
	items[1].LabelName = items[0].LabelName
	fieldsInEqualMethod++
	items[1].LabelID = items[0].LabelID
	fieldsInEqualMethod++
	items[1].Broken = items[0].Broken
	fieldsInEqualMethod++
	items[1].Exclude = items[0].Exclude
	fieldsInEqualMethod++
	items[1].RequireAll = items[0].RequireAll
	fieldsInEqualMethod++

	assert.Equal(t, fieldsInEqualMethod, numberOfFields,
		"Does cmp.Equal for ConfigurationProfileLabel needs to be updated for new/updated field(s)?")
	assert.True(t, cmp.Equal(items[0], items[1]))
}

func TestValidateNoSecretsInProfileName(t *testing.T) {
	testCases := []struct {
		name       string
		xmlContent string
		expectErr  bool
		errMsg     string
	}{
		{
			name: "no secrets",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>Test Profile</string>
    <key>PayloadIdentifier</key>
    <string>com.test.profile</string>
</dict>
</plist>`,
			expectErr: false,
		},
		{
			name: "secret in PayloadDisplayName",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>Test $FLEET_SECRET_PASSWORD Profile</string>
    <key>PayloadIdentifier</key>
    <string>com.test.profile</string>
</dict>
</plist>`,
			expectErr: true,
			errMsg:    "PayloadDisplayName cannot contain FLEET_SECRET variables",
		},
		{
			name: "multiple PayloadDisplayNames with secret in one",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>Main Profile</string>
    <key>PayloadContent</key>
    <array>
        <dict>
            <key>PayloadDisplayName</key>
            <string>Sub Profile $FLEET_SECRET_KEY</string>
        </dict>
    </array>
</dict>
</plist>`,
			expectErr: true,
			errMsg:    "PayloadDisplayName cannot contain FLEET_SECRET variables",
		},
		{
			name: "secret in other field not PayloadDisplayName",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>Test Profile</string>
    <key>PayloadDescription</key>
    <string>Description with $FLEET_SECRET_VALUE</string>
</dict>
</plist>`,
			expectErr: false,
		},
		{
			name: "whitespace in PayloadDisplayName value",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>PayloadDisplayName</key>
    <string>   Test Profile   </string>
</dict>
</plist>`,
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNoSecretsInProfileName([]byte(tc.xmlContent))
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsMacIdentifier(t *testing.T) {
	cases := []struct {
		product string
		want    bool
		wantErr bool
	}{
		// --- MacBookPro ---
		{product: "MacBookPro16,1", want: true},
		{product: "MacBookPro16,4", want: true},
		{product: "MacBookPro17,1", want: true},
		{product: "MacBookPro18,3", want: true},
		{product: "MacBookPro18,4", want: true},

		// --- MacBookAir ---
		{product: "MacBookAir9,1", want: true},
		{product: "MacBookAir10,1", want: true},
		{product: "MacBookAir14,2", want: true},

		// --- Macmini ---
		{product: "Macmini8,1", want: true},
		{product: "Macmini9,1", want: true},
		{product: "Macmini9,2", want: true},

		// --- iMac ---
		{product: "iMac20,1", want: true},
		{product: "iMac20,2", want: true},
		{product: "iMac21,1", want: true},
		{product: "iMac21,2", want: true},

		// --- MacBook (no suffix) — all x86, line discontinued before Apple Silicon ---
		{product: "MacBook10,1", want: true},
		{product: "MacBook9,1", want: true},

		// --- iMacPro — all x86, discontinued before Apple Silicon ---
		{product: "iMacPro1,1", want: true},

		// --- MacPro — old numbering, all x86 ---
		{product: "MacPro7,1", want: true},
		{product: "MacPro6,1", want: true},

		// --- Mac (bare prefix) — unified Apple Silicon naming ---
		{product: "Mac13,1", want: true},
		{product: "Mac13,2", want: true},
		{product: "Mac14,8", want: true},
		{product: "Mac16,10", want: true},

		// --- Non-Mac Apple devices — return false without error ---
		{product: "iPhone15,2", want: false},
		{product: "iPhone14,3", want: false},
		{product: "iPad13,18", want: false},
		{product: "iPodTouch9,1", want: false},

		// --- Virtual Mac machines ---
		{product: "VirtualMac2,1", want: true},

		// --- Error cases ---
		// Empty string
		{product: "", wantErr: true},
		// No comma separator
		{product: "MacBookPro18", wantErr: true},
		// Garbage input
		{product: "not-a-model", wantErr: true},
		// Non-Mac Apple devices that don't start with iPhone/iPod/iPad also returns silently with valid format
		{product: "AppleTV6,2", want: false},
		{product: "AppleTV14,1", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.product, func(t *testing.T) {
			got, _, _, err := IsMacIdentifier(tc.product)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestIsPrefixedIdentifier(t *testing.T) {
	cases := []struct {
		product string
		prefix  *string
		want    bool
		wantErr bool
	}{
		{product: "MacBookPro18,4", want: false},
		{product: "iPhone15,2", want: true},
		{product: "iPad10,1", want: true, prefix: new("iPad")},
		{product: "iPhone152", wantErr: true},
		{product: "", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.product, func(t *testing.T) {
			prefix := "iPhone"
			if tc.prefix != nil {
				prefix = *tc.prefix
			}
			got, _, _, err := IsPrefixedIdentifier(tc.product, prefix)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}

func TestMacSiliconAndA11DeviceCheck(t *testing.T) {
	type asResult struct{ wantAS, wantErr bool }
	type a11Result struct{ wantA11, wantErr bool }

	cases := []struct {
		product string
		as      asResult
		a11     a11Result
	}{
		// --- MacBookPro ---
		{product: "MacBookPro16,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "MacBookPro16,4", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "MacBookPro17,1", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "MacBookPro18,3", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "MacBookPro18,4", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},

		// --- MacBookAir ---
		{product: "MacBookAir9,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "MacBookAir10,1", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "MacBookAir14,2", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},

		// --- Macmini ---
		{product: "Macmini8,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "Macmini9,1", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},

		// --- iMac ---
		{product: "iMac20,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "iMac20,2", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "iMac21,1", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "iMac21,2", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},

		// --- Discontinued x86 lines ---
		{product: "MacBook10,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "iMacPro1,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "MacPro7,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},
		{product: "MacPro6,1", as: asResult{wantAS: false}, a11: a11Result{wantA11: false}},

		// --- Mac (bare prefix) — all Apple Silicon ---
		{product: "Mac13,1", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "Mac13,2", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "Mac14,8", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},
		{product: "Mac16,10", as: asResult{wantAS: true}, a11: a11Result{wantA11: false}},

		// --- Non-Mac iOS/iPadOS devices — A11+ capable, not Apple Silicon Macs ---
		{product: "iPhone10,3", a11: a11Result{wantA11: true}}, // iPhone X — first A11 device
		{product: "iPhone14,3", a11: a11Result{wantA11: true}},
		{product: "iPad13,18", a11: a11Result{wantA11: true}},
		{product: "iPhone8,1", a11: a11Result{wantA11: false}}, // A11 threshold not reached

		// --- Bad  ---
		{product: "", as: asResult{wantErr: true}, a11: a11Result{wantErr: true}},
		{product: "MacBookPro18", as: asResult{wantErr: true}, a11: a11Result{wantErr: true}},
		{product: "not-a-model", as: asResult{wantErr: true}, a11: a11Result{wantErr: true}},
	}

	for _, tc := range cases {
		t.Run(tc.product, func(t *testing.T) {
			gotAS, err := IsMacAppleSilicon(tc.product)
			if tc.as.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.as.wantAS, gotAS)
			}

			gotA11, err := IsA11ChipDevice(tc.product)
			if tc.a11.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.a11.wantA11, gotA11)
			}
		})
	}
}

func TestGetRawActivationValues(t *testing.T) {
	t.Run("valid activation", func(t *testing.T) {
		raw, err := GetRawActivationValues([]byte(`{
			"Type": "com.apple.activation.simple",
			"Identifier": "com.fleet.act.passcode",
			"Payload": {
				"StandardConfigurations": ["com.fleet.cfg.passcode"],
				"Predicate": "@status(os.version.major) >= 15"
			}
		}`))
		require.NoError(t, err)
		require.Equal(t, "com.apple.activation.simple", raw.Type)
		require.Equal(t, "com.fleet.act.passcode", raw.Identifier)
		require.Equal(t, []string{"com.fleet.cfg.passcode"}, raw.Payload.StandardConfigurations)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		_, err := GetRawActivationValues([]byte(`{"Type":`))
		require.Error(t, err)
		require.ErrorContains(t, err, "should include valid JSON")
	})
}

func TestMDMAppleRawActivationValidateUserProvided(t *testing.T) {
	const configIdentifier = "com.fleet.cfg.passcode"

	cases := []struct {
		name        string
		activation  MDMAppleRawActivation
		wantErr     bool
		errContains string
	}{
		{
			name: "valid activation",
			activation: rawActivation("com.apple.activation.simple", "com.fleet.act.passcode",
				configIdentifier),
		},
		{
			// Fleet accepts the whole com.apple.activation.* namespace so a new
			// Apple activation type doesn't need a Fleet change.
			name: "unknown activation type under the activation prefix is allowed",
			activation: rawActivation("com.apple.activation.something-new", "com.fleet.act.passcode",
				configIdentifier),
		},
		{
			name:        "missing type",
			activation:  rawActivation("", "com.fleet.act.passcode", configIdentifier),
			wantErr:     true,
			errContains: "The custom activation must include a Type.",
		},
		{
			name:        "whitespace type",
			activation:  rawActivation("      ", "com.fleet.act.passcode", configIdentifier),
			wantErr:     true,
			errContains: "The custom activation must include a Type.",
		},
		{
			name: "configuration type is not an activation",
			activation: rawActivation("com.apple.configuration.passcode.settings", "com.fleet.act.passcode",
				configIdentifier),
			wantErr:     true,
			errContains: "Only activation declarations (com.apple.activation.) are supported.",
		},
		{
			name:        "missing identifier",
			activation:  rawActivation("com.apple.activation.simple", "   ", configIdentifier),
			wantErr:     true,
			errContains: "The custom activation must include an Identifier.",
		},
		{
			name:        "identifier over Apple's 64 octet limit",
			activation:  rawActivation("com.apple.activation.simple", strings.Repeat("a", 65), configIdentifier),
			wantErr:     true,
			errContains: "Identifier must be 64 bytes or fewer.",
		},
		{
			name:        "no configurations referenced",
			activation:  rawActivation("com.apple.activation.simple", "com.fleet.act.passcode"),
			wantErr:     true,
			errContains: "The custom activation must reference the identifier of the configuration profile used to upload it.",
		},
		{
			name: "more than one configuration referenced",
			activation: rawActivation("com.apple.activation.simple", "com.fleet.act.passcode",
				configIdentifier, "com.fleet.cfg.firewall"),
			wantErr:     true,
			errContains: "The custom activation can only have one referenced configuration profile.",
		},
		{
			name: "references a different configuration",
			activation: rawActivation("com.apple.activation.simple", "com.fleet.act.passcode",
				"com.fleet.cfg.firewall"),
			wantErr:     true,
			errContains: `Expected "com.fleet.cfg.passcode", got "com.fleet.cfg.firewall".`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.activation.ValidateUserProvided(configIdentifier)
			if c.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, c.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("all problems are reported at once", func(t *testing.T) {
		act := rawActivation("com.apple.configuration.passcode.settings", "")
		err := act.ValidateUserProvided(configIdentifier)
		require.Error(t, err)

		var invalid *InvalidArgumentError
		require.ErrorAs(t, err, &invalid)
		require.Len(t, invalid.Errors, 3)
	})
}

func rawActivation(declType, identifier string, standardConfigurations ...string) MDMAppleRawActivation {
	var act MDMAppleRawActivation
	act.Type = declType
	act.Identifier = identifier
	act.Payload.StandardConfigurations = standardConfigurations
	return act
}
