package file

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildICNS constructs a minimal icns container with the given chunks.
// Each chunk is {tag, data}; the helper writes the 4-byte length (header+data)
// before the payload just like a real .icns file.
func buildICNS(t *testing.T, chunks ...struct {
	tag  string
	data []byte
},
) []byte {
	t.Helper()
	var body bytes.Buffer
	for _, c := range chunks {
		require.Len(t, c.tag, 4)
		body.WriteString(c.tag)
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(8+len(c.data))) //nolint:gosec // dismiss G115
		body.Write(lenBuf)
		body.Write(c.data)
	}

	var out bytes.Buffer
	out.WriteString("icns")
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(8+body.Len())) //nolint:gosec // dismiss G115
	out.Write(lenBuf)
	out.Write(body.Bytes())
	return out.Bytes()
}

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47}

func TestExtractPNGFromICNS(t *testing.T) {
	smallPNG := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0xAB}, 32)...)
	largePNG := append(append([]byte{}, pngMagic...), bytes.Repeat([]byte{0xCD}, 256)...)
	nonPNG := append([]byte{'i', 'c', 'p', '4'}, bytes.Repeat([]byte{0x00}, 16)...)

	t.Run("returns largest PNG", func(t *testing.T) {
		icns := buildICNS(t,
			struct {
				tag  string
				data []byte
			}{"ic07", smallPNG},
			struct {
				tag  string
				data []byte
			}{"ic09", nonPNG},
			struct {
				tag  string
				data []byte
			}{"ic10", largePNG},
		)

		out, err := ExtractPNGFromICNS(icns)
		require.NoError(t, err)
		assert.Equal(t, largePNG, out)
	})

	t.Run("no PNG chunks", func(t *testing.T) {
		icns := buildICNS(t, struct {
			tag  string
			data []byte
		}{"ic09", nonPNG})

		_, err := ExtractPNGFromICNS(icns)
		assert.ErrorContains(t, err, "no PNG data found")
	})

	t.Run("missing magic", func(t *testing.T) {
		_, err := ExtractPNGFromICNS([]byte("XXXX\x00\x00\x00\x08"))
		assert.ErrorContains(t, err, "not an icns file")
	})

	t.Run("too short", func(t *testing.T) {
		_, err := ExtractPNGFromICNS([]byte("icns"))
		assert.ErrorContains(t, err, "too short")
	})

	t.Run("truncated chunk length stops iteration", func(t *testing.T) {
		// Build a container whose header claims more bytes than exist,
		// so the loop should terminate cleanly rather than panic.
		icns := buildICNS(t, struct {
			tag  string
			data []byte
		}{"ic10", smallPNG})
		// Corrupt the chunk length to be larger than the file.
		binary.BigEndian.PutUint32(icns[8+4:8+8], 0xFFFFFFFF)

		_, err := ExtractPNGFromICNS(icns)
		assert.ErrorContains(t, err, "no PNG data found")
	})
}
