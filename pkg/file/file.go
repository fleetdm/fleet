package file

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
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

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog"
)

var ErrUnsupportedType = errors.New("unsupported file type")
var ErrInvalidTarball = errors.New("not a valid .tar.gz archive")

type InstallerMetadata struct {
	Name             string
	Version          string
	BundleIdentifier string
	SHASum           []byte
	Extension        string
	PackageIDs       []string
	UpgradeCode      string
}

// ExtractInstallerMetadata extracts the software name and version from the
// installer file and returns them along with the sha256 hash of the bytes. The
// format of the installer is determined based on the magic bytes of the content.
func ExtractInstallerMetadata(tfr *fleet.TempFileReader) (*InstallerMetadata, error) {
	br := bufio.NewReader(tfr)
	extension, err := typeFromBytes(br)
	if err != nil {
		return nil, err
	}
	if err := tfr.Rewind(); err != nil {
		return nil, err
	}

	var meta *InstallerMetadata
	switch extension {
	case "deb":
		meta, err = ExtractDebMetadata(tfr)
	case "rpm":
		meta, err = ExtractRPMMetadata(tfr)
	case "exe":
		meta, err = ExtractPEMetadata(tfr)
	case "pkg":
		meta, err = ExtractXARMetadata(tfr)
	case "msi":
		meta, err = ExtractMSIMetadata(tfr)
	case "tar.gz":
		meta, err = ValidateTarball(tfr)
		if err != nil {
			err = errors.Join(ErrInvalidTarball, err)
		}
	default:
		return nil, ErrUnsupportedType
	}

	if meta != nil {
		meta.Extension = extension
	}

	return meta, err
}

// typeFromBytes deduces the type from the magic bytes.
// See https://en.wikipedia.org/wiki/List_of_file_signatures.
func typeFromBytes(br *bufio.Reader) (string, error) {
	switch {
	case hasPrefix(br, []byte{0x78, 0x61, 0x72, 0x21}):
		return "pkg", nil
	case hasPrefix(br, []byte("!<arch>\ndebian")):
		return "deb", nil
	case hasPrefix(br, []byte{0xed, 0xab, 0xee, 0xdb}):
		return "rpm", nil
	case hasPrefix(br, []byte{0xd0, 0xcf}):
		return "msi", nil
	// will capture standalone gz files but will fail on tar read attempt, so good enough
	case hasPrefix(br, []byte{0x1f, 0x8b}):
		return "tar.gz", nil
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

// ExtractTarGz extracts the contents of the provided tar.gz file.
// This implementation uses os.* calls without permission checks, as we're
// running this operation in the context of fleetd running as root on a host
// (e.g. for installs), so we have different constraints than fleetctl building
// a package. destDir should be provided by the code rather than user input to
// avoid directory traversal attacks. maxFileSize indicates how large we want
// to allow the max file size to be when decompressing, as a zip bomb mitigation.
func ExtractTarGz(path string, destDir string, maxFileSize int64, logger zerolog.Logger) error {
	tarGzFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer tarGzFile.Close()

	gzipReader, err := gzip.NewReader(tarGzFile)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		switch {
		case err == nil:
			// OK
		case errors.Is(err, io.EOF):
			return nil
		default:
			return fmt.Errorf("tar reader: %w", err)
		}

		// Prevent zip-slip attack (which, combined with a trusted destDir, remediates the potential directory traversal
		// attack below)
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("invalid path in tar.gz: %q", header.Name)
		}

		targetPath := filepath.Join(destDir, header.Name) // nolint:gosec // see above notes on dir traversal

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, constant.DefaultDirMode); err != nil {
				return fmt.Errorf("mkdir %q: %w", header.Name, err)
			}
		case tar.TypeReg:
			err := func() error {
				// Make sure parent directory exists
				if err := os.MkdirAll(filepath.Dir(targetPath), constant.DefaultDirMode); err != nil {
					return fmt.Errorf("ensure parent dir exists %q: %w", header.Name, err)
				}

				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, header.FileInfo().Mode())
				if err != nil {
					return fmt.Errorf("failed to create %q: %w", header.Name, err)
				}
				defer outFile.Close()

				// CopyN call to avoid zip bomb DoS since we have less control over arbitrary .tar.gz archives
				// than in e.g. a TUF case.
				var readBytes int64
				chunkSize := int64(65536) // 64KiB
				for {
					if readBytes+chunkSize > maxFileSize {
						return fmt.Errorf("aborted extraction of oversized file after %d bytes", readBytes)
					}

					_, err := io.CopyN(outFile, tarReader, chunkSize)
					if err != nil {
						if err == io.EOF {
							break
						}
						return fmt.Errorf("failed to extract file %q inside %q: %w", header.Name, path, err)
					}
					readBytes += chunkSize
				}

				return nil
			}()
			if err != nil {
				return err
			}
		default:
			logger.Warn().Msgf("skipping unknown tar header flag type %d: %q", header.Typeflag, header.Name)
		}
	}
}
