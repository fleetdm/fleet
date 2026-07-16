package packaging

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1" //nolint:gosec // xar's on-disk checksum format uses SHA-1; not used for security
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// This file implements a minimal, pure-Go writer for the xar archive format,
// sufficient to produce macOS flat .pkg installers. It simulates macOS command
// `xar --compression none` invocation.
//
// A xar archive is:
//
//	[ 28-byte header ][ zlib-compressed TOC (XML) ][ heap ]
//
// The header points at the compressed TOC; the TOC is an XML description of the
// file tree whose <data>/<checksum> elements reference byte ranges ("offset" and
// "length") within the heap. The heap begins with the SHA-1 checksum of the
// compressed TOC (as declared by the TOC's own <checksum> element), followed by
// each file's contents. Files are stored uncompressed (encoding
// application/octet-stream), matching the previous `--compression none` behavior.
//
// Reference: the xar on-disk format: https://github.com/mackyle/xar.

const (
	xarMagic        uint32 = 0x78617221 // "xar!"
	xarHeaderSize   uint16 = 28
	xarVersion      uint16 = 1
	xarChecksumSHA1 uint32 = 1  // cksum_alg value for SHA-1
	xarChecksumSize int64  = 20 // size of a SHA-1 digest in bytes
)

// xarEntry is a node in the archive tree.
type xarEntry struct {
	name     string
	isDir    bool
	mode     os.FileMode
	data     []byte      // file contents (nil for directories)
	children []*xarEntry // populated for directories

	// Populated during heap layout (files only):
	id     int
	offset int64
	size   int64
	sha1   string
}

// writeXar walks srcDir and writes an uncompressed xar archive of its contents
// to dstPath. The archive tree is rooted at srcDir's children (srcDir itself is
// not included as a node), mirroring `xar -cf dst -C srcDir <entries...>`.
func writeXar(srcDir, dstPath string) error {
	entries, err := buildXarTree(srcDir)
	if err != nil {
		return fmt.Errorf("build xar tree: %w", err)
	}

	// Lay out the heap. Offset 0 is reserved for the compressed-TOC checksum,
	// so file data starts at xarChecksumSize.
	var heap bytes.Buffer
	cursor := xarChecksumSize
	nextID := 1
	if err := layoutXarHeap(entries, &heap, &cursor, &nextID); err != nil {
		return err
	}

	// Build and compress the TOC.
	toc := buildXarTOC(entries)
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(toc); err != nil {
		return fmt.Errorf("compress toc: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close toc writer: %w", err)
	}

	tocChecksum := sha1.Sum(compressed.Bytes()) //nolint:gosec // required by the xar format

	// Assemble the archive: header + compressed TOC + heap(checksum + data).
	var out bytes.Buffer
	if err := writeXarHeader(&out, len(compressed.Bytes()), len(toc)); err != nil {
		return err
	}
	out.Write(compressed.Bytes())
	out.Write(tocChecksum[:])
	out.Write(heap.Bytes())

	if err := os.WriteFile(dstPath, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write xar: %w", err)
	}
	return nil
}

// writeXarHeader writes the 28-byte big-endian xar header.
func writeXarHeader(w *bytes.Buffer, compressedTOCLen, uncompressedTOCLen int) error {
	fields := []any{
		xarMagic,
		xarHeaderSize,
		xarVersion,
		uint64(compressedTOCLen),   //nolint:gosec // slice length is non-negative
		uint64(uncompressedTOCLen), //nolint:gosec // slice length is non-negative
		xarChecksumSHA1,
	}
	for _, f := range fields {
		if err := binary.Write(w, binary.BigEndian, f); err != nil {
			return fmt.Errorf("write xar header: %w", err)
		}
	}
	return nil
}

// buildXarTree reads dir and returns its immediate children as xar entries,
// recursing into subdirectories. Entries are sorted by name for deterministic
// output.
func buildXarTree(dir string) ([]*xarEntry, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var entries []*xarEntry
	for _, de := range dirEntries {
		full := filepath.Join(dir, de.Name())
		info, err := de.Info()
		if err != nil {
			return nil, err
		}

		entry := &xarEntry{name: de.Name(), mode: info.Mode().Perm()}
		if de.IsDir() {
			entry.isDir = true
			children, err := buildXarTree(full)
			if err != nil {
				return nil, err
			}
			entry.children = children
		} else {
			data, err := os.ReadFile(full)
			if err != nil {
				return nil, err
			}
			entry.data = data
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries, nil
}

// layoutXarHeap walks the tree in depth-first order, appending each file's data
// to heap and recording its id, offset, size, and checksum. Directories consume
// no heap space but still receive an id. cursor tracks the next free heap offset.
func layoutXarHeap(entries []*xarEntry, heap *bytes.Buffer, cursor *int64, nextID *int) error {
	for _, e := range entries {
		e.id = *nextID
		*nextID++

		if e.isDir {
			if err := layoutXarHeap(e.children, heap, cursor, nextID); err != nil {
				return err
			}
			continue
		}

		sum := sha1.Sum(e.data) //nolint:gosec // required by the xar format
		e.sha1 = fmt.Sprintf("%x", sum)
		e.size = int64(len(e.data))
		e.offset = *cursor
		heap.Write(e.data)
		*cursor += e.size
	}
	return nil
}

// buildXarTOC renders the TOC XML for the archive tree.
func buildXarTOC(entries []*xarEntry) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<xar>\n")
	b.WriteString(" <toc>\n")
	b.WriteString(`  <checksum style="sha1">` + "\n")
	fmt.Fprintf(&b, "   <size>%d</size>\n", xarChecksumSize)
	b.WriteString("   <offset>0</offset>\n")
	b.WriteString("  </checksum>\n")
	for _, e := range entries {
		writeXarTOCEntry(&b, e, 2)
	}
	b.WriteString(" </toc>\n")
	b.WriteString("</xar>\n")
	return []byte(b.String())
}

// writeXarTOCEntry renders a single <file> element (recursing into directory
// children) at the given indentation depth.
func writeXarTOCEntry(b *strings.Builder, e *xarEntry, depth int) {
	ind := strings.Repeat(" ", depth)
	fmt.Fprintf(b, "%s<file id=\"%d\">\n", ind, e.id)
	fmt.Fprintf(b, "%s <name>%s</name>\n", ind, xarEscape(e.name))
	if e.isDir {
		fmt.Fprintf(b, "%s <type>directory</type>\n", ind)
	} else {
		fmt.Fprintf(b, "%s <type>file</type>\n", ind)
	}
	fmt.Fprintf(b, "%s <mode>0%o</mode>\n", ind, e.mode)
	fmt.Fprintf(b, "%s <uid>0</uid>\n", ind)
	fmt.Fprintf(b, "%s <gid>80</gid>\n", ind)

	if e.isDir {
		for _, c := range e.children {
			writeXarTOCEntry(b, c, depth+1)
		}
	} else {
		fmt.Fprintf(b, "%s <data>\n", ind)
		fmt.Fprintf(b, "%s  <archived-checksum style=\"sha1\">%s</archived-checksum>\n", ind, e.sha1)
		fmt.Fprintf(b, "%s  <extracted-checksum style=\"sha1\">%s</extracted-checksum>\n", ind, e.sha1)
		fmt.Fprintf(b, "%s  <size>%d</size>\n", ind, e.size)
		fmt.Fprintf(b, "%s  <offset>%d</offset>\n", ind, e.offset)
		fmt.Fprintf(b, "%s  <encoding style=\"application/octet-stream\"/>\n", ind)
		fmt.Fprintf(b, "%s  <length>%d</length>\n", ind, e.size)
		fmt.Fprintf(b, "%s </data>\n", ind)
	}
	fmt.Fprintf(b, "%s</file>\n", ind)
}

// xarEscape escapes the small set of characters that can appear in a file name
// and would otherwise be invalid in the TOC XML.
func xarEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
