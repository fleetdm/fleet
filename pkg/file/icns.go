package file

import (
	"encoding/binary"
	"fmt"
)

// ExtractPNGFromICNS parses an ICNS file and returns the largest embedded PNG image.
// Modern ICNS files (macOS 10.7+) store large icon sizes (256x256, 512x512, 1024x1024)
// as raw PNG data inside the container. The ICNS format is:
//
//	4 bytes: magic "icns"
//	4 bytes: total file length (big-endian)
//	repeated chunks:
//	  4 bytes: type tag (e.g. "ic10", "ic14")
//	  4 bytes: chunk length including header (big-endian)
//	  N bytes: data (PNG or other format)
func ExtractPNGFromICNS(data []byte) ([]byte, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("icns data too short")
	}
	if string(data[:4]) != "icns" {
		return nil, fmt.Errorf("not an icns file")
	}

	totalLen := min(int(binary.BigEndian.Uint32(data[4:8])), len(data))

	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47} // \x89PNG

	var bestPNG []byte
	offset := 8
	for offset+8 <= totalLen {
		chunkLen := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		if chunkLen < 8 || offset+chunkLen > totalLen {
			break
		}

		chunkData := data[offset+8 : offset+chunkLen]

		// Check if the chunk data starts with PNG magic bytes
		if len(chunkData) >= 4 && string(chunkData[:4]) == string(pngMagic) {
			// Keep the largest PNG we find
			if len(chunkData) > len(bestPNG) {
				bestPNG = chunkData
			}
		}

		offset += chunkLen
	}

	if bestPNG == nil {
		return nil, fmt.Errorf("no PNG data found in icns file")
	}

	return bestPNG, nil
}
