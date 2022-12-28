package fleet

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
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

	req := &http.Request{
		Header: map[string][]string{
			"Content-Type": {"application/x-apple-aspen-mdm-checkin"},
		},
		Method: http.MethodPost,
		Body:   io.NopCloser(strings.NewReader(xmlForTest("Authenticate", testSerial, testUDID, "MacBook Pro"))),
	}
	msg, err := decodeMDMAppleCheckinReq(req)
	require.NoError(t, err)
	require.NotNil(t, msg)
	msgAuth, ok := msg.(*mdm.Authenticate)
	require.True(t, ok)
	require.Equal(t, testSerial, msgAuth.SerialNumber)
	require.Equal(t, testUDID, msgAuth.UDID)
	require.Equal(t, "MacBook Pro", msgAuth.Model)
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
