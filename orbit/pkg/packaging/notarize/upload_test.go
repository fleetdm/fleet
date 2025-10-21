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
	childCommands["upload-success"] = testCmdUploadSuccess
	childCommands["upload-exit-status"] = testCmdUploadExitStatus
}

func TestUpload_success(t *testing.T) {
	uuid, err := upload(context.Background(), &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "upload-success"),
	})

	require.NoError(t, err)
	require.Equal(t, uuid, "cfd69166-8e2f-1397-8636-ec06f98e3597")
}

func TestUpload_exitStatus(t *testing.T) {
	uuid, err := upload(context.Background(), &Options{
		Logger:  hclog.L(),
		BaseCmd: childCmd(t, "upload-exit-status"),
	})

	require.Error(t, err)
	require.Empty(t, uuid)
}

// testCmdUploadSuccess mimicks a successful submission.
func testCmdUploadSuccess() int {
	fmt.Println(strings.TrimSpace(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
		<key>id</key>
		<string>cfd69166-8e2f-1397-8636-ec06f98e3597</string>
		<key>message</key>
		<string>Successfully uploaded file</string>
		<key>path</key>
		<string>/path/to/binary.zip</string>
</dict>
</plist>
`))
	return 0
}

// testCmdUploadExitStatus
func testCmdUploadExitStatus() int {
	return 1
}
