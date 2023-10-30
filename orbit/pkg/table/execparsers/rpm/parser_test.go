package rpm

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/rpm_info.txt
var rpm_info []byte

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
			input: []byte("\n\nNAme; fofo%&\n  release'     527g::\nname: tester\nVERSION:1.1.1\t\t\n   RelEase   :   5.el2\n\ndescription:\n\n\nThis is to test ^things.."),
			expected: []map[string]string{
				{
					"name":        "tester",
					"version":     "1.1.1",
					"release":     "5.el2",
					"description": "This is to test ^things..",
				},
			},
		},
		{
			name:  "rpm_info",
			input: rpm_info,
			expected: []map[string]string{
				{
					"name":         "apr-util",
					"version":      "1.5.2",
					"release":      "6.el7",
					"install date": "Mon 12 Dec 2022 11:59:35 PM MST",
					"group":        "System Environment/Libraries",
					"build date":   "Mon 09 Jun 2014 08:31:06 PM MDT",
					"summary":      "Apache Portable Runtime Utility library",
					"description":  "The mission of the Apache Portable Runtime (APR) is to provide a free library of C data structures and routines.  This library contains additional utility interfaces for APR; including support for XML, LDAP, database interfaces, URI parsing and more.",
				},
				{
					"name":         "autofs",
					"version":      "5.0.7",
					"release":      "116.el7_9",
					"install date": "Tue 13 Dec 2022 12:02:47 AM MST",
					"group":        "System Environment/Daemons",
					"build date":   "Tue 15 Dec 2020 09:27:45 AM MST",
					"summary":      "A tool for automatically mounting and unmounting filesystems",
					"description":  "autofs is a daemon which automatically mounts filesystems when you use them, and unmounts them later when you are not using them.  This can include network filesystems, CD-ROMs, floppies, and so forth.",
				},
				{
					"name":         "bind-libs",
					"version":      "9.11.4",
					"release":      "26.P2.el7_9.10",
					"install date": "Tue 13 Dec 2022 12:00:46 AM MST",
					"group":        "Unspecified",
					"build date":   "Tue 04 Oct 2022 01:09:32 AM MDT",
					"summary":      "Libraries used by the BIND DNS packages",
					"description":  "Contains heavyweight version of BIND suite libraries used by both named DNS server and utilities in bind-utils package.",
				},
				{
					"name":         "brave-browser",
					"version":      "1.45.133",
					"release":      "1",
					"install date": "Tue 20 Dec 2022 08:21:53 AM MST",
					"group":        "Applications/Internet",
					"build date":   "Thu 24 Nov 2022 04:45:49 PM MST",
					"summary":      "Brave Web Browser",
					"description":  "The web browser from Brave Browse faster by blocking ads and trackers that violate your privacy and cost you time and money.",
				},
				{
					"name":         "brave-keyring",
					"version":      "1.10",
					"release":      "1",
					"install date": "Tue 20 Dec 2022 08:21:42 AM MST",
					"group":        "Unspecified",
					"build date":   "Wed 18 May 2022 12:59:45 PM MDT",
					"summary":      "Brave Browser keyring and repository files",
					"description":  "The Brave keyring setup installs the keyring files necessary for validating packages. In the future it will install the yum.repos.d repository for for fetching the packages.",
				},
				{
					"name":         "firefox",
					"version":      "102.6.0",
					"release":      "1.el7.centos",
					"install date": "Tue 20 Dec 2022 08:39:11 AM MST",
					"group":        "Unspecified",
					"build date":   "Thu 15 Dec 2022 10:20:49 AM MST",
					"summary":      "Mozilla Firefox Web browser",
					"description":  "Mozilla Firefox is an open-source web browser, designed for standards compliance, performance and portability.",
				},
				{
					"name":         "java-1.8.0-openjdk",
					"version":      "1.8.0.352.b08",
					"release":      "2.el7_9",
					"install date": "Tue 13 Dec 2022 12:03:07 AM MST",
					"group":        "Development/Languages",
					"build date":   "Fri 21 Oct 2022 08:50:19 AM MDT",
					"summary":      "OpenJDK 8 Runtime Environment",
					"description":  "The OpenJDK 8 runtime environment.",
				},
				{
					"name":         "java-1.8.0-openjdk-headless",
					"version":      "1.8.0.352.b08",
					"release":      "2.el7_9",
					"install date": "Tue 13 Dec 2022 12:02:11 AM MST",
					"group":        "Development/Languages",
					"build date":   "Fri 21 Oct 2022 08:50:19 AM MDT",
					"summary":      "OpenJDK 8 Headless Runtime Environment",
					"description":  "The OpenJDK 8 runtime environment without audio and video support.",
				},
				{
					"name":         "openssl",
					"version":      "1.0.2k",
					"release":      "25.el7_9",
					"install date": "Tue 13 Dec 2022 12:00:55 AM MST",
					"group":        "System Environment/Libraries",
					"build date":   "Mon 28 Mar 2022 09:43:15 AM MDT",
					"summary":      "Utilities from the general purpose cryptography library with TLS implementation",
					"description":  "The OpenSSL toolkit provides support for secure communications between machines. OpenSSL includes a certificate management tool and shared libraries which provide various cryptographic algorithms and protocols.",
				},
				{
					"name":         "openssl-libs",
					"version":      "1.0.2k",
					"release":      "25.el7_9",
					"install date": "Tue 13 Dec 2022 12:00:01 AM MST",
					"group":        "System Environment/Libraries",
					"build date":   "Mon 28 Mar 2022 09:43:15 AM MDT",
					"summary":      "A general purpose cryptography library with TLS implementation",
					"description":  "OpenSSL is a toolkit for supporting cryptography. The openssl-libs package contains the libraries that are used by various applications which support cryptographic algorithms and protocols.",
				},
				{
					"name":         "osquery",
					"version":      "5.6.0",
					"release":      "1.linux",
					"install date": "Tue 13 Dec 2022 12:05:17 AM MST",
					"group":        "default",
					"build date":   "Mon 10 Oct 2022 11:04:49 AM MDT",
					"summary":      "osquery built using CMake",
					"description":  "osquery is an operating system instrumentation toolchain.",
				},
				{
					"name":         "perf",
					"version":      "3.10.0",
					"release":      "1160.81.1.el7",
					"install date": "Tue 20 Dec 2022 08:38:58 AM MST",
					"group":        "Development/System",
					"build date":   "Fri 16 Dec 2022 10:47:00 AM MST",
					"summary":      "Performance monitoring for the Linux kernel",
					"description":  "This package contains the perf tool, which enables performance monitoring of the Linux kernel.",
				},
				{
					"name":         "python",
					"version":      "2.7.5",
					"release":      "92.el7_9",
					"install date": "Tue 13 Dec 2022 12:00:03 AM MST",
					"group":        "Development/Languages",
					"build date":   "Tue 28 Jun 2022 09:55:39 AM MDT",
					"summary":      "An interpreted, interactive, object-oriented programming language",
					"description":  "Python is an interpreted, interactive, object-oriented programming language often compared to Tcl, Perl, Scheme or Java. Python includes modules, classes, exceptions, very high level dynamic data types and dynamic typing. Python supports interfaces to many system calls and libraries, as well as to various windowing systems (X11, Motif, Tk, Mac and MFC). Programmers can write new built-in modules for Python in C or C++. Python can be used as an extension language for applications that need a programmable interface. Note that documentation for Python is provided in the python-docs package. This package provides the \"python\" executable; most of the actual implementation is within the \"python-libs\" package.",
				},
				{
					"name":         "sudo",
					"version":      "1.8.23",
					"release":      "10.el7_9.2",
					"install date": "Tue 13 Dec 2022 12:05:13 AM MST",
					"group":        "Applications/System",
					"build date":   "Thu 14 Oct 2021 06:29:07 AM MDT",
					"summary":      "Allows restricted root access for specified users",
					"description":  "Sudo (superuser do) allows a system administrator to give certain users (or groups of users) the ability to run some (or all) commands as root while logging all commands and arguments. Sudo operates on a per-command basis.  It is not a replacement for the shell.  Features include: the ability to restrict what commands a user may run on a per-host basis, copious logging of each command (providing a clear audit trail of who did what), a configurable timeout of the sudo command, and the ability to use the same configuration file (sudoers) on many different machines.",
				},
				{
					"name":         "zlib",
					"version":      "1.2.7",
					"release":      "20.el7_9",
					"install date": "Mon 12 Dec 2022 11:59:14 PM MST",
					"group":        "System Environment/Libraries",
					"build date":   "Thu 12 May 2022 08:58:15 AM MDT",
					"summary":      "The compression and decompression library",
					"description":  "Zlib is a general-purpose, patent-free, lossless data compression library which is used by many different programs.",
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
