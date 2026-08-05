package msi

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// harvest.go walks the orbit root directory and produces the same model
// heat.exe emitted for it (heat dir root -gg -g1 -cg OrbitFiles -scom -sfrag
// -srd -sreg -dr ORBITROOT -ke, plus the TransformHeat SDDL injection):
// a component per file with WiX-compatible dir/cmp/fil identifiers and
// generated GUIDs, and a CreateFolder component per empty directory.

const orbitRootRef = "ORBITROOT"

// harvestedDir is a directory below the orbit root that heat would author
// (the root itself is suppressed by -srd and becomes a reference to the
// ORBITROOT directory from the main authoring).
type harvestedDir struct {
	ID       string
	ParentID string // orbitRootRef for top-level directories
	Name     string
	// DefaultDir is the MSI DefaultDir column value ("name" or "short|name").
	DefaultDir string
}

// harvestedFile is one file component.
type harvestedFile struct {
	ComponentID   string
	ComponentGUID string
	FileID        string
	DirID         string // directory the component installs to
	// FileName is the MSI FileName column value ("name" or "short|name").
	FileName string
	Path     string // source path on disk
	Size     int64
	ModTime  time.Time
	Version  string // from the PE version resource, empty if unversioned
	Language string
	Hash     [16]byte // MsiGetFileHash-style MD5 (zeros for empty files)
}

// harvestedEmptyDir is a CreateFolder component for a directory with no
// files (heat -ke).
type harvestedEmptyDir struct {
	ComponentID   string
	ComponentGUID string
	DirID         string
}

// harvestedComponent is one component in heat's ComponentGroup order:
// either a file component or an empty-directory CreateFolder component.
type harvestedComponent struct {
	File     *harvestedFile
	EmptyDir *harvestedEmptyDir
}

func (c harvestedComponent) id() string {
	if c.File != nil {
		return c.File.ComponentID
	}
	return c.EmptyDir.ComponentID
}

func (c harvestedComponent) dirID() string {
	if c.File != nil {
		return c.File.DirID
	}
	return c.EmptyDir.DirID
}

// harvest is the result of walking the root: components in heat's
// ComponentGroup order and the harvested directories.
type harvest struct {
	Components []harvestedComponent
	Dirs       []*harvestedDir // in walk order (parents before children)
	dirByID    map[string]*harvestedDir
}

// harvestRoot walks rootDir mirroring heat's traversal: within a directory,
// files sort before subdirectories, each alphabetically (case-insensitive).
// Directory identity hashes use the chain rooted at wixIdentifier("dir",
// "ORBITROOT") — the id heat generated for the suppressed root.
func harvestRoot(rootDir string) (*harvest, error) {
	h := &harvest{dirByID: map[string]*harvestedDir{}}
	rootID := wixIdentifier("dir", orbitRootRef)

	var walk func(dirPath, dirID, exposedParentID string) (fileCount int, err error)
	walk = func(dirPath, dirID, exposedParentID string) (int, error) {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return 0, fmt.Errorf("read dir %s: %w", dirPath, err)
		}
		var files, dirs []os.DirEntry
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e)
			} else {
				files = append(files, e)
			}
		}
		sortEntries := func(list []os.DirEntry) {
			sort.SliceStable(list, func(a, b int) bool {
				return strings.ToLower(list[a].Name()) < strings.ToLower(list[b].Name())
			})
		}
		sortEntries(files)
		sortEntries(dirs)

		count := 0
		for _, f := range files {
			path := filepath.Join(dirPath, f.Name())
			info, err := f.Info()
			if err != nil {
				return 0, fmt.Errorf("stat %s: %w", path, err)
			}
			fileID := wixIdentifier("fil", dirID, f.Name())
			compID := wixIdentifier("cmp", dirID, fileID)
			guid, err := newGUID()
			if err != nil {
				return 0, err
			}
			ver, err := peFileVersion(path)
			if err != nil {
				return 0, err
			}
			hash, err := msiFileHash(path, info.Size())
			if err != nil {
				return 0, err
			}
			h.Components = append(h.Components, harvestedComponent{File: &harvestedFile{
				ComponentID:   compID,
				ComponentGUID: guid,
				FileID:        fileID,
				DirID:         exposedParentID,
				FileName:      msiFileName(f.Name(), true, "File", compID),
				Path:          path,
				Size:          info.Size(),
				ModTime:       info.ModTime(),
				Version:       ver.Version,
				Language:      ver.Language,
				Hash:          hash,
			}})
			count++
		}

		for _, d := range dirs {
			childID := wixIdentifier("dir", dirID, d.Name())
			// Generated 8.3 short names hash the PARENT directory id
			// (Compiler.CompileDirectoryElement passes parentId).
			hd := &harvestedDir{
				ID:         childID,
				ParentID:   exposedParentID,
				Name:       d.Name(),
				DefaultDir: msiFileName(d.Name(), false, "Directory", exposedParentID),
			}
			h.Dirs = append(h.Dirs, hd)
			h.dirByID[childID] = hd
			n, err := walk(filepath.Join(dirPath, d.Name()), childID, childID)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				guid, err := newGUID()
				if err != nil {
					return 0, err
				}
				h.Components = append(h.Components, harvestedComponent{EmptyDir: &harvestedEmptyDir{
					ComponentID:   wixIdentifier("cmp", childID),
					ComponentGUID: guid,
					DirID:         childID,
				}})
			}
			count += n
		}
		return count, nil
	}

	if _, err := walk(rootDir, rootID, orbitRootRef); err != nil {
		return nil, err
	}
	return h, nil
}

// parentChain returns the harvested directories from dirID up to (but not
// including) ORBITROOT, deepest first. Used to emit Directory rows the way
// the WiX linker resolves them (walking up from each component).
func (h *harvest) parentChain(dirID string) []*harvestedDir {
	var chain []*harvestedDir
	for dirID != orbitRootRef {
		d := h.dirByID[dirID]
		if d == nil {
			break
		}
		chain = append(chain, d)
		dirID = d.ParentID
	}
	return chain
}
