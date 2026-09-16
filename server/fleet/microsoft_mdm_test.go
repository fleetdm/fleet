package fleet

import (
	"encoding/xml"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsMDMCommand(t *testing.T) {
	cases := []struct {
		desc    string
		raw     string
		wantCmd SyncMLCmd
		wantErr string
	}{
		{"not xml", "zzz", SyncMLCmd{}, "The payload isn't valid XML"},
		{"multi Exec top-level", `<Exec></Exec><Exec></Exec>`, SyncMLCmd{}, "You can run only a single <Exec> command"},
		{"not Exec", `<Get></Get>`, SyncMLCmd{}, "You can run only <Exec> command type"},
		{"valid Exec", `<Exec><Item><Target><LocURI>./test</LocURI></Target></Item></Exec>`, SyncMLCmd{
			XMLName: xml.Name{Local: "Exec"},
			Items: []CmdItem{
				{Target: ptr.String("./test")},
			},
		}, ""},
		{"valid Exec with spaces", `
			<Exec>
				<Item>
					<Target>
						<LocURI>./test</LocURI>
					</Target>
				</Item>
			</Exec>`, SyncMLCmd{
			XMLName: xml.Name{Local: "Exec"},
			Items: []CmdItem{
				{Target: ptr.String("./test")},
			},
		}, ""},
		{"Exec with multiple Items", `
			<Exec>
				<Item>
					<Target>
						<LocURI>./test</LocURI>
					</Target>
				</Item>
				<Item>
					<Target>
						<LocURI>./test2</LocURI>
					</Target>
				</Item>
			</Exec>`, SyncMLCmd{}, "You can run only a single <Exec> command"},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got, err := ParseWindowsMDMCommand([]byte(c.raw))
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, c.wantCmd, *got)
			}
		})
	}
}

func TestBuildMDMWindowsProfilePayloadFromMDMResponse(t *testing.T) {
	tests := []struct {
		name            string
		cmd             MDMWindowsCommand
		statuses        map[string]SyncMLCmd
		hostUUID        string
		expectedError   string
		expectedPayload *MDMWindowsProfilePayload
	}{
		{
			name: "no commands found",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
			},
			statuses:      map[string]SyncMLCmd{},
			hostUUID:      "host-uuid",
			expectedError: "no commands found in profile",
		},
		{
			name: "missing status for command",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand:  []byte(`<Atomic><Replace></Replace></Atomic>`),
			},
			statuses:      map[string]SyncMLCmd{},
			hostUUID:      "host-uuid",
			expectedError: "missing status for root command",
		},
		{
			name: "bad xml replace",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand:  []byte(`<Atomic><Replace><</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String(syncml.CmdStatusAtomicFailed)},
			},
			hostUUID:      "host-uuid",
			expectedError: "XML syntax error",
		},
		{
			name: "bad xml add",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand:  []byte(`<Atomic><Add><</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String(syncml.CmdStatusAtomicFailed)},
			},
			hostUUID:      "host-uuid",
			expectedError: "XML syntax error",
		},
		{
			name: "all operations succeded",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>bar</CmdID><Target><LocURI>./Device/Baz</LocURI></Target></Replace>
					<Add><CmdID>baz</CmdID><Target><LocURI>./Device/Baz</LocURI></Target></Add>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String("200")},
				"bar": {CmdID: CmdID{Value: "bar"}, Data: ptr.String("200")},
				"baz": {CmdID: CmdID{Value: "baz"}, Data: ptr.String("200")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryVerified,
				Detail:      "",
				CommandUUID: "foo",
			},
		},
		{
			name: "two operations failed",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>bar</CmdID><Item><Target><LocURI>./Device/Baz</LocURI></Target></Item></Replace>
					<Replace><CmdID>baz</CmdID><Item><Target><LocURI>./Bad/Loc</LocURI></Target></Item></Replace>
					<Add><CmdID>other</CmdID><Item><Target><LocURI>./Bad/Other</LocURI></Target></Item></Add>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo":   {CmdID: CmdID{Value: "foo"}, Data: ptr.String(syncml.CmdStatusAtomicFailed)},
				"bar":   {CmdID: CmdID{Value: "bar"}, Data: ptr.String(syncml.CmdStatusOK)},
				"baz":   {CmdID: CmdID{Value: "baz"}, Data: ptr.String(syncml.CmdStatusBadRequest)},
				"other": {CmdID: CmdID{Value: "other"}, Data: ptr.String(syncml.CmdStatusBadRequest)},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryFailed,
				Detail:      "./Device/Baz: status 200, ./Bad/Loc: status 400, ./Bad/Other: status 400",
				CommandUUID: "foo",
			},
		},
		{
			name: "scep profile gets verified",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>bar</CmdID><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP</LocURI></Target></Replace>
					<Add><CmdID>baz</CmdID><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP</LocURI></Target></Add>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String("200")},
				"bar": {CmdID: CmdID{Value: "bar"}, Data: ptr.String("200")},
				"baz": {CmdID: CmdID{Value: "baz"}, Data: ptr.String("200")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryVerified,
				Detail:      "",
				CommandUUID: "foo",
			},
		},
		{
			name: "full user-scoped profile gets verified",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>bar</CmdID><Target><LocURI>./User/My-Custom-Loc-URI-Path</LocURI></Target></Replace>
					<Add><CmdID>baz</CmdID><Target><LocURI>./User/My-Custom-Loc-URI-Path-Second</LocURI></Target></Add>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String("200")},
				"bar": {CmdID: CmdID{Value: "bar"}, Data: ptr.String("200")},
				"baz": {CmdID: CmdID{Value: "baz"}, Data: ptr.String("200")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryVerified,
				Detail:      "",
				CommandUUID: "foo",
			},
		},
		{
			name: "mix of user-scoped profile and device-scoped profile gets verifying",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>foobar</CmdID><Target><LocURI>./Vendor/My-Custom-Loc-URI-Path-First</LocURI></Target></Replace>
					<Replace><CmdID>bar</CmdID><Target><LocURI>./Device/My-Custom-Loc-URI-Path</LocURI></Target></Replace>
					<Add><CmdID>baz</CmdID><Target><LocURI>./User/My-Custom-Loc-URI-Path-Second</LocURI></Target></Add>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo":    {CmdID: CmdID{Value: "foo"}, Data: ptr.String("200")},
				"bar":    {CmdID: CmdID{Value: "bar"}, Data: ptr.String("200")},
				"foobar": {CmdID: CmdID{Value: "foobar"}, Data: ptr.String("200")},
				"baz":    {CmdID: CmdID{Value: "baz"}, Data: ptr.String("200")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryVerified,
				Detail:      "",
				CommandUUID: "foo",
			},
		},
		{
			name: "multiple non-atomic commands with a failure",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Add>
					<CmdID>foo</CmdID>
					<Item>
						<Target><LocURI>./Device/First</LocURI></Target>
					</Item>
				</Add>
				<Replace>
					<CmdID>bar</CmdID>
					<Item>
						<Target><LocURI>./Device/Second</LocURI></Target>
					</Item>
				</Replace>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: CmdID{Value: "foo"}, Data: ptr.String("200")},
				"bar": {CmdID: CmdID{Value: "bar"}, Data: ptr.String("400")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryFailed,
				Detail:      "./Device/First: status 200, ./Device/Second: status 400",
				CommandUUID: "foo",
			},
		},
		{
			// The signature measured on hardware: the SCEP root node Add is rejected, its sibling reports 216
			// (atomic rollback), and the Atomic itself reports 507.
			name: "user-channel write rejected with 500",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand: []byte(`<Atomic><CmdID>cmd-1</CmdID>` +
					`<Add><CmdID>add-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x</LocURI></Target></Item></Add>` +
					`<Add><CmdID>add-2</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x/Install/ServerURL</LocURI></Target></Item></Add>` +
					`</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"cmd-1": {Data: new(syncml.CmdStatusAtomicFailed)},
				"add-1": {Data: new(syncml.CmdStatusCommandFailed), Cmd: new("Add")},
				"add-2": {Data: new(syncml.CmdStatusAtomicRollbackAccepted), Cmd: new("Add")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID: "host-uuid",
				Status:   &MDMDeliveryFailed,
				Detail: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x: status 500, " +
					"./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x/Install/ServerURL: status 216",
				CommandUUID:         "cmd-1",
				UserChannelRejected: true,
			},
		},
		{
			// 404 observed live on a device-bound (fleetd) enrollment with nobody signed in: the "./User/..." node has no
			// user to resolve to. On a failed install it counts as a user-context rejection.
			name: "user-channel write rejected with 404",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand:  []byte(`<Replace><CmdID>rep-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/InternetExplorer/A</LocURI></Target></Item></Replace>`),
			},
			statuses: map[string]SyncMLCmd{
				"rep-1": {Data: new(syncml.CmdStatusNotFound), Cmd: new("Replace")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:            "host-uuid",
				Status:              &MDMDeliveryFailed,
				Detail:              "./User/Vendor/MSFT/Policy/Config/InternetExplorer/A: status 404",
				CommandUUID:         "cmd-1",
				UserChannelRejected: true,
			},
		},
		{
			// The same 404 on a device-channel target is an ordinary failure, not a user-context rejection.
			name: "device-channel 404 is not a user-context rejection",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand:  []byte(`<Replace><CmdID>rep-1</CmdID><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/InternetExplorer/A</LocURI></Target></Item></Replace>`),
			},
			statuses: map[string]SyncMLCmd{
				"rep-1": {Data: new(syncml.CmdStatusNotFound), Cmd: new("Replace")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryFailed,
				Detail:      "./Device/Vendor/MSFT/Policy/Config/InternetExplorer/A: status 404",
				CommandUUID: "cmd-1",
			},
		},
		{
			name: "user-channel write rejected with 405",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand:  []byte(`<Replace><CmdID>rep-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/Experience/A</LocURI></Target></Item></Replace>`),
			},
			statuses: map[string]SyncMLCmd{
				"rep-1": {Data: new(syncml.CmdStatusNotAllowed), Cmd: new("Replace")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:            "host-uuid",
				Status:              &MDMDeliveryFailed,
				Detail:              "./User/Vendor/MSFT/Policy/Config/Experience/A: status 405",
				CommandUUID:         "cmd-1",
				UserChannelRejected: true,
			},
		},
		{
			// 418 means the node already existed, which only happens when the user channel IS writable, so an Atomic
			// that rolled back because of one must not read as a user-context rejection. The already-exists resend
			// path recovers from it.
			name: "atomic rollback from a nested 418 is not a user-channel rejection",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand: []byte(`<Atomic><CmdID>cmd-1</CmdID>` +
					`<Add><CmdID>add-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/A</LocURI></Target></Item></Add>` +
					`</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"cmd-1": {Data: new(syncml.CmdStatusAtomicFailed)},
				"add-1": {Data: new(syncml.CmdStatusAlreadyExists), Cmd: new("Add")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:            "host-uuid",
				Status:              &MDMDeliveryFailed,
				Detail:              "./User/Vendor/MSFT/Policy/Config/A: status 418",
				CommandUUID:         "cmd-1",
				UserChannelRejected: false,
			},
		},
		{
			// A SCEP profile's user-channel write is the Exec on Install/Enroll, so leaving Exec out of the failure
			// scan would both hide the failing command and miss the rejection.
			name: "user-channel Exec rejected inside an atomic",
			cmd: MDMWindowsCommand{
				CommandUUID: "cmd-1",
				RawCommand: []byte(`<Atomic><CmdID>cmd-1</CmdID>` +
					`<Add><CmdID>add-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x</LocURI></Target></Item></Add>` +
					`<Exec><CmdID>exec-1</CmdID><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x/Install/Enroll</LocURI></Target></Item></Exec>` +
					`</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"cmd-1":  {Data: new(syncml.CmdStatusAtomicFailed)},
				"add-1":  {Data: new(syncml.CmdStatusAtomicRollbackAccepted), Cmd: new("Add")},
				"exec-1": {Data: new(syncml.CmdStatusCommandFailed), Cmd: new("Exec")},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID: "host-uuid",
				Status:   &MDMDeliveryFailed,
				Detail: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x: status 216, " +
					"./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x/Install/Enroll: status 500",
				CommandUUID:         "cmd-1",
				UserChannelRejected: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(tt.cmd, tt.statuses, tt.hostUUID, false)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedPayload, payload)
			}
		})
	}
}

func TestWindowsResponseToDeliveryStatus(t *testing.T) {
	tests := []struct {
		name     string
		resp     string
		expected MDMDeliveryStatus
	}{
		{
			name:     "response starts with 2",
			resp:     "202",
			expected: MDMDeliveryVerified,
		},
		{
			name:     "bad requests",
			resp:     "400",
			expected: MDMDeliveryFailed,
		},
		{
			name:     "errors",
			resp:     "500",
			expected: MDMDeliveryFailed,
		},
		{
			name:     "empty response",
			resp:     "",
			expected: MDMDeliveryPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WindowsResponseToDeliveryStatus(tt.resp)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestCmdIDMarshalXML(t *testing.T) {
	tests := []struct {
		name        string
		cmdID       CmdID
		expectedXML string
		expectError bool
	}{
		{
			name: "WithComment",
			cmdID: CmdID{
				Value:               "123",
				IncludeFleetComment: true,
			},
			expectedXML: "<!-- CmdID generated by Fleet --><CmdID>123</CmdID>",
			expectError: false,
		},
		{
			name: "WithoutComment",
			cmdID: CmdID{
				Value:               "456",
				IncludeFleetComment: false,
			},
			expectedXML: "<CmdID>456</CmdID>",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := xml.MarshalIndent(tt.cmdID, "", "  ")
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedXML, string(output))
			}
		})
	}
}

func TestCmdIDUnmarshalXML(t *testing.T) {
	tests := []struct {
		name        string
		xmlData     string
		expectedCmd CmdID
		expectError bool
	}{
		{
			name:        "ValidCmdID",
			xmlData:     "<CmdID>123</CmdID>",
			expectedCmd: CmdID{Value: "123"},
			expectError: false,
		},
		{
			name:        "InvalidXML",
			xmlData:     "<CmdID>invalid",
			expectedCmd: CmdID{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd CmdID
			err := xml.Unmarshal([]byte(tt.xmlData), &cmd)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedCmd, cmd)
			}
		})
	}
}

func TestBuildDeleteCommandFromProfileBytes(t *testing.T) {
	tests := []struct {
		name        string
		profileXML  string
		expectError string
		expectNil   bool
		// checkFn is called on the resulting command for custom assertions
		checkFn func(t *testing.T, cmd *MDMWindowsCommand)
	}{
		{
			name: "single Replace command",
			profileXML: `<Replace>
				<CmdID>1</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Browser/AllowDoNotTrack</LocURI></Target>
					<Meta><Format xmlns="syncml:metinf">int</Format></Meta>
					<Data>1</Data>
				</Item>
			</Replace>`,
			checkFn: func(t *testing.T, cmd *MDMWindowsCommand) {
				require.Equal(t, "test-uuid-123", cmd.CommandUUID)
				require.Empty(t, cmd.TargetLocURI)
				cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
				require.NoError(t, err)
				require.Len(t, cmds, 1)
				require.Equal(t, CmdDelete, cmds[0].XMLName.Local)
				require.Equal(t, "./Device/Vendor/MSFT/Policy/Config/Browser/AllowDoNotTrack", cmds[0].GetTargetURI())
			},
		},
		{
			name: "atomic profile with multiple Replace commands produces individual Deletes (not Atomic)",
			profileXML: `<Atomic>
				<CmdID>1</CmdID>
				<Replace>
					<CmdID>2</CmdID>
					<Item>
						<Target><LocURI>./Device/Vendor/MSFT/BitLocker/RequireStorageCardEncryption</LocURI></Target>
						<Data>1</Data>
					</Item>
				</Replace>
				<Replace>
					<CmdID>3</CmdID>
					<Item>
						<Target><LocURI>./Device/Vendor/MSFT/BitLocker/RequireDeviceEncryption</LocURI></Target>
						<Data>1</Data>
					</Item>
				</Replace>
			</Atomic>`,
			checkFn: func(t *testing.T, cmd *MDMWindowsCommand) {
				// Delete commands should NOT be wrapped in <Atomic>. Removal
				// is best-effort, and Atomic would cause all to fail if one fails.
				cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
				require.NoError(t, err)
				require.Len(t, cmds, 2)
				for _, c := range cmds {
					require.Equal(t, CmdDelete, c.XMLName.Local)
				}
				require.Equal(t, "./Device/Vendor/MSFT/BitLocker/RequireStorageCardEncryption", cmds[0].GetTargetURI())
				require.Equal(t, "./Device/Vendor/MSFT/BitLocker/RequireDeviceEncryption", cmds[1].GetTargetURI())
			},
		},
		{
			name: "multiple top-level Replace commands (non-atomic)",
			profileXML: `<Replace>
				<CmdID>1</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ActiveHoursStart</LocURI></Target>
					<Data>8</Data>
				</Item>
			</Replace>
			<Replace>
				<CmdID>2</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/ActiveHoursEnd</LocURI></Target>
					<Data>17</Data>
				</Item>
			</Replace>`,
			checkFn: func(t *testing.T, cmd *MDMWindowsCommand) {
				cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
				require.NoError(t, err)
				require.Len(t, cmds, 2)
				for _, c := range cmds {
					require.Equal(t, CmdDelete, c.XMLName.Local)
				}
				require.Equal(t, "./Device/Vendor/MSFT/Policy/Config/Update/ActiveHoursStart", cmds[0].GetTargetURI())
				require.Equal(t, "./Device/Vendor/MSFT/Policy/Config/Update/ActiveHoursEnd", cmds[1].GetTargetURI())
			},
		},
		{
			name: "atomic profile with Add and Exec commands skips Exec",
			profileXML: `<Atomic>
				<CmdID>1</CmdID>
				<Add>
					<CmdID>2</CmdID>
					<Item>
						<Target><LocURI>./Device/Vendor/MSFT/VPNv2/MyVPN/ProfileXML</LocURI></Target>
						<Data>vpn-config</Data>
					</Item>
				</Add>
				<Exec>
					<CmdID>3</CmdID>
					<Item>
						<Target><LocURI>./Device/Vendor/MSFT/VPNv2/MyVPN/Connect</LocURI></Target>
					</Item>
				</Exec>
			</Atomic>`,
			checkFn: func(t *testing.T, cmd *MDMWindowsCommand) {
				// Only the Add should produce a Delete; Exec is skipped.
				// Delete is NOT wrapped in Atomic (best-effort removal).
				cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
				require.NoError(t, err)
				require.Len(t, cmds, 1)
				require.Equal(t, CmdDelete, cmds[0].XMLName.Local)
				require.Equal(t, "./Device/Vendor/MSFT/VPNv2/MyVPN/ProfileXML", cmds[0].GetTargetURI())
			},
		},
		{
			name: "single Exec command returns nil",
			profileXML: `<Exec>
				<CmdID>1</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/RemoteWipe/doWipe</LocURI></Target>
				</Item>
			</Exec>`,
			expectNil: true,
		},
		{
			name: "atomic profile with only Exec commands returns nil",
			profileXML: `<Atomic>
				<CmdID>1</CmdID>
				<Exec>
					<CmdID>2</CmdID>
					<Item>
						<Target><LocURI>./Device/Vendor/MSFT/RemoteWipe/doWipe</LocURI></Target>
					</Item>
				</Exec>
			</Atomic>`,
			expectNil: true,
		},
		{
			name:       "empty profile returns nil",
			profileXML: "",
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := BuildDeleteCommandFromProfileBytes([]byte(tt.profileXML), "test-uuid-123", "test-profile-uuid")
			switch {
			case tt.expectError != "":
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectError)
			case tt.expectNil:
				require.NoError(t, err)
				require.Nil(t, cmd)
			default:
				require.NoError(t, err)
				require.NotNil(t, cmd)
				if tt.checkFn != nil {
					tt.checkFn(t, cmd)
				}
			}
		})
	}
}

func TestBuildDeleteCommandExcludesProtectedLocURIs(t *testing.T) {
	// Bug 005: When two profiles target the same LocURI and one is deleted,
	// the <Delete> should NOT include LocURIs that are still targeted by
	// another active profile.

	t.Run("single Replace with protected LocURI returns nil", func(t *testing.T) {
		profileXML := `<Replace>
			<CmdID>1</CmdID>
			<Item>
				<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target>
				<Data>5</Data>
			</Item>
		</Replace>`
		exclude := map[string]struct{}{
			"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": {},
		}
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "test-uuid", "test-profile-uuid", exclude)
		require.NoError(t, err)
		require.Nil(t, cmd, "should return nil when the only LocURI is protected")
	})

	t.Run("multi-Replace with one protected LocURI", func(t *testing.T) {
		profileXML := `<Replace>
			<CmdID>1</CmdID>
			<Item>
				<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target>
				<Data>5</Data>
			</Item>
		</Replace>
		<Replace>
			<CmdID>2</CmdID>
			<Item>
				<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled</LocURI></Target>
				<Data>0</Data>
			</Item>
		</Replace>`
		// Only MaxInactivityTimeDeviceLock is protected; DevicePasswordEnabled should still get a <Delete>
		exclude := map[string]struct{}{
			"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": {},
		}
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "test-uuid", "test-profile-uuid", exclude)
		require.NoError(t, err)
		require.NotNil(t, cmd, "should generate a command for the non-protected LocURI")

		cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
		require.NoError(t, err)
		require.Len(t, cmds, 1, "should have exactly one Delete command")
		require.Equal(t, CmdDelete, cmds[0].XMLName.Local)
		require.Equal(t, "./Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled", cmds[0].GetTargetURI())
	})

	t.Run("atomic with all protected LocURIs returns nil", func(t *testing.T) {
		profileXML := `<Atomic>
			<CmdID>1</CmdID>
			<Replace>
				<CmdID>2</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/BitLocker/A</LocURI></Target>
					<Data>1</Data>
				</Item>
			</Replace>
			<Replace>
				<CmdID>3</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/BitLocker/B</LocURI></Target>
					<Data>1</Data>
				</Item>
			</Replace>
		</Atomic>`
		exclude := map[string]struct{}{
			"./Device/Vendor/MSFT/BitLocker/A": {},
			"./Device/Vendor/MSFT/BitLocker/B": {},
		}
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "test-uuid", "test-profile-uuid", exclude)
		require.NoError(t, err)
		require.Nil(t, cmd, "should return nil when all atomic LocURIs are protected")
	})

	t.Run("atomic with partial protection keeps unprotected", func(t *testing.T) {
		profileXML := `<Atomic>
			<CmdID>1</CmdID>
			<Replace>
				<CmdID>2</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/BitLocker/A</LocURI></Target>
					<Data>1</Data>
				</Item>
			</Replace>
			<Replace>
				<CmdID>3</CmdID>
				<Item>
					<Target><LocURI>./Device/Vendor/MSFT/BitLocker/B</LocURI></Target>
					<Data>1</Data>
				</Item>
			</Replace>
		</Atomic>`
		exclude := map[string]struct{}{
			"./Device/Vendor/MSFT/BitLocker/A": {},
		}
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "test-uuid", "test-profile-uuid", exclude)
		require.NoError(t, err)
		require.NotNil(t, cmd)

		// Delete is NOT wrapped in Atomic (best-effort removal).
		cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
		require.NoError(t, err)
		require.Len(t, cmds, 1, "should only have one Delete for the unprotected URI")
		require.Equal(t, CmdDelete, cmds[0].XMLName.Local)
		require.Equal(t, "./Device/Vendor/MSFT/BitLocker/B", cmds[0].GetTargetURI())
	})

	t.Run("no exclusions works as before", func(t *testing.T) {
		profileXML := `<Replace>
			<CmdID>1</CmdID>
			<Item>
				<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock</LocURI></Target>
				<Data>5</Data>
			</Item>
		</Replace>`
		// No exclude parameter — should generate the Delete normally
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "test-uuid", "test-profile-uuid")
		require.NoError(t, err)
		require.NotNil(t, cmd)
	})

	t.Run("SCEP variable in LocURI is substituted with profile UUID", func(t *testing.T) {
		profileXML := `<Add><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI></Target></Item></Add>
<Add><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/ServerURL</LocURI></Target><Data>https://example.com</Data></Item></Add>`
		cmd, err := BuildDeleteCommandFromProfileBytes([]byte(profileXML), "cmd-uuid", "w-my-profile-uuid")
		require.NoError(t, err)
		require.NotNil(t, cmd)

		cmds, err := UnmarshallMultiTopLevelXMLProfile(cmd.RawCommand)
		require.NoError(t, err)
		require.Len(t, cmds, 2)
		// The $FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID should be replaced with the profile UUID
		require.Equal(t, "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/w-my-profile-uuid", cmds[0].GetTargetURI())
		require.Equal(t, "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/w-my-profile-uuid/Install/ServerURL", cmds[1].GetTargetURI())
		// Must NOT contain the variable literal
		require.NotContains(t, string(cmd.RawCommand), "$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID")
	})
}

func TestWindowsResponseToDeliveryStatusForRemove(t *testing.T) {
	tests := []struct {
		resp     string
		expected MDMDeliveryStatus
	}{
		{syncml.CmdStatusOK, MDMDeliveryVerified},
		{syncml.CmdStatusAcceptedForProcessing, MDMDeliveryVerified},
		{syncml.CmdStatusAtomicRollbackAccepted, MDMDeliveryVerified},
		{syncml.CmdStatusNotFound, MDMDeliveryVerified},      // setting not on device
		{syncml.CmdStatusNotAllowed, MDMDeliveryVerified},    // read-only node per OMA-DM spec
		{syncml.CmdStatusCommandFailed, MDMDeliveryVerified}, // Windows returns this for non-deletable CSP nodes
		{syncml.CmdStatusBadRequest, MDMDeliveryFailed},      // genuine error
		{syncml.CmdStatusAtomicFailed, MDMDeliveryFailed},    // genuine error
		{"", MDMDeliveryPending},
	}

	for _, tt := range tests {
		t.Run(tt.resp, func(t *testing.T) {
			got := WindowsResponseToDeliveryStatusForRemove(tt.resp)
			require.Equal(t, tt.expected, got)
		})
	}
}

// TestBuildMDMWindowsProfilePayloadFromMDMResponseRemoveOperation covers how a removal's SyncML statuses become a delivery
// status, a detail string, and the user-channel rejection signal.
func TestBuildMDMWindowsProfilePayloadFromMDMResponseRemoveOperation(t *testing.T) {
	t.Parallel()

	const (
		deviceNode  = "./Device/Vendor/MSFT/Policy/Config/A"
		deviceNodeB = "./Device/Vendor/MSFT/Policy/Config/B"
		userNode    = "./User/Vendor/MSFT/Policy/Config/C"
	)
	deleteCmd := func(cmdID, locURI string) string {
		return `<Delete><CmdID>` + cmdID + `</CmdID><Item><Target><LocURI>` + locURI + `</LocURI></Target></Item></Delete>`
	}

	for _, tc := range []struct {
		name         string
		raw          string
		statuses     map[string]SyncMLCmd
		wantStatus   MDMDeliveryStatus
		wantRejected bool
		wantDetail   string
	}{
		{
			name:       "device node is read-only",
			raw:        deleteCmd("del-1", deviceNode),
			statuses:   map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusNotAllowed)}},
			wantStatus: MDMDeliveryVerified,
		},
		{
			name: "device node deleted, second already gone",
			raw:  deleteCmd("del-1", deviceNode) + deleteCmd("del-2", deviceNodeB),
			statuses: map[string]SyncMLCmd{
				"del-1": {Data: new(syncml.CmdStatusOK)},
				"del-2": {Data: new(syncml.CmdStatusNotFound)},
			},
			wantStatus: MDMDeliveryVerified,
		},
		{
			name: "device node does not support delete",
			raw:  deleteCmd("del-1", deviceNode) + deleteCmd("del-2", deviceNodeB),
			statuses: map[string]SyncMLCmd{
				"del-1": {Data: new(syncml.CmdStatusOK)},
				"del-2": {Data: new(syncml.CmdStatusCommandFailed)},
			},
			wantStatus: MDMDeliveryVerified,
		},
		{
			name:         "user node not allowed",
			raw:          deleteCmd("del-1", userNode),
			statuses:     map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusNotAllowed)}},
			wantStatus:   MDMDeliveryVerified,
			wantRejected: true,
		},
		{
			name:         "user node command failed",
			raw:          deleteCmd("del-1", userNode),
			statuses:     map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusCommandFailed)}},
			wantStatus:   MDMDeliveryVerified,
			wantRejected: true,
		},
		{
			// 404 on a user node is ambiguous: genuinely gone, or unreachable because nobody is signed in (observed live
			// on a device-bound enrollment). The payload flags it and the enrollment's user context state decides: held
			// while a user context is still awaited (the sign-out race; the gate never sends removals in that state
			// otherwise), completed as a real removal the rest of the time.
			name:         "user node not found flags the ambiguity for the state check",
			raw:          deleteCmd("del-1", userNode),
			statuses:     map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusNotFound)}},
			wantStatus:   MDMDeliveryVerified,
			wantRejected: true,
		},
		{
			name:       "user node deleted cleanly",
			raw:        deleteCmd("del-1", userNode),
			statuses:   map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusOK)}},
			wantStatus: MDMDeliveryVerified,
		},
		{
			// Deletes are never wrapped in <Atomic>, so a mixed removal partially succeeds: the device node goes, the user
			// node does not, and the whole command still reads as verified.
			name: "mixed removal with only the user node rejected",
			raw:  deleteCmd("del-1", deviceNode) + deleteCmd("del-2", userNode),
			statuses: map[string]SyncMLCmd{
				"del-1": {Data: new(syncml.CmdStatusOK)},
				"del-2": {Data: new(syncml.CmdStatusNotAllowed)},
			},
			wantStatus:   MDMDeliveryVerified,
			wantRejected: true,
		},
		{
			// Fleet does not build atomic removals, but the response parser handles them, so the nested walk is covered too.
			name: "atomic removal with a rejected user node",
			raw:  `<Atomic><CmdID>cmd-1</CmdID>` + deleteCmd("sub-1", userNode) + `</Atomic>`,
			statuses: map[string]SyncMLCmd{
				"cmd-1": {Data: new(syncml.CmdStatusNotAllowed)},
				"sub-1": {Data: new(syncml.CmdStatusNotAllowed)},
			},
			wantStatus:   MDMDeliveryVerified,
			wantRejected: true,
		},
		{
			// A code outside the best-effort set is a real failure, and only then does the per-LocURI text reach the detail.
			name:       "removal genuinely failed",
			raw:        deleteCmd("del-1", userNode),
			statuses:   map[string]SyncMLCmd{"del-1": {Data: new(syncml.CmdStatusBadRequest)}},
			wantStatus: MDMDeliveryFailed,
			wantDetail: userNode + ": status " + syncml.CmdStatusBadRequest,
		},
		{
			name: "failed removal also carrying a user-channel rejection",
			raw:  deleteCmd("del-1", deviceNode) + deleteCmd("del-2", userNode),
			statuses: map[string]SyncMLCmd{
				"del-1": {Data: new(syncml.CmdStatusBadRequest)},
				"del-2": {Data: new(syncml.CmdStatusNotAllowed)},
			},
			wantStatus:   MDMDeliveryFailed,
			wantRejected: true,
			wantDetail: deviceNode + ": status " + syncml.CmdStatusBadRequest + ", " +
				userNode + ": status " + syncml.CmdStatusNotAllowed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(
				MDMWindowsCommand{CommandUUID: "cmd-1", RawCommand: []byte(tc.raw)}, tc.statuses, "host-1", true)
			require.NoError(t, err)

			require.Equal(t, tc.wantStatus, *payload.Status)
			require.Equal(t, tc.wantRejected, payload.UserChannelRejected)
			// A removal that reads as success must not carry per-LocURI status text into the admin-visible detail.
			require.Equal(t, tc.wantDetail, payload.Detail)
		})
	}
}

func TestExtractLocURIsFromProfileBytes(t *testing.T) {
	t.Parallel()
	t.Run("atomic profile", func(t *testing.T) {
		xml := `<Atomic><Replace><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Replace><Replace><Item><Target><LocURI>./Device/B</LocURI></Target></Item></Replace></Atomic>`
		uris := ExtractLocURIsFromProfileBytes([]byte(xml))
		require.Equal(t, []string{"./Device/A", "./Device/B"}, uris)
	})

	t.Run("non-atomic profile", func(t *testing.T) {
		xml := `<Replace><Item><Target><LocURI>./Device/X</LocURI></Target></Item></Replace><Replace><Item><Target><LocURI>./Device/Y</LocURI></Target></Item></Replace>`
		uris := ExtractLocURIsFromProfileBytes([]byte(xml))
		require.Equal(t, []string{"./Device/X", "./Device/Y"}, uris)
	})

	t.Run("exec commands excluded", func(t *testing.T) {
		xml := `<Replace><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Replace><Exec><Item><Target><LocURI>./Device/Enroll</LocURI></Target></Item></Exec>`
		uris := ExtractLocURIsFromProfileBytes([]byte(xml))
		require.Equal(t, []string{"./Device/A"}, uris)
	})

	t.Run("empty profile", func(t *testing.T) {
		uris := ExtractLocURIsFromProfileBytes([]byte(""))
		require.Nil(t, uris)
	})

	t.Run("delete commands excluded", func(t *testing.T) {
		xml := `<Replace><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Replace><Delete><Item><Target><LocURI>./Device/B</LocURI></Target></Item></Delete>`
		uris := ExtractLocURIsFromProfileBytes([]byte(xml))
		require.Equal(t, []string{"./Device/A"}, uris)
	})

	t.Run("surrounding whitespace trimmed", func(t *testing.T) {
		// A formatting-only spelling change (e.g. an editor reflowing the LocURI text) must not make the same node look like a
		// different URI: edit-diffing would otherwise emit a <Delete> for a node the new version still enforces.
		xml := "<Replace><Item><Target><LocURI>\n\t\t./Device/A </LocURI></Target></Item></Replace><Atomic><Add><Item><Target><LocURI> ./Device/B\n</LocURI></Target></Item></Add></Atomic>"
		uris := ExtractLocURIsFromProfileBytes([]byte(xml))
		require.Equal(t, []string{"./Device/A", "./Device/B"}, uris)
	})
}

func TestWindowsProfileScopeFromBytes(t *testing.T) {
	t.Parallel()

	scepUserProfile := `<Add><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI></Target></Item></Add>` +
		`<Exec><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI></Target></Item></Exec>`
	scepDeviceProfile := `<Add><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI></Target></Item></Add>` +
		`<Exec><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI></Target></Item></Exec>`

	testCases := []struct {
		name    string
		profile string
		want    WindowsProfileScope
	}{
		{
			name:    "device scoped",
			profile: `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Experience/AllowCortana</LocURI></Target></Item></Replace>`,
			want:    WindowsProfileScopeDevice,
		},
		{
			name:    "scope-less spelling is device scoped",
			profile: `<Replace><Item><Target><LocURI>./Vendor/MSFT/Policy/Config/Experience/AllowCortana</LocURI></Target></Item></Replace>`,
			want:    WindowsProfileScopeDevice,
		},
		{
			name:    "user scoped",
			profile: `<Replace><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/Experience/AllowWindowsSpotlight</LocURI></Target></Item></Replace>`,
			want:    WindowsProfileScopeUser,
		},
		{
			name: "mixed scope is user scoped",
			profile: `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/A</LocURI></Target></Item></Replace>` +
				`<Replace><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/B</LocURI></Target></Item></Replace>`,
			want: WindowsProfileScopeUser,
		},
		{
			// Text-splitting cases. A raw prefix match on the profile bytes would miss these, which is why
			// classification parses the profile instead of scanning it. See the differential fixed in #49715.
			name:    "user target split by a CDATA section",
			profile: `<Replace><Item><Target><LocURI><![CDATA[./User/Vendor/MSFT/Policy/Config/A]]></LocURI></Target></Item></Replace>`,
			want:    WindowsProfileScopeUser,
		},
		{
			name:    "user target split by a comment",
			profile: `<Replace><Item><Target><LocURI>./User<!-- c -->/Vendor/MSFT/Policy/Config/A</LocURI></Target></Item></Replace>`,
			want:    WindowsProfileScopeUser,
		},
		{
			name: "atomic nested user target",
			profile: `<Atomic><Replace><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Replace>` +
				`<Add><Item><Target><LocURI>./User/B</LocURI></Target></Item></Add></Atomic>`,
			want: WindowsProfileScopeUser,
		},
		{
			// An Exec on a user node is a user-channel write even though it sets no persistent value, so unlike
			// ExtractLocURIsFromProfileBytes the classifier must not skip Exec.
			name:    "exec-only user target",
			profile: `<Exec><Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/abc/Install/Enroll</LocURI></Target></Item></Exec>`,
			want:    WindowsProfileScopeUser,
		},
		{
			name:    "scep user profile, atomic-wrapped on the fly",
			profile: scepUserProfile,
			want:    WindowsProfileScopeUser,
		},
		{
			name:    "scep device profile",
			profile: scepDeviceProfile,
			want:    WindowsProfileScopeDevice,
		},
		{
			name:    "whitespace around the loc uri",
			profile: "<Replace><Item><Target><LocURI>\n\t ./User/Vendor/MSFT/Policy/Config/A \n</LocURI></Target></Item></Replace>",
			want:    WindowsProfileScopeUser,
		},
		{
			// Delete is a verb an authored profile can carry, and a Delete against a user node is still a
			// user-channel write.
			name:    "delete targeting the user channel",
			profile: `<Delete><Item><Target><LocURI>./User/Vendor/MSFT/Policy/Config/A</LocURI></Target></Item></Delete>`,
			want:    WindowsProfileScopeUser,
		},
		{
			name: "atomic nested delete targeting the user channel",
			profile: `<Atomic><Replace><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Replace>` +
				`<Delete><Item><Target><LocURI>./User/B</LocURI></Target></Item></Delete></Atomic>`,
			want: WindowsProfileScopeUser,
		},
		{
			// A variable in a node name or value cannot change the channel, so it must not force the gate on. This
			// is the shape every real SCEP profile has.
			name:    "fleet variable below the scope segment stays device scoped",
			profile: `<Add><Item><Target><LocURI>./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</LocURI></Target></Item></Add>`,
			want:    WindowsProfileScopeDevice,
		},
		{
			name:    "empty profile",
			profile: "",
			want:    WindowsProfileScopeDevice,
		},
		{
			name:    "unparseable profile",
			profile: `<Replace><Item><Target><LocURI>./User/A`,
			want:    WindowsProfileScopeDevice,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, WindowsProfileScopeFromBytes([]byte(tc.profile)))
		})
	}
}

func TestCanonicalLocURI(t *testing.T) {
	t.Parallel()

	// All device-scoped spellings of the same node collapse to one comparison form.
	for _, spelling := range []string{
		"./Device/Vendor/MSFT/Policy/Config/X",
		"./Vendor/MSFT/Policy/Config/X",
		"Device/Vendor/MSFT/Policy/Config/X",
		"Vendor/MSFT/Policy/Config/X",
		" ./Device/Vendor/MSFT/Policy/Config/X\n",
	} {
		require.Equal(t, "Vendor/MSFT/Policy/Config/X", CanonicalLocURI(spelling), "spelling: %q", spelling)
	}
	// User scope is explicit and stays distinct from device scope.
	require.Equal(t, "User/Vendor/MSFT/X", CanonicalLocURI("./User/Vendor/MSFT/X"))
	// Case is preserved: CSP paths are documented case-sensitive.
	require.Equal(t, "vendor/msft/x", CanonicalLocURI("./Device/vendor/msft/x"))
	// Only a whole "Device" segment is a scope marker, not a prefix of the first segment.
	require.Equal(t, "DeviceLock/X", CanonicalLocURI("./DeviceLock/X"))
}

func TestLocURITargetsReservedNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		locURI   string
		reserved string
		want     bool
	}{
		{name: "explicit device scope", locURI: "./Device/Vendor/MSFT/BitLocker/RequireDeviceEncryption", reserved: syncml.FleetBitLockerTargetLocURI, want: true},
		{name: "scope-less (regression #48752)", locURI: "Vendor/MSFT/BitLocker/RequireDeviceEncryption", reserved: syncml.FleetBitLockerTargetLocURI, want: true},
		{name: "user scope matches via Contains, not a prefix check", locURI: "./User/Vendor/MSFT/BitLocker/Foo", reserved: syncml.FleetBitLockerTargetLocURI, want: true},
		{name: "surrounding whitespace", locURI: " Vendor/MSFT/BitLocker/Foo ", reserved: syncml.FleetBitLockerTargetLocURI, want: true},
		// Boundary safety: a longer sibling segment that merely shares the reserved-node prefix must not match, on either end.
		{name: "left boundary: node ending in Vendor is not reserved", locURI: "Custom/SomeVendor/MSFT/BitLocker/Foo", reserved: syncml.FleetBitLockerTargetLocURI, want: false},
		{name: "right boundary: BitLockerCustom sibling is not reserved", locURI: "Vendor/MSFT/BitLockerCustom/Foo", reserved: syncml.FleetBitLockerTargetLocURI, want: false},
		{name: "unrelated node", locURI: "./Device/Vendor/MSFT/DMClient/Foo", reserved: syncml.FleetBitLockerTargetLocURI, want: false},
		// The reserved node itself (no descendant leaf) matches (node-inclusive).
		{name: "bare BitLocker node matches", locURI: "./Device/Vendor/MSFT/BitLocker", reserved: syncml.FleetBitLockerTargetLocURI, want: true},
		// One positive smoke per reserved constant so a future typo/rename is caught.
		{name: "OS update", locURI: "Vendor/MSFT/Policy/Config/Update/AllowAutoUpdate", reserved: syncml.FleetOSUpdateTargetLocURI, want: true},
		{name: "RemoteWipe operation", locURI: "./Device/Vendor/MSFT/RemoteWipe/doWipe", reserved: syncml.FleetRemoteWipeTargetLocURI, want: true},
		// RemoteWipe is a wipe-only subtree: the bare node matches (node-inclusive), a sibling does not (boundary).
		{name: "bare RemoteWipe node matches (wipe-only subtree)", locURI: "Vendor/MSFT/RemoteWipe", reserved: syncml.FleetRemoteWipeTargetLocURI, want: true},
		{name: "RemoteWipeCustom sibling is not reserved", locURI: "Vendor/MSFT/RemoteWipeCustom/doWipe", reserved: syncml.FleetRemoteWipeTargetLocURI, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, LocURITargetsReservedNode(tt.locURI, tt.reserved))
		})
	}
}

func TestSyncMLCmdIsPremium(t *testing.T) {
	t.Parallel()

	newExecCmd := func(locURI string) SyncMLCmd {
		return SyncMLCmd{
			XMLName: xml.Name{Local: "Exec"},
			Items:   []CmdItem{{Target: new(locURI)}},
		}
	}
	tests := []struct {
		name   string
		locURI string
		want   bool
	}{
		{name: "explicit device wipe", locURI: "./Device/Vendor/MSFT/RemoteWipe/doWipe", want: true},
		{name: "scope-less wipe (regression #48752)", locURI: "Vendor/MSFT/RemoteWipe/doWipe", want: true},
		{name: "non-wipe command", locURI: "./DevDetail/SwV", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, newExecCmd(tt.locURI).IsPremium())
		})
	}
}

func TestProfileTargetsReservedLocURI(t *testing.T) {
	t.Parallel()

	osUpdate := syncml.FleetOSUpdateTargetLocURI
	tests := []struct {
		name   string
		syncML string
		want   bool
	}{
		{
			name:   "scoped OS update profile (fast path)",
			syncML: `<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Update/AllowAutoUpdate</LocURI></Target></Item></Replace>`,
			want:   true,
		},
		{
			name:   "scope-less OS update profile (regression #48752)",
			syncML: `<Replace><Item><Target><LocURI>Vendor/MSFT/Policy/Config/Update/AllowAutoUpdate</LocURI></Target></Item></Replace>`,
			want:   true,
		},
		{
			name:   "scope-less OS update inside Atomic",
			syncML: `<Atomic><Replace><Item><Target><LocURI>Vendor/MSFT/Policy/Config/Update/AllowAutoUpdate</LocURI></Target></Item></Replace></Atomic>`,
			want:   true,
		},
		{
			name:   "non-OS-update profile",
			syncML: `<Replace><Item><Target><LocURI>Vendor/MSFT/BitLocker/RequireDeviceEncryption</LocURI></Target></Item></Replace>`,
			want:   false,
		},
		{
			name:   "node ending in Update is not reserved",
			syncML: `<Replace><Item><Target><LocURI>Custom/Config/UpdatePolicy/AllowAutoUpdate</LocURI></Target></Item></Replace>`,
			want:   false,
		},
		{
			// Sibling segment sharing the reserved prefix must not be flagged (mentions the node name so the quick-reject
			// filter passes, forcing the boundary-aware per-LocURI check to make the call).
			name:   "UpdateExtra sibling segment is not reserved",
			syncML: `<Replace><Item><Target><LocURI>Vendor/MSFT/Policy/Config/UpdateExtra/AllowAutoUpdate</LocURI></Target></Item></Replace>`,
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ProfileTargetsReservedLocURI([]byte(tt.syncML), osUpdate))
		})
	}
}

func TestIsFleetInternalCmdID(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want bool
	}{
		{"empty string", "", false},
		{"random UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"unrelated prefix", "foo-internal-bar", false},
		{"prefix only", FleetInternalCmdIDPrefix, true},
		{"devdetail link probe", FleetInternalCmdIDPrefix + "devdetail-smbios-serial", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsFleetInternalCmdID(tc.in))
		})
	}
}

func TestEnrollmentVersionAtLeast(t *testing.T) {
	const minVersion = syncml.MinSupportedEnrollmentVersion // "4.0"

	for _, tc := range []struct {
		name    string
		version string
		want    bool
		wantErr string
	}{
		{"minimum version", "4.0", true, ""},
		{"historically supported 5.0", "5.0", true, ""},
		{"historically supported 7.0", "7.0", true, ""},
		{"newer 8.0", "8.0", true, ""},
		{"windows 11 25H2 9.0", "9.0", true, ""},
		{"double-digit major 10.0 above 9.0", "10.0", true, ""},
		{"below minimum 3.0", "3.0", false, ""},
		{"below minimum 3.9", "3.9", false, ""},
		{"higher minor same major", "4.1", true, ""},
		{"major only equal", "4", true, ""},
		{"empty", "", false, "version is empty"},
		{"non-numeric", "abc", false, `invalid version component "abc"`},
		{"non-numeric minor", "4.x", false, `invalid version component "x"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := enrollmentVersionAtLeast(tc.version, minVersion)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
