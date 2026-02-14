package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

/*
Usage:

go run . -file ../../ee/cis/linux/cis-policy-queries.yml -ssh-user sharon -ssh-host 172.16.196.132 105
go run . -file ../../ee/cis/linux/cis-policy-queries.yml -ssh-user sharon -ssh-host 172.16.196.132 15 20
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

		// Your requested behavior:
		// If no automated PASS command -> print message and continue
		if p.Manual || len(p.PassCmd) == 0 {
			fmt.Println("i couldn't check this policy, continuing\n")
			continue
		}

		fmt.Println("PASS command(s):")
		for _, cmd := range p.PassCmd {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()

		msg := fmt.Sprintf("Policy %d: i am running the pass command", p.Index)

		fmt.Printf("About to run: ssh %s@%s -p %d \"echo %s\"\n",
			*sshUser, *sshHost, *sshPort, msg)

		out, err := sshEcho(*sshUser, *sshHost, *sshPort, msg)
		if err != nil {
			fmt.Printf("SSH ERROR: %v\n\n", err)
			continue
		}
		fmt.Printf("SSH OK: %s\n\n", strings.TrimSpace(out))
	}
}

/* ---------------- SSH ---------------- */

func sshEcho(user, host string, port int, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errors.New("empty message")
	}
	target := fmt.Sprintf("%s@%s", user, host)

	args := []string{
		"-i", os.ExpandEnv("$HOME/.ssh/policyqa"),
		"-p", strconv.Itoa(port),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		target,
		"echo " + shellQuote(message),
	}

	cmd := exec.Command("ssh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}
	return stdout.String(), nil
}

func shellQuote(s string) string {
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
			p.Index = len(out) + 1 // policy index = count of policy docs
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
	// Only accept if kind: policy exists in doc
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
		// Leave spec when indentation goes back to <= spec indent
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
	// Find "query: |" and then capture all following lines that are more indented
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
				// stop when indentation returns to baseIndent or less and it's not blank
				if strings.TrimSpace(l) != "" && countIndent(l) <= baseIndent {
					break
				}
				// strip one indentation level (2 spaces typical), but keep relative formatting
				b.WriteString(strings.TrimRight(l, " \t"))
				b.WriteString("\n")
			}
			return b.String()
		}
	}
	return ""
}

func extractCommentText(lines []string) []string {
	// Keep only comment lines, stripping leading spaces and the leading "#"
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
		lc := strings.ToLower(t)
		if strings.HasPrefix(lc, "sudo ") ||
			strings.HasPrefix(lc, "modprobe ") ||
			strings.HasPrefix(lc, "rm ") ||
			strings.HasPrefix(lc, "bash ") ||
			strings.HasPrefix(lc, "chown ") ||
			strings.HasPrefix(lc, "chmod ") ||
			strings.Contains(t, "2>/dev/null") ||
			strings.Contains(t, " >/") ||
			strings.Contains(t, " && ") ||
			strings.Contains(t, " <<EOF") {
			cmds = append(cmds, t)
		}
	}
	return cmds
}

func detectManual(lines []string, name string) bool {
	// conservative: any "Manual" in name or a "purpose: Manual..." line
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
