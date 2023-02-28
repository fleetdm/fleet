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
		{&emptyOutput, nil, nil, ErrNotFound},
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
