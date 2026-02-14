package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

/*
Examples:

# Default (safe): prints PASS cmds, does NOT execute them; runs osquery check
go run . -file ../../ee/cis/linux/cis-policy-queries.yml -ssh-user sharon -ssh-host 172.16.196.132 105

# Execute mode: runs PASS cmds over SSH, then runs osquery check
go run . -file ../../ee/cis/linux/cis-policy-queries.yml -ssh-user sharon -ssh-host 172.16.196.132 --execute-pass 105
*/

type Policy struct {
	Index   int
	Name    string
	Query   string
	Manual  bool
	PassCmd []string
	FailCmd []string
}

func main() {
	filePath := flag.String("file", "", "YAML file path")
	sshUser := flag.String("ssh-user", "", "SSH user")
	sshHost := flag.String("ssh-host", "", "SSH host")
	sshPort := flag.Int("ssh-port", 22, "SSH port")
	executePass := flag.Bool("execute-pass", false, "actually execute PASS command(s) over SSH (default: false)")
	flag.Parse()

	if *filePath == "" || *sshUser == "" || *sshHost == "" {
		fmt.Println("Missing required flags.")
		os.Exit(1)
	}

	start, end, ranged, err := parseRange(flag.Args())
	exitIfErr(err)

	policies, err := parsePoliciesFromFile(*filePath)
	exitIfErr(err)

	toRun := applyRange(policies, start, end, ranged)
	if len(toRun) == 0 {
		fmt.Println("No policies matched.")
		return
	}

	for _, p := range toRun {
		fmt.Printf("[%d] %s\n", p.Index, p.Name)

		if p.Manual || len(p.PassCmd) == 0 {
			fmt.Println("i couldn't check this policy, continuing\n")
			continue
		}

		// Always show PASS commands
		fmt.Println("PASS command(s):")
		for _, cmd := range p.PassCmd {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()

		// Apply PASS commands only when explicitly requested
		passApplyFailed := false

		if *executePass {
			fmt.Println("Executing PASS command(s) on VM...")

			for _, cmdToRun := range p.PassCmd {
				fmt.Printf("About to run on VM: %s\n", cmdToRun)

				stdout, stderr, err := sshRun(*sshUser, *sshHost, *sshPort, cmdToRun)
				if err != nil {
					fmt.Printf("********** FAIL **********\n")
					fmt.Printf("PASS command failed: %v\n", err)
					if strings.TrimSpace(stderr) != "" {
						fmt.Printf("stderr:\n%s\n", strings.TrimSpace(stderr))
					}
					if strings.TrimSpace(stdout) != "" {
						fmt.Printf("stdout:\n%s\n", strings.TrimSpace(stdout))
					}
					fmt.Println()
					passApplyFailed = true
					break
				}

				if strings.TrimSpace(stdout) != "" {
					fmt.Printf("stdout:\n%s\n", strings.TrimSpace(stdout))
				}
				if strings.TrimSpace(stderr) != "" {
					fmt.Printf("stderr:\n%s\n", strings.TrimSpace(stderr))
				}
				fmt.Println()
			}
		} else {
			// Dry-run marker only (no mutations)
			msg := fmt.Sprintf("Policy %d: (dry-run) would run PASS command(s)", p.Index)
			fmt.Printf("About to run: ssh %s@%s -p %d \"echo %s\"\n", *sshUser, *sshHost, *sshPort, msg)

			stdout, stderr, err := sshRun(*sshUser, *sshHost, *sshPort, "echo "+shellQuote(msg))
			if err != nil {
				fmt.Printf("SSH ERROR: %v\n", err)
				if strings.TrimSpace(stderr) != "" {
					fmt.Printf("SSH STDERR: %s\n", strings.TrimSpace(stderr))
				}
				if strings.TrimSpace(stdout) != "" {
					fmt.Printf("SSH STDOUT: %s\n", strings.TrimSpace(stdout))
				}
				fmt.Println()
				continue
			}
			fmt.Println()
		}

		// If PASS apply failed, don't run osquery check
		if passApplyFailed {
			continue
		}

		// Now run osquery check
		fmt.Println("Running osquery check…")
		pass, raw, stderr, err := sshOsquery(*sshUser, *sshHost, *sshPort, p.Query)
		if err != nil {
			fmt.Printf("********** FAIL **********\n")
			fmt.Printf("osquery error: %v\n", err)
			if strings.TrimSpace(stderr) != "" {
				fmt.Printf("osquery stderr:\n%s\n", strings.TrimSpace(stderr))
			}
			fmt.Println()
			continue
		}

		if pass {
			fmt.Printf("********** PASS **********\n\n")
		} else {
			fmt.Printf("********** FAIL **********\n")
			fmt.Printf("osquery returned no rows\n")
			fmt.Printf("osquery output: %s\n\n", strings.TrimSpace(raw))
		}
	}
}

/* ---------------- SSH ---------------- */

// sshRun executes a remote shell command via SSH.
// Returns stdout, stderr, and error (if exit code nonzero or ssh failed).
func sshRun(user, host string, port int, remoteCmd string) (string, string, error) {
	target := fmt.Sprintf("%s@%s", user, host)

	fmt.Printf(
		"\nAbout to run on target machine %s as user %s:\n  %s\n\n",
		host,
		user,
		remoteCmd,
	)

	args := []string{
		"-i", os.ExpandEnv("$HOME/.ssh/policyqa"),
		"-p", strconv.Itoa(port),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
	}

	// Only allocate a TTY when running sudo, so sudo can prompt for a password.
	if strings.Contains(remoteCmd, "sudo") {
		args = append(args, "-tt")
	}

	args = append(args, target, remoteCmd)

	cmd := exec.Command("ssh", args...)
	var stdout, stderr bytes.Buffer

	isSudo := strings.Contains(remoteCmd, "sudo")
	if isSudo {
		// Interactive: show prompts/output live, and still capture for logs.
		cmd.Stdin = os.Stdin
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	} else {
		// Non-interactive: capture output only.
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func sshOsquery(user, host string, port int, query string) (bool, string, string, error) {
	if strings.TrimSpace(query) == "" {
		return false, "", "", errors.New("empty osquery query")
	}

	remoteCmd := fmt.Sprintf("osqueryi --json %s", shellQuote(query))
	stdout, stderr, err := sshRun(user, host, port, remoteCmd)
	if err != nil {
		return false, "", stderr, err
	}

	out := strings.TrimSpace(stdout)

	// osquery --json returns [] for zero rows
	if out == "[]" || out == "" {
		return false, out, stderr, nil
	}

	return true, out, stderr, nil
}

func shellQuote(s string) string {
	// Safe single-quote quoting for sh:
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

/* ---------------- Parsing policies from raw text ---------------- */

var (
	reDocSep = regexp.MustCompile(`^\s*---\s*$`)
	reKind   = regexp.MustCompile(`^\s*kind:\s*policy\s*$`)
)

func parsePoliciesFromFile(path string) ([]Policy, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 20*1024*1024)

	var (
		docLines []string
		out      []Policy
	)

	flush := func() error {
		if len(docLines) == 0 {
			return nil
		}
		p, ok := parsePolicyDoc(docLines)
		if ok {
			p.Index = len(out) + 1
			out = append(out, p)
		}
		docLines = nil
		return nil
	}

	for sc.Scan() {
		line := sc.Text()
		if reDocSep.MatchString(line) {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}
		docLines = append(docLines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}

	return out, nil
}

func parsePolicyDoc(lines []string) (Policy, bool) {
	isPolicy := false
	for _, l := range lines {
		if reKind.MatchString(l) {
			isPolicy = true
			break
		}
	}
	if !isPolicy {
		return Policy{}, false
	}

	name := extractSpecName(lines)
	query := extractQueryBlock(lines)

	comments := extractCommentText(lines)
	passBlock := extractBlock(comments, "pass (run):")
	failBlock := extractBlock(comments, "fail (run):")

	passCmd := distillCommands(passBlock)
	failCmd := distillCommands(failBlock)

	manual := detectManual(lines, name)

	return Policy{
		Name:    nonEmpty(name, "<missing spec.name>"),
		Query:   strings.TrimSpace(query),
		Manual:  manual,
		PassCmd: passCmd,
		FailCmd: failCmd,
	}, true
}

func extractSpecName(lines []string) string {
	inSpec := false
	specIndent := -1

	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" || isCommentLine(raw) {
			continue
		}

		indent := countIndent(raw)
		trim := strings.TrimSpace(raw)

		if trim == "spec:" {
			inSpec = true
			specIndent = indent
			continue
		}
		if !inSpec {
			continue
		}
		if indent <= specIndent {
			inSpec = false
			specIndent = -1
			continue
		}

		if strings.HasPrefix(strings.TrimSpace(raw), "name:") {
			v := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "name:"))
			return strings.Trim(v, `"'`)
		}
	}
	return ""
}

func extractQueryBlock(lines []string) string {
	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		if isCommentLine(raw) {
			continue
		}
		trim := strings.TrimSpace(raw)
		if trim == "query: |" {
			baseIndent := countIndent(raw)
			var b strings.Builder
			for j := i + 1; j < len(lines); j++ {
				l := lines[j]
				if strings.TrimSpace(l) != "" && countIndent(l) <= baseIndent {
					break
				}
				b.WriteString(strings.TrimRight(l, " \t"))
				b.WriteString("\n")
			}
			return b.String()
		}
	}
	return ""
}

func extractCommentText(lines []string) []string {
	var out []string
	for _, raw := range lines {
		s := strings.TrimLeft(raw, " \t")
		if strings.HasPrefix(s, "#") {
			s = strings.TrimPrefix(s, "#")
			s = strings.TrimLeft(s, " \t")
			out = append(out, s)
		}
	}
	return out
}

func extractBlock(commentLines []string, marker string) []string {
	marker = strings.ToLower(marker)

	start := -1
	for i, l := range commentLines {
		if strings.Contains(strings.ToLower(l), marker) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil
	}

	var out []string
	for i := start; i < len(commentLines); i++ {
		low := strings.ToLower(strings.TrimSpace(commentLines[i]))
		if strings.Contains(low, "pass (run):") ||
			strings.Contains(low, "fail (run):") ||
			strings.Contains(low, "expected:") ||
			strings.Contains(low, "where:") {
			break
		}
		if strings.TrimSpace(commentLines[i]) != "" {
			out = append(out, commentLines[i])
		}
	}
	return out
}

func distillCommands(lines []string) []string {
	var cmds []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		t = strings.TrimPrefix(t, "-")
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		// Keep as-is; PASS commands are meant to be run.
		cmds = append(cmds, t)
	}
	return cmds
}

func detectManual(lines []string, name string) bool {
	if strings.Contains(strings.ToLower(name), "manual") {
		return true
	}
	for _, raw := range lines {
		if isCommentLine(raw) {
			continue
		}
		if strings.Contains(strings.ToLower(strings.TrimSpace(raw)), "purpose: manual") {
			return true
		}
	}
	return false
}

/* ---------------- Range ---------------- */

func parseRange(args []string) (int, int, bool, error) {
	if len(args) == 0 {
		return 0, 0, false, nil
	}
	if len(args) == 1 {
		v, err := strconv.Atoi(args[0])
		if err != nil || v <= 0 {
			return 0, 0, false, errors.New("invalid index")
		}
		return v, v, true, nil
	}
	if len(args) == 2 {
		a, err1 := strconv.Atoi(args[0])
		b, err2 := strconv.Atoi(args[1])
		if err1 != nil || err2 != nil || a <= 0 || b <= 0 {
			return 0, 0, false, errors.New("invalid range")
		}
		if a > b {
			a, b = b, a
		}
		return a, b, true, nil
	}
	return 0, 0, false, errors.New("too many args")
}

func applyRange(policies []Policy, start, end int, ranged bool) []Policy {
	if !ranged {
		return policies
	}
	var out []Policy
	for _, p := range policies {
		if p.Index >= start && p.Index <= end {
			out = append(out, p)
		}
	}
	return out
}

/* ---------------- Helpers ---------------- */

func isCommentLine(s string) bool {
	t := strings.TrimLeft(s, " \t")
	return strings.HasPrefix(t, "#")
}

func countIndent(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += 4
			continue
		}
		break
	}
	return n
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func exitIfErr(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
