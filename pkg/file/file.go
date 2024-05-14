package file

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/secure"
)

var ErrUnsupportedType = errors.New("unsupported file type")

// ExtractInstallerMetadata extracts the software name and version from the
// installer file and returns them along with the sha256 hash of the bytes. The
// format of the installer is determined based on the magic bytes of the content.
func ExtractInstallerMetadata(r io.Reader) (name, version, extension string, shaSum []byte, err error) {
	br := bufio.NewReader(r)
	extension, err = typeFromBytes(br)
	if err != nil {
		return "", "", "", nil, err
	}

	switch extension {
	case "deb":
		name, version, shaSum, err = ExtractDebMetadata(br)
	case "exe":
		name, version, shaSum, err = ExtractPEMetadata(br)
	case "pkg":
		name, version, shaSum, err = ExtractXARMetadata(br)
	case "msi":
		name, version, shaSum, err = ExtractMSIMetadata(br)
	default:
		return "", "", "", nil, ErrUnsupportedType
	}

	return name, version, extension, shaSum, err
}

func typeFromBytes(br *bufio.Reader) (string, error) {
	switch {
	case hasPrefix(br, []byte{0x78, 0x61, 0x72, 0x21}):
		return "pkg", nil
	case hasPrefix(br, []byte("!<arch>\ndebian")):
		return "deb", nil
	case hasPrefix(br, []byte{0xd0, 0xcf}):
		return "msi", nil
	case hasPrefix(br, []byte("MZ")):
		if blob, _ := br.Peek(0x3e); len(blob) == 0x3e {
			reloc := binary.LittleEndian.Uint16(blob[0x3c:0x3e])
			if blob, err := br.Peek(int(reloc) + 4); err == nil {
				if bytes.Equal(blob[reloc:reloc+4], []byte("PE\x00\x00")) {
					return "exe", nil
				}
			}
		}
		fallthrough
	default:
		return "", ErrUnsupportedType
	}
}

func hasPrefix(br *bufio.Reader, blob []byte) bool {
	d, _ := br.Peek(len(blob))
	if len(d) < len(blob) {
		return false
	}
	return bytes.Equal(d, blob)
}

// Copy copies the file from srcPath to dstPath, using the provided permissions.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func Copy(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src for copy: %w", err)
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), os.ModeDir|perm); err != nil {
		return fmt.Errorf("create dst dir for copy: %w", err)
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("open dst for copy: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy src to dst: %w", err)
	}
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("sync dst after copy: %w", err)
	}

	return nil
}

// Copy copies the file from srcPath to dstPath, using the permissions of the original file.
//
// Note that on Windows the permissions support is limited in Go's file functions.
func CopyWithPerms(srcPath, dstPath string) error {
	stat, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("get permissions for copy: %w", err)
	}

	return Copy(srcPath, dstPath, stat.Mode().Perm())
}

// Exists returns whether the file exists and is a regular file.
func Exists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("check file exists: %w", err)
	}

	return info.Mode().IsRegular(), nil
}

// Dos2UnixNewlines takes a string containing Windows-style newlines (\r\n) and
// converts them to Unix-style newlines (\n). It returns the converted string.
func Dos2UnixNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func ExtractFilenameFromURLPath(p string, defaultExtension string) string {
	u, err := url.Parse(p)
	if err != nil {
		return ""
	}

	invalid := map[string]struct{}{
		"":  {},
		".": {},
		"/": {},
	}

	b := path.Base(u.Path)
	if _, ok := invalid[b]; ok {
		return ""
	}

	if _, ok := invalid[path.Ext(b)]; ok {
		return fmt.Sprintf("%s.%s", b, defaultExtension)
	}

	return b
}
