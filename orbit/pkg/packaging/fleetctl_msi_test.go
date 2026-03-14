package packaging

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetctlMsiWxs(t *testing.T) {
	wxsPath := filepath.Join("..", "..", "..", "tools", "build-fleetctl-msi", "fleetctl.wxs")
	content, err := os.ReadFile(wxsPath)
	require.NoError(t, err, "fleetctl.wxs should exist at tools/build-fleetctl-msi/fleetctl.wxs")

	// Verify it parses as valid XML
	d := xml.NewDecoder(bytes.NewReader(content))
	for {
		_, err := d.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err, "fleetctl.wxs should be valid XML")
	}

	s := string(content)
	assert.Contains(t, s, "ProgramFiles64Folder", "should install to 64-bit Program Files")
	assert.Contains(t, s, `Name="Fleet"`, "should have Fleet top-level directory")
	assert.Contains(t, s, `Name="fleetctl"`, "should have fleetctl subdirectory")
	assert.Contains(t, s, "fleetctl.exe", "should reference fleetctl.exe")
	assert.Contains(t, s, `Name="PATH"`, "should have PATH environment element")
	assert.Contains(t, s, `System="yes"`, "PATH should be system-wide (HKLM)")
	assert.Contains(t, s, `Part="last"`, "PATH should append rather than replace")
	assert.Contains(t, s, `Permanent="no"`, "PATH entry should be removed on uninstall")
	assert.Contains(t, s, "BroadcastEnvironmentChange", "should broadcast PATH change without requiring reboot")
	assert.Contains(t, s, `InstallScope="perMachine"`, "should install per machine not per user")
	assert.Contains(t, s, "MajorUpgrade", "should have MajorUpgrade element for upgrade support")
}
