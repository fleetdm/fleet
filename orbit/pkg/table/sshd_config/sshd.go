package sshd_config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/rs/zerolog/log"
)

// sshdTimeout bounds each sshd invocation. A config dump is near-instant; this
// only guards against a hung process holding up the query.
const sshdTimeout = 15 * time.Second

// sshdPaths lists where sshd is installed on this platform. $PATH is never
// searched, and a candidate is only run if resolveTrusted accepts it: osquery
// runs this table as root, so a user who could replace the binary, or repoint a
// symlink to it, could run code as root.
var sshdPaths = map[string][]string{
	"darwin": {"/usr/sbin/sshd"},
	// /run/current-system/sw/bin is where NixOS links system packages.
	"linux": {"/usr/sbin/sshd", "/usr/bin/sshd", "/sbin/sshd", "/usr/local/sbin/sshd", "/run/current-system/sw/bin/sshd"},
}[runtime.GOOS]

// resolveSSHD returns the sshd at candidate if it is safe to run as root. It is
// a variable so tests can run fake sshd scripts, which aren't owned by root.
var resolveSSHD = func(candidate string) (sshdBinary, error) {
	path, info, err := resolveTrusted(candidate)
	if err != nil {
		return sshdBinary{}, err
	}
	return newSSHDBinary(path, info)
}

// unsupportedFlag matches the error sshd prints when it predates -G (added in
// OpenSSH 9.3). The wording comes from whichever getopt sshd was linked
// against: BSD libc says "illegal", glibc "invalid", OpenSSH's bundled getopt
// "unknown" (with no program name, ending in \r\n), and musl "unrecognized".
// sshd follows it with its usage text, which is matched as well in case a
// getopt words it some other way. Both are anchored to the start of a line so
// that a connection spec sshd echoes back ("Invalid test mode specification
// ...") can't match.
var unsupportedFlag = regexp.MustCompile(`(?im)^([^\s:]*: )?(illegal|invalid|unknown|unrecognized) option(:| --) '?G'?\r?$|^usage: sshd`)

// lacksG holds the sshdBinary.id of each binary that rejected -G, so later
// dumps go straight to -T instead of failing -G first every time. Only a run
// without a connection spec sets it, since a spec is query input that sshd may
// echo into stderr.
var lacksG sync.Map

type sshdBinary struct {
	// path has symlinks resolved, so the file that was checked is the one run.
	path string
	// id changes when the binary is replaced, such as by a package upgrade.
	id string
}

type configEntry struct {
	Keyword string
	Value   string
}

// findSSHD returns the first installed sshd that resolveSSHD accepts.
func findSSHD() (sshdBinary, bool) {
	for _, candidate := range sshdPaths {
		sshd, err := resolveSSHD(candidate)
		if err == nil {
			return sshd, true
		}
		if !errors.Is(err, fs.ErrNotExist) {
			log.Debug().Err(err).Msg("skipping sshd candidate")
		}
	}
	return sshdBinary{}, false
}

func newSSHDBinary(path string, info fs.FileInfo) (sshdBinary, error) {
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return sshdBinary{}, fmt.Errorf("%s: not an executable file", path)
	}
	return sshdBinary{
		path: path,
		id:   fmt.Sprintf("%s:%d:%d", path, info.Size(), info.ModTime().UnixNano()),
	}, nil
}

// dumpConfig returns sshd's effective configuration. It prefers -G, which only
// parses the configuration, and falls back to -T on sshd builds too old to have
// -G. -T prints the same dump but also runs sshd's startup checks, so it fails
// on a host whose host keys haven't been generated yet, and on Debian and
// Ubuntu while sshd is stopped (no /run/sshd).
//
// A connection spec needs -T even when -G is available: despite the man page,
// every sshd from 9.3 on rejects -C unless -T is also given. With both flags,
// -G still dumps the configuration and exits before -T loads any host keys.
func dumpConfig(ctx context.Context, sshd sshdBinary, spec string) (string, error) {
	if _, legacy := lacksG.Load(sshd.id); !legacy {
		args := []string{"-G"}
		if spec != "" {
			args = append(args, "-T", "-C", spec)
		}
		output, stderr, err := runSSHD(ctx, sshd.path, args)
		if err == nil {
			return output, nil
		}
		if !unsupportedFlag.MatchString(stderr) {
			return "", sshdError(sshd.path, args, stderr, err)
		}
		if spec == "" {
			lacksG.Store(sshd.id, struct{}{})
		}
	}

	args := []string{"-T"}
	if spec != "" {
		args = append(args, "-C", spec)
	}
	output, stderr, err := runSSHD(ctx, sshd.path, args)
	if err != nil {
		return "", sshdError(sshd.path, args, stderr, err)
	}
	return output, nil
}

func runSSHD(ctx context.Context, sshdPath string, args []string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, sshdTimeout)
	defer cancel()

	log.Debug().Str("path", sshdPath).Strs("args", args).Msg("running sshd for sshd_config table")

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, sshdPath, args...)
	// sshd needs no environment to dump its configuration; don't pass it orbit's.
	cmd.Env = []string{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// maxErrorDetail caps each part of a query error, in runes.
const maxErrorDetail = 512

// sshdError reports what sshd printed, since the exit status alone ("exit
// status 255") says nothing about which line of the configuration is wrong.
// The command and sshd's output can both contain the query's connection spec,
// so they are sanitized before they reach query results and logs.
func sshdError(sshdPath string, args []string, stderr string, err error) error {
	command := sanitizeDetail(sshdPath + " " + strings.Join(args, " "))
	if msg := sanitizeDetail(stderr); msg != "" {
		return fmt.Errorf("%s: %s", command, msg)
	}
	return fmt.Errorf("%s: %w", command, err)
}

// sanitizeDetail replaces non-printable characters such as terminal escapes
// with '?', collapses whitespace including newlines to single spaces, and
// truncates to maxErrorDetail runes.
func sanitizeDetail(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPrint(r) {
			return r
		}
		return '?'
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	if runes := []rune(s); len(runes) > maxErrorDetail {
		s = string(runes[:maxErrorDetail]) + "..."
	}
	return s
}

// parseConfig splits sshd's "keyword value" dump into entries, in the order
// sshd printed them. Keywords that may repeat (HostKey, ListenAddress,
// AcceptEnv, Subsystem, ...) produce one entry per line.
//
// Keywords are lowercased: sshd printed them lowercase until OpenSSH 10.4
// switched to mixed case, and lowercasing keeps queries working on both.
func parseConfig(output string) []configEntry {
	var entries []configEntry
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		keyword, value, _ := strings.Cut(line, " ")
		entries = append(entries, configEntry{
			Keyword: strings.ToLower(keyword),
			Value:   strings.TrimSpace(value),
		})
	}
	return entries
}
