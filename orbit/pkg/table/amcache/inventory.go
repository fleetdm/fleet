//go:build windows

package amcache

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/rs/zerolog"
	"www.velocidex.com/golang/regparser"
)

// The Root\Inventory* keys of a modern (Windows 10 1809+) Amcache hive, one per table.
const (
	keyApplicationFile     = "InventoryApplicationFile"
	keyApplication         = "InventoryApplication"
	keyApplicationShortcut = "InventoryApplicationShortcut"
	keyDriverBinary        = "InventoryDriverBinary"
	keyDriverPackage       = "InventoryDriverPackage"
	keyDevicePnp           = "InventoryDevicePnp"
	keyDeviceContainer     = "InventoryDeviceContainer"
)

var inventoryKeys = [...]string{
	keyApplicationFile,
	keyApplication,
	keyApplicationShortcut,
	keyDriverBinary,
	keyDriverPackage,
	keyDevicePnp,
	keyDeviceContainer,
}

// record is one subkey of an inventory key. values holds the key's values by registry value
// name, decoded only as far as a string; column-specific decoding happens in the table layer.
type record struct {
	name      string
	lastWrite int64
	values    map[string]string
}

// inventory holds only the keys the hive actually has; a missing key serves zero rows.
type inventory struct {
	legacy bool
	keys   map[string][]record
}

var (
	errNotAmcache      = errors.New("amcache: hive has no Amcache keys")
	errDecodedTooLarge = errors.New("amcache: decoded values exceed the budget for this hive")
)

// columnValues is, per inventory key, the set of registry value names some column reads.
// Other values are never decoded or kept.
var columnValues = func() map[string]map[string]struct{} {
	out := make(map[string]map[string]struct{}, len(specs))
	for _, s := range specs {
		names := make(map[string]struct{}, len(s.cols))
		for _, c := range s.cols {
			if c.src != "" {
				names[c.src] = struct{}{}
			}
		}
		out[s.key] = names
	}
	return out
}()

// parseState is what one parse threads through the record decoders.
type parseState struct {
	logger  zerolog.Logger
	reg     *regparser.Registry
	hiveLen int64
	// budget is the remaining decoded bytes. Cells can be shared by any number of records, so
	// without it a small crafted hive decodes into unbounded memory.
	budget int64
}

// regparser's walkers allocate from file-declared counts without bounds, so these caps are
// enforced before each call.
const (
	// A 36 KiB hive whose root declares one subkey can make CM_KEY_NODE.Subkeys materialize
	// millions of nodes, so subkey counts are checked against the index structure itself.
	maxSubkeys = 1_000_000
	// The largest real value count seen on a record is 34.
	maxValues = 256
	// Floor for the decoded-bytes budget, which is otherwise twice the hive size.
	minDecodedBudget = 16 << 20
	// A real Hwids value on InventoryDriverPackage is ~1 MB.
	maxValueBytes = 1 << 20
	// A real subkey index is ri -> lh, i.e. depth 1.
	maxIndexDepth = 32
)

// regf cell signatures.
const (
	sigNk = 0x6b6e
	sigLf = 0x666c
	sigLh = 0x686c
	sigRi = 0x6972
	sigLi = 0x696c
	sigDb = 0x6264
)

var (
	errIndexCycle     = errors.New("amcache: subkey index cell revisited")
	errIndexDepth     = errors.New("amcache: subkey index nested past the depth cap")
	errTooManySubkeys = errors.New("amcache: subkey index declares more slots than the cap admits")
)

// Each warning fires at most once per process, so a hive that is corrupt in the same way
// thousands of times logs one line.
const (
	warnLegacy = iota
	warnSubkeyIndex
	warnValueCount
	warnValueSize
	warnValueShape
	warnRecordPanic
	nWarnConds
)

var warnOnce [nWarnConds]sync.Once

// warn must never be given names or values read from the hive.
func warn(logger zerolog.Logger, cond int, format string, args ...any) {
	warnOnce[cond].Do(func() { logger.Warn().Msgf(format, args...) })
}

// indexSlots returns an upper bound on how many key nodes CM_KEY_NODE.Subkeys would return,
// without materializing them. regparser's CM_KEY_INDEX.Subkeys recurses with no depth limit
// or cycle detection, so an ri cell that references itself overflows the goroutine stack,
// which is fatal and can't be recovered. The depth cap bounds path length; the visited set
// bounds path count (a chain of N distinct ri cells otherwise walks 2^N frames).
func indexSlots(reg *regparser.Registry, off uint32, limit int) (int, error) {
	type frame struct {
		off   uint32
		depth int
	}
	seen := make(map[uint32]struct{}, 16)
	stack := []frame{{off, 0}}
	total := 0
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.depth > maxIndexDepth {
			return 0, errIndexDepth
		}
		if _, dup := seen[f.off]; dup {
			return 0, errIndexCycle
		}
		seen[f.off] = struct{}{}

		cell := reg.Profile.HCELL(reg.Reader, 0x1000+int64(f.off))
		switch cell.Signature() {
		case sigLf, sigLh:
			total += int(cell.KeyIndexFast().Count())
		case sigRi, sigLi:
			idx := cell.KeyIndex()
			for _, child := range regparser.ParseSafeArray_uint32(
				reg.Reader, idx.Offset+reg.Profile.Off_CM_KEY_INDEX_List, int(idx.Count())) {
				switch reg.Profile.HCELL(reg.Reader, 0x1000+int64(child)).Signature() {
				case sigNk:
					total++
				case sigLf, sigLh, sigRi, sigLi:
					stack = append(stack, frame{child, f.depth + 1})
				}
			}
		}
		if total > limit || len(stack) > limit {
			return 0, errTooManySubkeys
		}
	}
	return total, nil
}

// subkeys is the only caller of CM_KEY_NODE.Subkeys, so the bound check can't be bypassed.
func subkeys(reg *regparser.Registry, n *regparser.CM_KEY_NODE) ([]*regparser.CM_KEY_NODE, error) {
	off := n.SubKeyLists()[0]
	// A leaf key has 0xFFFFFFFF here, which resolves past EOF to a zero signature.
	if reg.Profile.HCELL(reg.Reader, 0x1000+int64(off)).Signature() == 0 {
		return nil, nil
	}
	if _, err := indexSlots(reg, off, maxSubkeys); err != nil {
		return nil, err
	}
	return n.Subkeys(), nil
}

// parseInventory extracts the inventory keys from an acquired hive. It classifies Root's
// children in one pass instead of calling regparser's OpenKey per key, which would walk
// unbounded subkey indexes on every path component.
func parseInventory(ctx context.Context, logger zerolog.Logger, hive []byte) (*inventory, error) {
	reg, err := regparser.NewRegistry(bytes.NewReader(hive))
	if err != nil {
		return nil, fmt.Errorf("amcache: base block: %w", err)
	}
	hiveLen := int64(len(hive))

	base := reg.Profile.HCELL(reg.Reader, 0x1000+int64(reg.BaseBlock.RootCell())).KeyNode()
	if base == nil {
		return nil, errNotAmcache
	}
	top, err := subkeys(reg, base)
	if err != nil {
		return nil, fmt.Errorf("amcache: base block subkey index: %w", err)
	}
	var root *regparser.CM_KEY_NODE
	for _, n := range top {
		if strings.EqualFold(nodeName(reg, n), "Root") {
			root = n
			break
		}
	}
	if root == nil {
		return nil, errNotAmcache
	}
	children, err := subkeys(reg, root)
	if err != nil {
		return nil, fmt.Errorf("amcache: Root subkey index: %w", err)
	}

	found := make(map[string]*regparser.CM_KEY_NODE, len(inventoryKeys))
	var legacy bool
	for _, n := range children {
		name := nodeName(reg, n)
		for _, k := range inventoryKeys {
			if strings.EqualFold(name, k) {
				found[k] = n
				break
			}
		}
		if strings.EqualFold(name, "File") || strings.EqualFold(name, "Programs") {
			legacy = true
		}
	}
	if len(found) == 0 {
		if legacy {
			warn(logger, warnLegacy, "amcache: hive uses the pre-Windows 10 1809 format, which is not supported; tables return zero rows")
			return &inventory{legacy: true, keys: map[string][]record{}}, nil
		}
		return nil, errNotAmcache
	}

	st := &parseState{logger: logger, reg: reg, hiveLen: hiveLen, budget: max(2*hiveLen, minDecodedBudget)}
	inv := &inventory{keys: make(map[string][]record, len(found))}
	for _, k := range inventoryKeys {
		n, ok := found[k]
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		subs, err := subkeys(reg, n)
		if err != nil {
			warn(logger, warnSubkeyIndex, "amcache: %s: subkey index rejected (%v); that key serves zero rows", k, err)
			continue
		}
		recs := make([]record, 0, len(subs))
		// A real index never lists the same key cell twice.
		seen := make(map[int64]struct{}, len(subs))
		for _, s := range subs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if _, dup := seen[s.Offset]; dup {
				continue
			}
			seen[s.Offset] = struct{}{}
			r, ok := st.parseRecord(s, columnValues[k])
			if st.budget < 0 {
				return nil, errDecodedTooLarge
			}
			if ok {
				recs = append(recs, r)
			}
		}
		// The walk follows the lh hash order; sorting makes output stable across parses.
		slices.SortStableFunc(recs, func(a, b record) int { return strings.Compare(a.name, b.name) })
		inv.keys[k] = recs
	}
	return inv, nil
}

// parseRecord builds one record from one subkey, decoding only the wanted values. A malformed
// record is skipped and its siblings survive; the recover is a backstop for regparser panics not
// covered by the guards.
func (st *parseState) parseRecord(n *regparser.CM_KEY_NODE, wanted map[string]struct{}) (rec record, ok bool) {
	logger, reg := st.logger, st.reg
	defer func() {
		if p := recover(); p != nil {
			warn(logger, warnRecordPanic, "amcache: a record was skipped after a parser panic (%T); later panics are not logged", p)
			rec, ok = record{}, false
		}
	}()

	rec = record{name: nodeName(reg, n), lastWrite: lastWrite(reg, n), values: map[string]string{}}
	// Names count too: distinct key cells can declare overlapping 64 KiB names.
	st.budget -= int64(len(rec.name))
	if st.budget < 0 {
		return record{}, false
	}

	if count := n.ValueList().Count(); count > maxValues {
		warn(logger, warnValueCount, "amcache: a record declares %d values, over the %d cap; its values are dropped", count, maxValues)
		return rec, true
	}
	vals := n.Values()
	rec.values = make(map[string]string, len(wanted))
	for _, v := range vals {
		name := valueName(reg, v)
		if _, ok := wanted[name]; !ok {
			continue
		}
		s, decoded := st.decodeValue(v)
		st.budget -= int64(len(s))
		if st.budget < 0 {
			return record{}, false
		}
		if decoded {
			rec.values[name] = s
		}
	}
	return rec, true
}

func valueName(reg *regparser.Registry, v *regparser.CM_KEY_VALUE) string {
	if v.NameLength() == 0 {
		return ""
	}
	name := regparser.ParseSafeArray_byte(reg.Reader, v.Offset+reg.Profile.Off_CM_KEY_VALUE_Name, int(v.NameLength()))
	if v.Flags()&0x0001 != 0 { // VALUE_COMP_NAME
		return latin1(name)
	}
	return utf16le(name)
}

// decodeValue renders a registry value as a string, or reports that it was skipped (REG_BINARY,
// unknown types, and malformed values).
func (st *parseState) decodeValue(v *regparser.CM_KEY_VALUE) (string, bool) {
	logger, reg, hiveLen := st.logger, st.reg, st.hiveLen
	raw := v.DataLength()
	size := raw & 0x7fffffff
	// ValueData reads one byte at a time until EOF, driven by the declared length.
	if size > maxValueBytes {
		warn(logger, warnValueSize, "amcache: a value declares %d bytes, over the %d byte cap; the value is dropped", size, maxValueBytes)
		return "", false
	}
	inline := raw&0x80000000 != 0

	switch v.Type() {
	case regparser.REG_DWORD, regparser.REG_DWORD_BIG_ENDIAN, regparser.REG_QWORD:
		// ValueData checks only the declared size and then indexes the bytes it actually read, so
		// a size/type mismatch, an inline QWORD (inline yields 4 bytes) or a data cell outside the
		// hive would panic.
		want := int64(4)
		if v.Type() == regparser.REG_QWORD {
			want = 8
		}
		if int64(size) != want {
			warn(logger, warnValueShape, "amcache: an integer value declares %d bytes for a %d byte type; the value is dropped", size, want)
			return "", false
		}
		if inline {
			if want != 4 {
				warn(logger, warnValueShape, "amcache: a %d byte integer value is marked inline; the value is dropped", want)
				return "", false
			}
		} else {
			cell := reg.Profile.HCELL(reg.Reader, 0x1000+int64(v.Data()))
			if cell.Signature() == sigDb || cell.Payload()+want > hiveLen {
				warn(logger, warnValueShape, "amcache: an integer value points at a data cell that cannot hold its %d bytes; the value is dropped", want)
				return "", false
			}
		}
		return strconv.FormatUint(v.ValueData().Uint64, 10), true

	case regparser.REG_SZ, regparser.REG_EXPAND_SZ:
		// Not ValueData().String: regparser's UTF-16 decoder has its BOM branches inverted, so a
		// value starting with U+FEFF (attacker-controlled PE metadata) comes back byte-swapped.
		s := utf16le(v.ValueData().Data)
		// regparser over-reads inline values to four bytes, so cut at the first NUL.
		if i := strings.IndexByte(s, 0); i >= 0 {
			s = s[:i]
		}
		return strings.ToValidUTF8(strings.TrimSpace(s), "�"), true

	case regparser.REG_MULTI_SZ:
		data := v.ValueData().Data
		out := make([]string, 0, 4)
		for i := 0; i+1 < len(data); {
			j := i
			for j+1 < len(data) && (data[j] != 0 || data[j+1] != 0) {
				j += 2
			}
			if p := utf16le(data[i:j]); p != "" {
				out = append(out, p)
			}
			i = j + 2
		}
		return strings.Join(out, ","), true
	}

	return "", false
}

// nodeName honors KEY_COMP_NAME, which regparser's Name ignores, so an uncompressed (UTF-16)
// name isn't returned as raw bytes.
func nodeName(reg *regparser.Registry, n *regparser.CM_KEY_NODE) string {
	name := regparser.ParseSafeArray_byte(reg.Reader, n.Offset+reg.Profile.Off_CM_KEY_NODE__Name, int(n.NameLength()))
	if n.Flags()&0x20 != 0 {
		return latin1(name)
	}
	return utf16le(name)
}

// latin1 decodes a compressed name, which is one byte per character over Latin-1 (not ASCII),
// so non-English hosts can have bytes >= 0x80.
func latin1(b []byte) string {
	r := make([]rune, len(b))
	for i, c := range b {
		r[i] = rune(c)
	}
	return string(r)
}

func utf16le(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

// FILETIME ticks (100 ns) from 1601-01-01 to 1970-01-01 and to 2100-01-01.
const (
	filetimeEpoch = 116444736000000000
	filetimeMax   = 157469184000000000
)

// lastWrite reads the raw FILETIME because regparser's LastWriteTime uses an unsigned
// subtraction that turns a zero FILETIME into the year 60056. 0 means unknown.
func lastWrite(reg *regparser.Registry, n *regparser.CM_KEY_NODE) int64 {
	ft := regparser.ParseUint64(reg.Reader, n.Offset+reg.Profile.Off_CM_KEY_NODE_LastWriteTime)
	if ft < filetimeEpoch || ft >= filetimeMax {
		return 0
	}
	return int64(ft/10000000) - filetimeEpoch/10000000
}
