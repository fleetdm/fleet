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

func unmarshal(w string) (WlanXmlProfile, error) {
	// This whole thing will be XML Encoded so step 1 is just to decode it
	var unescaped string
	err := xml.Unmarshal([]byte("<wlanxml>"+w+"</wlanxml>"), &unescaped)
	if err != nil {
		return WlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile to string: %w", err)
	}

	var profile WlanXmlProfile
	err = xml.Unmarshal([]byte(unescaped), &profile)
	if err != nil {
		return WlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile: %w", err)
	}
	if profile.XMLName.Local != "WLANProfile" {
		return WlanXmlProfile{}, fmt.Errorf("unmarshalling WLAN XML profile: expected <WLANProfile> tag, got <%s>", profile.XMLName.Local)
	}

	for i := 0; i < len(profile.SSIDConfig.SSID); i++ {
		profile.SSIDConfig.SSID[i].normalize()
	}
	profile.SSIDConfig.SSIDPrefix.normalize()

	return profile, nil
}

type WlanXmlProfile struct {
	XMLName    xml.Name                 `xml:"WLANProfile"`
	Name       string                   `xml:"name"`
	SSIDConfig WlanXmlProfileSSIDConfig `xml:"SSIDConfig"`
}

type WlanXmlProfileSSIDConfig struct {
	SSID         []WlanXmlProfileSSID `xml:"SSID"`
	SSIDPrefix   WlanXmlProfileSSID   `xml:"SSIDPrefix"`
	NonBroadcast bool                 `xml:"nonBroadcast"`
}

type WlanXmlProfileSSID struct {
	Hex  string `xml:"hex,omitempty"`
	Name string `xml:"name,omitempty"`
}

func (s *WlanXmlProfileSSID) normalize() {
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

func (s WlanXmlProfileSSID) Equal(b WlanXmlProfileSSID) bool {
	return s.Hex == b.Hex
}

// We have seen cases where Windows will "upgrade" a profile based on what it sees when it actually
// connects to a network, for instance if a profile specifies WPA2 but the interface and network
// support WPA3 it will "upgrade" the profile to WPA3 and that is what gets returned when querying
// the device. This behavior is undocumented but precludes comparing profiles too strictly.
// Because of this we have opted for a simple comparison that ensures a profile matching
// basic fields like name, SSID and (non-)broadcast status is considered equal.
func (a WlanXmlProfile) Equal(b WlanXmlProfile) bool {
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
	return slices.EqualFunc(a.SSIDConfig.SSID, b.SSIDConfig.SSID, func(i, j WlanXmlProfileSSID) bool {
		return i.Equal(j)
	})
}

// a profile may have multiple SSIDs.
func (a *WlanXmlProfile) sortSSIDs() {
	slices.SortFunc(a.SSIDConfig.SSID, func(i, j WlanXmlProfileSSID) int {
		return strings.Compare(i.Hex, j.Hex)
	})
}

// Generates a WLAN XML profile with the given SSID Config and name for use in our tests both. This
// is exported so it can be used by tests outside this package.
func GenerateWLANXMLProfileForTests(name string, ssidConfig WlanXmlProfileSSIDConfig) (string, error) {
	type wlanXmlProfileForTests struct {
		WlanXmlProfile
		MSM            string `xml:",innerxml"`
		ConnectionMode string `xml:"connectionMode"`
		ConnectionType string `xml:"connectionType"`
	}
	profile := wlanXmlProfileForTests{
		WlanXmlProfile: WlanXmlProfile{
			XMLName:    xml.Name{Local: "WLANProfile", Space: "http://www.microsoft.com/networking/WLAN/profile/v1"},
			Name:       name,
			SSIDConfig: ssidConfig,
		},
		ConnectionType: "ESS",
		ConnectionMode: "auto",
		MSM: `<security>
			<authEncryption>
				<authentication>WPA2PSK</authentication>
				<encryption>AES</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
			<sharedKey>
				<keyType>passPhrase</keyType>
				<protected>false</protected>
				<keyMaterial>sup3rs3cr3t</keyMaterial>
			</sharedKey>
		</security>`,
	}
	xmlBytes, err := xml.Marshal(profile)
	if err != nil {
		return "", fmt.Errorf("Error marshaling WLAN XML profile: %w", err)
	}

	var buffer strings.Builder
	err = xml.EscapeText(&buffer, xmlBytes)
	if err != nil {
		return "", fmt.Errorf("Error escaping marshalled WLAN XML profile: %w", err)
	}
	return buffer.String(), nil
}
