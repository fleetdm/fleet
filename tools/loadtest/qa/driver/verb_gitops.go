package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// gitopsApply is a customer's CI applying their GitOps repo: each run makes a small real change (a few scripts,
// one macOS profile in a few fleets, agent options in a few fleets), then runs the repo's apply command. The repo
// and the command are the operator's (env vars in the spec), because a real repo holds private config.
//
// The edits are marker lines the next run replaces, so the repo never drifts: scripts get a trailing
// "# fleet-driver edit <time>" line, profiles a "<!-- fleet-driver edit <time> -->" comment (it changes the
// profile's checksum, so Fleet re-delivers it), and agent options alternate one option between two values.
//
// With labels, the driver also owns a labels file the repo's default.yml includes (labels: - path: ...): manual
// labels whose host lists (UUIDs) it changes a little every run, the GitOps route to edited_label. The other manual
// labels in the repo should have no hosts key, so their membership stays with whoever sets it.
type gitopsApply struct {
	repo      string
	command   string
	pauseFile string
	busyFile  string
	timeout   time.Duration

	scripts      int
	scriptsGlob  string
	profiles     int
	profilesGlob string
	agentOptions int
	globalAOFile string
	fleetsGlob   string
	aoKey        string
	aoValues     [2]string
	labels       int
	labelsFile   string
	labelsPrefix string
	labelHosts   int
	labelChange  int
}

type gitopsLabel struct {
	Name                string   `yaml:"name"`
	Description         string   `yaml:"description"`
	LabelMembershipType string   `yaml:"label_membership_type"`
	Hosts               []string `yaml:"hosts"`
}

func newGitopsApply(p *params) (verb, error) {
	v := &gitopsApply{}
	var err error
	str := func(key, def string) string {
		if err != nil {
			return ""
		}
		var s string
		s, err = p.str(key, def)
		return os.ExpandEnv(s)
	}
	num := func(key string, def int) int {
		if err != nil {
			return 0
		}
		var n int
		n, err = p.integer(key, def)
		return n
	}
	v.repo = str("repo", "")
	v.command = str("command", "")
	v.pauseFile = str("pause_file", "")
	v.busyFile = str("busy_file", "")
	v.scripts = num("scripts", 3)
	v.scriptsGlob = str("scripts_glob", "lib/*/scripts/*")
	v.profiles = num("profiles", 0)
	v.profilesGlob = str("profiles_glob", "lib/*/apple/*.mobileconfig")
	v.agentOptions = num("agent_options", 0)
	v.fleetsGlob = str("fleets_glob", "fleets/*.yml")
	v.globalAOFile = str("global_agent_options_file", "")
	v.aoKey = str("agent_option", "distributed_tls_max_attempts")
	vals := str("agent_option_values", "3,4")
	v.labels = num("labels", 0)
	v.labelsFile = str("labels_file", "labels/driver.yml")
	v.labelsPrefix = str("labels_prefix", "driver-gitops-label-")
	v.labelHosts = num("label_hosts", 200)
	v.labelChange = num("label_change", 2)
	if err != nil {
		return nil, err
	}
	if v.timeout, err = p.duration("timeout", 45*time.Minute); err != nil {
		return nil, err
	}
	a, b, ok := strings.Cut(vals, ",")
	if !ok || a == b {
		return nil, fmt.Errorf("with.agent_option_values %q: want two different values, e.g. 3,4", vals)
	}
	v.aoValues = [2]string{strings.TrimSpace(a), strings.TrimSpace(b)}
	if v.scripts < 0 || v.profiles < 0 || v.agentOptions < 0 || v.labels < 0 || v.labelHosts < 1 || v.labelChange < 1 {
		return nil, errors.New("with: edit counts must be >= 0, label_hosts and label_change >= 1")
	}
	return v, nil
}

func (v *gitopsApply) setup(context.Context, *env) error {
	if v.repo == "" || v.command == "" {
		return errors.New("with.repo and with.command are required (set the env vars the spec names)")
	}
	if st, err := os.Stat(v.repo); err != nil || !st.IsDir() {
		return fmt.Errorf("repo %s: not a directory", v.repo)
	}
	for _, g := range []string{v.scriptsGlob, v.profilesGlob, v.fleetsGlob} {
		if _, err := filepath.Glob(filepath.Join(v.repo, g)); err != nil {
			return fmt.Errorf("glob %q: %w", g, err)
		}
	}
	return nil
}

func (v *gitopsApply) run(ctx context.Context, e *env) result {
	if v.pauseFile != "" {
		if _, err := os.Stat(v.pauseFile); err == nil {
			return result{skip: "pause file " + filepath.Base(v.pauseFile) + " exists"}
		}
	}
	if v.busyFile != "" {
		// Claim the busy file atomically, so two applies never overlap (another driver, a person's run). A claim
		// older than the timeout is a crashed run's and is taken over.
		f, err := os.OpenFile(v.busyFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644) //nolint:gosec // an empty marker file
		if errors.Is(err, fs.ErrExist) {
			if st, serr := os.Stat(v.busyFile); serr == nil && time.Since(st.ModTime()) < v.timeout {
				return result{skip: "another apply is running (" + filepath.Base(v.busyFile) + ")"}
			}
			f, err = os.OpenFile(v.busyFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644) //nolint:gosec // an empty marker file
		}
		if err != nil {
			return result{err: err}
		}
		_ = f.Close()
		defer os.Remove(v.busyFile)
	}
	stamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	edits, err := v.edit(stamp, e.hosts)
	detail := map[string]any{"edits": edits}
	if err != nil {
		return result{err: fmt.Errorf("edit repo: %w", err), detail: detail}
	}

	cctx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "/bin/sh", "-c", v.command) //nolint:gosec // the operator's own apply command
	cmd.Dir = v.repo
	out, runErr := cmd.CombinedOutput()
	if e.stateDir != "" {
		_ = os.MkdirAll(e.stateDir, 0o700)
		_ = os.WriteFile(filepath.Join(e.stateDir, "gitops-last.log"), out, 0o600)
	}
	if runErr == nil {
		return result{status: 200, detail: detail}
	}
	tail := strings.TrimSpace(string(out))
	if len(tail) > 400 {
		tail = "…" + tail[len(tail)-400:]
	}
	// A gateway error is the server's; anything else (validation, a refused apply) is the repo's or the setup's,
	// and mustn't count toward the broken-actions trip.
	status := 400
	if strings.Contains(tail, "502") || strings.Contains(tail, "504") || strings.Contains(tail, "Bad Gateway") {
		status = 502
	}
	return result{status: status, err: fmt.Errorf("apply: %w: %s", runErr, tail), detail: detail}
}

var (
	scriptMarker  = regexp.MustCompile(`(?m)^# fleet-driver edit .*\n?`)
	profileMarker = regexp.MustCompile(`<!-- fleet-driver edit [^>]*-->\n?`)
	plistOpen     = regexp.MustCompile(`<plist[^>]*>\n?`)
)

// edit makes this run's changes and returns how many files (labels for the labels file) of each kind it touched.
func (v *gitopsApply) edit(stamp string, hosts *hostCache) (map[string]int, error) {
	edits := map[string]int{}

	scripts, err := v.pick(v.scriptsGlob, v.scripts, false)
	if err != nil {
		return edits, err
	}
	for _, f := range scripts {
		if err := rewrite(f, func(s string) string {
			s = scriptMarker.ReplaceAllString(s, "")
			if !strings.HasSuffix(s, "\n") {
				s += "\n"
			}
			return s + "# fleet-driver edit " + stamp + "\n"
		}); err != nil {
			return edits, err
		}
		edits["scripts"]++
	}

	// One profile in each of `profiles` different directories: one fleet's profile set each.
	profiles, err := v.pick(v.profilesGlob, v.profiles, true)
	if err != nil {
		return edits, err
	}
	for _, f := range profiles {
		if err := rewrite(f, func(s string) string {
			s = profileMarker.ReplaceAllString(s, "")
			loc := plistOpen.FindStringIndex(s)
			if loc == nil {
				return s
			}
			return s[:loc[1]] + "<!-- fleet-driver edit " + stamp + " -->\n" + s[loc[1]:]
		}); err != nil {
			return edits, err
		}
		edits["profiles"]++
	}

	// Fleet files' agent options change config without an activity (only the fleet agent options API writes
	// edited_agent_options); the global file's change writes one.
	fleets, err := v.pick(v.fleetsGlob, v.agentOptions, false)
	if err != nil {
		return edits, err
	}
	if v.globalAOFile != "" {
		fleets = append(fleets, filepath.Join(v.repo, v.globalAOFile))
	}
	ao := regexp.MustCompile(`(?m)^(\s*` + regexp.QuoteMeta(v.aoKey) + `:\s*)(\S+)\s*$`)
	for _, f := range fleets {
		changed := false
		if err := rewrite(f, func(s string) string {
			loc := ao.FindStringSubmatchIndex(s)
			if loc == nil {
				return s
			}
			next := v.aoValues[0]
			if s[loc[4]:loc[5]] == v.aoValues[0] {
				next = v.aoValues[1]
			}
			changed = true
			return s[:loc[4]] + next + s[loc[5]:]
		}); err != nil {
			return edits, err
		}
		if changed {
			edits["agent_options"]++
		}
	}

	if v.labels > 0 && hosts != nil {
		n, err := v.editLabels(hosts)
		if err != nil {
			return edits, err
		}
		edits["labels"] = n
	}
	return edits, nil
}

// editLabels keeps labels_file at `labels` manual labels of about label_hosts hosts each, and swaps label_change
// hosts out and in on each of them, so every one is edited.
func (v *gitopsApply) editLabels(hosts *hostCache) (int, error) {
	path := filepath.Join(v.repo, v.labelsFile)
	var cur []gitopsLabel
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, &cur); err != nil {
			return 0, fmt.Errorf("%s: %w", v.labelsFile, err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return 0, err
	}
	byName := make(map[string]gitopsLabel, len(cur))
	for _, l := range cur {
		byName[l.Name] = l
	}
	uuids := func(n int) []string {
		var out []string
		for _, h := range hosts.pick(n, false) {
			if h.UUID != "" {
				out = append(out, h.UUID)
			}
		}
		return out
	}
	labels := make([]gitopsLabel, 0, v.labels)
	for i := 1; i <= v.labels; i++ {
		name := fmt.Sprintf("%s%02d", v.labelsPrefix, i)
		l, ok := byName[name]
		if !ok {
			l = gitopsLabel{
				Name:                name,
				Description:         "Manual label the load-test activity driver edits through GitOps host lists.",
				LabelMembershipType: "manual",
				Hosts:               uuids(v.labelHosts),
			}
		} else {
			members := make(map[string]struct{}, len(l.Hosts))
			for _, h := range l.Hosts {
				members[h] = struct{}{}
			}
			for _, j := range randPerm(len(l.Hosts))[:min(v.labelChange, len(l.Hosts))] {
				delete(members, l.Hosts[j])
			}
			added := 0
			for _, u := range uuids(4 * v.labelChange) {
				if _, in := members[u]; !in && added < v.labelChange {
					members[u] = struct{}{}
					added++
				}
			}
			l.Hosts = slices.Sorted(maps.Keys(members))
		}
		labels = append(labels, l)
	}
	b, err := yaml.Marshal(labels)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // a repo folder
		return 0, err
	}
	return len(labels), os.WriteFile(path, b, 0o644) //nolint:gosec // repo content, no secrets
}

// pick returns n random files matching glob under the repo; with perDir, at most one per directory.
func (v *gitopsApply) pick(glob string, n int, perDir bool) ([]string, error) {
	if n == 0 {
		return nil, nil
	}
	files, err := filepath.Glob(filepath.Join(v.repo, glob))
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]struct{}{}
	for _, i := range randPerm(len(files)) {
		f := files[i]
		if perDir {
			if _, ok := seen[filepath.Dir(f)]; ok {
				continue
			}
			seen[filepath.Dir(f)] = struct{}{}
		}
		out = append(out, f)
		if len(out) == n {
			break
		}
	}
	return out, nil
}

func rewrite(path string, fn func(string) string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	s := fn(string(b))
	if s == string(b) {
		return nil
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), st.Mode().Perm())
}
