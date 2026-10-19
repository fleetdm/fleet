//go:build windows

package amcache

import (
	"encoding/binary"
	"unicode/utf16"
)

// A minimal regf writer for tests: one hive bin, every cell allocated, fast (lh) subkey indexes
// and compressed (Latin-1) names.

type tkey struct {
	name      string
	lastWrite uint64 // FILETIME
	values    []tval
	subkeys   []tkey
}

type tval struct {
	name string
	typ  uint32
	data []byte
}

const (
	regSZ    = 1
	regDWORD = 4
	regQWORD = 11
)

func sz(s string) tval { return tval{typ: regSZ, data: utf16z(s)} }

func szv(name, s string) tval { v := sz(s); v.name = name; return v }

func dwordv(name string, n uint32) tval {
	return tval{name: name, typ: regDWORD, data: binary.LittleEndian.AppendUint32(nil, n)}
}

func qwordv(name string, n uint64) tval {
	return tval{name: name, typ: regQWORD, data: binary.LittleEndian.AppendUint64(nil, n)}
}

func utf16z(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return append(b, 0, 0)
}

// filetime converts unix seconds to a FILETIME.
func filetime(unix int64) uint64 { return uint64(unix)*10000000 + filetimeEpoch } // #nosec G115 -- test values are positive

type hiveWriter struct {
	bins []byte // hive bins data; offsets are relative to file offset 0x1000
}

// alloc appends an allocated cell holding payload and returns its offset.
func (w *hiveWriter) alloc(payload []byte) uint32 {
	size := (4 + len(payload) + 7) &^ 7
	off := uint32(len(w.bins)) // #nosec G115
	cell := make([]byte, size)
	binary.LittleEndian.PutUint32(cell, uint32(-int32(size))) // #nosec G115
	copy(cell[4:], payload)
	w.bins = append(w.bins, cell...)
	return off
}

func (w *hiveWriter) value(v tval) uint32 {
	p := make([]byte, 20+len(v.name))
	copy(p, "vk")
	binary.LittleEndian.PutUint16(p[2:], uint16(len(v.name))) // #nosec G115
	if len(v.data) <= 4 {
		binary.LittleEndian.PutUint32(p[4:], uint32(len(v.data))|0x80000000) // #nosec G115
		copy(p[8:12], v.data)
	} else {
		binary.LittleEndian.PutUint32(p[4:], uint32(len(v.data))) // #nosec G115
		binary.LittleEndian.PutUint32(p[8:], w.alloc(v.data))
	}
	binary.LittleEndian.PutUint32(p[12:], v.typ)
	binary.LittleEndian.PutUint16(p[16:], 1) // VALUE_COMP_NAME
	copy(p[20:], v.name)
	return w.alloc(p)
}

// lh writes a fast subkey index over the given nk cells.
func (w *hiveWriter) lh(children ...uint32) uint32 {
	p := make([]byte, 4+8*len(children))
	copy(p, "lh")
	binary.LittleEndian.PutUint16(p[2:], uint16(len(children))) // #nosec G115
	for i, c := range children {
		binary.LittleEndian.PutUint32(p[4+8*i:], c)
	}
	return w.alloc(p)
}

// key writes k and its descendants and returns the nk cell offset. subkeyList, when non-zero,
// overrides the subkey index offset.
func (w *hiveWriter) key(k tkey, flags uint16, subkeyList uint32) uint32 {
	var children []uint32
	for _, c := range k.subkeys {
		children = append(children, w.key(c, 0x20, 0))
	}
	list := uint32(0xFFFFFFFF)
	if len(children) > 0 {
		list = w.lh(children...)
	}
	if subkeyList != 0 {
		list = subkeyList
	}

	valList := uint32(0xFFFFFFFF)
	if len(k.values) > 0 {
		var offs []byte
		for _, v := range k.values {
			offs = binary.LittleEndian.AppendUint32(offs, w.value(v))
		}
		valList = w.alloc(offs)
	}

	p := make([]byte, 76+len(k.name))
	copy(p, "nk")
	binary.LittleEndian.PutUint16(p[2:], flags)
	binary.LittleEndian.PutUint64(p[4:], k.lastWrite)
	binary.LittleEndian.PutUint32(p[20:], uint32(len(children))) // #nosec G115
	binary.LittleEndian.PutUint32(p[28:], list)
	binary.LittleEndian.PutUint32(p[32:], 0xFFFFFFFF)
	binary.LittleEndian.PutUint32(p[36:], uint32(len(k.values))) // #nosec G115
	binary.LittleEndian.PutUint32(p[40:], valList)
	binary.LittleEndian.PutUint32(p[48:], 0xFFFFFFFF)
	binary.LittleEndian.PutUint16(p[72:], uint16(len(k.name))) // #nosec G115
	copy(p[76:], k.name)
	return w.alloc(p)
}

// hive assembles the base block and a single hive bin, padded to a 4096-byte multiple.
func (w *hiveWriter) hive(rootCell uint32, seq1, seq2 uint32) []byte {
	binSize := (len(w.bins) + 4095) &^ 4095
	out := make([]byte, 0x1000+binSize)

	copy(out, "regf")
	binary.LittleEndian.PutUint32(out[4:], seq1)
	binary.LittleEndian.PutUint32(out[8:], seq2)
	binary.LittleEndian.PutUint32(out[20:], 1) // major
	binary.LittleEndian.PutUint32(out[24:], 5) // minor
	binary.LittleEndian.PutUint32(out[32:], 1) // format
	binary.LittleEndian.PutUint32(out[36:], rootCell)
	binary.LittleEndian.PutUint32(out[40:], uint32(binSize)) // #nosec G115
	binary.LittleEndian.PutUint32(out[44:], 1)

	bin := out[0x1000:]
	copy(bin, w.bins)
	copy(bin, "hbin")
	binary.LittleEndian.PutUint32(bin[8:], uint32(binSize)) // #nosec G115
	return out
}

// newHiveWriter reserves the 0x20-byte hbin header, so offsets returned by alloc are relative to
// the start of hive bins data, as the format requires.
func newHiveWriter() *hiveWriter { return &hiveWriter{bins: make([]byte, 0x20, 4096)} }

// buildHive returns a hive whose root contains a single "Root" key with the given children.
func buildHive(keys ...tkey) []byte {
	w := newHiveWriter()
	root := w.key(tkey{name: "{11517B7C-E79D-4e20-961B-75A811715ADD}", subkeys: []tkey{{name: "Root", subkeys: keys}}}, 0x2c, 0)
	return w.hive(root, 1, 1)
}
