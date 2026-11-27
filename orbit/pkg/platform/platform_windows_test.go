//go:build windows
// +build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestUUIDNotPresent(t *testing.T) {
	uuidBytes := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := isValidUUID(uuidBytes)

	assert.NotNil(t, err, "UUID not present: ")
}

func TestUUIDNotSet(t *testing.T) {
	uuidBytes := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	_, err := isValidUUID(uuidBytes)

	assert.NotNil(t, err, "UUID not set: ")
}

func TestUUIDInvalidSize(t *testing.T) {
	uuidBytes := []byte{0x11, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	_, err := isValidUUID(uuidBytes)

	assert.NotNil(t, err, "UUID validation size: ")
}

func TestUUIDValid(t *testing.T) {
	uuidBytes := []byte{0x11, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00}
	_, err := isValidUUID(uuidBytes)

	assert.Nil(t, err, "UUID validation error: ")
}

func TestUUIDStringsIncludedInSectionPayload(t *testing.T) {
	payloadWithUUIDAndStrings := []byte{0x11, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x73, 0x6d, 0x62, 0x69, 0x6f, 0x73, 0x00}
	_, err := isValidUUID(payloadWithUUIDAndStrings)

	assert.Nil(t, err, "UUID validation error: ")
}

func TestChmodRestrictFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restricted.txt")
	require.NoError(t, os.WriteFile(path, []byte("contents"), 0o600))
	require.NoError(t, ChmodRestrictFile(path))

	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	require.NoError(t, err)
	control, _, err := sd.Control()
	require.NoError(t, err)
	assert.NotZero(t, control&windows.SE_DACL_PROTECTED, "DACL must not inherit from the parent directory")

	dacl, _, err := sd.DACL()
	require.NoError(t, err)
	require.NotNil(t, dacl)
	var granted []string
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		require.NoError(t, windows.GetAce(dacl, i, &ace))
		if ace.Mask == 0 {
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)) //nolint:gosec // G103: the ACE stores its SID inline.
		granted = append(granted, sid.String())
		assert.True(t, sid.IsWellKnown(windows.WinLocalSystemSid) || sid.IsWellKnown(windows.WinBuiltinAdministratorsSid),
			"unexpected access granted to %s", sid.String())
	}
	assert.Len(t, granted, 2)
}
