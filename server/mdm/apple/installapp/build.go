// Package installapp builds Apple MDM InstallApplication command payloads.
//
// The shape varies by enrollment type: standard device enrollments include
// a ChangeManagementState=Managed entry, while Account-Driven User Enrolled
// (BYOD) hosts must omit it — Apple rejects the command with "The MDM request
// is invalid." otherwise.
package installapp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
)

// Input is the set of values needed to build a single InstallApplication
// command. Exactly one of ITunesStoreID (for VPP) or ManifestURL (for in-house
// .ipa) must be populated.
type Input struct {
	// CommandUUID is the UUID Apple uses to correlate command results.
	CommandUUID string
	// ITunesStoreID is the App Store adam ID for a VPP install. Apple's spec
	// expects an integer; the value is embedded verbatim inside <integer>...
	// </integer> so the caller is responsible for passing a numeric string.
	ITunesStoreID string
	// ManifestURL is the manifest endpoint for an in-house .ipa install.
	ManifestURL string
	// ManagementFlags is the Apple management-flag bitmask. Mobile (iOS/iPadOS)
	// devices use 1 (remove app on MDM removal); macOS uses 0.
	ManagementFlags int
	// IsUserEnrollment indicates this command targets an Account-Driven User
	// Enrolled (BYOD) host. When true, ChangeManagementState is omitted.
	IsUserEnrollment bool
}

func (in Input) validate() error {
	if in.CommandUUID == "" {
		return errors.New("installapp: CommandUUID is required")
	}
	hasAdam := in.ITunesStoreID != ""
	hasManifest := in.ManifestURL != ""
	switch {
	case hasAdam && hasManifest:
		return errors.New("installapp: ITunesStoreID and ManifestURL are mutually exclusive")
	case !hasAdam && !hasManifest:
		return errors.New("installapp: one of ITunesStoreID or ManifestURL is required")
	}
	return nil
}

// BuildInstallApplicationXML returns a plist-encoded InstallApplication
// command for the given input. The output is a complete <?xml ...?> document
// suitable for nanomdm storage.
func BuildInstallApplicationXML(in Input) (string, error) {
	if err := in.validate(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
        <key>InstallAsManaged</key>
        <true/>
        <key>ManagementFlags</key>
        <integer>`)
	fmt.Fprintf(&buf, "%d", in.ManagementFlags)
	buf.WriteString(`</integer>
`)
	if !in.IsUserEnrollment {
		buf.WriteString(`        <key>ChangeManagementState</key>
        <string>Managed</string>
`)
	}
	buf.WriteString(`        <key>Options</key>
        <dict>
            <key>PurchaseMethod</key>
            <integer>1</integer>
        </dict>
        <key>RequestType</key>
        <string>InstallApplication</string>
`)
	if in.ITunesStoreID != "" {
		buf.WriteString(`        <key>iTunesStoreID</key>
        <integer>`)
		// ITunesStoreID is documented as integer, but Fleet's columns store it
		// as a string. Embedding via xml.EscapeText avoids breaking the plist if
		// a non-conforming value appears in test fixtures or future schemas.
		_ = xml.EscapeText(&buf, []byte(in.ITunesStoreID))
		buf.WriteString(`</integer>
`)
	} else {
		buf.WriteString(`        <key>ManifestURL</key>
        <string>`)
		_ = xml.EscapeText(&buf, []byte(in.ManifestURL))
		buf.WriteString(`</string>
`)
	}
	buf.WriteString(`    </dict>
    <key>CommandUUID</key>
    <string>`)
	_ = xml.EscapeText(&buf, []byte(in.CommandUUID))
	buf.WriteString(`</string>
</dict>
</plist>`)

	return buf.String(), nil
}
