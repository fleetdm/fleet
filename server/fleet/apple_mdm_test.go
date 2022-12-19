package fleet

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMDMAppleCheckinReq(t *testing.T) {
	expected := "application/x-apple-aspen-mdm-checkin"

	// should be true
	req := &http.Request{
		Header: map[string][]string{
			"Content-Type": {expected},
		},
	}
	require.True(t, isMDMAppleCheckinReq(req))

	// should be false
	req = &http.Request{
		Header: map[string][]string{
			"Content-Type": {"x-apple-aspen-deviceinfo"},
		},
	}
	require.False(t, isMDMAppleCheckinReq(req))
}

func TestDecodeMDMAppleCheckinRequest(t *testing.T) {
	testSerial := "test-serial"
	testUDID := "test-udid"

	// decode host details from request XML if MessageType is "Authenticate"
	req := &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUDID, "MacBook Pro"))),
	}
	host := &MDMAppleHostDetails{}
	err := decodeMDMAppleCheckinReq(req, host)
	require.NoError(t, err)
	require.Equal(t, testSerial, host.SerialNumber)
	require.Equal(t, testUDID, host.UDID)
	// require.Equal(t, "MacBook Pro", host.Model)

	// do nothing if MessageType is not "Authenticate"
	req = &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("TokenUpdate", testSerial, testUDID, "MacBook Pro"))),
	}
	host = &MDMAppleHostDetails{}
	err = decodeMDMAppleCheckinReq(req, host)
	require.NoError(t, err)
	require.Empty(t, host.SerialNumber)
	require.Empty(t, host.UDID)
}

func xmlForTest(msgType string, serial string, udid string, model string) string {
	return fmt.Sprintf(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>MessageType</key>
	<string>%s</string>
	<key>SerialNumber</key>
	<string>%s</string>
	<key>UDID</key>
	<string>%s</string>
	<key>Model</key>
	<string>%s</string>
</dict>
</plist>`, msgType, serial, udid, model)
}
