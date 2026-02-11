package acme

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"os"

	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
)

const (
	// DirectoryPath is the path for the ACME directory service.
	DirectoryPath = "acme/acme/directory" // TODO: this will need to be updated to use the Fleet server URL when we have the ACME endpoints routed in the server.
)

func GenerateEnrollmentProfileMobileconfig(orgName, fleetURL, deviceSerial, topic string) ([]byte, error) {
	discoveryURL, err := ResolveAppleACMEURL()
	if err != nil {
		return nil, fmt.Errorf("resolve Apple SCEP url: %w", err)
	}
	serverURL, err := apple_mdm.ResolveAppleMDMURL(fleetURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple MDM url: %w", err)
	}

	var buf bytes.Buffer
	if err := acmeEnrollmentProfileMobileconfigTemplate.Funcs(funcMap).Execute(&buf, struct {
		Organization     string
		DirectoryURL     string
		Topic            string
		ServerURL        string
		ClientIdentifier string
		SerialTemplate   string
	}{
		Organization:     orgName,
		DirectoryURL:     discoveryURL,
		Topic:            topic,
		ServerURL:        serverURL,
		ClientIdentifier: deviceSerial,
		SerialTemplate:   `%SerialNumber%`, // Apple replaces this placeholder with the device's serial number during enrollment
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// TODO: Figure out why the generated profile escaopes the left angle bracket in the opening
	// `<?xml` tag and remove the need for this replacement.
	return bytes.Replace(buf.Bytes(), []byte("&lt;"), []byte("<"), 1), nil
}

// TODO: this will need to be updated to use the Fleet server URL when we have the ACME endpoints routed in the server.
func ResolveAppleACMEURL() (string, error) {
	base := os.Getenv("FLEET_DEV_STEP_CA_SERVER")
	if base == "" {
		return "", fmt.Errorf("FLEET_DEV_STEP_CA_SERVER environment variable is not set")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse FLEET_DEV_STEP_CA_SERVER: %w", err)
	}
	u.Path = DirectoryPath
	return u.String(), nil
}

var funcMap = map[string]any{
	"xml": mobileconfig.XMLEscapeString,
}

// acmeEnrollmentProfileMobileconfigTemplate is the template Fleet uses to assemble a .mobileconfig enrollment profile to serve to devices.
//
// During a profile replacement, the system updates payloads with the same PayloadIdentifier and
// PayloadUUID in the old and new profiles.
var acmeEnrollmentProfileMobileconfigTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>Attest</key>
			<true/>
			<key>ClientIdentifier</key>
			<string>{{ .ClientIdentifier | xml }}</string>
			<key>DirectoryURL</key>
			<string>{{ .DirectoryURL | xml }}</string>
			<key>HardwareBound</key>
			<true/>
			<key>KeySize</key>
			<integer>384</integer>
			<key>KeyType</key>
			<string>ECSECPrimeRandom</string>
			<key>PayloadDisplayName</key>
			<string>Fleet Identity ACME</string>
			<key>PayloadIdentifier</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadType</key>
			<string>com.apple.security.acme</string>
			<key>PayloadUUID</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>Subject</key>
			<array>
				<array>
					<array>
						<string>CN</string>
						<string>{{ .SerialTemplate | xml }}</string>
					</array>
				</array>
			</array>
		</dict>
		<dict>
			<key>AccessRights</key>
			<integer>8191</integer>
			<key>CheckOutWhenRemoved</key>
			<true/>
			<key>IdentityCertificateUUID</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadIdentifier</key>
			<string>com.fleetdm.fleet.mdm.apple.mdm</string>
			<key>PayloadType</key>
			<string>com.apple.mdm</string>
			<key>PayloadUUID</key>
			<string>29713130-1602-4D27-90C9-B822A295E44E</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ServerCapabilities</key>
			<array>
				<string>com.apple.mdm.per-user-connections</string>
				<string>com.apple.mdm.bootstraptoken</string>
			</array>
			<key>ServerURL</key>
			<string>{{ .ServerURL | xml }}</string>
			<key>SignMessage</key>
			<true/>
			<key>Topic</key>
			<string>{{ .Topic | xml }}</string>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>{{ .Organization | xml }} enrollment</string>
	<key>PayloadIdentifier</key>
	<string>` + apple_mdm.FleetPayloadIdentifier + `</string>
	<key>PayloadOrganization</key>
	<string>{{ .Organization | xml }}</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))
