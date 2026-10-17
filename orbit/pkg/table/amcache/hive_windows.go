//go:build windows

package amcache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"www.velocidex.com/golang/go-ntfs/parser"
)

// readHive reads the Amcache hive and replays its transaction logs in memory if it's dirty.
// Data returned with a nil error is always clean: a torn hive parses into rows that look as
// authoritative as correct ones.
func readHive(ctx context.Context) (hiveResult, error) {
	s, err := acquireHive(ctx)
	if err != nil {
		return hiveResult{}, err
	}
	res := hiveResult{data: s.data, raw: s.raw}

	dirty, err := hiveDirty(res.data)
	if err != nil {
		return hiveResult{}, err
	}
	if !dirty {
		return res, nil
	}
	res.replayed = true

	if err := ctx.Err(); err != nil {
		return hiveResult{}, err
	}
	// One unreadable log isn't fatal because the other may hold the whole chain. Ending up with
	// a hive the logs didn't clean is.
	var logs [][]byte
	var logErrs []error
	for _, suffix := range []string{".LOG1", ".LOG2"} {
		b, err := s.readLog(ctx, suffix)
		if err != nil {
			logErrs = append(logErrs, fmt.Errorf("amcache: %s%s: %w", s.path, suffix, err))
			continue
		}
		logs = append(logs, b)
	}
	out, err := replayLogs(res.data, logs...)
	if err != nil {
		return hiveResult{}, err
	}
	stillDirty, err := hiveDirty(out)
	if err != nil {
		return hiveResult{}, err
	}
	if stillDirty {
		return hiveResult{}, errors.Join(append(logErrs,
			fmt.Errorf("amcache: %s is dirty and the transaction logs did not recover it", s.path))...)
	}
	res.data = out
	return res, nil
}

type hiveSource struct {
	drive string
	path  string
	data  []byte
	raw   bool
}

func acquireHive(ctx context.Context) (hiveSource, error) {
	dir, path, err := hivePath()
	if err != nil {
		return hiveSource{}, err
	}
	s := hiveSource{drive: dir[:2], path: path}

	data, err := readPlain(ctx, path, maxHiveBytes)
	switch {
	case err == nil:
		s.data = data
		return s, nil
	case errors.Is(err, fs.ErrNotExist):
		return hiveSource{}, errHiveNotFound
	case errors.Is(err, windows.ERROR_SHARING_VIOLATION), errors.Is(err, windows.ERROR_LOCK_VIOLATION):
		// The appraiser has the hive open. A raw volume handle is privileged and EDR products
		// flag it, so it's only opened for this one failure.
		raw, err := readRaw(ctx, s.drive, path, maxHiveBytes)
		if err != nil {
			return hiveSource{}, err
		}
		s.data, s.raw = raw, true
		return s, nil
	default:
		return hiveSource{}, err
	}
}

// readLog reads a transaction log through the same route that produced the hive.
func (s hiveSource) readLog(ctx context.Context, suffix string) ([]byte, error) {
	if s.raw {
		return readRaw(ctx, s.drive, s.path+suffix, maxLogBytes)
	}
	return readPlain(ctx, s.path+suffix, maxLogBytes)
}

// readChunk is the unit between context checks, so a slow disk (or an AV scanner) can't keep a
// read running long past the load deadline.
const readChunk = 1 << 20

// hivePath gets the Windows directory from the API rather than the environment, which the
// caller controls.
func hivePath() (dir, path string, err error) {
	dir, err = windows.GetSystemWindowsDirectory()
	if err != nil {
		return "", "", fmt.Errorf("amcache: system windows directory: %w", err)
	}
	// dir[:2] is about to be concatenated into a \\.\ device path.
	if len(dir) < 3 || dir[2] != '\\' || dir[1] != ':' ||
		!(dir[0] >= 'A' && dir[0] <= 'Z' || dir[0] >= 'a' && dir[0] <= 'z') {
		return "", "", fmt.Errorf("amcache: unexpected windows directory %q", dir)
	}
	return dir, filepath.Join(dir, `AppCompat\Programs\Amcache.hve`), nil
}

// statHive returns the hive's modification time and size. os.Stat goes through
// GetFileAttributesEx, which takes no handle, so it works while the appraiser holds the hive
// open.
func statHive() (time.Time, int64, error) {
	_, path, err := hivePath()
	if err != nil {
		return time.Time{}, 0, err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, 0, err
	}
	return fi.ModTime(), fi.Size(), nil
}

// readPlain returns errors unwrapped so the caller can classify sharing violations.
func readPlain(ctx context.Context, path string, limit int64) ([]byte, error) {
	f, err := os.Open(path) // #nosec G304 -- fixed path under GetSystemWindowsDirectory
	if err != nil {
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("amcache: %s is not a regular file", path)
	}
	if st.Size() > limit {
		return nil, fmt.Errorf("amcache: %s is %d bytes, over the %d byte cap", path, st.Size(), limit)
	}
	// limit+1 detects a file that grew between the Stat and the read.
	r := io.LimitReader(f, limit+1)
	buf := make([]byte, 0, st.Size()+1)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := r.Read(buf[len(buf):min(cap(buf), len(buf)+readChunk)])
		buf = buf[:len(buf)+n]
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	if int64(len(buf)) > limit {
		return nil, fmt.Errorf("amcache: %s is over the %d byte cap", path, limit)
	}
	return buf, nil
}

// readRaw reads winPath straight from the NTFS volume behind drive (e.g. "C:"), bypassing the
// share-mode check.
func readRaw(ctx context.Context, drive, winPath string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	vol, err := os.Open(`\\.\` + drive) // #nosec G304 -- drive letter comes from GetSystemWindowsDirectory
	if err != nil {
		return nil, fmt.Errorf("amcache: open volume %s: %w", drive, err)
	}
	defer vol.Close()

	// A raw volume handle rejects unaligned reads, so all reads go through 1 MiB pages.
	pr, err := parser.NewPagedReader(vol, 1<<20, 16)
	if err != nil {
		return nil, fmt.Errorf("amcache: paged reader on %s: %w", drive, err)
	}
	ntfs, err := parser.GetNTFSContext(pr, 0)
	if err != nil {
		return nil, fmt.Errorf("amcache: ntfs context on %s: %w", drive, err)
	}
	defer ntfs.Close()

	// GetDataForPath treats the first colon as an alternate data stream separator, so the
	// drive letter must be stripped.
	rng, err := parser.GetDataForPath(ntfs, filepath.ToSlash(winPath[2:]))
	if err != nil {
		return nil, fmt.Errorf("amcache: raw lookup of %s: %w", winPath, err)
	}
	n := parser.RangeSize(rng)
	if n <= 0 || n > limit {
		return nil, fmt.Errorf("amcache: raw %s is %d bytes, outside (0, %d]", winPath, n, limit)
	}
	buf := make([]byte, n)
	for off := int64(0); off < n; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(off+readChunk, n)
		// io.EOF with a full chunk is success; a short read must not pass as a smaller hive.
		got, err := rng.ReadAt(buf[off:end], off)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("amcache: raw read of %s: %w", winPath, err)
		}
		if int64(got) != end-off {
			return nil, fmt.Errorf("amcache: raw read of %s returned %d of %d bytes at offset %d", winPath, got, end-off, off)
		}
		off = end
	}
	return buf, nil
}
