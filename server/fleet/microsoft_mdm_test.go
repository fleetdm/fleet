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
		exclude := map[string]bool{
			"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": true,
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
		exclude := map[string]bool{
			"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock": true,
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
		exclude := map[string]bool{
			"./Device/Vendor/MSFT/BitLocker/A": true,
			"./Device/Vendor/MSFT/BitLocker/B": true,
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
		exclude := map[string]bool{
			"./Device/Vendor/MSFT/BitLocker/A": true,
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

func TestBuildMDMWindowsProfilePayloadFromMDMResponseRemoveOperation(t *testing.T) {
	t.Run("atomic remove with 405 is success", func(t *testing.T) {
		cmd := MDMWindowsCommand{
			CommandUUID: "cmd-1",
			RawCommand:  []byte(`<Atomic><CmdID>cmd-1</CmdID><Delete><CmdID>sub-1</CmdID><Item><Target><LocURI>./Device/Test</LocURI></Target></Item></Delete></Atomic>`),
		}
		statuses := map[string]SyncMLCmd{
			"cmd-1": {Data: new(syncml.CmdStatusNotAllowed)},
		}
		payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(cmd, statuses, "host-1", true)
		require.NoError(t, err)
		require.Equal(t, MDMDeliveryVerified, *payload.Status)
	})

	t.Run("non-atomic remove with mixed 200 and 404", func(t *testing.T) {
		cmdStr := new("Replace")
		cmd := MDMWindowsCommand{
			CommandUUID: "cmd-1",
			RawCommand: []byte(`<Delete><CmdID>del-1</CmdID><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Delete>` +
				`<Delete><CmdID>del-2</CmdID><Item><Target><LocURI>./Device/B</LocURI></Target></Item></Delete>`),
		}
		statuses := map[string]SyncMLCmd{
			"del-1": {Data: new(syncml.CmdStatusOK), Cmd: cmdStr},
			"del-2": {Data: new(syncml.CmdStatusNotFound), Cmd: cmdStr},
		}
		payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(cmd, statuses, "host-1", true)
		require.NoError(t, err)
		require.Equal(t, MDMDeliveryVerified, *payload.Status)
	})

	t.Run("non-atomic remove with 500 succeeds (best-effort)", func(t *testing.T) {
		cmdStr := new("Replace")
		cmd := MDMWindowsCommand{
			CommandUUID: "cmd-1",
			RawCommand: []byte(`<Delete><CmdID>del-1</CmdID><Item><Target><LocURI>./Device/A</LocURI></Target></Item></Delete>` +
				`<Delete><CmdID>del-2</CmdID><Item><Target><LocURI>./Device/B</LocURI></Target></Item></Delete>`),
		}
		statuses := map[string]SyncMLCmd{
			"del-1": {Data: new(syncml.CmdStatusOK), Cmd: cmdStr},
			"del-2": {Data: new(syncml.CmdStatusCommandFailed), Cmd: cmdStr},
		}
		payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(cmd, statuses, "host-1", true)
		require.NoError(t, err)
		require.Equal(t, MDMDeliveryVerified, *payload.Status)
	})
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
}
