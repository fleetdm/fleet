package client

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// mobileconfigForTest generates a minimal .mobileconfig plist for use in tests.
func mobileconfigForTest(name, identifier string, vars ...string) []byte {
	var varsStr strings.Builder
	for i, v := range vars {
		if !strings.HasPrefix(v, "FLEET_VAR_") {
			v = "FLEET_VAR_" + v
		}
		varsStr.WriteString(fmt.Sprintf("<key>Var %d</key><string>$%s</string>", i, v))
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>%s</string>
	<key>PayloadIdentifier</key>
	<string>%s</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>%s</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	%s
</dict>
</plist>
`, name, identifier, uuid.New().String(), varsStr.String()))
}

// syncMLForTest generates a minimal SyncML XML snippet for use in tests.
func syncMLForTest(locURI string) []byte {
	return []byte(fmt.Sprintf(`
<Add>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Add>
<Replace>
  <Item>
    <Target>
      <LocURI>%s</LocURI>
    </Target>
  </Item>
</Replace>`, locURI, locURI))
}
