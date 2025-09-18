package fleetd_pacman_packages

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test_data.txt
var data string

func TestParsePacmanQiOutput(t *testing.T) {
	packages := parsePacmanQiOutput(data)

	require.Len(t, packages, 4)

	pkg := packages[0]
	require.Equal(t, "xxhash", pkg["name"])
	require.Equal(t, "0.8.3-1", pkg["version"])
	require.Equal(t, "Extremely fast non-cryptographic hash algorithm", pkg["description"])
	require.Equal(t, "x86_64", pkg["arch"])
	require.Equal(t, "https://cyan4973.github.io/xxHash/", pkg["url"])
	require.Equal(t, "GPL2  BSD", pkg["licenses"])
	require.Equal(t, "", pkg["groups"])
	require.Equal(t, "libxxhash.so=0-64", pkg["provides"])
	require.Equal(t, "glibc", pkg["depends_on"])
	require.Equal(t, "", pkg["optional_deps"])
	require.Equal(t, "debugedit  libplacebo", pkg["required_by"])
	require.Equal(t, "limine-snapper-sync", pkg["optional_for"])
	require.Equal(t, "", pkg["conflicts_with"])
	require.Equal(t, "", pkg["replaces"])
	require.Equal(t, "400.10 KiB", pkg["installed_size"])
	require.Equal(t, "Christian Hesse <eworm@archlinux.org>", pkg["packager"])
	require.Equal(t, "Mon 06 Jan 2025 02:26:25 AM EST", pkg["build_date"])
	require.Equal(t, "Tue 16 Sep 2025 01:55:37 PM EDT", pkg["install_date"])
	require.Equal(t, "Installed as a dependency for another package", pkg["install_reason"])
	require.Equal(t, "Yes", pkg["install_script"])
	require.Equal(t, "Signature", pkg["validated_by"])

	pkg = packages[1]
	require.Equal(t, "xz", pkg["name"])
	require.Equal(t, "5.8.1-1", pkg["version"])
	require.Equal(t, "Library and command line tools for XZ and LZMA compressed files", pkg["description"])
	require.Equal(t, "x86_64", pkg["arch"])
	require.Equal(t, "https://tukaani.org/xz/", pkg["url"])
	require.Equal(t, "GPL  LGPL  custom", pkg["licenses"])
	require.Equal(t, "", pkg["groups"])
	require.Equal(t, "liblzma.so=5-64", pkg["provides"])
	require.Equal(t, "sh", pkg["depends_on"])
	require.Equal(t, "", pkg["optional_deps"])
	require.Equal(t, "base  ffmpeg  file  imagemagick  karchive  kmod  libarchive  libelf  libtiff  libunwind  libxml2  libxmlb  libxslt  systemd  systemd-libs  zstd", pkg["required_by"])
	require.Equal(t, "mkinitcpio  python", pkg["optional_for"])
	require.Equal(t, "", pkg["conflicts_with"])
	require.Equal(t, "", pkg["replaces"])
	require.Equal(t, "2.92 MiB", pkg["installed_size"])
	require.Equal(t, "Levente Polyak <anthraxx@archlinux.org>", pkg["packager"])
	require.Equal(t, "Thu 03 Apr 2025 12:43:12 PM EDT", pkg["build_date"])
	require.Equal(t, "Tue 16 Sep 2025 01:55:31 PM EDT", pkg["install_date"])
	require.Equal(t, "Installed as a dependency for another package", pkg["install_reason"])
	require.Equal(t, "No", pkg["install_script"])
	require.Equal(t, "Signature", pkg["validated_by"])

	pkg = packages[2]
	require.Equal(t, "zeromq", pkg["name"])
	require.Equal(t, "4.3.5-2", pkg["version"])
	require.Equal(t, "Fast messaging system built on sockets. C and C++ bindings. aka 0MQ, ZMQ.", pkg["description"])
	require.Equal(t, "x86_64", pkg["arch"])
	require.Equal(t, "http://www.zeromq.org", pkg["url"])
	require.Equal(t, "MPL2", pkg["licenses"])
	require.Equal(t, "", pkg["groups"])
	require.Equal(t, "libzmq.so=5-64", pkg["provides"])
	require.Equal(t, "glibc  gnutls  gcc-libs  util-linux  libsodium  libpgm", pkg["depends_on"])
	require.Equal(t, "cppzmq: C++ binding for libzmq", pkg["optional_deps"])
	require.Equal(t, "ffmpeg", pkg["required_by"])
	require.Equal(t, "", pkg["optional_for"])
	require.Equal(t, "", pkg["conflicts_with"])
	require.Equal(t, "", pkg["replaces"])
	require.Equal(t, "3.05 MiB", pkg["installed_size"])
	require.Equal(t, "George Rawlinson <grawlinson@archlinux.org>", pkg["packager"])
	require.Equal(t, "Mon 23 Oct 2023 07:48:59 PM EDT", pkg["build_date"])
	require.Equal(t, "Tue 16 Sep 2025 01:57:33 PM EDT", pkg["install_date"])
	require.Equal(t, "Explicitly installed", pkg["install_reason"])
	require.Equal(t, "No", pkg["install_script"])
	require.Equal(t, "Signature", pkg["validated_by"])

	pkg = packages[3]
	require.Equal(t, "incomplete", pkg["name"])
	require.Equal(t, "3", pkg["version"])
	require.Equal(t, "", pkg["description"])
	require.Equal(t, "any", pkg["arch"])
	require.Equal(t, "", pkg["url"])
	require.Equal(t, "", pkg["licenses"])
	require.Equal(t, "", pkg["groups"])
	require.Equal(t, "", pkg["provides"])
	require.Equal(t, "", pkg["depends_on"])
	require.Equal(t, "", pkg["optional_deps"])
	require.Equal(t, "", pkg["required_by"])
	require.Equal(t, "", pkg["optional_for"])
	require.Equal(t, "", pkg["conflicts_with"])
	require.Equal(t, "", pkg["replaces"])
	require.Equal(t, "", pkg["installed_size"])
	require.Equal(t, "", pkg["packager"])
	require.Equal(t, "", pkg["build_date"])
	require.Equal(t, "", pkg["install_date"])
	require.Equal(t, "", pkg["install_reason"])
	require.Equal(t, "", pkg["install_script"])
	require.Equal(t, "", pkg["validated_by"])
}
