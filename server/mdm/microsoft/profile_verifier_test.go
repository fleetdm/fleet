package microsoft_mdm

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLoopHostMDMLocURIs(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()

	ds.GetHostMDMProfilesExpectedForVerificationFunc = func(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
		return map[string]*fleet.ExpectedMDMProfile{
			"N1": {Name: "N1", RawProfile: syncml.ForTestWithData(map[string]string{"L1": "D1"})},
			"N2": {Name: "N2", RawProfile: syncml.ForTestWithData(map[string]string{"L2": "D2"})},
			"N3": {Name: "N3", RawProfile: syncml.ForTestWithData(map[string]string{"L3": "D3", "L3.1": "D3.1"})},
		}, nil
	}
	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
		return document, nil
	}

	type wantStruct struct {
		locURI      string
		data        string
		profileUUID string
		uniqueHash  string
	}
	got := []wantStruct{}
	err := LoopOverExpectedHostProfiles(ctx, ds, &fleet.Host{}, func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string) {
		got = append(got, wantStruct{
			locURI:      locURI,
			data:        data,
			profileUUID: profile.Name,
			uniqueHash:  hash,
		})
	})
	require.NoError(t, err)
	require.ElementsMatch(
		t,
		[]wantStruct{
			{"L1", "D1", "N1", "1255198959"},
			{"L2", "D2", "N2", "2736786183"},
			{"L3", "D3", "N3", "894211447"},
			{"L3.1", "D3.1", "N3", "3410477854"},
		},
		got,
	)
}

func TestHashLocURI(t *testing.T) {
	testCases := []struct {
		name           string
		profileName    string
		locURI         string
		expectNotEmpty bool
	}{
		{
			name:           "basic functionality",
			profileName:    "profile1",
			locURI:         "uri1",
			expectNotEmpty: true,
		},
		{
			name:           "empty strings",
			profileName:    "",
			locURI:         "",
			expectNotEmpty: true,
		},
		{
			name:           "special characters",
			profileName:    "profile!@#",
			locURI:         "uri$%^",
			expectNotEmpty: true,
		},
		{
			name:           "long string input",
			profileName:    string(make([]rune, 1000)),
			locURI:         string(make([]rune, 1000)),
			expectNotEmpty: true,
		},
		{
			name:           "non-ASCII characters",
			profileName:    "プロファイル",
			locURI:         "URI",
			expectNotEmpty: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash := HashLocURI(tc.profileName, tc.locURI)
			if tc.expectNotEmpty {
				require.NotEmpty(t, hash, "hash should not be empty")
			}
		})
	}
}

func TestVerifyHostMDMProfilesErrors(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	host := &fleet.Host{}

	err := VerifyHostMDMProfiles(ctx, ds, host, []byte{})
	require.ErrorIs(t, err, io.EOF)
}

func TestVerifyHostMDMProfilesHappyPaths(t *testing.T) {
	cases := []struct {
		name              string
		hostProfiles      []hostProfile
		report            []osqueryReport
		toVerify          []string
		toFail            []string
		toRetry           []string
		withinGracePeriod bool
	}{
		{
			name:         "profile reported, but host doesn't have any",
			hostProfiles: nil,
			report:       []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify:     []string{},
			toFail:       []string{},
			toRetry:      []string{},
		},
		{
			name: "single profile reported and verified",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "D1"}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: "single profile with secret variables reported and verified",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "$FLEET_SECRET_VALUE"}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: "Get succeeds but has missing data",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "D1"}), 0},
				{"N2", syncml.ForTestWithData(map[string]string{"L2": "D2"}), 1},
				{"N3", syncml.ForTestWithData(map[string]string{"L3": "D3"}), 0},
				{"N4", syncml.ForTestWithData(map[string]string{"L4": "D4"}), 1},
			},
			report: []osqueryReport{
				{"N1", "200", "L1", ""},
				{"N2", "200", "L2", ""},
				{"N3", "200", "L3", "D3"},
				{"N4", "200", "L4", "D4"},
			},
			toVerify: []string{"N3", "N4"},
			toFail:   []string{"N2"},
			toRetry:  []string{"N1"},
		},
		{
			name: "Get fails",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "D1"}), 0},
				{"N2", syncml.ForTestWithData(map[string]string{"L2": "D2"}), 1},
				{"N3", syncml.ForTestWithData(map[string]string{"L3": "D3"}), 0},
				{"N4", syncml.ForTestWithData(map[string]string{"L4": "D4"}), 1},
			},
			report: []osqueryReport{
				{"N1", "400", "L1", ""},
				{"N2", "500", "L2", ""},
				{"N3", "200", "L3", "D3"},
				{"N4", "200", "L4", "D4"},
			},
			toVerify: []string{"N3", "N4"},
			toFail:   []string{"N2"},
			toRetry:  []string{"N1"},
		},
		{
			name: "missing report",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "D1"}), 0},
				{"N2", syncml.ForTestWithData(map[string]string{"L2": "D2"}), 1},
			},
			report:   []osqueryReport{},
			toVerify: []string{},
			toFail:   []string{"N2"},
			toRetry:  []string{"N1"},
		},
		{
			name: "profiles with multiple locURIs",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{"L1": "D1", "L1.1": "D1.1"}), 0},
				{"N2", syncml.ForTestWithData(map[string]string{"L2": "D2", "L2.1": "D2.1"}), 1},
				{"N3", syncml.ForTestWithData(map[string]string{"L3": "D3", "L3.1": "D3.1"}), 0},
				{"N4", syncml.ForTestWithData(map[string]string{"L4": "D4", "L4.1": "D4.1"}), 1},
			},
			report: []osqueryReport{
				{"N1", "400", "L1", ""},
				{"N1", "200", "L1.1", "D1.1"},
				{"N2", "500", "L2", ""},
				{"N2", "200", "L2.1", "D2.1"},
				{"N3", "200", "L3", "D3"},
				{"N3", "200", "L3.1", "D3.1"},
				{"N4", "200", "L4", "D4"},
			},
			toVerify: []string{"N3"},
			toFail:   []string{"N2", "N4"},
			toRetry:  []string{"N1"},
		},
		{
			name: "single profile with CDATA reported and verified",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{
					"L1": `
      <![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
				}), 0},
			},
			report: []osqueryReport{{"N1", "200", "L1",
				"&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: "single profile with CDATA to retry",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData(map[string]string{
					"L1": `
      <![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
				}), 0},
			},
			report: []osqueryReport{{"N1", "200", "L1",
				"&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var msg fleet.SyncML
			msg.Xmlns = syncml.SyncCmdNamespace
			msg.SyncHdr = fleet.SyncHdr{
				VerDTD:    syncml.SyncMLSupportedVersion,
				VerProto:  syncml.SyncMLVerProto,
				SessionID: "2",
				MsgID:     "2",
			}
			for _, p := range tt.report {
				ref := HashLocURI(p.Name, p.LocURI)
				msg.AppendCommand(fleet.MDMRaw, fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
					CmdRef:  &ref,
					Data:    ptr.String(p.Status),
				})

				// the protocol can respond with only a `Status`
				// command if the status failed
				if p.Status != "200" || p.Data != "" {
					msg.AppendCommand(fleet.MDMRaw, fleet.SyncMLCmd{
						XMLName: xml.Name{Local: fleet.CmdResults},
						CmdID:   fleet.CmdID{Value: uuid.NewString()},
						CmdRef:  &ref,
						Items: []fleet.CmdItem{
							{Target: ptr.String(p.LocURI), Data: &fleet.RawXmlData{Content: p.Data}},
						},
					})
				}
			}

			ds := new(mock.Store)
			ds.GetHostMDMProfilesExpectedForVerificationFunc = func(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
				installDate := host.DetailUpdatedAt.Add(-2 * time.Hour)
				if tt.withinGracePeriod {
					installDate = host.DetailUpdatedAt
				}
				out := map[string]*fleet.ExpectedMDMProfile{}
				for _, profile := range tt.hostProfiles {
					out[profile.Name] = &fleet.ExpectedMDMProfile{
						Name:                profile.Name,
						RawProfile:          profile.RawContents,
						EarliestInstallDate: installDate,
					}
				}
				return out, nil
			}

			ds.UpdateHostMDMProfilesVerificationFunc = func(ctx context.Context, host *fleet.Host, toVerify []string, toFail []string, toRetry []string) error {
				require.ElementsMatch(t, tt.toVerify, toVerify, "profiles to verify don't match")
				require.ElementsMatch(t, tt.toFail, toFail, "profiles to fail don't match")
				require.ElementsMatch(t, tt.toRetry, toRetry, "profiles to retry don't match")
				return nil
			}

			ds.GetHostMDMProfilesRetryCountsFunc = func(ctx context.Context, host *fleet.Host) ([]fleet.HostMDMProfileRetryCount, error) {
				out := []fleet.HostMDMProfileRetryCount{}
				for _, profile := range tt.hostProfiles {
					out = append(out, fleet.HostMDMProfileRetryCount{
						ProfileName: profile.Name,
						Retries:     profile.RetryCount,
					})
				}
				return out, nil
			}

			ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
				return strings.ReplaceAll(document, "$FLEET_SECRET_VALUE", "D1"), nil
			}

			out, err := xml.Marshal(msg)
			require.NoError(t, err)
			require.NoError(t, VerifyHostMDMProfiles(context.Background(), ds, &fleet.Host{DetailUpdatedAt: time.Now()}, out))
			require.True(t, ds.UpdateHostMDMProfilesVerificationFuncInvoked)
			require.True(t, ds.GetHostMDMProfilesExpectedForVerificationFuncInvoked)
			ds.UpdateHostMDMProfilesVerificationFuncInvoked = false
			ds.GetHostMDMProfilesExpectedForVerificationFuncInvoked = false
		})
	}
}

// osqueryReport is used by TestVerifyHostMDMProfilesHappyPaths
type osqueryReport struct {
	Name   string
	Status string
	LocURI string
	Data   string
}

// hostProfile is used by TestVerifyHostMDMProfilesHappyPaths
type hostProfile struct {
	Name        string
	RawContents []byte
	RetryCount  uint
}
