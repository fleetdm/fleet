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
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/wlanxml"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func generateTestWlanProfiles(t *testing.T) (wlanXMLOriginalProfile, wlanXMLModifiedProfile string) {
	wlanSSIDConfig := wlanxml.WlanXmlProfileSSIDConfig{
		SSID: []wlanxml.WlanXmlProfileSSID{
			{
				Name: "Test",
			},
		},
		NonBroadcast: false,
	}
	var err error
	wlanXMLOriginalProfile, err = wlanxml.GenerateWLANXMLProfileForTests("Test", wlanSSIDConfig)
	require.NoError(t, err)
	wlanSSIDConfig.NonBroadcast = true
	wlanXMLModifiedProfile, err = wlanxml.GenerateWLANXMLProfileForTests("Test", wlanSSIDConfig)
	require.NoError(t, err)
	return
}

func TestLoopHostMDMLocURIs(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()

	ds.GetHostMDMProfilesExpectedForVerificationFunc = func(ctx context.Context, host *fleet.Host) (map[string]*fleet.ExpectedMDMProfile, error) {
		return map[string]*fleet.ExpectedMDMProfile{
			"N1": {Name: "N1", RawProfile: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}})},
			"N2": {Name: "N2", RawProfile: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L2", Data: "D2"}})},
			"N3": {Name: "N3", RawProfile: syncml.ForTestWithData([]syncml.TestCommand{
				{Verb: "Replace", LocURI: "L3", Data: "D3"},
				{Verb: "Add", LocURI: "L3.1", Data: "D3.1"},
			})},
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

	err := VerifyHostMDMProfiles(ctx, log.NewNopLogger(), ds, host, []byte{})
	require.ErrorIs(t, err, io.EOF)
}

func TestVerifyHostMDMProfilesHappyPaths(t *testing.T) {
	wlanXMLOriginalProfile, wlanXMLModifiedProfile := generateTestWlanProfiles(t)
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
			name: `single "Replace" profile reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `single "Add" profile reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L1", Data: "D1"}}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `single "Replace" profile with secret variables reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "$FLEET_SECRET_VALUE"}}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `single "Add" profile with secret variables reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L1", Data: "$FLEET_SECRET_VALUE"}}), 0},
			},
			report:   []osqueryReport{{"N1", "200", "L1", "D1"}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: "Get succeeds but has missing data",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}}), 0},
				{"N2", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L2", Data: "D2"}}), 1},
				{"N3", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L3", Data: "D3"}}), 0},
				{"N4", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L4", Data: "D4"}}), 1},
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
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}}), 0},
				{"N2", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L2", Data: "D2"}}), 1},
				{"N3", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L3", Data: "D3"}}), 0},
				{"N4", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L4", Data: "D4"}}), 1},
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
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L1", Data: "D1"}}), 0},
				{"N2", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "L2", Data: "D2"}}), 1},
				{"N3", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L3", Data: "D3"}}), 0},
				{"N4", syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Add", LocURI: "L4", Data: "D4"}}), 1},
			},
			report:   []osqueryReport{},
			toVerify: []string{},
			toFail:   []string{"N2", "N4"},
			toRetry:  []string{"N1", "N3"},
		},
		{
			name: "profiles with multiple locURIs",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{
					{Verb: "Replace", LocURI: "L1", Data: "D1"},
					{Verb: "Add", LocURI: "L1.1", Data: "D1.1"},
				}), 0},
				{"N2", syncml.ForTestWithData([]syncml.TestCommand{
					{Verb: "Add", LocURI: "L2", Data: "D2"},
					{Verb: "Replace", LocURI: "L2.1", Data: "D2.1"},
				}), 1},
				{"N3", syncml.ForTestWithData([]syncml.TestCommand{
					{Verb: "Add", LocURI: "L3", Data: "D3"},
					{Verb: "Add", LocURI: "L3.1", Data: "D3.1"},
				}), 0},
				{"N4", syncml.ForTestWithData([]syncml.TestCommand{
					{Verb: "Replace", LocURI: "L4", Data: "D4"},
					{Verb: "Replace", LocURI: "L4.1", Data: "D4.1"},
				}), 1},
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
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Replace",
					LocURI: "L1",
					Data: `
      <![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				"&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},

		{
			name: `single profile with CDATA to retry`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Replace",
					LocURI: "L1",
					Data: `
      <![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				"&lt;Enabled/&gt;&lt;Data id=\"EnableScriptBlockInvocationLogging\" value=\"true\"/&gt;&lt;Data id=\"ExecutionPolicy\" value=\"AllSigned\"/&gt;&lt;Data id=\"Listbox_ModuleNames\" value=\"*\"/&gt;&lt;Data id=\"OutputDirectory\" value=\"false\"/&gt;&lt;Data id=\"SourcePathForUpdateHelp\" value=\"false\"/&gt;",
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},

		{
			name: `single "Replace" profile with wireless XML reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Replace",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLOriginalProfile,
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `single "Replace" profile with wireless XML to retry`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Replace",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLModifiedProfile,
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
		{
			name: `single "Replace" profile with wireless XML to fail`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Replace",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 1},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLModifiedProfile,
			}},
			toVerify: []string{},
			toFail:   []string{"N1"},
			toRetry:  []string{},
		},

		{
			name: `single "Add" profile with wireless XML reported and verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Add",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLOriginalProfile,
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `single "Add" profile with wireless XML to retry`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Add",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLModifiedProfile,
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
		{
			name: `single "Add" profile with wireless XML to fail`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{{
					Verb:   "Add",
					LocURI: "L1",
					Data:   wlanXMLOriginalProfile,
				}}), 1},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				wlanXMLModifiedProfile,
			}},
			toVerify: []string{},
			toFail:   []string{"N1"},
			toRetry:  []string{},
		},
		{
			name: `win32/desktop bridge ADMX profile 404s but is marked verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{
					{
						Verb:   "Replace",
						LocURI: "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/employee/Policy/employeeAdmxFilename",
						Data:   "some ADMX policy file data",
					},
					{
						Verb:   "Replace",
						LocURI: "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/company",
						Data:   `<![CDATA[<enabled/> <data id="company" value="foocorp"/>]]>`,
					},
				}), 0},
			},
			report: []osqueryReport{{
				"N1", "404", "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/employee/Policy/employeeAdmxFilename", "",
			}, {
				"N1", "404", "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/company", "",
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
		},
		{
			name: `VPNV2 profile xml returns 200 but differently formatted data and is marked verified`,
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{
					{
						Verb:   "Add",
						LocURI: "./Vendor/MSFT/VPNv2/fleet-test-vpn/ProfileXML",
						Data: `&lt;VPNProfile&gt;
    &lt;ProfileName&gt;fleet-test-vpn&lt;/ProfileName&gt;
&lt;/VPNProfile&gt;`,
					},
				}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "./Vendor/MSFT/VPNv2/fleet-test-vpn/ProfileXML", `&lt;VPNProfile&gt;&lt;ProfileName&gt;fleet-test-vpn&lt;/ProfileName&gt;&lt;/VPNProfile&gt;`,
			}},
			toVerify: []string{"N1"},
			toFail:   []string{},
			toRetry:  []string{},
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
			require.NoError(t,
				VerifyHostMDMProfiles(context.Background(), log.NewNopLogger(), ds, &fleet.Host{DetailUpdatedAt: time.Now()}, out))
			require.True(t, ds.UpdateHostMDMProfilesVerificationFuncInvoked)
			require.True(t, ds.GetHostMDMProfilesExpectedForVerificationFuncInvoked)
			ds.UpdateHostMDMProfilesVerificationFuncInvoked = false
			ds.GetHostMDMProfilesExpectedForVerificationFuncInvoked = false
		})
	}
}

func TestIsWin32OrDesktopBridgeADMXCSP(t *testing.T) {
	testCases := []struct {
		name     string
		locURI   string
		expected bool
	}{
		{
			name:     "ADMX desktop bridge",
			locURI:   "",
			expected: false,
		},
		{
			name:     "properly formatted ADMX win32/desktop bridge app locURI with a specific category",
			locURI:   "./Device/Vendor/MSFT/Policy/Config/ContosoCompanyApp~Policy~ParentCategoryArea~Category1/L_PolicyConfigurationMode",
			expected: true,
		},
		{
			name:     "properly formatted ADMX win32/desktop bridge app with the default category",
			locURI:   "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/Subteam",
			expected: true,
		},
		{
			name:     "Base ADMXInstall node for app",
			locURI:   "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetTestApp",
			expected: false,
		},
		{
			name:     "ADMXInstall node with ADMX policy filename",
			locURI:   "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetTestApp/Policy/FleetTestAppAdmxFilename",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsWin32OrDesktopBridgeADMXCSP(tc.locURI)
			require.Equal(t, tc.expected, result)
			result = IsWin32OrDesktopBridgeADMXCSP(strings.ToUpper(tc.locURI))
			require.Equal(t, tc.expected, result, "Expected same result for uppercased locURI")
			result = IsWin32OrDesktopBridgeADMXCSP(strings.ToLower(tc.locURI))
			require.Equal(t, tc.expected, result, "Expected same result for lowercased locURI")

			// a locURI starting with "./Vendor/" is implicitly device scoped, so we should
			// get the same result if we expliclty scope it with ./Device/Vendor/
			if strings.HasPrefix(tc.locURI, "./Vendor/") {
				explicitlyDeviceScopedLocURI := strings.Replace(tc.locURI, "./Vendor/", "./Device/Vendor/", 1)
				result = IsWin32OrDesktopBridgeADMXCSP(explicitlyDeviceScopedLocURI)
				require.Equal(t, tc.expected, result, "Expected same result for explicitly and implicitly device scoped locURIs")
				result = IsWin32OrDesktopBridgeADMXCSP(strings.ToUpper(tc.locURI))
				require.Equal(t, tc.expected, result, "Expected same result for uppercased locURI when explicitly device scoped")
				result = IsWin32OrDesktopBridgeADMXCSP(strings.ToLower(tc.locURI))
				require.Equal(t, tc.expected, result, "Expected same result for lowercased locURI when explicitly device scoped")
			}
		})
	}
}

func TestIsADMXInstallConfigOperationCSP(t *testing.T) {
	testCases := []struct {
		name     string
		locURI   string
		expected bool
	}{
		{
			name:     "Base ADMXInstall node for app",
			locURI:   "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetTestApp",
			expected: true,
		},
		{
			name:     "ADMXInstall node with ADMX policy filename",
			locURI:   "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetTestApp/Policy/FleetTestAppAdmxFilename",
			expected: true,
		},
		{
			name:     "empty string",
			locURI:   "",
			expected: false,
		},
		// User-scoped ConfigOperations are not supported per Microsoft documentation
		{
			name:     "Unsupported User-scoped ADMXInstall node",
			locURI:   "./User/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/FleetTestApp",
			expected: false,
		},
		// Neither of these are valid or supported paths and should definitely not be picked up by
		// this logic
		{
			name:     "Similar looking ADMX path but not ADMXInstall",
			locURI:   "./Vendor/MSFT/Policy/ConfigOperations/ADMX/Something",
			expected: false,
		},
		{
			name:     "ADMXInstall under /Config/ path",
			locURI:   "./Vendor/MSFT/Policy/Config/ADMXInstall",
			expected: false,
		},
		{
			name:     "Partial path but not full ADMXInstall",
			locURI:   "./Vendor/MSFT/Policy",
			expected: false,
		},
		{
			name:     "Another Partial path but not full ADMXInstall",
			locURI:   "./Vendor/MSFT",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsADMXInstallConfigOperationCSP(tc.locURI)
			require.Equal(t, tc.expected, result)
			if strings.HasPrefix(tc.locURI, "./Vendor/") {
				explicitlyDeviceScopedLocURI := strings.Replace(tc.locURI, "./Vendor/", "./Device/Vendor/", 1)
				result = IsADMXInstallConfigOperationCSP(explicitlyDeviceScopedLocURI)
				require.Equal(t, tc.expected, result, "Expected same result for explicitly and implicitly device scoped locURIs")
			}
		})
	}
}

func TestIsVPNV2CSP(t *testing.T) {
	testCases := []struct {
		name     string
		locURI   string
		expected bool
	}{
		{
			name:     "VPNV2 CSP with ProfileXML",
			locURI:   "./Vendor/MSFT/VPNv2/fleet-test-vpn/ProfileXML",
			expected: true,
		},
		{
			name:     "VPNV2 CSP with ProfileXML",
			locURI:   "./User/Vendor/MSFT/VPNv2/fleet-test-vpn/ProfileXML",
			expected: true,
		},
		{
			name:     "VPNV2 CSP deeper path",
			locURI:   "./User/Vendor/MSFT/VPNv2/fleet-test-vpn/sometest/someothertest/anXMLThing",
			expected: true,
		},
		{
			name:     "VPN",
			locURI:   "./User/Vendor/MSFT/VPN/fleet-test-vpn",
			expected: false,
		},
		{
			name:     "Unrelated policy path",
			locURI:   "./Vendor/MSFT/Policy/VPNv2",
			expected: false,
		},
		{
			name:     "Incomplete VPNv2 path",
			locURI:   "./Vendor/MSFT/VPNv2",
			expected: false,
		},
		{
			name:     "empty string",
			locURI:   "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsVPNV2CSP(tc.locURI)
			require.Equal(t, tc.expected, result)
			if strings.HasPrefix(tc.locURI, "./Vendor/") {
				explicitlyDeviceScopedLocURI := strings.Replace(tc.locURI, "./Vendor/", "./Device/Vendor/", 1)
				result = IsVPNV2CSP(explicitlyDeviceScopedLocURI)
				require.Equal(t, tc.expected, result, "Expected same result for explicitly and implicitly device scoped locURIs")
			}
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
