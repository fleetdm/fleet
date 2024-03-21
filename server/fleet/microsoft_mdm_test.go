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
			name: "missing status for command",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
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
				Status:      &MDMDeliveryVerifying,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := BuildMDMWindowsProfilePayloadFromMDMResponse(tt.cmd, tt.statuses, tt.hostUUID)

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
			expected: MDMDeliveryVerifying,
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
