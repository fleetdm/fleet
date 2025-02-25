package packaging

import "text/template"

// Best reference I could find:
// http://s.sudre.free.fr/Stuff/Ivanhoe/FLAT.html
var macosPackageInfoTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<pkg-info format-version="2" identifier="{{.Identifier}}.base.pkg" version="{{.Version}}" install-location="/" auth="root">
  <payload/>
  <scripts>
    <postinstall file="./postinstall"/>
  </scripts>
  <bundle-version>
  </bundle-version>
</pkg-info>
`))

// This template is used to generate a Distribution Definition file, which
// controls the experience of the installer (the default dir, what options the
// user has, etc.)
//
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
    {{/* base.pkg specified here is the foldername that contains the package contents */}}
    <pkg-ref id="{{.Identifier}}.base.pkg" version="{{.Version}}" auth="root">#base.pkg</pkg-ref>
    {{/* this ref is collapsed with the previous, having a bundle version helps our notarization tools */}}
    <pkg-ref id="{{.Identifier}}.base.pkg">
      <bundle-version>
        <bundle id="{{.Identifier}}" path="" />
      </bundle-version>
    </pkg-ref>
	<options customize="never" hostArchitectures="arm64,x86_64" require-scripts="false" allow-external-scripts="false" />
</installer-gui-script>
`))

var macosPostinstallTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`#!/bin/bash

ln -sf /opt/orbit/bin/orbit/macos/{{.OrbitChannel}}/orbit /opt/orbit/bin/orbit/orbit
ln -sf /opt/orbit/bin/orbit/orbit /usr/local/bin/orbit
{{ if .LegacyVarLibSymlink }}
# Symlink needed to support old versions of orbit.
ln -sf /opt/orbit /var/lib/orbit
{{- end }}

{{ if .StartService -}}
DAEMON_LABEL="com.fleetdm.orbit"
DAEMON_PLIST="/Library/LaunchDaemons/${DAEMON_LABEL}.plist"

# Stop the previous desktop agent
pkill fleet-desktop || true
# Remove any pre-existing version of the config
launchctl bootout "system/${DAEMON_LABEL}"

# Make sure the launch daemon is enabled before we try to bootstrap it
launchctl enable "system/${DAEMON_LABEL}"

# Add the daemon to the launchd system.
#
# We add retries because we've seen "launchctl bootstrap" fail
# if the service is still running after bootout (in case the
# service takes a bit longer to terminate gracefully).
# We've seen this when deploying the package via an MDM server.
#
count=0
while ! launchctl bootstrap system "${DAEMON_PLIST}"; do
	sleep 1
	((count++))
	if [[ $count -eq 30 ]]; then
		echo "Failed to bootstrap system ${DAEMON_PLIST}"
		exit 1
	fi
	echo "Retrying launchctl bootstrap..."
done
echo "Successfully bootstrap system ${DAEMON_PLIST}"

# Force the daemon to start
launchctl kickstart "system/${DAEMON_LABEL}"
{{- end }}
`))

// Note it's important not to start the orbit binary in
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
		<string>/opt/orbit/fleet.pem</string>
		{{- end }}
		{{- if .EnrollSecret }}
		<key>ORBIT_ENROLL_SECRET_PATH</key>
		<string>/opt/orbit/secret.txt</string>
		{{- end }}
		{{- if .FleetURL }}
		<key>ORBIT_FLEET_URL</key>
		<string>{{ .FleetURL }}</string>
		{{- end }}
		{{- if .UseSystemConfiguration }}
		<key>ORBIT_USE_SYSTEM_CONFIGURATION</key>
		<string>{{ .UseSystemConfiguration }}</string>
		{{- end }}
		{{- if .EnableScripts }}
		<key>ORBIT_ENABLE_SCRIPTS</key>
		<string>{{ .EnableScripts }}</string>
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
		{{- if .UpdateTLSServerCertificate }}
		<key>ORBIT_UPDATE_TLS_CERTIFICATE</key>
		<string>/opt/orbit/update.pem</string>
		{{- end }}
		{{- if .Desktop }}
		<key>ORBIT_FLEET_DESKTOP</key>
		<string>true</string>
		<key>ORBIT_DESKTOP_CHANNEL</key>
		<string>{{ .DesktopChannel }}</string>
		{{- if .FleetDesktopAlternativeBrowserHost }}
		<key>ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST</key>
		<string>{{ .FleetDesktopAlternativeBrowserHost }}</string>
		{{- end }}
		{{- end }}
		<key>ORBIT_UPDATE_INTERVAL</key>
		<string>{{ .OrbitUpdateInterval }}</string>
		{{- if and (ne .HostIdentifier "") (ne .HostIdentifier "uuid") }}
		<key>ORBIT_HOST_IDENTIFIER</key>
		<string>{{ .HostIdentifier }}</string>
		{{- end }}
		{{- if .DisableKeystore }}
		<key>ORBIT_DISABLE_KEYSTORE</key>
		<string>true</string>
		{{- end }}
		{{- if .OsqueryDB }}
		<key>ORBIT_OSQUERY_DB</key>
		<string>{{ .OsqueryDB }}</string>
		{{- end }}
	</dict>
	<key>KeepAlive</key>
	<true/>
	<key>Label</key>
	<string>com.fleetdm.orbit</string>
	<key>ProgramArguments</key>
	<array>
		<string>/opt/orbit/bin/orbit/orbit</string>
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
