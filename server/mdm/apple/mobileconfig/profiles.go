package mobileconfig

import "text/template"

// FleetdProfileOptions are the keys required to execute a
// FleetdProfileTemplate.
type FleetdProfileOptions struct {
	PayloadType  string
	EnrollSecret string
	ServerURL    string
}

// FleetdProfileTemplate is used to configure orbit's EnrollSecret and
// ServerURL for packages that have been generated without those values.
//
// This is useful when you want to have a single, publicly accessible (possibly
// signed + notarized) fleetd package that you can use for different
// teams/servers.
//
// Internally, this is used by Fleet MDM to configure the installer delivered
// to hosts during DEP enrollment.
var FleetdProfileTemplate = template.Must(template.New("").Option("missingkey=error").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <array>
      <dict>
        <key>EnrollSecret</key>
        <string>{{ .EnrollSecret }}</string>
        <key>FleetURL</key>
        <string>{{ .ServerURL }}</string>
        <key>EnableScripts</key>
        <true />
        <key>PayloadDisplayName</key>
        <string>Fleetd configuration</string>
        <key>PayloadIdentifier</key>
        <string>{{ .PayloadType }}</string>
        <key>PayloadType</key>
        <string>{{ .PayloadType }}</string>
        <key>PayloadUUID</key>
        <string>476F5334-D501-4768-9A31-1A18A4E1E807</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
      </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>Fleetd configuration</string>
    <key>PayloadIdentifier</key>
    <string>{{ .PayloadType }}</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>0C6AFB45-01B6-4E19-944A-123CD16381C7</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadDescription</key>
    <string>Default configuration for the fleetd agent.</string>
  </dict>
</plist>
`))
