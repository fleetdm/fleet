package installers

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// SharedFile is one installer under the share root, with ready-to-paste
// commands for the machine that will install it.
type SharedFile struct {
	Path    string `json:"path"` // relative to the root, slash-separated
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	Command string `json:"command"`
	Shell   string `json:"shell"` // what the command runs in: "sh" or "powershell"
}

func isInstaller(p string) bool {
	return InstallerExt(p) != ""
}

// BaseURL is a server's address as another machine dials it.
func BaseURL(lanIP string, port uint16) string {
	host := lanIP
	if host == "" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(int(port))) + "/"
}

var pythonServe = regexp.MustCompile(` http\.server (\d+) --directory (.+)$`)

// PythonServedURL is the URL at which python http.server, run as command
// (`python3 -m http.server <port> --directory <dir>`) in cwd, serves root, or
// "" when root isn't inside the directory it serves.
func PythonServedURL(command, cwd, root, lanIP string) string {
	m := pythonServe.FindStringSubmatch(command)
	if m == nil {
		return ""
	}
	port, err := strconv.ParseUint(m[1], 10, 16)
	if err != nil {
		return ""
	}
	dir := m[2]
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(cwd, dir)
	}
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(root))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	u := BaseURL(lanIP, uint16(port))
	if rel != "." {
		u += escapePath(filepath.ToSlash(rel)) + "/"
	}
	return u
}

// ListShared lists the installers under root (one level of server folders
// plus root itself) with download URLs and install commands for baseURL. With
// no baseURL (not sharing) only paths and sizes are filled in.
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
		f := SharedFile{Path: filepath.ToSlash(rel), Size: info.Size()}
		if baseURL != "" {
			f.URL = baseURL + escapePath(f.Path)
			f.Command, f.Shell = InstallCommand(path.Base(f.Path), f.URL)
		}
		out = append(out, f)
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
	switch InstallerExt(name) {
	case ".pkg.tar.zst":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.pkg.tar.zst" '%s' && sudo pacman -U --noconfirm "$d/fleetd.pkg.tar.zst"`, fileURL), "sh"
	case ".deb":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.deb" '%s' && sudo apt-get install -y "$d/fleetd.deb"`, fileURL), "sh"
	case ".rpm":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.rpm" '%s' && sudo dnf install -y "$d/fleetd.rpm"`, fileURL), "sh"
	case ".pkg":
		return fmt.Sprintf(`d=$(mktemp -d) && curl -fLo "$d/fleetd.pkg" '%s' && sudo installer -pkg "$d/fleetd.pkg" -target /`, fileURL), "sh"
	case ".msi":
		// Stop on a failed download, and download to a fresh name, so msiexec
		// never installs an older package left in %TEMP%. Hiding the progress
		// bar makes Invoke-WebRequest many times faster on Windows PowerShell.
		return fmt.Sprintf(`$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue'; $f = Join-Path $env:TEMP ('fleetd-' + [guid]::NewGuid() + '.msi'); Invoke-WebRequest -UseBasicParsing -Uri '%s' -OutFile $f; Start-Process msiexec.exe -Wait -ArgumentList '/i', ('"' + $f + '"'), '/qn'`, fileURL), "powershell"
	}
	return "", ""
}
