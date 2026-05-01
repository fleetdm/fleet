package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watchedRoots is the hardcoded list of top-level directories whose Go files
// can affect the `fleet` and `fleetctl` binaries (Approach A from the PR
// design discussion). If Fleet adds a new top-level Go directory the binary
// imports, add it here. Paths are joined with the repo root at runtime.
var watchedRoots = []string{
	"server",
	"cmd/fleet",
	"cmd/fleetctl",
	"ee",
	"pkg",
}

// watchedRootFiles are individual files at the repo root we also care about.
// fsnotify watches directories, so we register the repo root and filter
// events to just these names.
var watchedRootFiles = map[string]bool{
	"go.mod": true,
	"go.sum": true,
}

// excludedDirs are skipped during the recursive walk and ignored in event
// paths. `vendor` is checked-in deps, `build` is artifacts, `node_modules`
// belongs to webpack, `.git` is git internals.
var excludedDirs = map[string]bool{
	"vendor":       true,
	"build":        true,
	"node_modules": true,
	".git":         true,
}

const watcherDebounce = 500 * time.Millisecond

// watcherTrigger is what the watcher sends out: a human-readable reason
// (used in the dashboard step list) and the underlying changed files for
// debugging.
type watcherTrigger struct {
	Reason string
	Files  []string
}

// watcher wraps fsnotify with the path filter, debounced batching, and
// recursive-add behavior we need. It does NOT know about engine state
// (running / paused) — the engine consumes the trigger channel and
// decides whether to act.
type watcher struct {
	fs       *fsnotify.Watcher
	repoRoot string
	out      chan watcherTrigger

	mu      sync.Mutex
	pending map[string]struct{} // keyed by path-relative-to-repo-root
	timer   *time.Timer

	cancel context.CancelFunc
	done   chan struct{}
}

// newWatcher constructs a watcher and registers every directory under the
// hardcoded roots. Returns an error if fsnotify can't be initialized or any
// of the directories don't exist.
func newWatcher(repoRoot string) (*watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &watcher{
		fs:       fsw,
		repoRoot: repoRoot,
		out:      make(chan watcherTrigger, 8),
		pending:  map[string]struct{}{},
		done:     make(chan struct{}),
	}

	// Repo root itself — needed to catch go.mod / go.sum writes.
	if err := fsw.Add(repoRoot); err != nil {
		_ = fsw.Close()
		return nil, err
	}
	for _, sub := range watchedRoots {
		root := filepath.Join(repoRoot, sub)
		if err := w.addDirRecursive(root); err != nil {
			_ = fsw.Close()
			return nil, err
		}
	}
	return w, nil
}

// Start kicks off the event loop in a goroutine. Triggers are sent on the
// returned channel.
func (w *watcher) Start(ctx context.Context) <-chan watcherTrigger {
	ctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	go w.run(ctx)
	return w.out
}

// Stop halts the watcher and closes the trigger channel.
func (w *watcher) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	<-w.done
	_ = w.fs.Close()
}

func (w *watcher) run(ctx context.Context) {
	defer close(w.done)
	for {
		select {
		case <-ctx.Done():
			w.mu.Lock()
			if w.timer != nil {
				w.timer.Stop()
				w.timer = nil
			}
			w.mu.Unlock()
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.handleEvent(ev)
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			// fsnotify errors are usually recoverable noise (a watch
			// expired, kernel buffer overflowed). Don't crash the
			// watcher over them.
			_ = err
		}
	}
}

// handleEvent processes a single fsnotify event: filter it, update the
// pending set, and (re)arm the debounce timer.
func (w *watcher) handleEvent(ev fsnotify.Event) {
	// On directory creation, recursively add the new dir so its children
	// are watched too. Skip the rest of the event handling for that path
	// since dirs themselves aren't .go files.
	if ev.Op&fsnotify.Create != 0 && isDir(ev.Name) {
		_ = w.addDirRecursive(ev.Name)
		return
	}

	if !w.shouldFireOn(ev.Name) {
		return
	}
	// We only care about modifications: Write, Create, Rename, Remove.
	// Chmod alone (e.g. `touch -h`) shouldn't trigger a rebuild.
	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return
	}

	rel, err := filepath.Rel(w.repoRoot, ev.Name)
	if err != nil {
		rel = ev.Name
	}

	w.mu.Lock()
	w.pending[rel] = struct{}{}
	if w.timer == nil {
		w.timer = time.AfterFunc(watcherDebounce, w.fire)
	} else {
		w.timer.Reset(watcherDebounce)
	}
	w.mu.Unlock()
}

// fire is called after the debounce window with no further events. It
// drains the pending set, builds the trigger reason, and sends.
func (w *watcher) fire() {
	w.mu.Lock()
	files := make([]string, 0, len(w.pending))
	for f := range w.pending {
		files = append(files, f)
	}
	w.pending = map[string]struct{}{}
	w.timer = nil
	w.mu.Unlock()

	if len(files) == 0 {
		return
	}
	sort.Strings(files)

	trig := watcherTrigger{
		Reason: buildReason(files),
		Files:  files,
	}
	select {
	case w.out <- trig:
	default:
		// Channel full — engine hasn't drained yet. Drop the trigger;
		// the next event will fire one anyway.
	}
}

// shouldFireOn returns true if this path is one we'd rebuild for: a Go
// source file, or go.mod / go.sum at the repo root. Excludes dotfiles,
// editor backup files (~), and anything under vendor/build/node_modules/.git.
func (w *watcher) shouldFireOn(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") {
		return false
	}

	// go.mod / go.sum only count when they're at the repo root.
	if watchedRootFiles[base] {
		return filepath.Dir(path) == w.repoRoot
	}

	if !strings.HasSuffix(base, ".go") {
		return false
	}

	// Reject paths that pass through any excluded directory.
	rel, err := filepath.Rel(w.repoRoot, path)
	if err != nil {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if excludedDirs[part] {
			return false
		}
	}
	return true
}

// addDirRecursive walks root and registers every directory with fsnotify,
// skipping excluded names and dotfiles.
func (w *watcher) addDirRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// A permissions error or vanished dir mid-walk shouldn't
			// kill watcher startup. Just skip it.
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		base := d.Name()
		if excludedDirs[base] {
			return fs.SkipDir
		}
		if strings.HasPrefix(base, ".") && path != root {
			return fs.SkipDir
		}
		return w.fs.Add(path)
	})
}

func buildReason(files []string) string {
	switch len(files) {
	case 0:
		return "files changed"
	case 1:
		return files[0] + " changed"
	default:
		return files[0] + " changed (+" + strconv.Itoa(len(files)-1) + " others)"
	}
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
