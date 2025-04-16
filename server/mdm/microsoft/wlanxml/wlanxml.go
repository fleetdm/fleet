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
	// in some cases Windows converts it to uppercase when writing a policy to the system which was provided
	// with uppercase alpha characters.
	for i := 0; i < len(policy.SSIDConfig.SSID); i++ {
		policy.SSIDConfig.SSID[i].normalize()
	}
	policy.SSIDConfig.SSIDPrefix.normalize()

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
	SSIDPrefix   wlanXmlPolicySSID `xml:"SSIDPrefix"`
	NonBroadcast bool              `xml:"nonBroadcast"`
}

type wlanXmlPolicySSID struct {
	Hex  string `xml:"hex"`
	Name string `xml:"name"`
}

func (s *wlanXmlPolicySSID) normalize() {
	// Microsoft's documentation says that the hex representation overrides the Name when both are
	// present. In testing, if a policy is provided with only the Name and not the hex
	// representation, Microsoft generates Hex and it is present in the policy returned. As such we
	// will convert name to hex on the way in for use in comparisons
	if s.Hex == "" && s.Name != "" {
		s.Hex = fmt.Sprintf("%x", s.Name)
	}

	// Most of the policy settings are case sensitive however the hex representation of the SSID is not and
	// in some cases Windows converts it to uppercase when writing a policy to the system which was provided
	// with uppercase alpha characters.
	s.Hex = strings.ToUpper(s.Hex)
}

func (a wlanXmlPolicySSID) Equal(b wlanXmlPolicySSID) bool {
	return a.Hex == b.Hex
}

// We have seen cases where Windows will "upgrade" a profile based on what it sees when it actually
// connects to a network, for instance if a profile specifies WPA2 but the interface and network
// support WPA3 it will "upgrade" the profile to WPA3 and that is what gets returned when querying
// the device. This behavior is undocumented but precludes comparing profiles too strictly.
// Because of this we have opted for a simple comparison that ensures a profile matching
// basic fields like name, SSID and (non-)broadcast status is considered equal.
func (a wlanXmlPolicy) Equal(b wlanXmlPolicy) bool {
	if a.Name != b.Name {
		return false
	}

	if a.SSIDConfig.NonBroadcast != b.SSIDConfig.NonBroadcast {
		return false
	}

	if !a.SSIDConfig.SSIDPrefix.Equal(b.SSIDConfig.SSIDPrefix) {
		return false
	}

	if len(a.SSIDConfig.SSID) != len(b.SSIDConfig.SSID) {
		return false
	}

	a.sortSSIDs()
	b.sortSSIDs()
	for i := range a.SSIDConfig.SSID {
		if !a.SSIDConfig.SSID[i].Equal(b.SSIDConfig.SSID[i]) {
			return false
		}
	}
	return true
}

// a profile may have multiple SSIDs.
func (a *wlanXmlPolicy) sortSSIDs() {
	slices.SortFunc(a.SSIDConfig.SSID, func(i, j wlanXmlPolicySSID) int {
		return strings.Compare(i.Hex, j.Hex)
	})
}
