package mdm

import "testing"

// MDM check-in and command-result messages are always XML.
func TestDecodeXMLOnly(t *testing.T) {
	nonXML := []byte("bplist00\xd1\x01\x02")
	if _, err := DecodeCheckin(nonXML); err == nil {
		t.Error("DecodeCheckin: want error for non-XML input, got nil")
	}
	if _, err := DecodeCommandResults(nonXML); err == nil {
		t.Error("DecodeCommandResults: want error for non-XML input, got nil")
	}

	checkin := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MessageType</key>
	<string>Authenticate</string>
	<key>UDID</key>
	<string>00000000-1111</string>
	<key>Topic</key>
	<string>com.apple.mgmt.External.test</string>
</dict>
</plist>`)
	if _, err := DecodeCheckin(checkin); err != nil {
		t.Errorf("DecodeCheckin rejected valid XML: %v", err)
	}

	result := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CommandUUID</key>
	<string>abc-123</string>
	<key>Status</key>
	<string>Acknowledged</string>
</dict>
</plist>`)
	if _, err := DecodeCommandResults(result); err != nil {
		t.Errorf("DecodeCommandResults rejected valid XML: %v", err)
	}
}
