package msi

import (
	"crypto/md5" //nolint:gosec // WiX-compatible identifier hashing, not used for security
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// utf16LEBytes returns the UTF-16 little-endian encoding of s, matching
// .NET's Encoding.Unicode used by WiX when hashing identifier data.
func utf16LEBytes(s string) []byte {
	codes := utf16.Encode([]rune(s))
	out := make([]byte, 2*len(codes))
	for i, c := range codes {
		out[2*i] = byte(c) //nolint:gosec // UTF-16 code unit split into bytes
		out[2*i+1] = byte(c >> 8)
	}
	return out
}

// wixIdentifier generates an identifier the same way as WiX v3's
// Common.GenerateIdentifier (as used by heat.exe): prefix followed by the
// uppercase hex MD5 of the UTF-16LE arguments joined with "|".
//
// heat generates directory ids as wixIdentifier("dir", parentID, name),
// file ids as wixIdentifier("fil", dirID, fileName), component ids as
// wixIdentifier("cmp", dirID, fileID), and, with -srd, the suppressed root
// directory id as wixIdentifier("dir", directoryRef).
func wixIdentifier(prefix string, args ...string) string {
	sum := md5.Sum(utf16LEBytes(strings.Join(args, "|"))) //nolint:gosec
	return prefix + strings.ToUpper(fmt.Sprintf("%x", sum))
}

// wixShortName generates an 8.3-compliant short file/directory name the same
// way as WiX v3's CompilerCore.GenerateShortName: base64 of the MD5 of the
// UTF-16LE hash data (lowercased long name joined with args by "|"), first 8
// characters with '/'→'_' and '+'→'-', optionally keeping up to a 4-char
// extension (including the dot), all lowercased.
//
// WiX calls this with args ("Directory", parentDirectoryID) for directories
// (keepExtension=false) and ("File", componentID) for files
// (keepExtension=true).
func wixShortName(longName string, keepExtension bool, args ...string) string {
	data := strings.Join(append([]string{strings.ToLower(longName)}, args...), "|")
	sum := md5.Sum(utf16LEBytes(data)) //nolint:gosec
	short := base64.StdEncoding.EncodeToString(sum[:])[:8]
	short = strings.ReplaceAll(short, "/", "_")
	short = strings.ReplaceAll(short, "+", "-")
	if keepExtension {
		ext := filepath.Ext(strings.ToLower(longName))
		if len(ext) > 4 {
			ext = ext[:4]
		}
		short += ext
	}
	return strings.ToLower(short)
}

// needsShortName reports whether name is not already a valid 8.3 short name,
// in which case the MSI Filename/DefaultDir column carries a
// "short|long" pair. This mirrors WiX's CompilerCore.IsValidShortFilename.
func needsShortName(name string) bool {
	base := name
	ext := ""
	if i := strings.LastIndex(name, "."); i >= 0 {
		base, ext = name[:i], name[i+1:]
	}
	if len(base) == 0 || len(base) > 8 || len(ext) > 3 {
		return true
	}
	if strings.Count(name, ".") > 1 {
		return true
	}
	// Characters not allowed in a short name (subset relevant to our inputs;
	// anything unexpected errs on the side of generating a short name).
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '.', '_', '-', '$', '~', '!', '#', '%', '&', '\'', '(', ')', '@', '^', '`', '{', '}':
			continue
		}
		return true
	}
	return false
}

// msiFileName returns the value for an MSI Filename/DefaultDir column:
// the name itself if 8.3-safe, otherwise "generatedshort|long".
func msiFileName(longName string, keepExtension bool, args ...string) string {
	if !needsShortName(longName) {
		return longName
	}
	return wixShortName(longName, keepExtension, args...) + "|" + longName
}

// newGUID returns a fresh random (version 4) GUID formatted the way WiX
// emits generated GUIDs: uppercase, with braces.
func newGUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate GUID: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return strings.ToUpper(fmt.Sprintf("{%x-%x-%x-%x-%x}", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])), nil
}
