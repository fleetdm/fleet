package msi

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundtrip_test.go contains minimal CFB and CAB readers used to validate
// the writers' output structurally, guarding the binary-format code against
// regressions without external tooling.

// --- minimal CFB reader ---

type cfbFile struct {
	streams map[string][]byte
}

func readCFB(t *testing.T, raw []byte) *cfbFile {
	t.Helper()
	le := binary.LittleEndian
	require.GreaterOrEqual(t, len(raw), 512)
	require.Equal(t, []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}, raw[:8])
	sectorSize := 1 << le.Uint16(raw[30:32])
	miniSize := 1 << le.Uint16(raw[32:34])
	require.Equal(t, 512, sectorSize)
	require.Equal(t, 64, miniSize)
	numFAT := int(le.Uint32(raw[44:48]))
	firstDir := le.Uint32(raw[48:52])
	miniCutoff := int(le.Uint32(raw[56:60]))
	firstMiniFAT := le.Uint32(raw[60:64])

	sector := func(n uint32) []byte {
		start := 512 + int(n)*sectorSize
		require.LessOrEqual(t, start+sectorSize, len(raw), "sector out of range")
		return raw[start : start+sectorSize]
	}

	var difat []uint32
	for i := 0; i < 109 && i < numFAT; i++ {
		difat = append(difat, le.Uint32(raw[76+4*i:80+4*i]))
	}
	var fat []uint32
	for _, s := range difat {
		sec := sector(s)
		for i := 0; i < sectorSize/4; i++ {
			fat = append(fat, le.Uint32(sec[4*i:4*i+4]))
		}
	}
	chain := func(start uint32, size int) []byte {
		var buf bytes.Buffer
		visited := 0
		for s := start; s != 0xFFFFFFFE; s = fat[s] {
			buf.Write(sector(s))
			visited++
			require.Less(t, visited, len(fat)+1, "FAT chain loop")
		}
		if size >= 0 {
			require.GreaterOrEqual(t, buf.Len(), size)
			return buf.Bytes()[:size]
		}
		return buf.Bytes()
	}

	dir := chain(firstDir, -1)
	type entry struct {
		name  string
		typ   byte
		start uint32
		size  int
	}
	var entries []entry
	for i := 0; i+128 <= len(dir); i += 128 {
		e := dir[i : i+128]
		nameLen := int(le.Uint16(e[64:66]))
		if nameLen == 0 {
			continue
		}
		var name []rune
		for j := 0; j+2 <= nameLen-2; j += 2 {
			name = append(name, rune(le.Uint16(e[j:j+2])))
		}
		entries = append(entries, entry{
			name:  string(name),
			typ:   e[66],
			start: le.Uint32(e[116:120]),
			size:  int(le.Uint32(e[120:124])),
		})
	}
	require.NotEmpty(t, entries)
	require.Equal(t, byte(5), entries[0].typ, "first entry must be the root storage")

	var miniStream []byte
	if entries[0].size > 0 {
		miniStream = chain(entries[0].start, entries[0].size)
	}
	var miniFAT []uint32
	if firstMiniFAT != 0xFFFFFFFE {
		mf := chain(firstMiniFAT, -1)
		for i := 0; i+4 <= len(mf); i += 4 {
			miniFAT = append(miniFAT, le.Uint32(mf[i:i+4]))
		}
	}
	miniChain := func(start uint32, size int) []byte {
		var buf bytes.Buffer
		for s := start; s != 0xFFFFFFFE; s = miniFAT[s] {
			buf.Write(miniStream[int(s)*miniSize : int(s+1)*miniSize])
		}
		return buf.Bytes()[:size]
	}

	out := &cfbFile{streams: map[string][]byte{}}
	for _, e := range entries[1:] {
		if e.typ != 2 {
			continue
		}
		switch {
		case e.size == 0:
			out.streams[e.name] = nil
		case e.size < miniCutoff:
			out.streams[e.name] = miniChain(e.start, e.size)
		default:
			out.streams[e.name] = chain(e.start, e.size)
		}
	}
	return out
}

func TestWriteCFBRoundTrip(t *testing.T) {
	big := bytes.Repeat([]byte{0xAB, 0xCD, 0xEF}, 30000) // 90KB, FAT sectors
	small := []byte("hello mini stream")                 // mini stream
	boundary := bytes.Repeat([]byte{0x42}, cfbMiniCutoff) // exactly at cutoff: FAT
	streams := []cfbStream{
		{Name: "empty", Data: nil},
		{Name: "small", Data: small},
		{Name: "boundary", Data: boundary},
		{Name: "big", Data: big},
		{Name: encodeStreamName("!_Tables"), Data: []byte{1, 0}},
	}
	var buf bytes.Buffer
	require.NoError(t, writeCFB(&buf, streams))
	require.Equal(t, 0, buf.Len()%512, "CFB must be sector aligned")

	got := readCFB(t, buf.Bytes())
	require.Len(t, got.streams, len(streams))
	assert.Empty(t, got.streams["empty"])
	assert.Equal(t, small, got.streams["small"])
	assert.Equal(t, boundary, got.streams["boundary"])
	assert.Equal(t, big, got.streams["big"])
	assert.Equal(t, []byte{1, 0}, got.streams[encodeStreamName("!_Tables")])
}

// --- minimal CAB reader ---

type cabEntry struct {
	name      string
	size      uint32
	folder    int
	folderOff uint32
}

// readCab parses a cabinet, verifies every CFDATA checksum and MSZIP frame,
// and returns the entries plus each folder's decompressed data.
func readCab(t *testing.T, raw []byte) ([]cabEntry, [][]byte) {
	t.Helper()
	require.NotNil(t, raw)
	le := binary.LittleEndian
	require.Equal(t, "MSCF", string(raw[:4]))
	require.Equal(t, uint32(len(raw)), le.Uint32(raw[8:12]), "cbCabinet") //nolint:gosec // test cab well under 4GB
	coffFiles := le.Uint32(raw[16:20])
	nFolders := int(le.Uint16(raw[26:28]))
	nFiles := int(le.Uint16(raw[28:30]))

	type folderHdr struct {
		off    uint32
		blocks int
	}
	var folders []folderHdr
	for i := range nFolders {
		off := 36 + 8*i
		require.Equal(t, uint16(1), le.Uint16(raw[off+6:off+8]), "compression must be MSZIP")
		folders = append(folders, folderHdr{le.Uint32(raw[off : off+4]), int(le.Uint16(raw[off+4 : off+6]))})
	}

	var entries []cabEntry
	pos := int(coffFiles)
	for range nFiles {
		size := le.Uint32(raw[pos : pos+4])
		folderOff := le.Uint32(raw[pos+4 : pos+8])
		folder := int(le.Uint16(raw[pos+8 : pos+10]))
		pos += 16
		end := bytes.IndexByte(raw[pos:], 0)
		require.GreaterOrEqual(t, end, 0)
		entries = append(entries, cabEntry{string(raw[pos : pos+end]), size, folder, folderOff})
		pos += end + 1
	}

	folderData := make([][]byte, 0, len(folders))
	for _, f := range folders {
		var out bytes.Buffer
		pos := int(f.off)
		var window []byte
		for b := 0; b < f.blocks; b++ {
			csum := le.Uint32(raw[pos : pos+4])
			cbData := int(le.Uint16(raw[pos+4 : pos+6]))
			cbUncomp := int(le.Uint16(raw[pos+6 : pos+8]))
			require.LessOrEqual(t, cbData, cabBlockSize+12, "MSZIP block size limit")
			payload := raw[pos+8 : pos+8+cbData]
			require.Equal(t, csum, cabChecksum(payload, cabChecksum(raw[pos+4:pos+8], 0)), "CFDATA checksum")
			require.Equal(t, []byte("CK"), payload[:2])
			fr := flate.NewReaderDict(bytes.NewReader(payload[2:]), window)
			frame, err := io.ReadAll(fr)
			require.NoError(t, err)
			require.Len(t, frame, cbUncomp)
			out.Write(frame)
			if len(frame) >= cabBlockSize {
				window = frame[len(frame)-cabBlockSize:]
			} else {
				window = append(window, frame...)
				if len(window) > cabBlockSize {
					window = window[len(window)-cabBlockSize:]
				}
			}
			pos += 8 + cbData
		}
		folderData = append(folderData, out.Bytes())
	}
	return entries, folderData
}

func TestWriteCabRoundTrip(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) cabFile {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, data, 0o600))
		h, err := msiFileHash(p, int64(len(data)))
		require.NoError(t, err)
		return cabFile{Key: name, Path: p, Size: int64(len(data)), Hash: h}
	}

	// content large enough to span multiple 32KB MSZIP frames, plus
	// incompressible data to exercise the stored-block fallback
	big := bytes.Repeat([]byte("fleet osquery installer payload "), 3000) // ~96KB
	incompressible := make([]byte, 70000)
	for i := range incompressible {
		incompressible[i] = byte(i*7919 + i>>3) //nolint:gosec // intentional wrap for pseudo-random bytes
	}

	files := []cabFile{
		write("aaa.bin", big),
		write("bbb.txt", []byte("small file")),
		write("ccc.dup", big), // duplicate of aaa.bin: same size and hash
		write("ddd.rand", incompressible),
		write("eee.empty", nil),
	}
	// aaa.bin has a duplicate later, so it must start its own folder; the
	// non-adjacent duplicate ccc.dup must close its folder after being
	// recorded.
	raw, err := writeCab(files)
	require.NoError(t, err)

	entries, folderData := readCab(t, raw)
	require.Len(t, entries, 5)

	byName := map[string]cabEntry{}
	for _, e := range entries {
		byName[e.name] = e
	}
	// entry order must match input (sequence) order
	for i, want := range []string{"aaa.bin", "bbb.txt", "ccc.dup", "ddd.rand", "eee.empty"} {
		assert.Equal(t, want, entries[i].name)
	}
	// the duplicate points at the original's data
	assert.Equal(t, byName["aaa.bin"].folder, byName["ccc.dup"].folder)
	assert.Equal(t, byName["aaa.bin"].folderOff, byName["ccc.dup"].folderOff)

	// every entry's bytes must round-trip
	for _, e := range entries {
		data := folderData[e.folder][e.folderOff : e.folderOff+e.size]
		switch e.name {
		case "aaa.bin", "ccc.dup":
			assert.Equal(t, big, data, e.name)
		case "bbb.txt":
			assert.Equal(t, []byte("small file"), data)
		case "ddd.rand":
			assert.Equal(t, incompressible, data)
		case "eee.empty":
			assert.Empty(t, data)
		}
	}
}

func TestHarvestRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "bin", "orbit", "windows-arm64", "stable"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "staging"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "certs.pem"), []byte("certs"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte("s3cret"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "bin", "orbit", "windows-arm64", "stable", "orbit.exe"), []byte("not a real PE"), 0o600))

	h, err := harvestRoot(root)
	require.NoError(t, err)

	// components come in heat's order: root files alphabetically, then
	// subdirectory trees, then empty-directory components
	require.Len(t, h.Components, 4)
	require.NotNil(t, h.Components[0].File)
	assert.Equal(t, "filBD4431EBD09EAC47887B6474705D94B7", h.Components[0].File.FileID, "certs.pem id must match WiX/heat")
	require.NotNil(t, h.Components[1].File)
	assert.Equal(t, "secret.txt", filepath.Base(h.Components[1].File.Path))
	require.NotNil(t, h.Components[2].File)
	assert.Equal(t, "orbit.exe", filepath.Base(h.Components[2].File.Path))
	require.NotNil(t, h.Components[3].EmptyDir)
	assert.Equal(t, "cmp3721680D11CEA9BEE8173E0F0A00BD49", h.Components[3].EmptyDir.ComponentID, "staging component id must match WiX/heat")

	// windows-arm64 needs a generated 8.3 short name; its parent chain walks
	// back up to ORBITROOT
	orbitExe := h.Components[2].File
	chain := h.parentChain(orbitExe.DirID)
	require.Len(t, chain, 4) // stable, windows-arm64, orbit, bin
	assert.Equal(t, "2-lee-0b|windows-arm64", chain[1].DefaultDir)
	assert.Equal(t, "ORBITROOT", chain[3].ParentID)
}

// TestBuildSmoke builds a full MSI from a synthetic root and validates the
// container: all expected streams present, the cabinet decodes with valid
// checksums, and the payload round-trips.
func TestBuildSmoke(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin", "orbit", "windows-arm64", "stable")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "orbit.exe"), bytes.Repeat([]byte("orbit!"), 10000), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte("s3cret"), 0o600))

	out := filepath.Join(t.TempDir(), "out.msi")
	opt := Options{
		Architecture:        "arm64",
		Version:             "1.0.0",
		FleetURL:            "https://fleet.example.com",
		EnrollSecret:        true,
		DesktopChannel:      "stable",
		OrbitChannel:        "stable",
		OsquerydChannel:     "stable",
		OrbitUpdateInterval: "15m0s",
		NativePlatform:      "windows-arm64",
	}
	require.NoError(t, Build(opt, root, out))

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	cfb := readCFB(t, raw)

	for _, name := range []string{
		"!_Tables", "!_StringPool", "!_StringData", "!_Columns", "!_Validation",
		"!Property", "!File", "!Component", "!Directory", "!ServiceInstall",
		"cab1.cab", "Binary.WixCA", "Binary.WixCA_A64",
	} {
		assert.Contains(t, cfb.streams, encodeStreamName(name), name)
	}
	assert.Contains(t, cfb.streams, "\x05SummaryInformation")

	// the cabinet must decode cleanly with valid checksums
	entries, folderData := readCab(t, cfb.streams[encodeStreamName("cab1.cab")])
	require.Len(t, entries, 3) // orbit.exe (authored), harvested orbit.exe dup, secret.txt
	var total int
	for _, e := range entries {
		got := folderData[e.folder][e.folderOff : e.folderOff+e.size]
		require.Len(t, got, int(e.size))
		total += int(e.size)
	}
	require.NotZero(t, total)

	// the embedded custom-action DLLs must be intact
	assert.Equal(t, wixCADLL, cfb.streams[encodeStreamName("Binary.WixCA")])
	assert.Equal(t, wixCAA64DLL, cfb.streams[encodeStreamName("Binary.WixCA_A64")])
}
