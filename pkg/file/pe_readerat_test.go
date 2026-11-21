package file

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/fleetdm/fleet/v4/server/fleet"
	saferwall "github.com/saferwall/pe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saferwallVersionResources runs the saferwall-based parsing used by
// ExtractPEMetadata, as the reference implementation.
func saferwallVersionResources(t *testing.T, path string) ([]map[string]string, error) {
	t.Helper()
	pep, err := saferwall.New(path, &saferwall.Options{
		OmitExportDirectory:       true,
		OmitImportDirectory:       true,
		OmitExceptionDirectory:    true,
		OmitSecurityDirectory:     true,
		OmitRelocDirectory:        true,
		OmitDebugDirectory:        true,
		OmitArchitectureDirectory: true,
		OmitGlobalPtrDirectory:    true,
		OmitTLSDirectory:          true,
		OmitLoadConfigDirectory:   true,
		OmitBoundImportDirectory:  true,
		OmitIATDirectory:          true,
		OmitDelayImportDirectory:  true,
		OmitCLRHeaderDirectory:    true,
	})
	if err != nil {
		return nil, err
	}
	defer pep.Close()
	if err := pep.Parse(); err != nil {
		return nil, err
	}
	return pep.ParseVersionResourcesForEntries()
}

func parsePEBytes(b []byte) ([]map[string]string, error) {
	return parsePEVersionResources(bytes.NewReader(b), int64(len(b)))
}

func assertPEParity(t *testing.T, path string) []map[string]string {
	t.Helper()
	want, wantErr := saferwallVersionResources(t, path)

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	st, err := f.Stat()
	require.NoError(t, err)

	got, err := parsePEVersionResources(f, st.Size())
	if wantErr != nil {
		require.ErrorIs(t, err, ErrInvalidInstaller, "reference parser failed with: %v", wantErr)
		return nil
	}
	require.NoError(t, err)
	assert.Equal(t, want, got)
	return got
}

func TestParsePEVersionResourcesFixtures(t *testing.T) {
	t.Parallel()

	fixtures := []string{
		"testdata/software-installers/hello-world-installer.exe",
		"../../server/service/testdata/software-installers/hello-world-installer.exe",
	}
	// Executables shipped with the Go toolchain (no version resources).
	for _, name := range []string{"gcc-386-mingw-exec", "gcc-386-mingw-no-symbols-exec", "gcc-amd64-mingw-exec"} {
		fixtures = append(fixtures, filepath.Join(runtime.GOROOT(), "src", "debug", "pe", "testdata", name))
	}
	// Real-world installers added locally (not in git, see TestExtractInstallerMetadata).
	local, err := filepath.Glob(filepath.Join("testdata", "installers", "*.exe"))
	require.NoError(t, err)
	fixtures = append(fixtures, local...)

	for _, path := range fixtures {
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			if _, err := os.Stat(path); err != nil {
				t.Skipf("fixture not available: %v", err)
			}
			assertPEParity(t, path)
		})
	}

	got := assertPEParity(t, fixtures[0])
	require.NotEmpty(t, got)
	// Values are space-padded in this fixture; ExtractPEMetadata trims them.
	assert.Equal(t, "Hello world", strings.TrimSpace(got[0]["ProductName"]))
	assert.Equal(t, "1.0.0", strings.TrimSpace(got[0]["ProductVersion"]))
}

func TestParsePEVersionResourcesSynthetic(t *testing.T) {
	t.Parallel()

	english := versionInfo(
		stringFileInfo(stringTable("040904b0",
			"CompanyName", "Fleet",
			"FileVersion", "1.2.3.4",
			"ProductName", "Fleet Test ",
			"ProductVersion", "1.2.3",
			"Comments", "",
			"OriginalFilename", "héllo-ü.exe",
		)),
		varFileInfo(),
	)
	// VarFileInfo first and lower-case keys, as in Opera's SFX installer.
	sfx := versionInfo(
		varFileInfo(),
		stringFileInfo(stringTable("000004b0",
			"productname", "Opera installer",
			"ProductVersion", "110.0",
		)),
	)
	// A second StringTable is ignored by saferwall.
	multiTable := versionInfo(stringFileInfo(
		stringTable("040904b0", "ProductName", "First", "FileVersion", "1.0"),
		stringTable("040704b0", "ProductName", "Zweite", "FileVersion", "2.0"),
	))
	sevenZip := versionInfo(stringFileInfo(stringTable("040904b0",
		"ProductName", "7-Zip",
		"ProductVersion", "24.08",
	)))

	cases := []struct {
		name  string
		opts  peBuildOpts
		blobs [][]byte
	}{
		{name: "single language", blobs: [][]byte{english}},
		{name: "multiple languages", blobs: [][]byte{english, sfx, sevenZip}},
		{name: "multiple string tables", blobs: [][]byte{multiTable}},
		{name: "unaligned blob is rejected by both", opts: peBuildOpts{misalign: 2}, blobs: [][]byte{english, sfx}},
		{name: "other resource types", opts: peBuildOpts{extraRootEntries: true}, blobs: [][]byte{english}},
		{name: "PE32", opts: peBuildOpts{pe32: true}, blobs: [][]byte{english}},
		{name: "no version resource", opts: peBuildOpts{noVersion: true}},
		{name: "no resource directory", opts: peBuildOpts{noResources: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			b := buildPE(t, c.opts, c.blobs...)
			path := filepath.Join(t.TempDir(), "test.exe")
			require.NoError(t, os.WriteFile(path, b, 0o600))
			got := assertPEParity(t, path)
			if len(c.blobs) > 0 && c.opts.misalign == 0 {
				require.Len(t, got, len(c.blobs))
			}
		})
	}

	got, err := parsePEBytes(buildPE(t, peBuildOpts{}, english, sfx, multiTable))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, map[string]string{
		"CompanyName":      "Fleet",
		"FileVersion":      "1.2.3.4",
		"ProductName":      "Fleet Test ",
		"ProductVersion":   "1.2.3",
		"Comments":         "",
		"OriginalFilename": "héllo-ü.exe",
	}, got[0])
	assert.Equal(t, map[string]string{"productname": "Opera installer", "ProductVersion": "110.0"}, got[1])
	assert.Equal(t, map[string]string{"ProductName": "First", "FileVersion": "1.0"}, got[2])
}

type countingReaderAt struct {
	r     io.ReaderAt
	reads int
	bytes int64
}

func (c *countingReaderAt) ReadAt(p []byte, off int64) (int, error) {
	c.reads++
	c.bytes += int64(len(p))
	return c.r.ReadAt(p, off)
}

func TestParsePEVersionResourcesMinimalReads(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile("testdata/software-installers/hello-world-installer.exe")
	require.NoError(t, err)
	cr := &countingReaderAt{r: bytes.NewReader(b)}
	got, err := parsePEVersionResources(cr, int64(len(b)))
	require.NoError(t, err)
	require.NotEmpty(t, got)
	// headers, 3 directories (header + entries each), 1 data entry, 1 blob
	assert.LessOrEqual(t, cr.reads, 10)
	assert.Less(t, cr.bytes, int64(16<<10))
}

type errReaderAt struct{ err error }

func (e errReaderAt) ReadAt([]byte, int64) (int, error) { return 0, e.err }

func TestParsePEVersionResourcesMalformed(t *testing.T) {
	t.Parallel()

	blob := versionInfo(stringFileInfo(stringTable("040904b0", "ProductName", "X", "ProductVersion", "1")))
	valid := buildPE(t, peBuildOpts{}, blob)
	layout := peTestLayout(1)

	mutate := func(f func(b []byte)) []byte {
		b := bytes.Clone(valid)
		f(b)
		return b
	}
	put32 := func(b []byte, off uint32, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }
	put16 := func(b []byte, off uint32, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }

	cases := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"not a PE", []byte("hello world, this is not a PE file at all")},
		{"truncated DOS header", valid[:32]},
		{"truncated NT headers", valid[:0x80]},
		{"truncated resources", valid[:peTestRsrcOff+layout.dataEntries+4]},
		{"truncated version blob", valid[:peTestRsrcOff+layout.blobs+20]},
		{"huge e_lfanew", mutate(func(b []byte) { put32(b, 0x3c, 0x7FFFFFF0) })},
		{"too many sections", mutate(func(b []byte) { put16(b, 0x46, 0xFFFF) })},
		{"resource RVA outside file", mutate(func(b []byte) { put32(b, peTestDataDirOff(false, 2), 0x7FFF0000) })},
		{"too many root entries", mutate(func(b []byte) { put16(b, peTestRsrcOff+14, 0xFFFF) })},
		{"version entry not a directory", mutate(func(b []byte) { put32(b, peTestRsrcOff+20, layout.nameDir) })},
		{"directory loop", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.nameDir+20, 0x80000000) })},
		{"language entry is a directory", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.langDir+20, 0x80000000|layout.dataEntries) })},
		{"data entry outside file", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.langDir+20, 0x7FFFFFF0) })},
		{"blob outside file", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.dataEntries, 0x7FFFFFF0) })},
		{"huge blob size", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.dataEntries+4, 0xFFFFFFF0) })},
		{"blob size past EOF", mutate(func(b []byte) { put32(b, peTestRsrcOff+layout.dataEntries+4, 0x00100000) })},
		{"bad VS_VERSION_INFO key", mutate(func(b []byte) { b[peTestRsrcOff+layout.blobs+6] = 'X' })},
		{"bad fixed file info signature", mutate(func(b []byte) { b[peTestRsrcOff+layout.blobs+40] ^= 0xFF })},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePEBytes(c.data)
			require.ErrorIs(t, err, ErrInvalidInstaller)
			assert.Nil(t, got)
		})
	}

	t.Run("reader error", func(t *testing.T) {
		t.Parallel()
		readErr := errors.New("network down")
		_, err := parsePEVersionResources(errReaderAt{err: readErr}, 1<<20)
		require.ErrorIs(t, err, readErr)
		require.NotErrorIs(t, err, ErrInvalidInstaller)
	})

	t.Run("zero-length string table does not hang", func(t *testing.T) {
		t.Parallel()
		// Length of the first StringTable: offset 92 (StringFileInfo) + 36.
		b := mutate(func(b []byte) { put16(b, peTestRsrcOff+layout.blobs+92+36, 0) })
		got, err := parsePEBytes(b)
		require.NoError(t, err)
		assert.Equal(t, []map[string]string{{}}, got)
	})

	t.Run("random mutations", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // deterministic test input
		for range 5000 {
			b := bytes.Clone(valid)
			for range 1 + rng.IntN(8) {
				i := rng.IntN(len(b))
				if rng.IntN(2) == 0 {
					b[i] = byte(rng.Uint32())
				} else {
					b[i] = []byte{0x00, 0xFF, 0x7F, 0x80}[rng.IntN(4)]
				}
			}
			if rng.IntN(4) == 0 {
				b = b[:rng.IntN(len(b))]
			}
			_, err := parsePEBytes(b)
			if err != nil {
				require.ErrorIs(t, err, ErrInvalidInstaller)
			}
		}
	})
}

// --- synthetic PE builder ---

func utf16z(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u)+2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[2*i:], c)
	}
	return b
}

type vsNode struct {
	key         string
	value       []byte
	valueLength uint16
	text        bool
	children    []vsNode
}

func pad4(b *bytes.Buffer) {
	for b.Len()%4 != 0 {
		b.WriteByte(0)
	}
}

func (n vsNode) encode() []byte {
	var b bytes.Buffer
	b.Write(make([]byte, 6))
	b.Write(utf16z(n.key))
	pad4(&b)
	b.Write(n.value)
	for _, c := range n.children {
		pad4(&b)
		b.Write(c.encode())
	}
	out := b.Bytes()
	binary.LittleEndian.PutUint16(out, uint16(len(out))) //nolint:gosec // test data is small
	binary.LittleEndian.PutUint16(out[2:], n.valueLength)
	if n.text {
		binary.LittleEndian.PutUint16(out[4:], 1)
	}
	return out
}

func versionInfo(children ...vsNode) []byte {
	ff := make([]byte, 52)
	binary.LittleEndian.PutUint32(ff, 0xFEEF04BD)
	binary.LittleEndian.PutUint32(ff[4:], 0x00010000)
	return vsNode{key: "VS_VERSION_INFO", value: ff, valueLength: 52, children: children}.encode()
}

func stringFileInfo(tables ...vsNode) vsNode {
	return vsNode{key: "StringFileInfo", text: true, children: tables}
}

func stringTable(lang string, kv ...string) vsNode {
	n := vsNode{key: lang, text: true}
	for i := 0; i+1 < len(kv); i += 2 {
		v := utf16z(kv[i+1])
		n.children = append(n.children, vsNode{key: kv[i], value: v, valueLength: uint16(len(v) / 2), text: true}) //nolint:gosec // test data is small
	}
	return n
}

func varFileInfo() vsNode {
	return vsNode{key: "VarFileInfo", text: true, children: []vsNode{
		{key: "Translation", value: []byte{0x09, 0x04, 0xb0, 0x04}, valueLength: 4},
	}}
}

type peBuildOpts struct {
	pe32             bool
	misalign         int // added to every blob's offset to break DWORD alignment
	extraRootEntries bool
	noVersion        bool
	noResources      bool
}

const (
	peTestRsrcOff = 0x200
	peTestRsrcRVA = 0x1000
)

// peTestDataDirOff returns the file offset of a data directory entry.
func peTestDataDirOff(pe32 bool, idx uint32) uint32 {
	optOff := uint32(0x40 + 4 + 20)
	if pe32 {
		return optOff + 96 + 8*idx
	}
	return optOff + 112 + 8*idx
}

type peLayout struct {
	nameDir, langDir, dataEntries, blobs uint32
}

// peTestLayout returns offsets (relative to the resource section) used by
// buildPE without extra root entries: root dir (1 entry), name dir (1 entry),
// language dir (n entries), data entries, blobs.
func peTestLayout(n int) peLayout {
	l := peLayout{nameDir: 16 + 8}
	l.langDir = l.nameDir + 16 + 8
	l.dataEntries = l.langDir + 16 + 8*uint32(n) //nolint:gosec // test data is small
	l.blobs = l.dataEntries + 16*uint32(n)       //nolint:gosec // test data is small
	return l
}

func buildPE(t *testing.T, opts peBuildOpts, blobs ...[]byte) []byte {
	t.Helper()
	le := binary.LittleEndian

	var rsrc bytes.Buffer
	dirHeader := func(b *bytes.Buffer, n int) {
		var h [16]byte
		le.PutUint16(h[14:], uint16(n)) //nolint:gosec // test data is small
		b.Write(h[:])
	}
	entry := func(b *bytes.Buffer, name, off uint32) {
		var e [8]byte
		le.PutUint32(e[:], name)
		le.PutUint32(e[4:], off)
		b.Write(e[:])
	}

	if opts.extraRootEntries {
		// root: RT_ICON (3) -> empty dir, RT_VERSION (16) -> name dir, RT_MANIFEST (24) -> empty dir
		emptyDir := uint32(16 + 3*8)
		nameDir := emptyDir + 16
		langDir := nameDir + 16 + 8
		dataEntries := langDir + 16 + 8*uint32(len(blobs)) //nolint:gosec // test data is small
		dirHeader(&rsrc, 3)
		entry(&rsrc, 3, 0x80000000|emptyDir)
		entry(&rsrc, 16, 0x80000000|nameDir)
		entry(&rsrc, 24, 0x80000000|emptyDir)
		dirHeader(&rsrc, 0)
		writeVersionDirs(&rsrc, nameDir, langDir, dataEntries, opts, blobs)
	} else {
		l := peTestLayout(len(blobs))
		if opts.noVersion {
			dirHeader(&rsrc, 1)
			entry(&rsrc, 3, 0x80000000|l.nameDir)
			dirHeader(&rsrc, 0)
		} else {
			dirHeader(&rsrc, 1)
			entry(&rsrc, 16, 0x80000000|l.nameDir)
			writeVersionDirs(&rsrc, l.nameDir, l.langDir, l.dataEntries, opts, blobs)
		}
	}

	rsrcLen := uint32(rsrc.Len()) //nolint:gosec // test data is small
	rawSize := (rsrcLen + 0x1FF) &^ 0x1FF

	var b bytes.Buffer
	dos := make([]byte, 0x40)
	dos[0], dos[1] = 'M', 'Z'
	le.PutUint32(dos[0x3c:], 0x40)
	b.Write(dos)
	b.WriteString("PE\x00\x00")

	fh := pe.FileHeader{Machine: pe.IMAGE_FILE_MACHINE_AMD64, NumberOfSections: 1, Characteristics: 0x22}
	var opt any
	var dataDirs *[16]pe.DataDirectory
	if opts.pe32 {
		fh.Machine = pe.IMAGE_FILE_MACHINE_I386
		oh := &pe.OptionalHeader32{Magic: 0x10b, ImageBase: 0x400000}
		oh.SectionAlignment, oh.FileAlignment, oh.SizeOfImage, oh.SizeOfHeaders, oh.NumberOfRvaAndSizes = 0x1000, 0x200, 0x2000+rawSize, 0x200, 16
		oh.MajorSubsystemVersion, oh.Subsystem = 6, 2
		opt, dataDirs = oh, &oh.DataDirectory
	} else {
		oh := &pe.OptionalHeader64{Magic: 0x20b, ImageBase: 0x140000000}
		oh.SectionAlignment, oh.FileAlignment, oh.SizeOfImage, oh.SizeOfHeaders, oh.NumberOfRvaAndSizes = 0x1000, 0x200, 0x2000+rawSize, 0x200, 16
		oh.MajorSubsystemVersion, oh.Subsystem = 6, 2
		opt, dataDirs = oh, &oh.DataDirectory
	}
	if !opts.noResources {
		dataDirs[pe.IMAGE_DIRECTORY_ENTRY_RESOURCE] = pe.DataDirectory{VirtualAddress: peTestRsrcRVA, Size: rsrcLen}
	}
	fh.SizeOfOptionalHeader = uint16(binary.Size(opt)) //nolint:gosec // fixed size
	require.NoError(t, binary.Write(&b, le, fh))
	require.NoError(t, binary.Write(&b, le, opt))

	sh := pe.SectionHeader32{
		VirtualSize:      rsrcLen,
		VirtualAddress:   peTestRsrcRVA,
		SizeOfRawData:    rawSize,
		PointerToRawData: peTestRsrcOff,
		Characteristics:  0x40000040,
	}
	copy(sh.Name[:], ".rsrc")
	require.NoError(t, binary.Write(&b, le, sh))
	require.LessOrEqual(t, b.Len(), peTestRsrcOff)
	b.Write(make([]byte, peTestRsrcOff-b.Len()))
	b.Write(rsrc.Bytes())
	b.Write(make([]byte, int(rawSize-rsrcLen)))
	return b.Bytes()
}

func writeVersionDirs(rsrc *bytes.Buffer, nameDir, langDir, dataEntries uint32, opts peBuildOpts, blobs [][]byte) {
	le := binary.LittleEndian
	var h [16]byte

	if uint32(rsrc.Len()) != nameDir { //nolint:gosec // test data is small
		panic("unexpected name dir offset")
	}
	le.PutUint16(h[14:], 1)
	rsrc.Write(h[:])
	var e [8]byte
	le.PutUint32(e[:], 1)
	le.PutUint32(e[4:], 0x80000000|langDir)
	rsrc.Write(e[:])

	le.PutUint16(h[14:], uint16(len(blobs))) //nolint:gosec // test data is small
	rsrc.Write(h[:])
	for i := range blobs {
		le.PutUint32(e[:], uint32(0x409+i))           //nolint:gosec // test data is small
		le.PutUint32(e[4:], dataEntries+16*uint32(i)) //nolint:gosec // test data is small
		rsrc.Write(e[:])
	}

	blobOff := dataEntries + 16*uint32(len(blobs)) //nolint:gosec // test data is small
	var offsets []uint32
	for _, blob := range blobs {
		blobOff = (blobOff+3)&^3 + uint32(opts.misalign) //nolint:gosec // test data is small
		offsets = append(offsets, blobOff)
		blobOff += uint32(len(blob)) //nolint:gosec // test data is small
	}
	for i, blob := range blobs {
		var d [16]byte
		le.PutUint32(d[:], peTestRsrcRVA+offsets[i])
		le.PutUint32(d[4:], uint32(len(blob))) //nolint:gosec // test data is small
		rsrc.Write(d[:])
	}
	for i, blob := range blobs {
		rsrc.Write(make([]byte, int(offsets[i])-rsrc.Len()))
		rsrc.Write(blob)
	}
}

// TestParsePEVersionResourcesFMAInstallers checks parity with saferwall on a
// few small real-world .exe installers of Fleet-maintained apps (current
// versions of ee/maintained-apps/outputs/<slug>/windows.json), which can't be
// committed for license reasons. It downloads them, so it is opt-in:
//
//	FLEET_PE_FMA_PARITY_TEST=1 go test ./pkg/file/ -run TestParsePEVersionResourcesFMAInstallers
//
// Installers are cached in FLEET_PE_FMA_PARITY_CACHE (default:
// fleet-pe-fma-installers in the user cache directory).
func TestParsePEVersionResourcesFMAInstallers(t *testing.T) {
	// Small (under 5 MB) .exe installers.
	slugs := []string{"bitwarden", "hwmonitor", "cpu-z"}
	const maxFMAParityBytes int64 = 5 << 20

	if os.Getenv("FLEET_PE_FMA_PARITY_TEST") == "" {
		t.Skip("set FLEET_PE_FMA_PARITY_TEST=1 to run (downloads installers)")
	}
	cacheDir := os.Getenv("FLEET_PE_FMA_PARITY_CACHE")
	if cacheDir == "" {
		userCache, err := os.UserCacheDir()
		require.NoError(t, err)
		cacheDir = filepath.Join(userCache, "fleet-pe-fma-installers")
	}
	require.NoError(t, os.MkdirAll(cacheDir, 0o755))

	type installer struct{ slug, url, sha256 string }
	var installers []installer
	for _, slug := range slugs {
		path := filepath.Join("..", "..", "ee", "maintained-apps", "outputs", slug, "windows.json")
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		var manifest struct {
			Versions []struct {
				InstallerURL string `json:"installer_url"`
				SHA256       string `json:"sha256"`
			} `json:"versions"`
		}
		require.NoError(t, json.Unmarshal(b, &manifest), path)
		require.NotEmpty(t, manifest.Versions, path)
		v := manifest.Versions[0]
		installers = append(installers, installer{slug: slug, url: v.InstallerURL, sha256: v.SHA256})
	}

	// Each installer is downloaded and checked in its own parallel subtest, so
	// results show up as they come (with -v). A vendor removing an old version
	// only skips that app.
	downloads := make(chan struct{}, 6)
	for _, inst := range installers {
		t.Run(inst.slug, func(t *testing.T) {
			t.Parallel()
			downloads <- struct{}{}
			path, err := downloadPEFixture(t, cacheDir, inst.slug, inst.url, inst.sha256, maxFMAParityBytes)
			<-downloads
			if err != nil {
				t.Skipf("installer not available: %v", err)
			}
			got := assertPEParity(t, path)

			// ExtractPEMetadata is what Fleet uses; it must accept every
			// installer the reference parser accepts.
			if got != nil {
				tfr, err := fleet.NewKeepFileReader(path)
				require.NoError(t, err)
				defer tfr.Close()
				meta, err := ExtractPEMetadata(tfr)
				require.NoError(t, err)
				t.Logf("name=%q version=%q", meta.Name, meta.Version)
			}
		})
	}
}

// downloadPEFixture downloads url to cacheDir unless it is already there, and
// checks its SHA256 (unless the manifest has none). It fails if the download
// is larger than maxBytes.
func downloadPEFixture(t *testing.T, cacheDir, slug, url, wantSHA256 string, maxBytes int64) (string, error) {
	path := filepath.Join(cacheDir, slug+".exe")
	checkSum := func() error {
		if wantSHA256 == "" || wantSHA256 == "no_check" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		if got := hex.EncodeToString(h.Sum(nil)); got != wantSHA256 {
			return fmt.Errorf("sha256 mismatch: got %s, want %s", got, wantSHA256)
		}
		return nil
	}
	if _, err := os.Stat(path); err == nil && checkSum() == nil {
		return path, nil
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, res.Status)
	}
	tmp, err := os.CreateTemp(cacheDir, slug+".*.part")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	n, err := io.Copy(tmp, io.LimitReader(res.Body, maxBytes+1))
	if err != nil {
		tmp.Close()
		return "", err
	}
	if n > maxBytes {
		tmp.Close()
		return "", fmt.Errorf("GET %s: larger than %d bytes", url, maxBytes)
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	return path, checkSum()
}
