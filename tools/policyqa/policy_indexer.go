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

type PolicyReportRow struct {
	Index int
	Name  string

	Mode        string // "check" or "exec"
	Manual      bool
	NoRealQuery bool

	Step1   string
	Step2   string
	Step3   string
	Step4   string
	Overall string // PASS / FAIL / SKIP / MANUAL

	Notes []string
}

func isNoRealQuery(q string) bool {
	// Detect obvious “fake” queries used as placeholders.
	// Example: SELECT 1 WHERE 0
	re := regexp.MustCompile(`(?i)\bselect\s+1\s+where\s+0\b`)
	return re.MatchString(q)
}

func missingTableName(stderr string) (string, bool) {
	re := regexp.MustCompile(`no such table:\s*([A-Za-z0-9_]+)`)
	m := re.FindStringSubmatch(stderr)
	if len(m) == 2 {
		return m[1], true
	}
	return "", false
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.SplitN(s, "\n", 2)
	return strings.TrimSpace(parts[0])
}

func printReport(report []PolicyReportRow) {
	fmt.Println("\n================ REPORT ================")
	fmt.Printf("%-5s %-6s %-7s %-7s %-18s %-14s %-6s %s\n", "IDX", "MODE", "MANUAL", "FAKE", "STEP2", "STEP4", "OVER", "NAME")
	for _, r := range report {
		man := "no"
		if r.Manual {
			man = "yes"
		}
		fake := "no"
		if r.NoRealQuery {
			fake = "yes"
		}
		s2 := r.Step2
		if s2 == "" {
			s2 = "-"
		}
		s4 := r.Step4
		if s4 == "" {
			s4 = "-"
		}
		over := r.Overall
		if over == "" {
			over = "-"
		}
		fmt.Printf("%-5d %-6s %-7s %-7s %-18s %-14s %-6s %s\n", r.Index, r.Mode, man, fake, s2, s4, over, r.Name)
		if len(r.Notes) > 0 {
			fmt.Printf("      notes: %s\n", strings.Join(r.Notes, "; "))
		}
	}
	fmt.Println("========================================")
	fmt.Println()
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

	report := make([]PolicyReportRow, 0, len(toRun))

	for _, p := range toRun {
		row := PolicyReportRow{
			Index:       p.Index,
			Name:        p.Name,
			Manual:      p.Manual || len(p.PassCmd) == 0,
			NoRealQuery: isNoRealQuery(p.Query),
			Mode:        "check",
		}

		fmt.Printf("[%d] %s\n", p.Index, p.Name)

		if row.Manual {
			row.Overall = "MANUAL"
			if p.Manual {
				row.Notes = append(row.Notes, "manual policy")
			}
			if len(p.PassCmd) == 0 {
				row.Notes = append(row.Notes, "no PASS commands")
			}
			if row.NoRealQuery {
				row.Notes = append(row.Notes, "no real query (select 1 where 0)")
			}
			fmt.Println("i couldn't check this policy, continuing")
			fmt.Println()
			report = append(report, row)
			continue
		}

		// Always show PASS/FAIL commands
		fmt.Println("PASS command(s):")
		for _, cmd := range p.PassCmd {
			fmt.Printf("  %s\n", cmd)
		}
		fmt.Println()

		fmt.Println("FAIL command(s):")
		if len(p.FailCmd) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, cmd := range p.FailCmd {
				fmt.Printf("  %s\n", cmd)
			}
		}
		fmt.Println()

		if *executePass {
			row.Mode = "exec"

			// STEP 1: Execute FAIL commands
			if len(p.FailCmd) > 0 {
				fmt.Println("STEP 1: Executing FAIL command(s) on VM...")
				for _, cmdToRun := range p.FailCmd {
					fmt.Printf("About to run on VM: %s\n\n", cmdToRun)
					stdout, stderr, err := sshRun(*sshUser, *sshHost, *sshPort, cmdToRun)
					if err != nil {
						row.Step1 = fmt.Sprintf("ERR(%v)", err)
						if strings.TrimSpace(stderr) != "" {
							row.Notes = append(row.Notes, "fail-cmd stderr: "+firstLine(stderr))
						}
						fmt.Printf("********** FAIL **********\n")
						fmt.Printf("FAIL command failed: %v\n", err)
						if strings.TrimSpace(stderr) != "" {
							fmt.Printf("stderr:\n%s\n", strings.TrimSpace(stderr))
						}
						if strings.TrimSpace(stdout) != "" {
							fmt.Printf("stdout:\n%s\n", strings.TrimSpace(stdout))
						}
						fmt.Println()
						// Keep going; still attempt checks/PASS to avoid stopping the batch.
						break
					}
				}
			} else {
				row.Step1 = "SKIP(no fail cmds)"
				fmt.Println("STEP 1: No FAIL command(s) found (skipping FAIL setup).")
				fmt.Println()
			}

			// STEP 2: Run query expecting FAIL
			fmt.Println("STEP 2: Running osquery check (expect FAIL). If it fails, it is OK.")
			fmt.Printf("About to run query on target machine %s:\n%s\n\n", *sshHost, strings.TrimSpace(p.Query))

			pass, raw, stderr, err := sshOsquery(*sshUser, *sshHost, *sshPort, p.Query)
			if err != nil {
				if tbl, ok := missingTableName(stderr); ok {
					row.Step2 = "SKIP(missing table " + tbl + ")"
					row.Overall = "SKIP"
					row.Notes = append(row.Notes, "missing table "+tbl)
					fmt.Printf("********** SKIP **********\n")
					fmt.Printf("osquery missing table: %s\n\n", tbl)
					report = append(report, row)
					continue
				}
				row.Step2 = "ERR(osquery)"
				row.Overall = "FAIL"
				fmt.Printf("********** FAIL **********\n")
				fmt.Printf("osquery error: %v\n", err)
				if strings.TrimSpace(stderr) != "" {
					fmt.Printf("osquery stderr:\n%s\n", strings.TrimSpace(stderr))
				}
				fmt.Println()
				report = append(report, row)
				continue
			}

			if pass {
				row.Step2 = "UNEXPECTED PASS"
				row.Notes = append(row.Notes, "step2 unexpected pass after FAIL setup")
				fmt.Printf("❌ UNEXPECTED: query PASSED after FAIL setup (this is NOT OK)\n")
				fmt.Printf("osquery output: %s\n\n", strings.TrimSpace(raw))
			} else {
				row.Step2 = "OK(expected fail)"
				fmt.Printf("✅ Expected: query FAILED after FAIL setup (this is OK)\n")
				fmt.Printf("osquery output: %s\n\n", strings.TrimSpace(raw))
			}

			// STEP 3: Execute PASS commands
			fmt.Println("STEP 3: Executing PASS command(s) on VM...")
			passCmdFailed := false
			for _, cmdToRun := range p.PassCmd {
				fmt.Printf("About to run on VM: %s\n\n", cmdToRun)
				stdout, stderr, err := sshRun(*sshUser, *sshHost, *sshPort, cmdToRun)
				if err != nil {
					passCmdFailed = true
					row.Step3 = fmt.Sprintf("ERR(%v)", err)
					if strings.TrimSpace(stderr) != "" {
						row.Notes = append(row.Notes, "pass-cmd stderr: "+firstLine(stderr))
					}
					fmt.Printf("********** FAIL **********\n")
					fmt.Printf("PASS command failed: %v\n", err)
					if strings.TrimSpace(stderr) != "" {
						fmt.Printf("stderr:\n%s\n", strings.TrimSpace(stderr))
					}
					if strings.TrimSpace(stdout) != "" {
						fmt.Printf("stdout:\n%s\n", strings.TrimSpace(stdout))
					}
					fmt.Println()
					break
				}
			}
			if passCmdFailed {
				row.Overall = "FAIL"
				report = append(report, row)
				continue
			}

			// STEP 4: Run query again expecting PASS
			fmt.Println("STEP 4: Running osquery check again (expect PASS).")
			fmt.Printf("About to run query on target machine %s:\n%s\n\n", *sshHost, strings.TrimSpace(p.Query))

			pass, raw, stderr, err = sshOsquery(*sshUser, *sshHost, *sshPort, p.Query)
			if err != nil {
				if tbl, ok := missingTableName(stderr); ok {
					row.Step4 = "SKIP(missing table " + tbl + ")"
					row.Overall = "SKIP"
					row.Notes = append(row.Notes, "missing table "+tbl)
					fmt.Printf("********** SKIP **********\n")
					fmt.Printf("osquery missing table: %s\n\n", tbl)
					report = append(report, row)
					continue
				}
				row.Step4 = "ERR(osquery)"
				row.Overall = "FAIL"
				fmt.Printf("********** FAIL **********\n")
				fmt.Printf("osquery error: %v\n", err)
				if strings.TrimSpace(stderr) != "" {
					fmt.Printf("osquery stderr:\n%s\n", strings.TrimSpace(stderr))
				}
				fmt.Println()
				report = append(report, row)
				continue
			}

			if pass {
				row.Step4 = "OK(pass)"
				if row.Step2 == "UNEXPECTED PASS" {
					row.Overall = "FAIL"
				} else {
					row.Overall = "PASS"
				}
				fmt.Printf("********** PASS **********\n\n")
			} else {
				row.Step4 = "UNEXPECTED FAIL"
				row.Overall = "FAIL"
				fmt.Printf("********** FAIL **********\n")
				fmt.Printf("osquery returned no rows\n")
				fmt.Printf("osquery output: %s\n\n", strings.TrimSpace(raw))
			}

			if row.NoRealQuery {
				row.Notes = append(row.Notes, "no real query (select 1 where 0)")
			}

			report = append(report, row)
			continue
		}

		// Default (safe): prints PASS cmds; does not execute; runs osquery check once.
		fmt.Println("Running osquery check…")
		fmt.Printf("About to run query on target machine %s:\n%s\n\n", *sshHost, strings.TrimSpace(p.Query))

		pass, raw, stderr, err := sshOsquery(*sshUser, *sshHost, *sshPort, p.Query)
		if err != nil {
			if tbl, ok := missingTableName(stderr); ok {
				row.Overall = "SKIP"
				row.Notes = append(row.Notes, "missing table "+tbl)
				fmt.Printf("********** SKIP **********\n")
				fmt.Printf("osquery missing table: %s\n\n", tbl)
				report = append(report, row)
				continue
			}
			row.Overall = "FAIL"
			fmt.Printf("********** FAIL **********\n")
			fmt.Printf("osquery error: %v\n", err)
			if strings.TrimSpace(stderr) != "" {
				fmt.Printf("osquery stderr:\n%s\n", strings.TrimSpace(stderr))
			}
			fmt.Println()
			report = append(report, row)
			continue
		}

		if pass {
			row.Overall = "PASS"
			fmt.Printf("********** PASS **********\n\n")
		} else {
			row.Overall = "FAIL"
			fmt.Printf("********** FAIL **********\n")
			fmt.Printf("osquery returned no rows\n")
			fmt.Printf("osquery output: %s\n\n", strings.TrimSpace(raw))
		}
		report = append(report, row)
	}

	printReport(report)
}

/* ---------------- SSH ---------------- */

// sshRun executes a remote shell command via SSH.
// Returns stdout, stderr, and error (if exit code nonzero or ssh failed).
func sshRun(user, host string, port int, remoteCmd string) (string, string, error) {
	target := fmt.Sprintf("%s@%s", user, host)

	args := []string{
		"-i", os.ExpandEnv("$HOME/.ssh/policyqa"),
		"-p", strconv.Itoa(port),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
	}

	// Only allocate a TTY when running sudo, so sudo can prompt (if needed).
	isSudo := strings.Contains(remoteCmd, "sudo")
	if isSudo {
		args = append(args, "-tt")
	}

	args = append(args, target, remoteCmd)

	fmt.Printf("\nAbout to run on target machine %s as user %s:\n  %s\n\n", host, user, remoteCmd)

	cmd := exec.Command("ssh", args...)
	var stdout, stderr bytes.Buffer

	if isSudo {
		// Interactive: show prompts/output live, and still capture for logs.
		cmd.Stdin = os.Stdin
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	} else {
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

	// osquery --json returns an empty JSON array for zero rows.
	// Sometimes it includes whitespace/newlines inside the brackets.
	compact := strings.ReplaceAll(out, "\n", "")
	compact = strings.ReplaceAll(compact, "\r", "")
	compact = strings.ReplaceAll(compact, "\t", "")
	compact = strings.ReplaceAll(compact, " ", "")

	if compact == "[]" || compact == "" {
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
		// Some CIS docs include human notes like "(optional) ...". Skip them.
		if strings.HasPrefix(strings.ToLower(t), "(optional)") {
			continue
		}
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
