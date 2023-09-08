//go:build darwin

package profiles

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
)

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
		wantOut bool
		wantErr error
	}{
		{
			"command error",
			nil,
			errors.New("some command error"),
			false,
			errors.New("some command error"),
		},
		{
			"empty output",
			ptr.String(""),
			nil,
			false,
			errors.New("parsing profiles output: expected at least 2 lines but got 1"),
		},
		{
			"null profile",
			ptr.String(`Device Enrollment configuration:
(null)
		`),
			nil,
			false,
			errors.New("parsing profiles output: received null device enrollment configuration"),
		},
		{
			"mismatch profile",
			ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	...
}
			`),
			nil,
			false,
			errors.New(`configuration web url: expected 'valid.com' but found 'test.example.com'`),
		},
		{
			"match profile",
			ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.example.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://valid.com?token=1234";
	...
}
			`),
			nil,
			false,
			nil,
		},
		{
			"mixed case match",
			ptr.String(`Device Enrollment configuration:
{
    AllowPairing = 1;
	AutoAdvanceSetup = 0;
	AwaitDeviceConfigured = 0;
	ConfigurationURL = "https://test.ExaMplE.com/mdm/apple/enroll?token=1234";
	ConfigurationWebURL = "https://vaLiD.com?tOken=1234";
	...
}
			`),
			nil,
			false,
			nil,
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
