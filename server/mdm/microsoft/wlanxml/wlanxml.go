// Package wlanxml handles WLAN Profiles for Microsoft MDM server.
// See: https://learn.microsoft.com/en-us/windows/win32/nativewifi/wlan-profileschema-schema
// for samples: https://learn.microsoft.com/en-us/windows/win32/nativewifi/wireless-profile-samples
// finally for multi-SSID uses: https://learn.microsoft.com/en-us/windows-hardware/drivers/mobilebroadband/handling-large-numbers-of-ssids
package wlanxml

import (
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
)

func IsWLANXML(text string) bool {
	// We try to unmarshal the string to see if it looks like a valid WLAN XML policy
	_, err := unmarshal(text)
	if err != nil {
		return false
	}
	return true
}

func Equal(a, b string) (bool, error) {
	aPolicy, err := unmarshal(a)
	if err != nil {
		return false, fmt.Errorf("unmarshalling WLAN XML policy a: %w", err)
	}
	bPolicy, err := unmarshal(b)
	if err != nil {
		return false, fmt.Errorf("unmarshalling WLAN XML policy b: %w", err)
	}
	return aPolicy.Equal(bPolicy), nil
}

func unmarshal(a string) (wlanXmlPolicy, error) {
	// This whole thing will be XML Encoded so step 1 is just to decode it
	var unescaped string
	err := xml.Unmarshal([]byte("<wlanxml>"+a+"</wlanxml>"), &unescaped)
	if err != nil {
		return wlanXmlPolicy{}, fmt.Errorf("unmarshalling WLAN XML policy to string: %w", err)
	}

	var policy wlanXmlPolicy
	err = xml.Unmarshal([]byte(unescaped), &policy)
	if err != nil {
		return wlanXmlPolicy{}, fmt.Errorf("unmarshalling WLAN XML policy: %w", err)
	}
	if policy.XMLName.Local != "WLANProfile" {
		return wlanXmlPolicy{}, fmt.Errorf("unmarshalling WLAN XML policy: expected <WLANProfile> tag, got <%s>", policy.XMLName.Local)
	}

	// Much of the policy settings are case sensitive however the hex representation of the SSID is not and
	// in some cases Windows seems to convert it to lowercase when writing the policy to the system
	for i := 0; i < len(policy.SSIDConfig.SSID); i++ {
		policy.SSIDConfig.SSID[i].Hex = strings.ToLower(policy.SSIDConfig.SSID[i].Hex)
	}
	return policy, nil
}

type wlanXmlPolicy struct {
	XMLName    xml.Name
	Name       string                  `xml:"name"`
	SSIDConfig wlanXmlPolicySSIDConfig `xml:"SSIDConfig"`
}

type wlanXmlPolicySSIDConfig struct {
	SSID []wlanXmlPolicySSID `xml:"SSID"`
	// Note that this field is optional so if we ever do more inspection of these policies
	// we likely need to
	SSIDPrefix   wlanXmlPolicySSIDPrefix `xml:"SSIDPrefix"`
	NonBroadcast bool                    `xml:"nonBroadcast"`
}

type wlanXmlPolicySSID struct {
	Hex  string `xml:"hex"`
	Name string `xml:"name"`
}

type wlanXmlPolicySSIDPrefix struct {
	Name string `xml:"name"`
}

// We have seen cases where Windows will "upgrade" a profile based on what it sees when it actually
// connects to a network, for instance if a profile specifies WPA2 but the interface and network
// support WPA3 it will "upgrade" the profile to WPA3 and that is what gets returned when querying
// the device. This behavior is undocumented but precludes comparing profiles too strictly.
// Because of this we have opted for a simple comparison that ensures a profile matching
// basic fields like name, SSID and (non-)broadcast status is considered equal.
func (a wlanXmlPolicy) Equal(b wlanXmlPolicy) bool {
	fmt.Printf("Comparing %s and %s\n", a.Name, b.Name)
	if a.Name != b.Name {
		return false
	}

	fmt.Printf("Comparing %t and %t\n", a.SSIDConfig.NonBroadcast, b.SSIDConfig.NonBroadcast)
	if a.SSIDConfig.NonBroadcast != b.SSIDConfig.NonBroadcast {
		return false
	}
	fmt.Printf("Comparing %s and %s\n", a.SSIDConfig.SSIDPrefix.Name, b.SSIDConfig.SSIDPrefix.Name)
	if a.SSIDConfig.SSIDPrefix.Name != b.SSIDConfig.SSIDPrefix.Name {
		return false
	}

	fmt.Printf("Comparing %d and %d\n", len(a.SSIDConfig.SSID), len(b.SSIDConfig.SSID))
	if len(a.SSIDConfig.SSID) != len(b.SSIDConfig.SSID) {
		return false
	}

	a.sortSSIDs()
	b.sortSSIDs()
	for i := range a.SSIDConfig.SSID {
		fmt.Printf("Comparing %s %s and %s %s\n", a.SSIDConfig.SSID[i].Hex, a.SSIDConfig.SSID[i].Name, b.SSIDConfig.SSID[i].Hex, b.SSIDConfig.SSID[i].Name)
		if !a.SSIDConfig.SSID[i].Equal(b.SSIDConfig.SSID[i]) {
			return false
		}
	}
	return true
}

func (a *wlanXmlPolicy) sortSSIDs() {
	slices.SortFunc(a.SSIDConfig.SSID, func(i, j wlanXmlPolicySSID) int {
		if i.Hex == j.Hex {
			return strings.Compare(i.Name, j.Name)
		}
		return strings.Compare(i.Hex, j.Hex)
	})
}

func (a wlanXmlPolicySSID) Equal(b wlanXmlPolicySSID) bool {
	return a.Hex == b.Hex && a.Name == b.Name
}
