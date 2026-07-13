package apple_mdm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/micromdm/plist"
)

// Limits applied to binary property lists before decoding. They are generous
// for the device-info plists these endpoints handle, which are a single flat
// dictionary of scalar values.
const (
	// binaryPlistMagic is the prefix that selects the binary plist decoder.
	// Apple doesn't fully document all versions, but "bplist00" and "bplist01" are known.
	// Using "bplist0" lets us accept any "bplist0<digit>" version header.
	binaryPlistMagic = "bplist0"
	plistTrailerSize = 32

	maxPlistObjects = 1 << 16 // distinct objects (offset-table size)
	maxPlistDepth   = 16      // reference nesting
	maxPlistNodes   = 1 << 16 // objects after references are expanded
)

var (
	errPlistTooComplex = errors.New("plist exceeds parsing limits")
	errMalformedPlist  = errors.New("malformed binary plist")
)

// BoundedPlistUnmarshal decodes a plist into v. Binary plists are first checked
// against the depth, object-count, and object-size limits above; XML plists are
// decoded directly (their size is bounded by the caller's body limit).
func BoundedPlistUnmarshal(data []byte, v any) error {
	if bytes.HasPrefix(data, []byte(binaryPlistMagic)) {
		if err := checkBinaryPlistBounds(data); err != nil {
			return err
		}
	}
	return plist.Unmarshal(data, v)
}

// checkBinaryPlistBounds walks a binary plist's object references, rejecting
// input that exceeds the limits or points outside the data region.
func checkBinaryPlistBounds(data []byte) error {
	// See comment on binaryPlistMagic above for why the +1 is needed
	if len(data) < len(binaryPlistMagic)+1+plistTrailerSize {
		return fmt.Errorf("%w: shorter than minimum size", errMalformedPlist)
	}

	// Trailer is the final 32 bytes (CFBinaryPlistTrailer).
	trailer := data[len(data)-plistTrailerSize:]
	offsetIntSize := trailer[6]
	objectRefSize := trailer[7]
	numObjects := binary.BigEndian.Uint64(trailer[8:16])
	rootObject := binary.BigEndian.Uint64(trailer[16:24])
	offsetTableOffset := binary.BigEndian.Uint64(trailer[24:32])

	if offsetIntSize == 0 || offsetIntSize > 8 || objectRefSize == 0 || objectRefSize > 8 {
		return fmt.Errorf("%w: invalid integer sizes", errMalformedPlist)
	}
	if numObjects == 0 {
		return fmt.Errorf("%w: no objects", errMalformedPlist)
	}
	if numObjects > maxPlistObjects {
		return fmt.Errorf("%w: %d objects", errPlistTooComplex, numObjects)
	}

	trailerStart := uint64(len(data) - plistTrailerSize) //nolint:gosec // dismiss G115, length is bounded above
	tableBytes := numObjects * uint64(offsetIntSize)
	if offsetTableOffset > trailerStart || tableBytes > trailerStart-offsetTableOffset {
		return fmt.Errorf("%w: offset table out of bounds", errMalformedPlist)
	}

	offsetTable := make([]uint64, numObjects)
	pos := offsetTableOffset
	for i := range offsetTable {
		offsetTable[i] = readUintBE(data[pos : pos+uint64(offsetIntSize)])
		pos += uint64(offsetIntSize)
	}

	b := &plistBounder{
		data:          data,
		offsetTable:   offsetTable,
		objectRefSize: objectRefSize,
		dataEnd:       offsetTableOffset, // objects live before the offset table
	}
	return b.visit(rootObject, 0)
}

type plistBounder struct {
	data          []byte
	offsetTable   []uint64
	objectRefSize uint8
	dataEnd       uint64
	nodes         int
}

// visit walks the object at index, recursing into array and dict references.
// The depth and node guards keep the walk itself bounded: a cycle stops at
// maxPlistDepth, and reference expansion stops at maxPlistNodes.
func (b *plistBounder) visit(index uint64, depth int) error {
	if depth > maxPlistDepth {
		return fmt.Errorf("%w: nesting deeper than %d", errPlistTooComplex, maxPlistDepth)
	}
	b.nodes++
	if b.nodes > maxPlistNodes {
		return fmt.Errorf("%w: more than %d expanded objects", errPlistTooComplex, maxPlistNodes)
	}
	if index >= uint64(len(b.offsetTable)) {
		return fmt.Errorf("%w: object ref %d out of range", errMalformedPlist, index)
	}
	offset := b.offsetTable[index]
	if offset >= b.dataEnd {
		return fmt.Errorf("%w: object offset out of range", errMalformedPlist)
	}

	// High nibble of the marker byte is the object type (CFBinaryPList.c).
	cur := plistCursor{data: b.data, pos: offset, end: b.dataEnd}
	marker, err := cur.readByte()
	if err != nil {
		return err
	}
	switch marker >> 4 {
	case 0xa: // array: count object refs
		count, err := cur.readCount(marker)
		if err != nil {
			return err
		}
		if count > cur.remaining()/uint64(b.objectRefSize) {
			return fmt.Errorf("%w: array refs exceed input", errMalformedPlist)
		}
		return b.visitRefs(&cur, count, depth)
	case 0xd: // dictionary: count key refs followed by count value refs
		count, err := cur.readCount(marker)
		if err != nil {
			return err
		}
		if count > cur.remaining()/uint64(b.objectRefSize)/2 {
			return fmt.Errorf("%w: dictionary refs exceed input", errMalformedPlist)
		}
		return b.visitRefs(&cur, 2*count, depth)
	case 0x2: // real: 1<<(low nibble) bytes follow inline
		nbytes := uint64(1) << (marker & 0xf)
		if cur.pos+nbytes < cur.pos || cur.pos+nbytes > b.dataEnd {
			return fmt.Errorf("%w: real size exceeds input", errMalformedPlist)
		}
		return nil
	case 0x4, 0x5: // data, ASCII string: count single-byte units
		return b.checkPayload(&cur, marker, 1)
	case 0x6: // UTF-16 string: count two-byte units
		return b.checkPayload(&cur, marker, 2)
	default:
		// Other types are fixed-size or non-recursive; nothing to bound.
		return nil
	}
}

func (b *plistBounder) visitRefs(cur *plistCursor, total uint64, depth int) error {
	for range total {
		ref, err := cur.readRef(b.objectRefSize)
		if err != nil {
			return err
		}
		if err := b.visit(ref, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// checkPayload verifies a variable-length scalar (data or string) declares a
// payload that fits within the object region.
func (b *plistBounder) checkPayload(cur *plistCursor, marker byte, unitSize uint64) error {
	count, err := cur.readCount(marker)
	if err != nil {
		return err
	}
	size := count * unitSize
	if size/unitSize != count {
		return fmt.Errorf("%w: object size overflow", errMalformedPlist)
	}
	if cur.pos+size < cur.pos || cur.pos+size > b.dataEnd {
		return fmt.Errorf("%w: object size exceeds input", errMalformedPlist)
	}
	return nil
}

// plistCursor reads forward within a single object's bytes, bounded by end.
type plistCursor struct {
	data []byte
	pos  uint64
	end  uint64
}

func (c *plistCursor) remaining() uint64 {
	if c.pos >= c.end {
		return 0
	}
	return c.end - c.pos
}

func (c *plistCursor) readByte() (byte, error) {
	if c.pos >= c.end {
		return 0, fmt.Errorf("%w: unexpected end of object data", errMalformedPlist)
	}
	v := c.data[c.pos]
	c.pos++
	return v, nil
}

func (c *plistCursor) readBytes(n uint64) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	if c.pos+n < c.pos || c.pos+n > c.end {
		return nil, fmt.Errorf("%w: unexpected end of object data", errMalformedPlist)
	}
	v := c.data[c.pos : c.pos+n]
	c.pos += n
	return v, nil
}

// readCount decodes the variable-length count used by data, strings, arrays, and dicts.
func (c *plistCursor) readCount(marker byte) (uint64, error) {
	if marker&0xf != 0xf {
		return uint64(marker & 0xf), nil
	}
	sizeMarker, err := c.readByte()
	if err != nil {
		return 0, err
	}
	nbytes := uint64(1) << (sizeMarker & 0x0f)
	if nbytes > 8 {
		return 0, fmt.Errorf("%w: invalid count size", errMalformedPlist)
	}
	buf, err := c.readBytes(nbytes)
	if err != nil {
		return 0, err
	}
	return readUintBE(buf), nil
}

func (c *plistCursor) readRef(size uint8) (uint64, error) {
	buf, err := c.readBytes(uint64(size))
	if err != nil {
		return 0, err
	}
	return readUintBE(buf), nil
}

func readUintBE(b []byte) uint64 {
	var n uint64
	for _, c := range b {
		n = n<<8 | uint64(c)
	}
	return n
}
