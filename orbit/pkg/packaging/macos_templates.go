package packaging

import "text/template"

// Best reference I could find:
// http://s.sudre.free.fr/Stuff/Ivanhoe/FLAT.html
var macosPackageInfoTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<pkg-info format-version="2" identifier="{{.Identifier}}.base.pkg" version="{{.Version}}" install-location="/" auth="root">
  <scripts>
    <postinstall file="./postinstall"/>
  </scripts>
  <bundle-version>
  </bundle-version>
</pkg-info>
`))

// Reference:
// https://developer.apple.com/library/archive/documentation/DeveloperTools/Reference/DistributionDefinitionRef/Chapters/Distribution_XML_Ref.html
var macosDistributionTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
	<title>Fleet osquery</title>
	<choices-outline>
	    <line choice="choiceBase"/>
    </choices-outline>
    <choice id="choiceBase" title="Fleet osquery" enabled="false" selected="true" description="Standard installation for Fleet osquery.">
        <pkg-ref id="{{.Identifier}}.base.pkg"/>
    </choice>
    <pkg-ref id="{{.Identifier}}.base.pkg" version="{{.Version}}" auth="root">#base.pkg</pkg-ref>
</installer-gui-script>
`))

var macosPostinstallTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`#!/bin/bash

ln -sf /var/lib/orbit/bin/orbit/macos/{{.OrbitChannel}}/orbit /var/lib/orbit/bin/orbit/orbit
ln -sf /var/lib/orbit/bin/orbit/orbit /usr/local/bin/orbit

{{ if .StartService -}}
DAEMON_LABEL="com.fleetdm.orbit"
DAEMON_PLIST="/Library/LaunchDaemons/${DAEMON_LABEL}.plist"

# Remove any pre-existing version of the config
launchctl bootout "system/${DAEMON_LABEL}"
# Add the daemon to the launchd system
launchctl bootstrap system "${DAEMON_PLIST}"
# Enable the daemon
launchctl enable "system/${DAEMON_LABEL}"
# Force the daemon to start
launchctl kickstart "system/${DAEMON_LABEL}"
{{- end }}
`))

// TODO set Nice?
//
//Note it's important not to start the orbit binary in
// `/usr/local/bin/orbit` because this is a path that users usually have write
// access to, and running that binary with launchd can become a privilege
// escalation vector.
var macosLaunchdTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnvironmentVariables</key>
	<dict>
		{{- if .Debug }}
		<key>ORBIT_DEBUG</key>
		<string>true</string>
		{{- end }}
		{{- if .Insecure }}
		<key>ORBIT_INSECURE</key>
		<string>true</string>
		{{- end }}
		{{- if .FleetCertificate }}
		<key>ORBIT_FLEET_CERTIFICATE</key>
		<string>/var/lib/orbit/fleet.pem</string>
		{{- end }}
		{{- if .EnrollSecret }}
		<key>ORBIT_ENROLL_SECRET_PATH</key>
		<string>/var/lib/orbit/secret.txt</string>
		{{- end }}
		{{- if .FleetURL }}
		<key>ORBIT_FLEET_URL</key>
		<string>{{ .FleetURL }}</string>
		{{- end }}
		{{- if .DisableUpdates }}
		<key>ORBIT_DISABLE_UPDATES</key>
		<string>true</string>
		{{- end }}
		<key>ORBIT_ORBIT_CHANNEL</key>
		<string>{{ .OrbitChannel }}</string>
		<key>ORBIT_OSQUERYD_CHANNEL</key>
		<string>{{ .OsquerydChannel }}</string>
		<key>ORBIT_UPDATE_URL</key>
		<string>{{ .UpdateURL }}</string>
	</dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>com.fleetdm.orbit</string>
	<key>ProgramArguments</key>
	<array>
		<string>/var/lib/orbit/bin/orbit/orbit</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardErrorPath</key>
	<string>/var/log/orbit/orbit.stderr.log</string>
	<key>StandardOutPath</key>
	<string>/var/log/orbit/orbit.stdout.log</string>
	<key>ThrottleInterval</key>
	<integer>10</integer>
</dict>
</plist>
`))
