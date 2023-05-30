package live_query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapBitfield(t *testing.T) {
	// empty
	assert.Equal(t, []byte{}, mapBitfield(nil))
	assert.Equal(t, []byte{}, mapBitfield([]uint{}))

	// one byte
	assert.Equal(t, []byte("\x80"), mapBitfield([]uint{0}))
	assert.Equal(t, []byte("\x40"), mapBitfield([]uint{1}))
	assert.Equal(t, []byte("\xc0"), mapBitfield([]uint{0, 1}))

	assert.Equal(t, []byte("\x08"), mapBitfield([]uint{4}))
	assert.Equal(t, []byte("\xf8"), mapBitfield([]uint{0, 1, 2, 3, 4}))
	assert.Equal(t, []byte("\xff"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7}))

	// two bytes
	assert.Equal(t, []byte("\x00\x80"), mapBitfield([]uint{8}))
	assert.Equal(t, []byte("\xff\x80"), mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8}))

	// more bytes
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 170}),
	)
	assert.Equal(
		t,
		[]byte("\xff\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00@\x00\x00\x00\x00\x00\x00 "),
		mapBitfield([]uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 113, 170}),
	)
	assert.Equal(
		t,
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01"),
		mapBitfield([]uint{79}),
	)
}
