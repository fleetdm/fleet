package microsoft_mdm

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
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
			"N4": {Name: "N4", RawProfile: syncml.ForTestWithData([]syncml.TestCommand{
				{Verb: "Replace", LocURI: "L4", Data: "<![CDATA[D4]]>"},
				{Verb: "Add", LocURI: "L4.1", Data: "<![CDATA[D4.1]]>"},
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
	err := LoopOverExpectedHostProfiles(ctx, log.NewNopLogger(), ds, &fleet.Host{}, func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string) {
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
			{"L4", "D4", "N4", "4141459399"},
			{"L4.1", "D4.1", "N4", "236794510"},
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
		existingProfiles  []fleet.HostMDMWindowsProfile
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
					Data: `<![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>`,
				}}), 0},
			},
			report: []osqueryReport{{
				"N1", "200", "L1",
				`<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>`,
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
				`<disabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="SourcePathForUpdateHelp" value="false"/>`,
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
			name: `win32/desktop bridge ADMX profile 404s and is marked for retry since it was previously undelivered`,
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
			existingProfiles: []fleet.HostMDMWindowsProfile{},
			report: []osqueryReport{{
				"N1", "404", "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/employee/Policy/employeeAdmxFilename", "",
			}, {
				"N1", "404", "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/company", "",
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
		{
			name: `win32/desktop bridge ADMX profile 404s and is marked for retry since it was previously nil delivery status`,
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
			existingProfiles: []fleet.HostMDMWindowsProfile{
				{
					ProfileUUID: "uuid-N1",
					Name:        "N1",
					Status:      nil,
				},
			},
			report: []osqueryReport{{
				"N1", "404", "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/employee/Policy/employeeAdmxFilename", "",
			}, {
				"N1", "404", "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/company", "",
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
		{
			name: `win32/desktop bridge ADMX profile 404s and is marked for retry since it was previously pending`,
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
			existingProfiles: []fleet.HostMDMWindowsProfile{
				{
					ProfileUUID: "uuid-N1",
					Name:        "N1",
					Status:      &fleet.MDMDeliveryPending,
				},
			},
			report: []osqueryReport{{
				"N1", "404", "./Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/employee/Policy/employeeAdmxFilename", "",
			}, {
				"N1", "404", "./Device/Vendor/MSFT/Policy/Config/employee~Policy~DefaultCategory/company", "",
			}},
			toVerify: []string{},
			toFail:   []string{},
			toRetry:  []string{"N1"},
		},
		{
			name: `win32/desktop bridge ADMX profile 404s but is marked verified since it was previously verifying`,
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
			existingProfiles: []fleet.HostMDMWindowsProfile{
				{
					ProfileUUID: "uuid-N1",
					Name:        "N1",
					Status:      &fleet.MDMDeliveryVerifying,
				},
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
			name: `win32/desktop bridge ADMX profile 404s but is marked verified since it was previously verifying`,
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
			existingProfiles: []fleet.HostMDMWindowsProfile{
				{
					ProfileUUID: "uuid-N1",
					Name:        "N1",
					Status:      &fleet.MDMDeliveryVerified,
				},
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
			name: "scep profile instantly verifies",
			hostProfiles: []hostProfile{
				{"N1", syncml.ForTestWithData([]syncml.TestCommand{
					{
						Verb: "Replace",
						LocURI: `
						./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/bogus-key-value`,
						Data: "non related data",
					},
				}), 0},
			},
			existingProfiles: []fleet.HostMDMWindowsProfile{
				{
					ProfileUUID: "uuid-N1",
					Name:        "N1",
					Status:      &fleet.MDMDeliveryPending,
				},
			},
			toVerify: []string{"N1"},
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
						ProfileUUID:         "uuid-" + profile.Name,
						Name:                profile.Name,
						RawProfile:          profile.RawContents,
						EarliestInstallDate: installDate,
					}
				}
				return out, nil
			}

			ds.GetHostMDMWindowsProfilesFunc = func(ctx context.Context, hostUUID string) ([]fleet.HostMDMWindowsProfile, error) {
				return tt.existingProfiles, nil
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
			require.True(t, ds.GetHostMDMWindowsProfilesFuncInvoked)
			ds.UpdateHostMDMProfilesVerificationFuncInvoked = false
			ds.GetHostMDMProfilesExpectedForVerificationFuncInvoked = false
			ds.GetHostMDMWindowsProfilesFuncInvoked = false
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

func TestPreprocessWindowsProfileContentsForVerification(t *testing.T) {
	ds := new(mock.Store)

	tests := []struct {
		name             string
		hostUUID         string
		profileContents  string
		expectedContents string
	}{
		{
			name:             "no fleet variables",
			hostUUID:         "test-uuid-123",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
		},
		{
			name:             "fleet variable without braces",
			hostUUID:         "test-uuid-456",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: test-uuid-456</Data></Item></Replace>`,
		},
		{
			name:             "fleet variable with braces",
			hostUUID:         "test-uuid-789",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: ${FLEET_VAR_HOST_UUID}</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: test-uuid-789</Data></Item></Replace>`,
		},
		{
			name:             "multiple fleet variables",
			hostUUID:         "test-uuid-abc",
			profileContents:  `<Replace><Item><Data>First: $FLEET_VAR_HOST_UUID, Second: ${FLEET_VAR_HOST_UUID}</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Data>First: test-uuid-abc, Second: test-uuid-abc</Data></Item></Replace>`,
		},
		{
			name:             "fleet variable with special XML characters in UUID",
			hostUUID:         "test<>&\"uuid",
			profileContents:  `<Replace><Item><Data>Device: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Data>Device: test&lt;&gt;&amp;&#34;uuid</Data></Item></Replace>`,
		},
		{
			name:             "fleet variable with apostrophe in UUID",
			hostUUID:         "test<>&\"'uuid",
			profileContents:  `<Replace><Data>ID: $FLEET_VAR_HOST_UUID</Data></Replace>`,
			expectedContents: `<Replace><Data>ID: test&lt;&gt;&amp;&#34;&#39;uuid</Data></Replace>`,
		},
		{
			name:             "unsupported variable ignored",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>ID: $FLEET_VAR_HOST_UUID, Other: $FLEET_VAR_UNSUPPORTED</Data></Replace>`,
			expectedContents: `<Replace><Data>ID: test-host-1234-uuid, Other: $FLEET_VAR_UNSUPPORTED</Data></Replace>`,
		},
		{
			name:             "fleet variable with CmdID in profile",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><CmdID>1</CmdID><Item><Data>Device ID: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			expectedContents: `<Replace><CmdID>1</CmdID><Item><Data>Device ID: test-host-1234-uuid</Data></Item></Replace>`,
		},
		{
			name:             "fleet variable with both formats in same profile",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>ID1: $FLEET_VAR_HOST_UUID, ID2: ${FLEET_VAR_HOST_UUID}</Data></Replace>`,
			expectedContents: `<Replace><Data>ID1: test-host-1234-uuid, ID2: test-host-1234-uuid</Data></Replace>`,
		},
		{
			name:             "skips scep windows id var",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>ID: $FLEET_VAR_HOST_UUID, SCEP: $FLEET_VAR_HOST_SCEP_WINDOWS_ID</Data></Replace>`,
			expectedContents: `<Replace><Data>ID: test-host-1234-uuid, SCEP: $FLEET_VAR_HOST_SCEP_WINDOWS_ID</Data></Replace>`,
		},
	}

	params := PreprocessingParameters{
		HostIDForUUIDCache: make(map[string]uint),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PreprocessWindowsProfileContentsForVerification(t.Context(), log.NewNopLogger(), ds, tt.hostUUID, uuid.NewString(), tt.profileContents, params)
			require.Equal(t, tt.expectedContents, result)
		})
	}
}

func TestPreprocessWindowsProfileContentsForDeployment(t *testing.T) {
	ds := new(mock.Store)

	scimUser := &fleet.ScimUser{
		UserName:   "test@idp.com",
		GivenName:  ptr.String("First"),
		FamilyName: ptr.String("Last"),
		Department: ptr.String("Department"),
		Groups: []fleet.ScimUserGroup{
			{
				ID:          1,
				DisplayName: "Group One",
			},
			{
				ID:          2,
				DisplayName: "Group Two",
			},
		},
	}

	baseSetup := func() {
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			if ds.GetAllCertificateAuthoritiesFunc == nil {
				return &fleet.GroupedCertificateAuthorities{
					CustomScepProxy: []fleet.CustomSCEPProxyCA{},
				}, nil
			}

			cas, err := ds.GetAllCertificateAuthoritiesFunc(ctx, includeSecrets)
			if err != nil {
				return nil, err
			}

			return fleet.GroupCertificateAuthoritiesByType(cas)
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://test-fleet.com",
				},
			}, nil
		}
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
			return []uint{42}, nil
		}

		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			if hostID == 42 {
				return scimUser, nil
			}

			return nil, fmt.Errorf("no scim user for host id %d", hostID)
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{}, nil
		}
	}

	// use the same uuid for all profile UUID actions
	profileUUID := uuid.NewString()

	tests := []struct {
		name             string
		hostUUID         string
		hostCmdUUID      string
		profileContents  string
		expectedContents string
		expectError      bool
		processingError  string                                                          // if set then we expect the error to be of type MicrosoftProfileProcessingError with this message
		setup            func()                                                          // Used for setting up datastore mocks.
		expect           func(t *testing.T, managedCerts []*fleet.MDMManagedCertificate) // Add more params as they need validation.
		freeTier         bool
	}{
		{
			name:             "no fleet variables",
			hostUUID:         "test-uuid-123",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
		},
		{
			name:             "host uuid fleet variable",
			hostUUID:         "test-uuid-456",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: test-uuid-456</Data></Item></Replace>`,
		},
		{
			name:             "scep windows certificate id",
			hostUUID:         "test-host-1234-uuid",
			hostCmdUUID:      "cmd-uuid-5678",
			profileContents:  `<Replace><Data>SCEP: $FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</Data></Replace>`,
			expectedContents: `<Replace><Data>SCEP: cmd-uuid-5678</Data></Replace>`,
		},
		{
			name:            "custom scep proxy url not usable in free tier",
			hostUUID:        "test-host-1234-uuid",
			hostCmdUUID:     "cmd-uuid-5678",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Custom SCEP integration requires a Fleet Premium license.",
			freeTier:        true,
		},
		{
			name:            "custom scep proxy url ca not found",
			hostUUID:        "test-host-1234-uuid",
			hostCmdUUID:     "cmd-uuid-5678",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Fleet couldn't populate $CUSTOM_SCEP_PROXY_URL_CERTIFICATE because CERTIFICATE certificate authority doesn't exist.",
		},
		{
			name:             "custom scep proxy url ca found and replaced",
			hostUUID:         "test-host-1234-uuid",
			hostCmdUUID:      "cmd-uuid-5678",
			profileContents:  `<Replace><Data>     $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectedContents: `<Replace><Data>https://test-fleet.com/mdm/scep/proxy/test-host-1234-uuid%2C` + profileUUID + `%2CCERTIFICATE%2Csupersecret</Data></Replace>`,
			setup: func() {
				ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
					return []*fleet.CertificateAuthority{
						{
							ID:        1,
							Name:      ptr.String("CERTIFICATE"),
							Type:      string(fleet.CATypeCustomSCEPProxy),
							URL:       ptr.String("https://scep.proxy.url/scep"),
							Challenge: ptr.String("supersecret"),
						},
					}, nil
				}
				ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
					return "supersecret", nil
				}
			},
			expect: func(t *testing.T, managedCerts []*fleet.MDMManagedCertificate) {
				require.Len(t, managedCerts, 1)
				require.Equal(t, "CERTIFICATE", managedCerts[0].CAName)
				require.Equal(t, fleet.CAConfigCustomSCEPProxy, managedCerts[0].Type)
			},
		},
		{
			name:            "custom scep challenge not usable in free tier",
			hostUUID:        "test-host-1234-uuid",
			hostCmdUUID:     "cmd-uuid-5678",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Custom SCEP integration requires a Fleet Premium license.",
			freeTier:        true,
		},
		{
			name:            "custom scep proxy challenge ca not found",
			hostUUID:        "test-host-1234-uuid",
			hostCmdUUID:     "cmd-uuid-5678",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Fleet couldn't populate $CUSTOM_SCEP_CHALLENGE_CERTIFICATE because CERTIFICATE certificate authority doesn't exist.",
		},
		{
			name:             "custom scep proxy challenge ca found and replaced",
			hostUUID:         "test-host-1234-uuid",
			hostCmdUUID:      "cmd-uuid-5678",
			profileContents:  `<Replace><Data>     $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectedContents: `<Replace><Data>supersecret</Data></Replace>`,
			setup: func() {
				ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
					return []*fleet.CertificateAuthority{
						{
							ID:        1,
							Name:      ptr.String("CERTIFICATE"),
							Type:      string(fleet.CATypeCustomSCEPProxy),
							URL:       ptr.String("https://scep.proxy.url/scep"),
							Challenge: ptr.String("supersecret"),
						},
					}, nil
				}
				ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
					return "supersecret", nil
				}
			},
		},
		{
			name:             "all idp variables",
			hostUUID:         "idp-host-uuid",
			hostCmdUUID:      "cmd-uuid-5678",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_USERNAME - $FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART - $FLEET_VAR_HOST_END_USER_IDP_GROUPS - $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT - $FLEET_VAR_HOST_END_USER_IDP_FULL_NAME</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: test@idp.com - test - Group One,Group Two - Department - First Last</Data></Item></Replace>`,
		},
		{
			name:            "missing groups on idp user",
			hostUUID:        "no-groups-idp",
			profileContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_GROUPS</Data></Item></Replace>`,
			expectError:     true,
			processingError: "There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.",
			setup: func() {
				scimUser.Groups = []fleet.ScimUserGroup{}
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					return scimUser, nil
				}
			},
		},
		{
			name:            "missing department on idp user",
			hostUUID:        "no-department-idp",
			profileContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT</Data></Item></Replace>`,
			expectError:     true,
			processingError: "There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.",
			setup: func() {
				scimUser.Department = nil
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					return scimUser, nil
				}
			},
		},
	}

	params := PreprocessingParameters{
		HostIDForUUIDCache: make(map[string]uint),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseSetup()
			if tt.setup != nil {
				tt.setup()
			}
			t.Cleanup(func() {
				ds = new(mock.Store) // Reset the mock datastore after each test, to avoid overlapping setups.
			})

			licenseInfo := &fleet.LicenseInfo{
				Tier: fleet.TierPremium,
			}
			if tt.freeTier {
				licenseInfo.Tier = fleet.TierFree
			}
			ctx := license.NewContext(t.Context(), licenseInfo)

			appConfig, err := ds.AppConfig(ctx)
			require.NoError(t, err)

			// Populate this one, in setup by mocking ds.GetAllCertificateAuthoritiesFunc if needed.
			groupedCAs, err := ds.GetGroupedCertificateAuthorities(ctx, true)
			require.NoError(t, err)

			managedCertificates := &[]*fleet.MDMManagedCertificate{}

			result, err := PreprocessWindowsProfileContentsForDeployment(ctx, log.NewNopLogger(), ds, appConfig, tt.hostUUID, tt.hostCmdUUID, profileUUID, groupedCAs, tt.profileContents, managedCertificates, params)
			if tt.expectError {
				require.Error(t, err)
				if tt.processingError != "" {
					var processingErr *MicrosoftProfileProcessingError
					require.ErrorAs(t, err, &processingErr, "expected ProfileProcessingError")
					require.Equal(t, tt.processingError, processingErr.Error())
				}
				return // do not verify profile contents if an error is expected
			}

			require.Equal(t, tt.expectedContents, result)
			require.NoError(t, err)

			if tt.expect != nil {
				tt.expect(t, *managedCertificates)
			}
		})
	}
}
