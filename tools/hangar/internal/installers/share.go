package installers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultSharePort is where the free-port search starts.
const DefaultSharePort uint16 = 8000

// maxRecentDownloads caps the download log the tab shows.
const maxRecentDownloads = 20

// ShareStatus is the share server's state for the tab.
type ShareStatus struct {
	Running   bool       `json:"running"`
	Port      uint16     `json:"port"`
	Root      string     `json:"root"`
	URL       string     `json:"url"` // http://<lan-ip>:<port>/, "" when stopped
	StartedAt int64      `json:"started_at_ms"`
	Recent    []Download `json:"recent"` // newest first
}

// Download is one file served to a client.
type Download struct {
	Path   string `json:"path"`
	Client string `json:"client"`
	At     int64  `json:"at_ms"`
}

// SharedFile is one installer under the share root, with ready-to-paste
// commands for the machine that will install it.
type SharedFile struct {
	Path    string `json:"path"` // relative to the root, slash-separated
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	Command string `json:"command"`
	Shell   string `json:"shell"` // what the command runs in: "sh" or "powershell"
}

// Share serves a folder of installers read-only over HTTP on all interfaces,
// so test machines on the LAN can download them. One per app.
type Share struct {
	mu        sync.Mutex
	srv       *http.Server
	rootFS    *os.Root
	port      uint16
	root      string
	startedAt time.Time
	recent    []Download
}

// Start serves root on port (all interfaces). It fails if already running or
// the port is taken.
func (s *Share) Start(root string, port uint16) error {
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("folder %s doesn't exist — build some installers first", root)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return errors.New("already sharing")
	}
	ln, err := listenAll(port)
	if err != nil {
		return fmt.Errorf("port %d: %w", port, err)
	}
	// os.Root refuses paths (including symlinks) that resolve outside root.
	rootFS, err := os.OpenRoot(root)
	if err != nil {
		ln.Close()
		return err
	}
	srv := &http.Server{ReadHeaderTimeout: 10 * time.Second}
	srv.Handler = s.handler(srv, rootFS)
	s.srv, s.rootFS, s.port, s.root, s.startedAt, s.recent = srv, rootFS, port, root, time.Now(), nil
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			if s.srv == srv {
				s.srv = nil
				rootFS.Close()
			}
			s.mu.Unlock()
		}
	}()
	return nil
}

// listenAll listens on port on all interfaces, after checking loopback too:
// on macOS a wildcard bind succeeds even when another process holds
// 127.0.0.1:port, which would leave localhost pointing at that process.
func listenAll(port uint16) (net.Listener, error) {
	lo, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(int(port)))
	if err != nil {
		return nil, err
	}
	lo.Close()
	return net.Listen("tcp", ":"+strconv.Itoa(int(port)))
}

// Stop shuts the server down (in-flight downloads get a few seconds).
func (s *Share) Stop() error {
	s.mu.Lock()
	srv, rootFS := s.srv, s.rootFS
	s.srv, s.rootFS = nil, nil
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	defer rootFS.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return srv.Close()
	}
	return nil
}

// Status reports the server's state; lanIP builds the URL.
func (s *Share) Status(lanIP string) ShareStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv == nil {
		return ShareStatus{Recent: []Download{}}
	}
	recent := make([]Download, len(s.recent))
	copy(recent, s.recent)
	return ShareStatus{
		Running:   true,
		Port:      s.port,
		Root:      s.root,
		URL:       BaseURL(lanIP, s.port),
		StartedAt: s.startedAt.UnixMilli(),
		Recent:    recent,
	}
}

// record logs a download from srv, unless srv has since been stopped (a slow
// download finishing after Stop mustn't show up in the next session).
func (s *Share) record(srv *http.Server, d Download) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != srv {
		return
	}
	s.recent = append([]Download{d}, s.recent...)
	if len(s.recent) > maxRecentDownloads {
		s.recent = s.recent[:maxRecentDownloads]
	}
}

// handler serves root read-only: GET/HEAD only, dotfiles hidden (temp files
// from an in-progress manifest write), directory listings on.
func (s *Share) handler(srv *http.Server, root *os.Root) http.Handler {
	files := http.FileServer(http.FS(dotHidingFS{root.FS()}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		files.ServeHTTP(rec, r)
		if r.Method == http.MethodGet && rec.status == http.StatusOK && isInstaller(r.URL.Path) {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			s.record(srv, Download{Path: strings.TrimPrefix(r.URL.Path, "/"), Client: host, At: time.Now().UnixMilli()})
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// dotHidingFS makes names starting with "." not exist.
type dotHidingFS struct{ fs.FS }

func (d dotHidingFS) Open(name string) (fs.File, error) {
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return nil, fs.ErrNotExist
		}
	}
	f, err := d.FS.Open(name)
	if err != nil {
		return nil, err
	}
	// Only wrap directories: *os.File implements ReadDir for every file, and
	// the wrapper hides Seek, which http.FileServer needs to sniff unknown
	// types (e.g. .rpm) and to serve ranges.
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if dir, ok := f.(fs.ReadDirFile); ok && info.IsDir() {
		return dotHidingDir{dir}, nil
	}
	return f, nil
}

type dotHidingDir struct{ fs.ReadDirFile }

func (d dotHidingDir) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := d.ReadDirFile.ReadDir(n)
	out := entries[:0]
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e)
		}
	}
	return out, err
}

func isInstaller(p string) bool {
	switch path.Ext(p) {
	case ".pkg", ".msi", ".deb", ".rpm":
		return true
	}
	return false
}

// BaseURL is the share's address as another machine dials it.
func BaseURL(lanIP string, port uint16) string {
	host := lanIP
	if host == "" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(int(port))) + "/"
}

// FreePort returns the first port at or after start that isn't in reserved
// and can be listened on, scanning 100 ports.
func FreePort(start uint16, reserved map[uint16]bool) (uint16, error) {
	for p := int(start); p < int(start)+100 && p <= 65535; p++ {
		if reserved[uint16(p)] {
			continue
		}
		ln, err := listenAll(uint16(p))
		if err != nil {
			continue
		}
		ln.Close()
		return uint16(p), nil
	}
	return 0, fmt.Errorf("no free port in %d–%d", start, int(start)+99)
}

// ListShared lists the installers under root (one level of server folders
// plus root itself) with download URLs and install commands for baseURL.
func ListShared(root, baseURL string) ([]SharedFile, error) {
	out := []SharedFile{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		depth := strings.Count(filepath.ToSlash(rel), "/")
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // the share won't serve links that leave the folder; don't offer them
		}
		if strings.HasPrefix(d.Name(), ".") && p != root {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root && depth >= 1 {
				return filepath.SkipDir
			}
			return nil
		}
		if !isInstaller(p) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		fileURL := baseURL + escapePath(relSlash)
		cmd, shell := InstallCommand(path.Base(relSlash), fileURL)
		out = append(out, SharedFile{Path: relSlash, URL: fileURL, Size: info.Size(), Command: cmd, Shell: shell})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return []SharedFile{}, nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

// escapePath percent-encodes every byte of each segment except unreserved
// characters — stricter than url.PathEscape (which leaves e.g. "$"), so a URL
// pasted into a shell or PowerShell command never carries syntax.
func escapePath(p string) string {
	var b strings.Builder
	for i, seg := range strings.Split(p, "/") {
		if i > 0 {
			b.WriteByte('/')
		}
		for _, c := range []byte(seg) {
			switch {
			case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '.', c == '_', c == '~':
				b.WriteByte(c)
			default:
				fmt.Fprintf(&b, "%%%02X", c)
			}
		}
	}
	return b.String()
}

// InstallCommand is a one-liner that downloads and installs an installer on
// the target OS. Linux and macOS commands use sudo; the Windows one must run
// in an elevated PowerShell.
//
// The file name never appears in the command (it comes from disk and could
// carry shell syntax); the download goes to a fixed name in a private temp
// dir. fileURL is percent-encoded, so it's safe inside single quotes.
func InstallCommand(name, fileURL string) (cmd, shell string) {
	switch path.Ext(name) {
	case ".deb":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.deb" '%s' && sudo apt-get install -y "$d/fleetd.deb"`, fileURL), "sh"
	case ".rpm":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.rpm" '%s' && sudo dnf install -y "$d/fleetd.rpm"`, fileURL), "sh"
	case ".pkg":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.pkg" '%s' && sudo installer -pkg "$d/fleetd.pkg" -target /`, fileURL), "sh"
	case ".msi":
		return fmt.Sprintf(`$f = Join-Path $env:TEMP 'fleetd-installer.msi'; Invoke-WebRequest -Uri '%s' -OutFile $f; Start-Process msiexec.exe -Wait -ArgumentList '/i', ('"' + $f + '"'), '/qn'`, fileURL), "powershell"
	}
	return "", ""
}
