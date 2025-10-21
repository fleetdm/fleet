package notarize

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"
)

func init() {
	childCommands["info-accepted"] = testCmdInfoAcceptedSubmission
	childCommands["info-invalid"] = testCmdInfoInvalidSubmission
}

func TestInfo_accepted(t *testing.T) {
	info, err := info(context.Background(), "foo", &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "info-accepted"),
	})

	require := require.New(t)
	require.NoError(err)
	require.Equal(info.RequestUUID, "32684f68-d63e-49ba-9234-25eeec84b369")
	require.Equal(info.Status, "Accepted")
	require.Equal(info.StatusMessage, "Successfully received submission info")
}

func TestInfo_invalid(t *testing.T) {
	info, err := info(context.Background(), "foo", &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "info-invalid"),
	})

	require := require.New(t)
	require.NoError(err)
	require.Equal(info.RequestUUID, "cfd69166-8e2f-1397-8636-ec06f98e3597")
	require.Equal(info.Status, "Invalid")
}

// testCmdInfoAcceptedSubmission mimicks an accepted submission.
func testCmdInfoAcceptedSubmission() int {
	fmt.Println(strings.TrimSpace(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
		<key>createdDate</key>
		<string>2023-08-01T08:22:19.939Z</string>
		<key>id</key>
		<string>32684f68-d63e-49ba-9234-25eeec84b369</string>
		<key>message</key>
		<string>Successfully received submission info</string>
		<key>name</key>
		<string>binary.zip</string>
		<key>status</key>
		<string>Accepted</string>
</dict>
</plist>
`))
	return 0
}

// testCmdInfoInvalidSubmission mimicks an invalid submission.
func testCmdInfoInvalidSubmission() int {
	fmt.Println(strings.TrimSpace(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
		<key>createdDate</key>
		<string>2023-08-01T08:12:11.193Z</string>
		<key>id</key>
		<string>cfd69166-8e2f-1397-8636-ec06f98e3597</string>
		<key>message</key>
		<string>Successfully received submission info</string>
		<key>name</key>
		<string>binary.zip</string>
		<key>status</key>
		<string>Invalid</string>
</dict>
</plist>
`))
	return 0
}
