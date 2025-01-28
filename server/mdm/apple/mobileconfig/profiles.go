package mobileconfig

import (
	"text/template"
)

var funcMap = map[string]any{
	"xml": XMLEscapeString,
}

// FleetdProfileOptions are the keys required to execute a
// FleetdProfileTemplate.
type FleetdProfileOptions struct {
	PayloadType  string
	PayloadName  string
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
var FleetdProfileTemplate = template.Must(template.New("").Funcs(funcMap).Option("missingkey=error").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <array>
      <dict>
        <key>EnrollSecret</key>
        <string>{{ .EnrollSecret | xml }}</string>
        <key>FleetURL</key>
        <string>{{ .ServerURL }}</string>
        <key>EnableScripts</key>
        <true />
        <key>PayloadDisplayName</key>
        <string>{{ .PayloadName }}</string>
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
    <string>{{ .PayloadName }}</string>
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

// FleetCARootTemplateOptions are the keys required to execute a
// FleetCARootTemplate.
type FleetCARootTemplateOptions struct {
	PayloadName       string
	PayloadIdentifier string
	Certificate       string
}

var FleetCARootTemplate = template.Must(template.New("").Option("missingkey=error").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <array>
      <dict>
        <key>PayloadCertificateFileName</key>
        <string>CertificateRoot</string>
        <key>PayloadContent</key>
        <data>{{ .Certificate }}</data>
        <key>PayloadDescription</key>
        <string>{{ .PayloadName }}</string>
        <key>PayloadDisplayName</key>
        <string>{{ .PayloadName }}</string>
        <key>PayloadIdentifier</key>
        <string>{{ .PayloadIdentifier }}.certpayload</string>
        <key>PayloadType</key>
        <string>com.apple.security.root</string>
        <key>PayloadUUID</key>
        <string>B295992E-861A-4F92-902-17BCF4E33C61</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
	<key>AllowAllAppsAccess</key>
	<false/>
      </dict>
    </array>
    <key>PayloadDisplayName</key>
    <string>{{ .PayloadName }}</string>
    <key>PayloadIdentifier</key>
    <string>{{ .PayloadIdentifier }}</string>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadUUID</key>
    <string>4F5428DE-05B6-4965-87AD-532CAFC35FCF</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
  </dict>
</plist>
`))

var OTAMobileConfigTemplate = template.Must(template.New("").Funcs(funcMap).Option("missingkey=error").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadContent</key>
    <dict>
      <key>URL</key>
      <string>{{ .URL }}</string>
      <key>DeviceAttributes</key>
      <array>
        <string>UDID</string>
        <string>VERSION</string>
        <string>PRODUCT</string>
	<string>SERIAL</string>
      </array>
    </dict>
    <key>PayloadOrganization</key>
    <string>{{ xml .Organization }}</string>
    <key>PayloadDisplayName</key>
    <string>{{ xml .Organization }} enrollment</string>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadUUID</key>
    <string>fdb376e5-b5bb-4d8c-829e-e90865f990c9</string>
    <key>PayloadIdentifier</key>
    <string>com.fleetdm.fleet.mdm.apple.ota</string>
    <key>PayloadType</key>
    <string>Profile Service</string>
  </dict>
</plist>`))
