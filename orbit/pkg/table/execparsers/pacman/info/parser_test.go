package pacman_info

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/pacman_info.txt
var pacman_info []byte

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name     string
		input    []byte
		expected []map[string]string
	}{
		{
			name:     "empty input",
			expected: make([]map[string]string, 0),
		},
		{
			name:  "malformed input",
			input: []byte("\n\t blah: fjiun3\n \n736 : \"foob\nName\t\t   ;  \n\nName:\ttester\nGroups  :  tee\nVersion: 1.0\nDescription     :     This is a test.\n%^82\nInstall Reason: No reason\nBuild Date\t:\tSun Nov 14 10:00:12 2021\nInstall Date		: 		Tue Jul 26 09:49:03 2022\n\n\n"),
			expected: []map[string]string{
				{
					"name":           "tester",
					"version":        "1.0",
					"description":    "This is a test.",
					"groups":         "tee",
					"build date":     "Sun Nov 14 10:00:12 2021",
					"install date":   "Tue Jul 26 09:49:03 2022",
					"install reason": "No reason",
				},
			},
		},
		{
			name:  "pacman_info",
			input: pacman_info,
			expected: []map[string]string{
				{
					"name":           "apr-util",
					"version":        "1.6.1-9",
					"description":    "The Apache Portable Runtime",
					"groups":         "None",
					"build date":     "Sat Nov 13 13:00:12 2021",
					"install date":   "Tue Jul 26 11:47:15 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "archlinux-keyring",
					"version":        "20220713-1",
					"description":    "Arch Linux PGP keyring",
					"groups":         "None",
					"build date":     "Wed Jul 13 02:57:11 2022",
					"install date":   "Tue Jul 26 11:46:30 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "base",
					"version":        "2-2",
					"description":    "Minimal package set to define a basic Arch Linux installation",
					"groups":         "None",
					"build date":     "Wed Nov 13 09:21:49 2019",
					"install date":   "Tue Jul 26 09:49:03 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "binutils",
					"version":        "2.38-6",
					"description":    "A set of programs to assemble and manipulate binary and object files",
					"groups":         "base-devel",
					"build date":     "Mon Jun 27 15:56:44 2022",
					"install date":   "Tue Jul 26 11:36:44 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "bzip2",
					"version":        "1.0.8-4",
					"description":    "A high-quality data compression program",
					"groups":         "None",
					"build date":     "Mon Nov  2 14:03:27 2020",
					"install date":   "Tue Jul 26 09:49:01 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "ca-certificates",
					"version":        "20210603-1",
					"description":    "Common CA certificates (default providers)",
					"groups":         "None",
					"build date":     "Thu Jun  3 13:36:41 2021",
					"install date":   "Tue Jul 26 09:49:02 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "cheese",
					"version":        "41.1-2",
					"description":    "Take photos and videos with your webcam, with fun graphical effects",
					"groups":         "gnome",
					"build date":     "Thu Feb 10 15:35:37 2022",
					"install date":   "Tue Jul 26 11:47:09 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "coreutils",
					"version":        "9.1-1",
					"description":    "The basic file, shell and text manipulation utilities of the GNU operating system",
					"groups":         "None",
					"build date":     "Sun Apr 17 12:21:13 2022",
					"install date":   "Tue Jul 26 09:49:01 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "gnome-shell",
					"version":        "1:42.3.1-1",
					"description":    "Next generation desktop shell",
					"groups":         "gnome",
					"build date":     "Mon Jul  4 17:01:26 2022",
					"install date":   "Tue Jul 26 11:47:11 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "gnome-software",
					"version":        "42.3-1",
					"description":    "GNOME Software Tools",
					"groups":         "gnome",
					"build date":     "Thu Jun 30 16:41:10 2022",
					"install date":   "Tue Jul 26 11:47:14 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "gnome-terminal",
					"version":        "3.44.1-1",
					"description":    "The GNOME Terminal Emulator",
					"groups":         "gnome",
					"build date":     "Sat May 28 11:26:02 2022",
					"install date":   "Tue Jul 26 11:47:14 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "linux",
					"version":        "5.18.14.arch1-1",
					"description":    "The Linux kernel and modules",
					"groups":         "None",
					"build date":     "Sat Jul 23 05:46:17 2022",
					"install date":   "Tue Jul 26 09:49:03 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "osquery",
					"version":        "5.3.0-2",
					"description":    "SQL powered operating system instrumentation, monitoring, and analytics",
					"groups":         "None",
					"build date":     "Thu Jun  2 13:11:23 2022",
					"install date":   "Tue Jul 26 12:19:07 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "pacman",
					"version":        "6.0.1-5",
					"description":    "A library-based package manager with dependency support",
					"groups":         "base-devel",
					"build date":     "Mon May  9 11:12:11 2022",
					"install date":   "Tue Jul 26 11:36:45 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "x264",
					"version":        "3:0.164.r3081.19856cc-2",
					"description":    "Open Source H264/AVC video encoder",
					"groups":         "None",
					"build date":     "Sun Mar  6 09:09:54 2022",
					"install date":   "Tue Jul 26 11:47:08 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "xorgproto",
					"version":        "2022.1-1",
					"description":    "combined X.Org X11 Protocol headers",
					"groups":         "None",
					"build date":     "Thu Apr 21 12:32:08 2022",
					"install date":   "Tue Jul 26 11:35:38 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "xz",
					"version":        "5.2.5-3",
					"description":    "Library and command line tools for XZ and LZMA compressed files",
					"groups":         "None",
					"build date":     "Thu Apr  7 13:44:20 2022",
					"install date":   "Tue Jul 26 09:49:01 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "yelp",
					"version":        "42.1-2",
					"description":    "Get help with GNOME",
					"groups":         "gnome",
					"build date":     "Sat Apr  2 17:30:17 2022",
					"install date":   "Tue Jul 26 11:47:14 2022",
					"install reason": "Explicitly installed",
				},
				{
					"name":           "zbar",
					"version":        "0.23.1-9",
					"description":    "Application and library for reading bar codes from various sources",
					"groups":         "None",
					"build date":     "Thu Dec  2 14:52:41 2021",
					"install date":   "Tue Jul 26 11:47:08 2022",
					"install reason": "Installed as a dependency for another package",
				},
				{
					"name":           "zlib",
					"version":        "1:1.2.12-2",
					"description":    "Compression library implementing the deflate compression method found in gzip and PKZIP",
					"groups":         "None",
					"build date":     "Sun Apr 24 00:19:33 2022",
					"install date":   "Tue Jul 26 09:49:01 2022",
					"install reason": "Installed as a dependency for another package",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			result, err := p.Parse(bytes.NewReader(tt.input))
			require.NoError(t, err, "unexpected error parsing input")

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}
