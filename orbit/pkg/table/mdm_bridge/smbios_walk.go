package mdm_bridge

// The SMBIOS table is a sequence of firmware-provided structures, each tagged with a type byte; type 1 is System Information,
// which holds the 16-byte system UUID at offset 8. Windows derives the expected Flags value from that UUID.
//
// embeddedModeFallback is what Windows hashes instead when its SMBIOS walk misses the type 1 structure. It's an undocumented
// constant from embeddedmodesvcapi.dll (10.0.26100.8115), confirmed on affected hosts via IsEmbeddedModeAllowed.
var embeddedModeFallback = []byte{0x57, 0x26, 0xdb, 0xcb, 0x8d, 0xce, 0x27, 0x47, 0x9d, 0x45, 0x1e, 0x18, 0x66, 0xf3, 0x7e, 0x21}

// findSMBIOSUUID returns the raw UUID bytes of the first type 1 structure in an SMBIOS structure table, or nil. With emulateWindowsBug
// it mirrors the walk in embeddedmodesvcapi.dll, which never resets its "previous byte was zero" state between structures: a
// string-less structure after the first one ends one byte early, and the walk can drift past type 1 or onto a stray 0x01 byte.
func findSMBIOSUUID(smbiosTable []byte, emulateWindowsBug bool) []byte {
	prevZero := false
	for i := 0; i+1 < len(smbiosTable); {
		if smbiosTable[i] == 1 {
			if i+24 > len(smbiosTable) {
				return nil
			}
			return smbiosTable[i+8 : i+24]
		}
		if !emulateWindowsBug {
			prevZero = false
		}
		// Skip the formatted area, then find the double zero that ends the string set.
		j := i + int(smbiosTable[i+1])
		for ; j < len(smbiosTable) && (smbiosTable[j] != 0 || !prevZero); j++ {
			prevZero = smbiosTable[j] == 0
		}
		if j >= len(smbiosTable) {
			return nil
		}
		i = j + 1
	}
	return nil
}
