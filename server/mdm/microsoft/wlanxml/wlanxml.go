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
	// We try to unmarshal the string to see if it looks like a valid WLAN XML profile
	_, err := unmarshal(text)
	return err == nil
}

func Equal(a, b string) (bool, error) {
	aProfile, err := unmarshal(a)
	if err != nil {
		return false, fmt.Errorf("unmarshalling WLAN XML profile a: %w", err)
	}
	bProfile, err := unmarshal(b)
	if err != nil {
		return false, fmt.Errorf("unmarshalling WLAN XML profile b: %w", err)
	}
	return aProfile.Equal(bProfile), nil
}

func unmarshal(w string) (wlanXmlProfile, error) {
	// This whole thing will be XML Encoded so step 1 is just to decode it
	var unescaped string
	err := xml.Unmarshal([]byte("<wlanxml>"+w+"</wlanxml>"), &unescaped)
	if err != nil {
		return wlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile to string: %w", err)
	}

	var profile wlanXmlProfile
	err = xml.Unmarshal([]byte(unescaped), &profile)
	if err != nil {
		return wlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile: %w", err)
	}
	if profile.XMLName.Local != "WLANProfile" {
		return wlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile: expected <WLANProfile> tag, got <%s>", profile.XMLName.Local)
	}

	for i := 0; i < len(profile.SSIDConfig.SSID); i++ {
		profile.SSIDConfig.SSID[i].normalize()
	}
	profile.SSIDConfig.SSIDPrefix.normalize()

	return profile, nil
}

type wlanXmlProfile struct {
	XMLName    xml.Name
	Name       string                   `xml:"name"`
	SSIDConfig wlanXmlProfileSSIDConfig `xml:"SSIDConfig"`
}

type wlanXmlProfileSSIDConfig struct {
	SSID         []wlanXmlProfileSSID `xml:"SSID"`
	SSIDPrefix   wlanXmlProfileSSID   `xml:"SSIDPrefix"`
	NonBroadcast bool                 `xml:"nonBroadcast"`
}

type wlanXmlProfileSSID struct {
	Hex  string `xml:"hex"`
	Name string `xml:"name"`
}

func (s *wlanXmlProfileSSID) normalize() {
	// Microsoft's documentation says that the hex representation overrides the Name when both are
	// present. In testing, if a profile is provided with only the Name and not the hex
	// representation, Microsoft generates Hex and it is present in the profile returned. As such we
	// will convert name to hex on the way in for use in comparisons
	if s.Hex == "" && s.Name != "" {
		s.Hex = fmt.Sprintf("%x", s.Name)
	}

	// Most of the profile settings are case sensitive however the hex representation of the SSID is not and
	// in some cases Windows converts it to uppercase when writing a profile to the system which was provided
	// with uppercase alpha characters.
	s.Hex = strings.ToUpper(s.Hex)
}

func (s wlanXmlProfileSSID) Equal(b wlanXmlProfileSSID) bool {
	return s.Hex == b.Hex
}

// We have seen cases where Windows will "upgrade" a profile based on what it sees when it actually
// connects to a network, for instance if a profile specifies WPA2 but the interface and network
// support WPA3 it will "upgrade" the profile to WPA3 and that is what gets returned when querying
// the device. This behavior is undocumented but precludes comparing profiles too strictly.
// Because of this we have opted for a simple comparison that ensures a profile matching
// basic fields like name, SSID and (non-)broadcast status is considered equal.
func (a wlanXmlProfile) Equal(b wlanXmlProfile) bool {
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
func (a *wlanXmlProfile) sortSSIDs() {
	slices.SortFunc(a.SSIDConfig.SSID, func(i, j wlanXmlProfileSSID) int {
		return strings.Compare(i.Hex, j.Hex)
	})
}
