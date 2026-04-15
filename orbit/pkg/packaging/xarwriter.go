package packaging

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1" //nolint:gosec // XAR format requires SHA-1 for TOC checksums
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// XAR file format constants.
const (
	xarWriterMagic      = 0x78617221 // "xar!"
	xarWriterHeaderSize = 28
	xarWriterVersion    = 1
	xarCksumSHA1        = 1
	xarSHA1Size         = 20
)

// xarWriterHeader is the 28-byte XAR file header (all fields big-endian).
type xarWriterHeader struct {
	Magic            uint32
	Size             uint16
	Version          uint16
	TOCCompressed    uint64
	TOCUncompressed  uint64
	CksumAlg         uint32
}

// writeXAR creates a XAR archive at outputPath from all files in flatDir,
// using no compression (matching `xar --compression none -cf`).
// This is a pure Go replacement for the xar command.
func writeXAR(flatDir, outputPath string) error {
	// Collect files from flatDir.
	root, err := buildXARTree(flatDir)
	if err != nil {
		return fmt.Errorf("build xar tree: %w", err)
	}

	// Build the heap: SHA-1 placeholder (20 bytes) + file data.
	var heap bytes.Buffer
	heap.Write(make([]byte, xarSHA1Size)) // Placeholder for TOC checksum.

	// Assign heap offsets and compute checksums for each file.
	if err := assignHeapOffsets(root, &heap); err != nil {
		return fmt.Errorf("assign heap offsets: %w", err)
	}

	// Generate TOC XML.
	tocXML := generateTOCXML(root)

	// Compress TOC with zlib.
	var tocCompressed bytes.Buffer
	zw, err := zlib.NewWriterLevel(&tocCompressed, zlib.BestCompression)
	if err != nil {
		return fmt.Errorf("create zlib writer: %w", err)
	}
	if _, err := zw.Write(tocXML); err != nil {
		return fmt.Errorf("write toc to zlib: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("close zlib writer: %w", err)
	}

	// Compute SHA-1 of compressed TOC and write it into the heap placeholder.
	tocHash := sha1.Sum(tocCompressed.Bytes()) //nolint:gosec
	heapBytes := heap.Bytes()
	copy(heapBytes[0:xarSHA1Size], tocHash[:])

	// Write the output file.
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	// Header.
	hdr := xarWriterHeader{
		Magic:           xarWriterMagic,
		Size:            xarWriterHeaderSize,
		Version:         xarWriterVersion,
		TOCCompressed:   uint64(tocCompressed.Len()), //nolint:gosec // G115: TOC size is bounded
		TOCUncompressed: uint64(len(tocXML)),       //nolint:gosec // G115: TOC size is bounded
		CksumAlg:        xarCksumSHA1,
	}
	if err := binary.Write(out, binary.BigEndian, &hdr); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Compressed TOC.
	if _, err := out.Write(tocCompressed.Bytes()); err != nil {
		return fmt.Errorf("write toc: %w", err)
	}

	// Heap.
	if _, err := out.Write(heapBytes); err != nil {
		return fmt.Errorf("write heap: %w", err)
	}

	return out.Sync()
}

// xarFileNode represents a file or directory in the XAR archive.
type xarFileNode struct {
	name     string
	isDir    bool
	mode     uint32 // Unix permission bits (e.g. 0644)
	children []*xarFileNode
	fullPath string // Filesystem path for reading content.

	// Set during heap construction for files.
	heapOffset int64
	size       int64
	sha1Hex    string
}

// buildXARTree builds a tree of xarFileNode from the flatDir directory.
func buildXARTree(flatDir string) (*xarFileNode, error) {
	root := &xarFileNode{name: "", isDir: true}
	nodeMap := map[string]*xarFileNode{"": root}

	var allPaths []string
	err := filepath.WalkDir(flatDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == flatDir {
			return nil
		}
		rel, relErr := filepath.Rel(flatDir, path)
		if relErr != nil {
			return relErr
		}
		allPaths = append(allPaths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort for deterministic output.
	sort.Strings(allPaths)

	for _, rel := range allPaths {
		fullPath := filepath.Join(flatDir, rel)
		info, err := os.Lstat(fullPath)
		if err != nil {
			return nil, err
		}

		node := &xarFileNode{
			name:     filepath.Base(rel),
			isDir:    info.IsDir(),
			mode:     uint32(info.Mode().Perm()),
			fullPath: fullPath,
		}

		parentRel := filepath.Dir(rel)
		if parentRel == "." {
			parentRel = ""
		}
		parent := nodeMap[parentRel]
		if parent == nil {
			return nil, fmt.Errorf("parent not found for %s", rel)
		}
		parent.children = append(parent.children, node)

		if node.isDir {
			nodeMap[rel] = node
		}
	}

	return root, nil
}

// assignHeapOffsets walks the tree, reads file data, writes it to the heap,
// and records offsets and checksums.
func assignHeapOffsets(root *xarFileNode, heap *bytes.Buffer) error {
	// DFS to match the order files appear in the TOC.
	var walk func(n *xarFileNode) error
	walk = func(n *xarFileNode) error {
		if !n.isDir && n.fullPath != "" {
			data, err := os.ReadFile(n.fullPath)
			if err != nil {
				return fmt.Errorf("read %s: %w", n.fullPath, err)
			}
			n.heapOffset = int64(heap.Len())
			n.size = int64(len(data))
			h := sha1.Sum(data) //nolint:gosec
			n.sha1Hex = hex.EncodeToString(h[:])
			heap.Write(data)
		}
		for _, c := range n.children {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

// generateTOCXML produces the TOC XML for the XAR archive.
func generateTOCXML(root *xarFileNode) []byte {
	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	buf.WriteString("<xar>\n")
	buf.WriteString(" <toc>\n")
	fmt.Fprintf(&buf, "  <creation-time>%s</creation-time>\n",
		time.Now().UTC().Format("2006-01-02T15:04:05"))
	buf.WriteString("  <checksum style=\"sha1\">\n")
	buf.WriteString("   <offset>0</offset>\n")
	fmt.Fprintf(&buf, "   <size>%d</size>\n", xarSHA1Size)
	buf.WriteString("  </checksum>\n")

	nextID := 1
	for _, c := range root.children {
		writeXARFileXML(&buf, c, &nextID, "  ")
	}

	buf.WriteString(" </toc>\n")
	buf.WriteString("</xar>\n")
	return buf.Bytes()
}

func writeXARFileXML(buf *bytes.Buffer, node *xarFileNode, nextID *int, indent string) {
	id := *nextID
	*nextID++

	fmt.Fprintf(buf, "%s<file id=\"%d\">\n", indent, id)

	if !node.isDir {
		fmt.Fprintf(buf, "%s <data>\n", indent)
		fmt.Fprintf(buf, "%s  <length>%d</length>\n", indent, node.size)
		fmt.Fprintf(buf, "%s  <encoding style=\"application/octet-stream\"/>\n", indent)
		fmt.Fprintf(buf, "%s  <offset>%d</offset>\n", indent, node.heapOffset)
		fmt.Fprintf(buf, "%s  <size>%d</size>\n", indent, node.size)
		fmt.Fprintf(buf, "%s  <extracted-checksum style=\"sha1\">%s</extracted-checksum>\n", indent, node.sha1Hex)
		fmt.Fprintf(buf, "%s  <archived-checksum style=\"sha1\">%s</archived-checksum>\n", indent, node.sha1Hex)
		fmt.Fprintf(buf, "%s </data>\n", indent)
	}

	fmt.Fprintf(buf, "%s <mode>0%o</mode>\n", indent, node.mode)

	if node.isDir {
		fmt.Fprintf(buf, "%s <type>directory</type>\n", indent)
	} else {
		fmt.Fprintf(buf, "%s <type>file</type>\n", indent)
	}

	fmt.Fprintf(buf, "%s <name>%s</name>\n", indent, xmlEscape(node.name))

	for _, c := range node.children {
		writeXARFileXML(buf, c, nextID, indent+" ")
	}

	fmt.Fprintf(buf, "%s</file>\n", indent)
}

// xmlEscape escapes special XML characters.
func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
