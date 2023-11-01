package fleet

import (
	"encoding/xml"
	"testing"

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
