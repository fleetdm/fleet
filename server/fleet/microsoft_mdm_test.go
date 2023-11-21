package fleet

import (
	"encoding/xml"
	"testing"

	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
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
			name: "bad xml",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand:  []byte(`<Atomic><Replace><</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: "foo", Data: ptr.String(microsoft_mdm.CmdStatusAtomicFailed)},
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
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: "foo", Data: ptr.String("200")},
				"bar": {CmdID: "bar", Data: ptr.String("200")},
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
			name: "one operation failed",
			cmd: MDMWindowsCommand{
				CommandUUID: "foo",
				RawCommand: []byte(`
				<Atomic>
					<CmdID>foo</CmdID>
					<Replace><CmdID>bar</CmdID><Target><LocURI>./Device/Baz</LocURI></Target></Replace>
					<Replace><CmdID>baz</CmdID><Target><LocURI>./Bad/Loc</LocURI></Target></Replace>
				</Atomic>`),
			},
			statuses: map[string]SyncMLCmd{
				"foo": {CmdID: "foo", Data: ptr.String(microsoft_mdm.CmdStatusAtomicFailed)},
				"bar": {CmdID: "bar", Data: ptr.String(microsoft_mdm.CmdStatusOK)},
				"baz": {CmdID: "baz", Data: ptr.String(microsoft_mdm.CmdStatusBadRequest)},
			},
			hostUUID: "host-uuid",
			expectedPayload: &MDMWindowsProfilePayload{
				HostUUID:    "host-uuid",
				Status:      &MDMDeliveryFailed,
				Detail:      "CmdID bar: status 200, CmdID baz: status 400",
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
