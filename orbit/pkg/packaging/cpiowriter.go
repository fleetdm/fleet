package packaging

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// writeCPIOGzip creates a gzip-compressed CPIO archive in ODC (old character)
// format from srcDir and writes it to dstPath. All entries are assigned the
// given uid and gid. This is a pure Go replacement for the
// `find . | cpio -o --format odc -R uid:gid | gzip -c` pipeline.
func writeCPIOGzip(srcDir, dstPath string, uid, gid uint32) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer dst.Close()

	gw := gzip.NewWriter(dst)
	defer gw.Close()

	if err := writeCPIO(gw, srcDir, uid, gid); err != nil {
		return err
	}

	if err := gw.Close(); err != nil {
		return fmt.Errorf("close gzip: %w", err)
	}

	return dst.Sync()
}

// writeCPIO writes a CPIO archive in ODC format to w.
func writeCPIO(w io.Writer, srcDir string, uid, gid uint32) error {
	// Collect all paths, sorted (matching `find .` output).
	var paths []string
	err := filepath.WalkDir(srcDir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			paths = append(paths, ".")
		} else {
			paths = append(paths, "./"+rel)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk dir: %w", err)
	}

	sort.Strings(paths)

	inode := uint32(1)
	for _, rel := range paths {
		fullPath := filepath.Join(srcDir, rel)
		info, statErr := os.Lstat(fullPath)
		if statErr != nil {
			return fmt.Errorf("stat %s: %w", rel, statErr)
		}

		mode := uint32(info.Mode().Perm())
		if info.IsDir() {
			mode |= 0o40000
		} else if info.Mode()&fs.ModeSymlink != 0 {
			mode |= 0o120000
		} else {
			mode |= 0o100000
		}

		var fileSize uint64
		var fileData []byte
		if info.Mode().IsRegular() {
			fileData, err = os.ReadFile(fullPath)
			if err != nil {
				return fmt.Errorf("read %s: %w", rel, err)
			}
			fileSize = uint64(len(fileData))
		}

		// Name includes the null terminator.
		name := rel + "\x00"

		// Write ODC header (76 bytes, all fixed-width octal ASCII).
		hdr := fmt.Sprintf("070707"+
			"%06o"+ // dev
			"%06o"+ // ino
			"%06o"+ // mode
			"%06o"+ // uid
			"%06o"+ // gid
			"%06o"+ // nlink
			"%06o"+ // rdev
			"%011o"+ // mtime
			"%06o"+ // namesize
			"%011o", // filesize
			0,                           // dev
			inode,                       // ino
			mode,                        // mode
			uid,                         // uid
			gid,                         // gid
			nlinksForMode(info),         // nlink
			0,                           // rdev
			info.ModTime().Unix(),       // mtime
			len(name),                   // namesize (includes null)
			fileSize,                    // filesize
		)
		inode++

		if _, err := io.WriteString(w, hdr); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
		if _, err := io.WriteString(w, name); err != nil {
			return fmt.Errorf("write name: %w", err)
		}
		if len(fileData) > 0 {
			if _, err := w.Write(fileData); err != nil {
				return fmt.Errorf("write data: %w", err)
			}
		}
	}

	// Write trailer entry.
	trailerName := "TRAILER!!!\x00"
	trailer := fmt.Sprintf("070707"+
		"%06o%06o%06o%06o%06o%06o%06o%011o%06o%011o",
		0, 0, 0, 0, 0, 0, 0, 0,
		len(trailerName),
		0,
	)
	if _, err := io.WriteString(w, trailer); err != nil {
		return fmt.Errorf("write trailer header: %w", err)
	}
	if _, err := io.WriteString(w, trailerName); err != nil {
		return fmt.Errorf("write trailer name: %w", err)
	}

	return nil
}

// nlinksForMode returns the appropriate nlink count for a filesystem entry.
// Directories get nlink=2 (self + "."), files get nlink=1.
func nlinksForMode(info fs.FileInfo) uint32 {
	if info.IsDir() {
		return 2
	}
	return 1
}
