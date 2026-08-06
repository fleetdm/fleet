package execuser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadEnvFromProcFile(t *testing.T) {
	dir := t.TempDir()
	environPath := filepath.Join(dir, "environ")

	testCases := []struct {
		name     string
		environ  string
		envVar   string
		expected string
	}{
		{
			name:     "DISPLAY found",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":0",
		},
		{
			name:     "DISPLAY :1",
			environ:  "HOME=/home/foo\x00DISPLAY=:1\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":1",
		},
		{
			name:     "DISPLAY not present",
			environ:  "HOME=/home/foo\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "empty environ",
			environ:  "",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "does not match prefix substring",
			environ:  "DISPLAY_NUM=5\x00DISPLAY=:2",
			envVar:   "DISPLAY",
			expected: ":2",
		},
		{
			name:     "other env var",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00WAYLAND_DISPLAY=wayland-0",
			envVar:   "WAYLAND_DISPLAY",
			expected: "wayland-0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(environPath, []byte(tc.environ), 0o644))
			result, err := readEnvFromProcFile(environPath, tc.envVar)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestReadEnvFromProcFileMissing(t *testing.T) {
	_, err := readEnvFromProcFile("/nonexistent/path/environ", "DISPLAY")
	require.Error(t, err)
}

func TestReadAllEnvFromProcFile(t *testing.T) {
	dir := t.TempDir()
	environPath := filepath.Join(dir, "environ")

	require.NoError(t, os.WriteFile(environPath,
		[]byte("HOME=/home/foo\x00XDG_DATA_DIRS=/a:/b\x00EMPTY=\x00NOEQUALS\x00LANG=en_US.UTF-8"),
		0o644))

	environ, err := readAllEnvFromProcFile(environPath)
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"HOME":          "/home/foo",
		"XDG_DATA_DIRS": "/a:/b",
		"EMPTY":         "",
		"LANG":          "en_US.UTF-8",
	}, environ)
}

func TestFilterSessionEnv(t *testing.T) {
	got := filterSessionEnv(map[string]string{
		// Carried over: what a GUI child needs from the session.
		"XDG_DATA_DIRS":       "/usr/share/gnome:/usr/share",
		"XDG_RUNTIME_DIR":     "/run/user/1000",
		"XDG_CURRENT_DESKTOP": "ubuntu:GNOME",
		"GTK_IM_MODULE":       "ibus",
		"QT_IM_MODULE":        "ibus",
		"GDK_BACKEND":         "wayland",
		"XMODIFIERS":          "@im=ibus",
		"XAUTHORITY":          "/home/foo/.Xauthority",
		"LANG":                "en_US.UTF-8",
		"LC_ALL":              "en_US.UTF-8",

		// Dropped: orbit sets its own, or they are shell artifacts.
		"DISPLAY":                  ":9",
		"WAYLAND_DISPLAY":          "wayland-9",
		"DBUS_SESSION_BUS_ADDRESS": "unix:path=/wrong",
		"LD_LIBRARY_PATH":          "/wrong",
		"LD_PRELOAD":               "/wrong",
		"PATH":                     "/wrong",
		"HOME":                     "/wrong",
		"SUDO_USER":                "root",
		"PWD":                      "/home/foo",
		"SHLVL":                    "1",
		"_":                        "/usr/bin/env",

		// Dropped: makes GTK load arbitrary modules into a process that
		// handles the end user's passphrase.
		"GTK_MODULES": "canberra-gtk-module",

		// Dropped: an empty value carries no information, and forwarding an
		// empty XDG_DATA_DIRS would suppress the default.
		"XDG_DATA_HOME": "",
		"LANGUAGE":      "",
	})

	require.Equal(t, []string{
		"GDK_BACKEND=wayland",
		"GTK_IM_MODULE=ibus",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
		"QT_IM_MODULE=ibus",
		"XAUTHORITY=/home/foo/.Xauthority",
		"XDG_CURRENT_DESKTOP=ubuntu:GNOME",
		"XDG_DATA_DIRS=/usr/share/gnome:/usr/share",
		"XDG_RUNTIME_DIR=/run/user/1000",
		"XMODIFIERS=@im=ibus",
	}, got)
}

func TestSessionEnvForChildFallback(t *testing.T) {
	// An unresolvable user has no session to harvest, so the specification
	// default must still be supplied: the GUI dialogs don't open without
	// XDG_DATA_DIRS set.
	require.Equal(t,
		[]string{"XDG_DATA_DIRS=" + xdgDataDirsDefault},
		sessionEnvForChild("not-a-uid", "DISPLAY", ":0"))
}

func TestGetUserSessionEnvNoMatchingSession(t *testing.T) {
	// A resolvable user whose processes are all on other displays must not have
	// an unrelated environment harvested, e.g. that of an `ssh -X` session.
	_, err := getUserSessionEnv("0", "DISPLAY", "display-that-cannot-exist")
	require.Error(t, err)
}

func TestPreferLocalDisplay(t *testing.T) {
	testCases := []struct {
		name     string
		displays []string
		expected string
	}{
		{
			name:     "local only",
			displays: []string{":0"},
			expected: ":0",
		},
		{
			// A user with an `ssh -X` session open has processes on a forwarded
			// display; the desktop session is the one to launch into.
			name:     "forwarded found before local",
			displays: []string{"localhost:10.0", ":0"},
			expected: ":0",
		},
		{
			name:     "local found before forwarded",
			displays: []string{":0", "localhost:10.0"},
			expected: ":0",
		},
		{
			name:     "host qualified forwarded display",
			displays: []string{"myhost.example.com:10.0", ":1"},
			expected: ":1",
		},
		{
			name:     "local display with a screen",
			displays: []string{"localhost:10.0", ":0.0"},
			expected: ":0.0",
		},
		{
			// Fall back to whatever is available, as before.
			name:     "forwarded only",
			displays: []string{"localhost:11.0", "localhost:10.0"},
			expected: "localhost:11.0",
		},
		{
			name:     "none",
			displays: nil,
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, preferLocalDisplay(tc.displays))
		})
	}
}

func TestWithXDGDataDirs(t *testing.T) {
	testCases := []struct {
		name     string
		env      []string
		expected []string
	}{
		{
			name:     "absent",
			env:      []string{"LANG=en_US.UTF-8"},
			expected: []string{"LANG=en_US.UTF-8", "XDG_DATA_DIRS=" + xdgDataDirsDefault},
		},
		{
			name:     "nothing to start from",
			env:      nil,
			expected: []string{"XDG_DATA_DIRS=" + xdgDataDirsDefault},
		},
		{
			name:     "already present",
			env:      []string{"XDG_DATA_DIRS=/session/value"},
			expected: []string{"XDG_DATA_DIRS=/session/value"},
		},
		{
			// An empty value would leave GTK without the schemas it needs, so it
			// must not suppress the default.
			name:     "present but empty",
			env:      []string{"XDG_DATA_DIRS="},
			expected: []string{"XDG_DATA_DIRS=", "XDG_DATA_DIRS=" + xdgDataDirsDefault},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, withXDGDataDirs(tc.env))
		})
	}
}

func TestSudoArgs(t *testing.T) {
	testCases := []struct {
		name         string
		noLoginShell bool
		expected     []string
	}{
		{
			name:         "login shell",
			noLoginShell: false,
			expected:     []string{"-n", "-i", "-u", "alice", "-H"},
		},
		{
			// Callers that read the command's stdout must not get a login
			// shell: its startup files would write to the same stream.
			name:         "no login shell",
			noLoginShell: true,
			expected:     []string{"-n", "-u", "alice", "-H"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, sudoArgs("alice", tc.noLoginShell))
		})
	}
}
