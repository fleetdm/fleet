//go:build linux

package go_packages

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePasswdHomeDirsValidUsers(t *testing.T) {
	input := strings.NewReader(
		"alice:x:1000:1000:Alice:/home/alice:/bin/bash\n" +
			"bob:x:1001:1001:Bob:/home/bob:/bin/zsh\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/alice", "/home/bob"}, dirs)
}

func TestParsePasswdHomeDirsFiltersSystemAccounts(t *testing.T) {
	input := strings.NewReader(
		"root:x:0:0:root:/root:/bin/bash\n" +
			"daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin\n" +
			"nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin\n" +
			"alice:x:1000:1000:Alice:/home/alice:/bin/bash\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	// nobody has uid 65534 >= 500 but has nologin shell, so filtered.
	// root and daemon have uid < 500, so filtered.
	require.Equal(t, []string{"/home/alice"}, dirs)
}

func TestParsePasswdHomeDirsFiltersNologinShells(t *testing.T) {
	input := strings.NewReader(
		"svc:x:1000:1000:Service:/home/svc:/usr/sbin/nologin\n" +
			"blocked:x:1001:1001:Blocked:/home/blocked:/bin/false\n" +
			"alice:x:1002:1002:Alice:/home/alice:/bin/bash\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/alice"}, dirs)
}

func TestParsePasswdHomeDirsFiltersInvalidHomes(t *testing.T) {
	input := strings.NewReader(
		"u1:x:1000:1000::/:/bin/bash\n" +
			"u2:x:1001:1001:::/bin/bash\n" +
			"u3:x:1002:1002::/dev/null:/bin/bash\n" +
			"alice:x:1003:1003:Alice:/home/alice:/bin/bash\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/alice"}, dirs)
}

func TestParsePasswdHomeDirsDeduplicates(t *testing.T) {
	input := strings.NewReader(
		"alice:x:1000:1000:Alice:/home/shared:/bin/bash\n" +
			"bob:x:1001:1001:Bob:/home/shared:/bin/bash\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/shared"}, dirs)
}

func TestParsePasswdHomeDirsMalformedLines(t *testing.T) {
	input := strings.NewReader(
		"short:entry\n" +
			"alice:x:1000:1000:Alice:/home/alice:/bin/bash\n" +
			"\n" +
			"no-fields\n",
	)
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Equal(t, []string{"/home/alice"}, dirs)
}

func TestParsePasswdHomeDirsEmpty(t *testing.T) {
	input := strings.NewReader("")
	dirs, err := parsePasswdHomeDirs(input)
	require.NoError(t, err)
	require.Nil(t, dirs)
}
