//go:build darwin

package profiles

import (
	"bytes"
	"errors"
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
		wantErr string
	}{
		{nil, testErr, nil, testErr.Error()},
		{ptr.String("invalid-json"), nil, nil, "unmarshaling configuration"},
		{ptr.String("{}"), nil, &fleet.MDMAppleFleetdConfig{}, ""},
		{
			ptr.String(`{"EnrollSecret": "ENROLL_SECRET", "FleetURL": "https://test.example.com", "EnableScripts": true}`),
			nil,
			&fleet.MDMAppleFleetdConfig{
				EnrollSecret:  "ENROLL_SECRET",
				FleetURL:      "https://test.example.com",
				EnableScripts: true,
			},
			"",
		},
		{
			ptr.String(`{"EnrollSecret": "ENROLL_SECRET", "FleetURL": "https://test.example.com", "EnableScripts": false}`),
			nil,
			&fleet.MDMAppleFleetdConfig{
				EnrollSecret:  "ENROLL_SECRET",
				FleetURL:      "https://test.example.com",
				EnableScripts: false,
			},
			"",
		},
		{
			ptr.String(`{"EnableScripts": true}`),
			nil,
			&fleet.MDMAppleFleetdConfig{EnableScripts: true},
			"",
		},
		{
			ptr.String(`{"EnrollSecret": "ENROLL_SECRET", "FleetURL": ""}`),
			nil,
			&fleet.MDMAppleFleetdConfig{EnrollSecret: "ENROLL_SECRET"},
			"",
		},
		{
			ptr.String(`{"EnrollSecret": "", "FleetURL": "https://test.example.com"}`),
			nil,
			&fleet.MDMAppleFleetdConfig{FleetURL: "https://test.example.com"},
			"",
		},
	}

	origExecScript := execScript
	t.Cleanup(func() { execScript = origExecScript })
	for _, c := range cases {
		execScript = func(script string) (*bytes.Buffer, error) {
			if c.cmdOut == nil {
				return nil, c.cmdErr
			}

			var buf bytes.Buffer
			buf.WriteString(*c.cmdOut)
			return &buf, nil
		}

		out, err := GetFleetdConfig()
		if c.wantErr != "" {
			require.ErrorContains(t, err, c.wantErr)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, c.wantOut, out)
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
