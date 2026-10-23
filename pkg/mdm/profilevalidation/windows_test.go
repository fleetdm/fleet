package profilevalidation

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsContractExample(t *testing.T) {
	raw, err := os.ReadFile("testdata/contract_example.json")
	require.NoError(t, err)
	var s ProfileValidationSchema
	require.NoError(t, json.Unmarshal(raw, &s))

	assert.Equal(t, WindowsVersion{
		Release: "DDFv2Feb2026",
		URL:     "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip",
		SHA256:  "bf667d895af4a8c8ab5a31065ce0e28ea2f8b649c4dc416f452f62fd1c42ff14",
	}, s.Version.Windows)
	assert.Equal(t, WindowsTemplatesVersion{SHA256: "0f0e", GeneratedAt: "2026-10-07T14:08:30.957Z"}, s.Version.WindowsTemplates)

	node := func(locURI string) *WindowsNode { return notNil(t, s.Windows[locURI]) }
	int1 := WindowsFormats{WindowsFormatInt}
	chr := WindowsFormats{WindowsFormatChr}

	assert.Equal(t, &WindowsNode{
		Format: int1, Scope: WindowsScopeDevice, Source: WindowsSourceWindows, Range: &Range{Min: new(0.0), Max: new(999.0)}, MinBuild: []string{"10.0.10240"},
	}, node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock"))
	assert.Equal(t, `^(\d\.)?(\d)$`, node("./Device/Vendor/MSFT/DMClient/Provider/*/SyncApplicationVersion").RegEx)
	assert.Equal(t, []string{"0x0", "0x1", "0x2", "0x4"}, node("./Device/Vendor/MSFT/DMClient/Provider/*/FirstSyncStatus/BlockInStatusPage").Flag)
	assert.Equal(t, []string{"0", "1"}, node("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type").Enum)
	assert.Equal(t, &WindowsNode{Format: chr, Scope: WindowsScopeDevice, Source: WindowsSourceWindows, ADMXBacked: true},
		node("./Device/Vendor/MSFT/Policy/Config/RemoteAssistance/SolicitedRemoteAssistance"))
	assert.Equal(t, WindowsSourceADMXTemplate,
		node("./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation").Source)

	xsd := node("./Device/Vendor/MSFT/Policy/Config/LocalUsersAndGroups/Configure").XSD
	assert.Contains(t, s.WindowsXSDs[xsd], "accessgroup")

	deprecated := node("./User/Vendor/MSFT/Policy/Config/Example/Deprecated")
	assert.Equal(t, WindowsScopeUser, deprecated.Scope)
	assert.Equal(t, &Range{Min: new(-1.0), Max: new(10.0)}, deprecated.Range)
	assert.Equal(t, []string{"10.0.19041", "10.0.19042.1", "99.9.99999"}, deprecated.MinBuild)
	assert.Equal(t, &WindowsDeprecated{Build: "10.0.22000"}, deprecated.Deprecated)

	list := node("./Device/Vendor/MSFT/Policy/Config/Example/List")
	assert.Equal(t, ",", list.Delimiter)
	// Deprecated without a build is still deprecated.
	assert.Equal(t, &WindowsDeprecated{}, list.Deprecated)

	loosened := node("./Device/Vendor/MSFT/Policy/Config/Example/Loosened")
	assert.Equal(t, WindowsFormats{WindowsFormatInt, WindowsFormatChr}, loosened.Format)
	assert.Equal(t, "Devices accept a string.", loosened.Override)
	assert.Equal(t, WindowsFormats{"decimal128"}, node("./Device/Vendor/MSFT/Example/FutureFormat").Format)

	// Re-encoding and decoding again is lossless for everything modeled.
	out, err := json.Marshal(s)
	require.NoError(t, err)
	var again ProfileValidationSchema
	require.NoError(t, json.Unmarshal(out, &again))
	assert.Equal(t, s, again)
}

func TestWindowsNodeEncoding(t *testing.T) {
	// Optional fields are omitted, a single format is a string, and a 0 bound is kept.
	out, err := json.Marshal(&WindowsNode{Format: WindowsFormats{WindowsFormatInt}, Scope: WindowsScopeDevice, Range: &Range{Min: new(0.0)}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"format": "int", "scope": "device", "range": {"min": 0}}`, string(out))

	out, err = json.Marshal(&WindowsNode{Format: WindowsFormats{WindowsFormatInt, WindowsFormatChr}, Scope: WindowsScopeUser, Deprecated: &WindowsDeprecated{}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"format": ["int", "chr"], "scope": "user", "deprecated": {}}`, string(out))
}

func TestWindowsFormatsUnmarshal(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    WindowsFormats
		wantErr bool
	}{
		{name: "single", in: `"chr"`, want: WindowsFormats{WindowsFormatChr}},
		{name: "list", in: `["int","chr"]`, want: WindowsFormats{WindowsFormatInt, WindowsFormatChr}},
		// Newer upstream data must never fail an older parser.
		{name: "unknown value carried", in: `"decimal128"`, want: WindowsFormats{"decimal128"}},
		{name: "number rejected", in: `1`, wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got WindowsFormats
			err := json.Unmarshal([]byte(c.in), &got)
			if c.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestWindowsNodeLookup(t *testing.T) {
	nodes := map[string]*WindowsNode{}
	add := func(locURI string) *WindowsNode {
		n := &WindowsNode{Format: WindowsFormats{WindowsFormatChr}, Scope: WindowsScopeDevice}
		nodes[locURI] = n
		return n
	}
	maxInactivity := add("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock")
	firewallType := add("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type")
	syncVersion := add("./Device/Vendor/MSFT/DMClient/Provider/*/SyncApplicationVersion")
	twoStars := add("./Device/Vendor/MSFT/Example/*/Group/*/Value")
	oneStar := add("./Device/Vendor/MSFT/Example/*/Group/Fixed/Value")
	literal := add("./Device/Vendor/MSFT/Example/Known/Group/Fixed/Value")
	nodes["./Device/Vendor/MSFT/Example/*/Null"] = nil
	s := &ProfileValidationSchema{Windows: nodes}

	cases := []struct {
		name   string
		locURI string
		want   *WindowsNode
	}{
		{name: "exact", locURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock", want: maxInactivity},
		{name: "dynamic segment", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules/MyRule/Action/Type", want: firewallType},
		{name: "dynamic segment with spaces", locURI: "./Device/Vendor/MSFT/DMClient/Provider/MS DM Server/SyncApplicationVersion", want: syncVersion},
		{name: "literal star matches its own pattern", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type", want: firewallType},
		{name: "exact beats star", locURI: "./Device/Vendor/MSFT/Example/Known/Group/Fixed/Value", want: literal},
		{name: "fewer stars win", locURI: "./Device/Vendor/MSFT/Example/Other/Group/Fixed/Value", want: oneStar},
		{name: "two stars when only they match", locURI: "./Device/Vendor/MSFT/Example/Other/Group/Else/Value", want: twoStars},
		{name: "empty segment doesn't match star", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules//Action/Type"},
		{name: "extra segment", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules/MyRule/Action/Type/Extra"},
		{name: "missing segment", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules/Action/Type"},
		// Root equivalence is the validator's call, not the lookup's.
		{name: "device root not equal to bare root", locURI: "./Device/Vendor/MSFT/Firewall/MdmStore/FirewallRules/MyRule/Action/Type"},
		{name: "case sensitive", locURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/maxinactivitytimedevicelock"},
		{name: "null node skipped", locURI: "./Device/Vendor/MSFT/Example/X/Null"},
		{name: "unknown", locURI: "./Device/Vendor/MSFT/Unknown"},
		{name: "empty", locURI: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Same(t, c.want, s.WindowsNode(c.locURI))
		})
	}

	// A schema without a Windows section doesn't panic.
	assert.Nil(t, (&ProfileValidationSchema{}).WindowsNode("./Device/Vendor/MSFT/Unknown"))
}
