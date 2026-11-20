package mdm_bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// smbiosStructure builds an SMBIOS structure: header (type, length, zero handle), formatted data, then the string set.
func smbiosStructure(structType byte, formatted []byte, strs ...string) []byte {
	out := append([]byte{structType, byte(4 + len(formatted)), 0, 0}, formatted...)
	for _, str := range strs {
		out = append(append(out, str...), 0)
	}
	if len(strs) == 0 {
		out = append(out, 0)
	}
	return append(out, 0)
}

func TestFindSMBIOSUUID(t *testing.T) {
	uuid := []byte{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf, 0xb0}
	systemInfo := smbiosStructure(1, append([]byte{1, 2, 3, 4}, uuid...), "Vendor", "Model")
	dellOEM := smbiosStructure(218, []byte{0xff, 0xff})

	testCases := []struct {
		name        string
		table       []byte
		windowsUUID []byte
		specUUID    []byte
	}{
		{
			name:        "type 0 then type 1",
			table:       slices.Concat(smbiosStructure(0, nil, "BIOS vendor", "1.0"), systemInfo),
			windowsUUID: uuid,
			specUUID:    uuid,
		},
		{
			name:        "string-less OEM structures before type 0 make Windows miss type 1",
			table:       slices.Concat(dellOEM, dellOEM, dellOEM, dellOEM, smbiosStructure(0, nil, "BIOS vendor", "1.0"), systemInfo),
			windowsUUID: nil,
			specUUID:    uuid,
		},
		{
			// Windows lands one byte early on the second zero of the string-less structure, reads the next type byte (2) as a length,
			// and stops on the 0x01 in the type 2 formatted area.
			name:        "Windows drifts onto a stray 0x01",
			table:       slices.Concat(smbiosStructure(0, nil, "a"), smbiosStructure(218, nil), smbiosStructure(2, []byte{1, 0}), systemInfo),
			windowsUUID: append([]byte{1, 2, 3, 4}, uuid[:12]...),
			specUUID:    uuid,
		},
		{
			name:        "type 1 UUID ends at the end of the table",
			table:       append([]byte{1, 0x18, 0, 0, 1, 2, 3, 4}, uuid...),
			windowsUUID: uuid,
			specUUID:    uuid,
		},
		{
			name:        "no type 1",
			table:       slices.Concat(smbiosStructure(0, nil, "BIOS vendor"), smbiosStructure(127, nil)),
			windowsUUID: nil,
			specUUID:    nil,
		},
		{
			name:        "truncated type 1",
			table:       []byte{1, 0x1b, 0, 0, 1},
			windowsUUID: nil,
			specUUID:    nil,
		},
		{
			name:        "empty table",
			table:       nil,
			windowsUUID: nil,
			specUUID:    nil,
		},
	}
	fallbackHash := sha256.Sum256(embeddedModeFallback)
	require.Equal(t, "2161fec15aac0f2a2957e96bb581e8084d383eefefc3f220702c3ae838746889", hex.EncodeToString(fallbackHash[:]))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.windowsUUID, findSMBIOSUUID(tc.table, true))
			require.Equal(t, tc.specUUID, findSMBIOSUUID(tc.table, false))
		})
	}
}
