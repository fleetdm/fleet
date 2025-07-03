//go:build darwin

package profiles

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestGetFleetdConfig(t *testing.T) {
	testErr := errors.New("test error")
	cases := []struct {
		cmdOut  *string
		cmdErr  error
		wantOut *fleet.MDMAppleFleetdConfig
		wantErr error
	}{
		{nil, testErr, nil, testErr},
		{ptr.String("invalid-xml"), nil, nil, io.EOF},
		{&emptyOutput, nil, &fleet.MDMAppleFleetdConfig{}, nil},
		{&withFleetdConfig, nil, &fleet.MDMAppleFleetdConfig{EnrollSecret: "ENROLL_SECRET", FleetURL: "https://test.example.com"}, nil},
	}

	origExecProfileCmd := execProfileCmd
	t.Cleanup(func() { execProfileCmd = origExecProfileCmd })
	for _, c := range cases {
		execProfileCmd = func() (*bytes.Buffer, error) {
			if c.cmdOut == nil {
				return nil, c.cmdErr
			}

			var buf bytes.Buffer
			buf.WriteString(*c.cmdOut)
			return &buf, nil
		}

		out, err := GetFleetdConfig()
		require.ErrorIs(t, err, c.wantErr)
		require.Equal(t, c.wantOut, out)
	}
}

var (
	emptyOutput = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict/>
</plist>`

	withFleetdConfig = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>_computerlevel</key>
	<array>
		<dict>
			<key>ProfileDescription</key>
			<string>test descripiton</string>
			<key>ProfileDisplayName</key>
			<string>test name</string>
			<key>ProfileIdentifier</key>
			<string>com.fleetdm.fleetd.config</string>
			<key>ProfileInstallDate</key>
			<string>2023-02-27 18:55:07 +0000</string>
			<key>ProfileItems</key>
			<array>
				<dict>
					<key>PayloadContent</key>
					<dict>
						<key>EnrollSecret</key>
						<string>ENROLL_SECRET</string>
						<key>FleetURL</key>
						<string>https://test.example.com</string>
					</dict>
					<key>PayloadDescription</key>
					<string>test description</string>
					<key>PayloadDisplayName</key>
					<string>test name</string>
					<key>PayloadIdentifier</key>
					<string>com.fleetdm.fleetd.config</string>
					<key>PayloadType</key>
					<string>com.fleetdm.fleetd</string>
					<key>PayloadUUID</key>
					<string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
					<key>PayloadVersion</key>
					<integer>1</integer>
				</dict>
			</array>
			<key>ProfileRemovalDisallowed</key>
			<string>true</string>
			<key>ProfileType</key>
			<string>Configuration</string>
			<key>ProfileUUID</key>
			<string>8D0F62E6-E24F-4B2F-AFA8-CAC1F07F4FDC</string>
			<key>ProfileVersion</key>
			<integer>1</integer>
		</dict>
	</array>
</dict>
</plist>`

	withFleetdConfigAndEnrollment = `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>_computerlevel</key>
	<array>
		<dict>
			<key>ProfileDescription</key>
			<string>test descripiton</string>
			<key>ProfileDisplayName</key>
			<string>test name</string>
			<key>ProfileIdentifier</key>
			<string>com.fleetdm.fleetd.config</string>
			<key>ProfileInstallDate</key>
			<string>2023-02-27 18:55:07 +0000</string>
			<key>ProfileItems</key>
			<array>
				<dict>
					<key>PayloadContent</key>
					<dict>
						<key>EnrollSecret</key>
						<string>ENROLL_SECRET</string>
						<key>FleetURL</key>
						<string>https://test.example.com</string>
					</dict>
					<key>PayloadDescription</key>
					<string>test description</string>
					<key>PayloadDisplayName</key>
					<string>test name</string>
					<key>PayloadIdentifier</key>
					<string>com.fleetdm.fleetd.config</string>
					<key>PayloadType</key>
					<string>com.fleetdm.fleetd</string>
					<key>PayloadUUID</key>
					<string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
					<key>PayloadVersion</key>
					<integer>1</integer>
				</dict>
			</array>
			<key>ProfileRemovalDisallowed</key>
			<string>true</string>
			<key>ProfileType</key>
			<string>Configuration</string>
			<key>ProfileUUID</key>
			<string>8D0F62E6-E24F-4B2F-AFA8-CAC1F07F4FDC</string>
			<key>ProfileVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>ProfileDisplayName</key>
			<string>f1337 enrollment</string>
			<key>ProfileIdentifier</key>
			<string>com.fleetdm.fleet.mdm.apple</string>
			<key>ProfileInstallDate</key>
			<string>2023-02-27 18:55:07 +0000</string>
			<key>ProfileItems</key>
			<array>
				<dict>
						<key>PayloadContent</key>
						<dict/>
						<key>PayloadIdentifier</key>
						<string>com.fleetdm.fleet.mdm.apple.scep</string>
						<key>PayloadType</key>
						<string>com.apple.security.scep</string>
						<key>PayloadUUID</key>
						<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
						<key>PayloadVersion</key>
						<integer>1</integer>
				</dict>
				<dict>
					<key>PayloadContent</key>
					<dict>
						<key>AccessRights</key>
						<integer>8191</integer>
						<key>CheckOutWhenRemoved</key>
						<true/>
						<key>EndUserEmail</key>
						<string>user@example.com</string>
						<key>ServerCapabilities</key>
						<array>
							<string>com.apple.mdm.per-user-connections</string>
							<string>com.apple.mdm.bootstraptoken</string>
						</array>
						<key>ServerURL</key>
						<string>https://test.example.com</string>
					</dict>
					<key>PayloadIdentifier</key>
					<string>com.fleetdm.fleet.mdm.apple.mdm</string>
					<key>PayloadType</key>
					<string>com.apple.mdm</string>
					<key>PayloadUUID</key>
					<string>29713130-1602-4D27-90C9-B822A295E44E</string>
					<key>PayloadVersion</key>
					<integer>1</integer>
				</dict>
			</array>
			<key>ProfileOrganization</key>
			<string>f1337</string>
			<key>ProfileType</key>
			<string>Configuration</string>
			<key>ProfileUUID</key>
			<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
			<key>ProfileVersion</key>
			<integer>1</integer>
	</dict>
</array>
</dict>
</plist>`
)

func TestCustomInstallerWorkflow(t *testing.T) {
	origExecProfileCmd := execProfileCmd
	t.Cleanup(func() { execProfileCmd = origExecProfileCmd })

	for _, c := range []struct {
		name      string
		mockOut   string
		wantEmail string
		wantErr   error
	}{
		{"happy path", withFleetdConfigAndEnrollment, "user@example.com", nil},
		{"empty profiles", emptyOutput, "", ErrNotFound},
		{"no enrollment payload", withFleetdConfig, "", ErrNotFound},
		{"wrong payload identifier", strings.Replace(withFleetdConfigAndEnrollment, mobileconfig.FleetEnrollmentPayloadIdentifier, "wrong-identifier", 1), "", ErrNotFound},
		{"no end user email key", strings.Replace(withFleetdConfigAndEnrollment, "EndUserEmail", "WrongKey", 1), "", ErrNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			execProfileCmd = func() (*bytes.Buffer, error) {
				var buf bytes.Buffer
				buf.WriteString(c.mockOut)
				return &buf, nil
			}

			gotContent, err := GetCustomEnrollmentProfileEndUserEmail()
			if c.wantErr != nil {
				require.ErrorIs(t, err, c.wantErr)
				require.Empty(t, gotContent)
			} else {
				require.NoError(t, err)
				require.Equal(t, "user@example.com", gotContent)
			}
		})
	}
}

func TestIsEnrolledInMDM(t *testing.T) {
	cases := []struct {
		cmdOut       *string
		cmdErr       error
		wantEnrolled bool
		wantURL      string
		wantErr      bool
	}{
		{nil, errors.New("test error"), false, "", true},
		{ptr.String(""), nil, false, "", false},
		{ptr.String(`
Enrolled via DEP: No
MDM enrollment: No
		`), nil, false, "", false},
		{
			ptr.String(`
Enrolled via DEP: Yes
MDM enrollment: Yes
MDM server: https://test.example.com
			`),
			nil,
			true,
			"https://test.example.com",
			false,
		},
		{
			ptr.String(`
Enrolled via DEP: Yes
MDM enrollment: Yes
MDM server /  https://test.example.com
			`),
			nil,
			true,
			"//test.example.com",
			false,
		},
		{
			ptr.String(`
Enrolled via DEP: Yes
MDM enrollment: Yes
MDM server: https://valid.com/mdm/apple/mdm
			`),
			nil,
			true,
			"https://valid.com/mdm/apple/mdm",
			false,
		},
	}

	origCmd := getMDMInfoFromProfilesCmd
	t.Cleanup(func() { getMDMInfoFromProfilesCmd = origCmd })
	for _, c := range cases {
		getMDMInfoFromProfilesCmd = func() ([]byte, error) {
			if c.cmdOut == nil {
				return nil, c.cmdErr
			}

			var buf bytes.Buffer
			buf.WriteString(*c.cmdOut)
			return []byte(*c.cmdOut), nil
		}

		enrolled, url, err := IsEnrolledInMDM()
		if c.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, c.wantEnrolled, enrolled)
		require.Equal(t, c.wantURL, url)
	}
}

func TestCheckAssignedEnrollmentProfile(t *testing.T) {
	fleetURL := "https://valid.com"
	cases := []struct {
		name    string
		cmdOut  *string
		cmdErr  error
		wantErr error
	}{
		{
			name:    "command error",
			cmdOut:  nil,
			cmdErr:  errors.New("some command error"),
			wantErr: errors.New("some command error"),
		},
		{
			name:    "empty output",
			cmdOut:  ptr.String(""),
			cmdErr:  nil,
			wantErr: errors.New("parsing profiles output: expected at least 2 lines but got 1"),
		},
		{
			name: "null profile",
			cmdOut: ptr.String(`Device Enrollment configuration:
(null)
		`),
			cmdErr:  nil,
			wantErr: errors.New("parsing profiles output: received null device enrollment configuration"),
		},
		{
			name: "mismatch profile",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	...
}
			`),
			cmdErr:  nil,
			wantErr: errors.New(`server url: expected 'valid.com' but found 'test.example.com'`),
		},
		{
			name: "match profile",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://valid.com?token=1234";
	...
}
			`),
			cmdErr:  nil,
			wantErr: nil,
		},
		{
			name: "mixed case match configuration web URL",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.ExaMplE.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://vaLiD.com?tOken=1234";
	...
}
			`),
			cmdErr:  nil,
			wantErr: nil,
		},
		{
			name: "mixed case match configuration URL but wrong configuration web URL",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
	AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://vaLiD.com?tOken=1234";
	ConfigurationWebURL = "https://test.ExaMplE.com/mdm/apple/enroll?token=1234";
	...
}
			`),
			cmdErr:  nil,
			wantErr: errors.New(`server url: expected 'valid.com' but found 'test.ExaMplE.com'`),
		},
		{
			name: "match configuration URL and empty configuration web URL",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
	AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://valid.com?token=1234";
	ConfigurationWebURL = "";
	...
}
			`),
			cmdErr:  nil,
			wantErr: nil,
		},
		{
			name: "mixed case match configuration web URL and empty configuration URL",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
	AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "";
	ConfigurationWebURL = "https://vaLiD.com?tOken=1234";
	...
}
			`),
			cmdErr:  nil,
			wantErr: nil,
		},

		{
			name: "unparseable URL",
			cmdOut: ptr.String(`Device Enrollment configuration:
{
	AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "://invalid-url";
	ConfigurationWebURL = "";
	...
}
			`),
			cmdErr:  nil,
			wantErr: errors.New("parsing profiles output: unable to parse server url"),
		},
	}

	origCmd := showEnrollmentProfileCmd
	t.Cleanup(func() { showEnrollmentProfileCmd = origCmd })
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			showEnrollmentProfileCmd = func() ([]byte, error) {
				if c.cmdOut == nil {
					return nil, c.cmdErr
				}
				var buf bytes.Buffer
				buf.WriteString(*c.cmdOut)
				return []byte(*c.cmdOut), nil
			}

			err := CheckAssignedEnrollmentProfile(fleetURL)
			if c.wantErr != nil {
				require.ErrorContains(t, err, c.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetProfilePayloadContent(t *testing.T) {
	origExecProfileCmd := execProfileCmd
	t.Cleanup(func() { execProfileCmd = origExecProfileCmd })

	execProfileCmd = func() (*bytes.Buffer, error) {
		var buf bytes.Buffer
		buf.WriteString(withFleetdConfigAndEnrollment)
		return &buf, nil
	}

	// mismatched int type is not acceptable
	_, err := getProfilePayloadContent[int]("com.fleetdm.fleet.mdm.apple.mdm")
	require.ErrorContains(t, err, "plist: cannot unmarshal dict into Go value of type int")

	// mismatched string type is not acceptable
	_, err = getProfilePayloadContent[string]("com.fleetdm.fleet.mdm.apple.mdm")
	require.ErrorContains(t, err, "plist: cannot unmarshal dict into Go value of type string")

	// mismatched bool type is not acceptable
	_, err = getProfilePayloadContent[bool]("com.fleetdm.fleet.mdm.apple.mdm")
	require.ErrorContains(t, err, "plist: cannot unmarshal dict into Go value of type bool")

	// mismatched slice type is not acceptable
	_, err = getProfilePayloadContent[[]string]("com.fleetdm.fleet.mdm.apple.mdm")
	require.ErrorContains(t, err, "plist: cannot unmarshal dict into Go value of type []string")

	// mismatched []byte type is not acceptable
	_, err = getProfilePayloadContent[[]byte]("com.fleetdm.fleet.mdm.apple.mdm")
	require.ErrorContains(t, err, "plist: cannot unmarshal dict into Go value of type []uint8")

	// mismatched struct type is acceptable, but result is empty
	type wrongStruct struct {
		foo string
		bar int
	}
	ws, err := getProfilePayloadContent[wrongStruct]("com.fleetdm.fleet.mdm.apple.mdm")
	require.NoError(t, err)
	require.NotNil(t, ws)
	require.Empty(t, ws.bar)
	require.Empty(t, ws.foo)

	// struct type is acceptable and returns the expected value type corresponds to the payload identifier
	c, err := getProfilePayloadContent[fleet.MDMAppleFleetdConfig]("com.fleetdm.fleetd.config")
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Equal(t, *c, fleet.MDMAppleFleetdConfig{EnrollSecret: "ENROLL_SECRET", FleetURL: "https://test.example.com"})

	// struct type is acceptable and returns the expected value type corresponds to the payload identifier
	e, err := getProfilePayloadContent[fleet.MDMCustomEnrollmentProfileItem]("com.fleetdm.fleet.mdm.apple.mdm")
	require.NoError(t, err)
	require.NotNil(t, e)
	require.Equal(t, *e, fleet.MDMCustomEnrollmentProfileItem{EndUserEmail: "user@example.com"})

	// map type is acceptable
	m, err := getProfilePayloadContent[map[string]any]("com.fleetdm.fleet.mdm.apple.mdm")
	require.NoError(t, err)
	require.NotNil(t, m)
	gotMap := *m
	v, ok := gotMap["EndUserEmail"]
	require.True(t, ok)
	require.Equal(t, "user@example.com", v)
	_, ok = gotMap["EnrollSecret"]
	require.False(t, ok)
	_, ok = gotMap["FleetURL"]
	require.False(t, ok)

	// map type is acceptable
	m2, err := getProfilePayloadContent[map[string]any]("com.fleetdm.fleetd.config")
	require.NoError(t, err)
	require.NotNil(t, m)
	gotMap = *m2
	v, ok = gotMap["EnrollSecret"]
	require.True(t, ok)
	require.Equal(t, "ENROLL_SECRET", v)
	v, ok = gotMap["FleetURL"]
	require.True(t, ok)
	require.Equal(t, "https://test.example.com", v)
	_, ok = gotMap["EndUserEmail"]
	require.False(t, ok)
}
