package amcache

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	notepadSha1 = "f1d2d2f924e990a2fd0b5bb7c5cfd5e1d8e4a3b8"
	hashOnly    = "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"
	driverSha1  = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	programID   = "0000a1b2c3d4e5f60718293a4b5c6d7e8f9012345678"
	containerID = "{0A1B2C3D-4E5F-6071-8293-A4B5C6D7E8F9}"
)

func fixtureHive() []byte {
	return buildHive(
		tkey{name: keyApplicationFile, subkeys: []tkey{
			{
				name:      "notepad.exe|5d8b7d2a",
				lastWrite: filetime(1700000000),
				values: []tval{
					szv("LowerCaseLongPath", `c:\windows\system32\notepad.exe`),
					szv("Name", "notepad.exe"),
					szv("FileId", "0000"+notepadSha1),
					szv("ProgramId", programID),
					szv("Publisher", "Microsoft Corporation"),
					szv("LinkDate", "04/05/2023 10:11:12"),
					qwordv("Size", 201216),
					dwordv("IsOsComponent", 1),
				},
			},
			{
				// Windows 11 style hash-only stub whose LinkDate is a version string.
				name:   "zzz|1",
				values: []tval{szv("FileId", "0000"+hashOnly), szv("LinkDate", "10.0.22621.1")},
			},
		}},
		tkey{name: keyApplication, subkeys: []tkey{
			{name: programID, values: []tval{szv("Name", "Notepad"), szv("InstallDate", "01/02/2024 03:04:05")}},
		}},
		tkey{name: keyDriverBinary, subkeys: []tkey{
			{name: "c:/windows/system32/drivers/evil.sys", values: []tval{
				szv("DriverId", "0000"+driverSha1),
				dwordv("DriverSigned", 0),
				dwordv("DriverInBox", 0),
				dwordv("DriverIsKernelMode", 1),
				dwordv("DriverTimeStamp", 0xF0000000),
				dwordv("DriverType", 0x80000000),
			}},
		}},
		tkey{name: keyDevicePnp, subkeys: []tkey{
			{name: "USBSTOR/DISK&VEN_ACME/0001", values: []tval{
				szv("Enumerator", "USBSTOR"),
				szv("ContainerId", containerID),
				szv("FirstInstallDate", "11-03-2024"),
				szv("LowerFilters", " a , b ,, c "),
			}},
		}},
		tkey{name: keyDeviceContainer, subkeys: []tkey{
			{name: containerID, values: []tval{szv("FriendlyName", "ACME Stick"), szv("IsConnected", "0")}},
		}},
	)
}

func specFor(t *testing.T, name string) spec {
	t.Helper()
	for _, s := range specs {
		if s.table == name {
			return s
		}
	}
	t.Fatalf("no table %s", name)
	return spec{}
}

func fixedStat() (time.Time, int64, error) { return time.Unix(1, 0), 1, nil }

func newTestTables(hive []byte) (*tables, *atomic.Int32) {
	var reads atomic.Int32
	read := func(context.Context) (hiveResult, error) {
		reads.Add(1)
		return hiveResult{data: hive}, nil
	}
	return newTables(zerolog.Nop(), read, fixedStat), &reads
}

func generate(t *testing.T, tt *tables, name string, qc table.QueryContext) []map[string]string {
	t.Helper()
	rows, err := tt.generator(specFor(t, name))(t.Context(), qc)
	require.NoError(t, err)
	return rows
}

func eq(col string, vals ...string) table.QueryContext {
	var cs []table.Constraint
	for _, v := range vals {
		cs = append(cs, table.Constraint{Operator: table.OperatorEquals, Expression: v})
	}
	return table.QueryContext{Constraints: map[string]table.ConstraintList{col: {Constraints: cs}}}
}

func TestTables(t *testing.T) {
	tt, reads := newTestTables(fixtureHive())

	files := generate(t, tt, "amcache_application_files", table.QueryContext{})
	require.Len(t, files, 2)
	notepad := files[0]
	assert.Equal(t, `c:\windows\system32\notepad.exe`, notepad["path"])
	assert.Equal(t, notepadSha1, notepad["sha1"])
	assert.Equal(t, programID, notepad["program_id"])
	assert.Equal(t, "201216", notepad["size"])
	assert.Equal(t, "1", notepad["os_component"])
	assert.Equal(t, "1680689472", notepad["link_time"])
	assert.Equal(t, "1700000000", notepad["last_write_time"])
	// Every column is present, even when the hive has no value for it.
	assert.Len(t, notepad, len(specFor(t, "amcache_application_files").cols))
	assert.Empty(t, notepad["long_path_hash"])

	stub := files[1]
	assert.Empty(t, stub["path"])
	assert.Equal(t, hashOnly, stub["sha1"])
	assert.Empty(t, stub["link_time"])
	assert.Empty(t, stub["last_write_time"])

	apps := generate(t, tt, "amcache_applications", table.QueryContext{})
	require.Len(t, apps, 1)
	assert.Equal(t, programID, apps[0]["program_id"])
	assert.Equal(t, "1704164645", apps[0]["install_time"])

	drivers := generate(t, tt, "amcache_driver_binaries", table.QueryContext{})
	require.Len(t, drivers, 1)
	assert.Equal(t, `c:\windows\system32\drivers\evil.sys`, drivers[0]["path"])
	assert.Equal(t, driverSha1, drivers[0]["sha1"])
	assert.Equal(t, "0", drivers[0]["signed"])
	assert.Equal(t, "1", drivers[0]["kernel_mode"])
	assert.Equal(t, "4026531840", drivers[0]["link_time"])
	assert.Equal(t, "2147483648", drivers[0]["driver_type"])

	pnp := generate(t, tt, "amcache_device_pnp", table.QueryContext{})
	require.Len(t, pnp, 1)
	assert.Equal(t, `USBSTOR\DISK&VEN_ACME\0001`, pnp[0]["device_instance_id"])
	assert.Equal(t, "{0a1b2c3d-4e5f-6071-8293-a4b5c6d7e8f9}", pnp[0]["container_id"])
	assert.Equal(t, "1730592000", pnp[0]["first_install_time"])
	assert.Equal(t, "a,b,c", pnp[0]["lower_filters"])

	containers := generate(t, tt, "amcache_device_containers", table.QueryContext{})
	require.Len(t, containers, 1)
	assert.Equal(t, pnp[0]["container_id"], containers[0]["container_id"])
	assert.Equal(t, "0", containers[0]["connected"])

	// Keys absent from the hive serve zero rows.
	assert.Empty(t, generate(t, tt, "amcache_driver_packages", table.QueryContext{}))
	assert.Empty(t, generate(t, tt, "amcache_application_shortcuts", table.QueryContext{}))

	assert.EqualValues(t, 1, reads.Load(), "all tables share one parse")
}

func TestPushdown(t *testing.T) {
	tt, _ := newTestTables(fixtureHive())
	const files = "amcache_application_files"

	rows := generate(t, tt, files, eq("sha1", hashOnly))
	require.Len(t, rows, 1)
	assert.Equal(t, hashOnly, rows[0]["sha1"])

	// The index is case-insensitive; SQLite applies the exact comparison afterwards.
	require.Len(t, generate(t, tt, files, eq("sha1", "F1D2D2F924E990A2FD0B5BB7C5CFD5E1D8E4A3B8")), 1)

	// Several equalities on one column are a union (IN list).
	require.Len(t, generate(t, tt, files, eq("sha1", hashOnly, notepadSha1, notepadSha1)), 2)

	assert.Empty(t, generate(t, tt, files, eq("sha1", driverSha1)))

	// Equalities across columns are an intersection.
	qc := eq("sha1", hashOnly, notepadSha1)
	qc.Constraints["path"] = table.ConstraintList{Constraints: []table.Constraint{
		{Operator: table.OperatorEquals, Expression: `c:\windows\system32\notepad.exe`},
	}}
	rows = generate(t, tt, files, qc)
	require.Len(t, rows, 1)
	assert.Equal(t, notepadSha1, rows[0]["sha1"])

	// Constraints that can't use the index return everything for SQLite to filter.
	like := table.QueryContext{Constraints: map[string]table.ConstraintList{"path": {Constraints: []table.Constraint{
		{Operator: table.OperatorLike, Expression: `c:\windows\%`},
	}}}}
	assert.Len(t, generate(t, tt, files, like), 2)
	assert.Len(t, generate(t, tt, files, eq("publisher", "nobody")), 2)
	assert.Len(t, generate(t, tt, files, eq("path", "")), 2)
}

func TestHiveNotFound(t *testing.T) {
	var reads atomic.Int32
	tt := newTables(zerolog.Nop(), func(context.Context) (hiveResult, error) {
		reads.Add(1)
		return hiveResult{}, errHiveNotFound
	}, fixedStat)

	assert.Empty(t, generate(t, tt, "amcache_application_files", table.QueryContext{}))
	assert.Empty(t, generate(t, tt, "amcache_device_pnp", table.QueryContext{}))
	assert.EqualValues(t, 1, reads.Load())
}

func TestFailuresAreCached(t *testing.T) {
	var reads atomic.Int32
	tt := newTables(zerolog.Nop(), func(context.Context) (hiveResult, error) {
		reads.Add(1)
		return hiveResult{}, errors.New("access denied")
	}, fixedStat)
	gen := tt.generator(specFor(t, "amcache_application_files"))

	for range 3 {
		_, err := gen(t.Context(), table.QueryContext{})
		require.ErrorContains(t, err, "access denied")
	}
	assert.EqualValues(t, 1, reads.Load())
}

func TestCanceledLoadIsNotCached(t *testing.T) {
	hive := fixtureHive()
	var reads atomic.Int32
	ctx, cancel := context.WithCancel(t.Context())
	tt := newTables(zerolog.Nop(), func(rctx context.Context) (hiveResult, error) {
		if reads.Add(1) == 1 {
			// The caller goes away mid-load.
			cancel()
			return hiveResult{}, rctx.Err()
		}
		return hiveResult{data: hive}, nil
	}, fixedStat)
	gen := tt.generator(specFor(t, "amcache_application_files"))

	_, err := gen(ctx, table.QueryContext{})
	require.ErrorIs(t, err, context.Canceled)

	rows, err := gen(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	assert.Len(t, rows, 2)
	assert.EqualValues(t, 2, reads.Load())
}

func TestHiveChangeReloads(t *testing.T) {
	var size atomic.Int64
	size.Store(1)
	var reads atomic.Int32
	hive := fixtureHive()
	tt := newTables(zerolog.Nop(), func(context.Context) (hiveResult, error) {
		reads.Add(1)
		return hiveResult{data: hive}, nil
	}, func() (time.Time, int64, error) { return time.Unix(1, 0), size.Load(), nil })

	generate(t, tt, "amcache_application_files", table.QueryContext{})
	generate(t, tt, "amcache_application_files", table.QueryContext{})
	assert.EqualValues(t, 1, reads.Load())

	size.Store(2)
	generate(t, tt, "amcache_application_files", table.QueryContext{})
	assert.EqualValues(t, 2, reads.Load())
}

func TestLegacyHive(t *testing.T) {
	tt, _ := newTestTables(buildHive(tkey{name: "File"}, tkey{name: "Programs"}))
	assert.Empty(t, generate(t, tt, "amcache_application_files", table.QueryContext{}))
}

func TestNotAmcache(t *testing.T) {
	tt, _ := newTestTables(buildHive(tkey{name: "Something"}))
	_, err := tt.generator(specFor(t, "amcache_application_files"))(t.Context(), table.QueryContext{})
	require.ErrorIs(t, err, errNotAmcache)
}

// A subkey index that references itself would make regparser recurse until the stack
// overflows, which can't be recovered. Only the affected key should be lost.
func TestSubkeyIndexCycle(t *testing.T) {
	w := newHiveWriter()
	ri := uint32(len(w.bins)) // #nosec G115
	p := make([]byte, 8)
	copy(p, "ri")
	binary.LittleEndian.PutUint16(p[2:], 1)
	binary.LittleEndian.PutUint32(p[4:], ri)
	require.Equal(t, ri, w.alloc(p))

	bad := w.key(tkey{name: keyApplicationFile}, 0x20, ri)
	good := w.key(tkey{name: keyApplication, subkeys: []tkey{{name: programID}}}, 0x20, 0)
	root := w.key(tkey{name: "Root"}, 0x20, w.lh(bad, good))
	base := w.key(tkey{name: "{base}"}, 0x2c, w.lh(root))

	tt, _ := newTestTables(w.hive(base, 1, 1))
	assert.Empty(t, generate(t, tt, "amcache_application_files", table.QueryContext{}))
	assert.Len(t, generate(t, tt, "amcache_applications", table.QueryContext{}), 1)
}

func TestReplayLogs(t *testing.T) {
	appHive := func(name string) []byte {
		return buildHive(tkey{name: keyApplication, subkeys: []tkey{{name: programID, values: []tval{szv("Name", name)}}}})
	}
	oldHive, newHive := appHive("old!"), appHive("new!")
	require.Len(t, newHive, len(oldHive))
	binSize := len(newHive) - 0x1000

	// The primary file is dirty: Sequence1 was bumped but the flush never completed.
	dirty := append([]byte(nil), oldHive...)
	binary.LittleEndian.PutUint32(dirty[4:], 2)
	isDirty, err := hiveDirty(dirty)
	require.NoError(t, err)
	require.True(t, isDirty)

	buildLog := func(entrySeq uint32) []byte {
		log := make([]byte, 0x200)
		copy(log, "regf")
		binary.LittleEndian.PutUint32(log[4:], 1)  // Sequence1
		binary.LittleEndian.PutUint32(log[28:], 6) // Type: new-format log
		entrySize := (40 + 8 + binSize + 511) &^ 511
		e := make([]byte, entrySize)
		copy(e, "HvLE")
		binary.LittleEndian.PutUint32(e[4:], uint32(entrySize)) // #nosec G115
		binary.LittleEndian.PutUint32(e[12:], entrySeq)
		binary.LittleEndian.PutUint32(e[16:], uint32(binSize)) // #nosec G115
		binary.LittleEndian.PutUint32(e[20:], 1)               // dirty page count
		binary.LittleEndian.PutUint32(e[44:], uint32(binSize)) // #nosec G115 -- page ref: offset 0, size binSize
		copy(e[48:], newHive[0x1000:])
		return append(log, e...)
	}

	out, err := replayLogs(dirty, buildLog(1))
	require.NoError(t, err)
	isDirty, err = hiveDirty(out)
	require.NoError(t, err)
	assert.False(t, isDirty)

	tt, _ := newTestTables(out)
	rows := generate(t, tt, "amcache_applications", table.QueryContext{})
	require.Len(t, rows, 1)
	assert.Equal(t, "new!", rows[0]["name"])

	// An entry out of sequence isn't applied, so the hive stays dirty.
	out, err = replayLogs(dirty, buildLog(5))
	require.NoError(t, err)
	isDirty, err = hiveDirty(out)
	require.NoError(t, err)
	assert.True(t, isDirty)
	assert.Equal(t, dirty, out)

	_, err = replayLogs(dirty[:100])
	require.Error(t, err)
}

func TestPanicIsRecoveredAndCached(t *testing.T) {
	var reads atomic.Int32
	tt := newTables(zerolog.Nop(), func(context.Context) (hiveResult, error) {
		reads.Add(1)
		panic("boom")
	}, fixedStat)
	gen := tt.generator(specFor(t, "amcache_application_files"))
	for range 2 {
		_, err := gen(t.Context(), table.QueryContext{})
		require.ErrorContains(t, err, "panicked")
	}
	assert.EqualValues(t, 1, reads.Load())
}

// A load that exceeds the deadline says something about this host, so it is cached rather
// than retried (and a raw volume read attempted) on every query.
func TestSlowLoadIsCached(t *testing.T) {
	var reads atomic.Int32
	tt := newTables(zerolog.Nop(), func(ctx context.Context) (hiveResult, error) {
		reads.Add(1)
		<-ctx.Done()
		return hiveResult{}, ctx.Err()
	}, fixedStat)
	tt.timeout = 10 * time.Millisecond
	gen := tt.generator(specFor(t, "amcache_application_files"))

	_, err := gen(t.Context(), table.QueryContext{})
	require.ErrorIs(t, err, errLoadTooSlow)
	_, err = gen(t.Context(), table.QueryContext{})
	require.ErrorIs(t, err, errLoadTooSlow)
	assert.EqualValues(t, 1, reads.Load())
}

func TestMalformedIntegerValuesAreDropped(t *testing.T) {
	hive := buildHive(tkey{name: keyApplicationFile, subkeys: []tkey{{name: "a", values: []tval{
		szv("FileId", "0000"+notepadSha1),
		// Declared as a DWORD but holding 8 bytes, which would make regparser index past the
		// bytes it read.
		{name: "Size", typ: regDWORD, data: make([]byte, 8)},
		// A QWORD stored inline only has 4 bytes.
		{name: "Usn", typ: regQWORD, data: make([]byte, 4)},
	}}}})
	tt, _ := newTestTables(hive)
	rows := generate(t, tt, "amcache_application_files", table.QueryContext{})
	require.Len(t, rows, 1)
	assert.Equal(t, notepadSha1, rows[0]["sha1"])
	assert.Empty(t, rows[0]["size"])
	assert.Empty(t, rows[0]["usn"])
}

func TestTablePlugins(t *testing.T) {
	plugins := TablePlugins(zerolog.Nop())
	require.Len(t, plugins, len(specs))
	names := make(map[string]struct{})
	for _, p := range plugins {
		names[p.Name()] = struct{}{}
		assert.Len(t, p.Routes(), len(specFor(t, p.Name()).cols))
	}
	assert.Len(t, names, len(specs))
}

func TestMultiSz(t *testing.T) {
	hive := buildHive(tkey{name: keyDevicePnp, subkeys: []tkey{{name: "x", values: []tval{
		{name: "LowerFilters", typ: 7, data: append(append(utf16z("a"), utf16z("b")...), 0, 0)},
	}}}})
	tt, _ := newTestTables(hive)
	rows := generate(t, tt, "amcache_device_pnp", table.QueryContext{})
	require.Len(t, rows, 1)
	assert.Equal(t, "a,b", rows[0]["lower_filters"])
}

// Cells are addressed by offset, so a crafted hive can make many records share one large value.
// Duplicated record cells are skipped, and distinct records that share data hit the decoded
// bytes budget instead of growing orbit's memory.
func TestSharedCellAmplification(t *testing.T) {
	w := newHiveWriter()
	data := utf16z(strings.Repeat("a", 400<<10))
	vk := make([]byte, 20+len("LowerCaseLongPath"))
	copy(vk, "vk")
	binary.LittleEndian.PutUint16(vk[2:], uint16(len("LowerCaseLongPath")))
	binary.LittleEndian.PutUint32(vk[4:], uint32(len(data))) // #nosec G115
	binary.LittleEndian.PutUint32(vk[8:], w.alloc(data))
	binary.LittleEndian.PutUint32(vk[12:], regSZ)
	binary.LittleEndian.PutUint16(vk[16:], 1)
	copy(vk[20:], "LowerCaseLongPath")
	valList := w.alloc(binary.LittleEndian.AppendUint32(nil, w.alloc(vk)))

	record := func(name string) uint32 {
		p := make([]byte, 76+len(name))
		copy(p, "nk")
		binary.LittleEndian.PutUint16(p[2:], 0x20)
		binary.LittleEndian.PutUint32(p[28:], 0xFFFFFFFF)
		binary.LittleEndian.PutUint32(p[36:], 1)
		binary.LittleEndian.PutUint32(p[40:], valList)
		binary.LittleEndian.PutUint16(p[72:], uint16(len(name))) // #nosec G115
		copy(p[76:], name)
		return w.alloc(p)
	}
	hiveWith := func(files uint32) []byte {
		root := w.key(tkey{name: "Root"}, 0x20, w.lh(files))
		return w.hive(w.key(tkey{name: "{base}"}, 0x2c, w.lh(root)), 1, 1)
	}

	one := record("one")
	dup := make([]uint32, 1000)
	for i := range dup {
		dup[i] = one
	}
	tt, _ := newTestTables(hiveWith(w.key(tkey{name: keyApplicationFile}, 0x20, w.lh(dup...))))
	rows := generate(t, tt, "amcache_application_files", table.QueryContext{})
	require.Len(t, rows, 1)
	assert.Len(t, rows[0]["path"], 400<<10)

	distinct := make([]uint32, 100)
	for i := range distinct {
		distinct[i] = record(fmt.Sprintf("r%03d", i))
	}
	tt, _ = newTestTables(hiveWith(w.key(tkey{name: keyApplicationFile}, 0x20, w.lh(distinct...))))
	_, err := tt.generator(specFor(t, "amcache_application_files"))(t.Context(), table.QueryContext{})
	require.ErrorIs(t, err, errDecodedTooLarge)
}
