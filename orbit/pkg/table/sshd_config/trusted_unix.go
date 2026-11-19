//go:build darwin || linux

package sshd_config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// maxSymlinks matches Linux's limit on symlinks followed in one path lookup.
const maxSymlinks = 40

// resolveTrusted resolves path one component at a time and returns the file it
// leads to, provided only root can change what it leads to: "/" and every
// directory, symlink, and file along the way must pass checkTrusted.
// filepath.EvalSymlinks isn't enough on its own, because a symlink in a
// user-writable directory could be repointed at any root-owned binary.
func resolveTrusted(path string) (string, fs.FileInfo, error) {
	if !filepath.IsAbs(path) {
		return "", nil, fmt.Errorf("%s: not an absolute path", path)
	}
	if err := lstatTrusted("/"); err != nil {
		return "", nil, err
	}

	resolved := "/"
	pending := strings.Split(path, "/")
	for links := 0; len(pending) > 0; {
		name := pending[0]
		pending = pending[1:]
		switch name {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}

		next := filepath.Join(resolved, name)
		info, err := os.Lstat(next)
		if err != nil {
			return "", nil, err
		}
		if err := checkTrusted(next, info); err != nil {
			return "", nil, err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			resolved = next
			continue
		}

		links++
		if links > maxSymlinks {
			return "", nil, fmt.Errorf("%s: too many levels of symbolic links", path)
		}
		target, err := os.Readlink(next)
		if err != nil {
			return "", nil, err
		}
		if filepath.IsAbs(target) {
			resolved = "/"
		}
		pending = append(strings.Split(target, "/"), pending...)
	}

	info, err := os.Lstat(resolved)
	if err != nil {
		return "", nil, err
	}
	return resolved, info, nil
}

func lstatTrusted(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return checkTrusted(path, info)
}

// checkTrusted requires that root owns the entry and that group and others
// can't write to it. A writable directory is allowed when it is sticky (like
// /tmp or /nix/store): there, only an entry's owner can rename or replace it,
// and each entry is checked in turn. A symlink's own permissions are ignored,
// since they are never enforced.
func checkTrusted(path string, info fs.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return fmt.Errorf("%s: not owned by root", path)
	}
	mode := info.Mode()
	sticky := mode.IsDir() && mode&fs.ModeSticky != 0
	if mode&fs.ModeSymlink == 0 && !sticky && mode.Perm()&0o022 != 0 {
		return fmt.Errorf("%s: writable by group or others", path)
	}
	return nil
}
