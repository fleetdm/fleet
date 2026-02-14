package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type policyDoc struct {
	Spec struct {
		Name  string `yaml:"name"`
		Query string `yaml:"query"`
	} `yaml:"spec"`
}

func main() {
	file := flag.String("file", "", "Path to CIS policy YAML file")
	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "usage: go run manual_run.go -file <policy yaml>")
		os.Exit(2)
	}

	b, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	docs := splitYAMLDocs(string(b))
	for _, rawDoc := range docs {
		rawDoc = strings.TrimSpace(rawDoc)
		if rawDoc == "" {
			continue
		}

		var pd policyDoc
		if err := yaml.Unmarshal([]byte(rawDoc), &pd); err != nil {
			// If a doc fails YAML parse (rare), still show separator so you can see it.
			fmt.Println("========================================")
			fmt.Printf("name: (failed to parse YAML doc: %v)\n\n", err)
			continue
		}

		passCmds, failCmds := extractQACommands(rawDoc)

		fmt.Println("========================================")
		fmt.Printf("name: %s\n\n", strings.TrimSpace(pd.Spec.Name))

		fmt.Println("---- run to pass ----")
		for _, c := range passCmds {
			fmt.Println(c)
		}
		fmt.Println()

		fmt.Println("---- run to fail ----")
		for _, c := range failCmds {
			fmt.Println(c)
		}
		fmt.Println()

		fmt.Println("---- osquery ----")
		fmt.Println(strings.TrimSpace(pd.Spec.Query))
		fmt.Println()
	}
}

// splitYAMLDocs splits a multi-document YAML file into raw document strings.
// Handles files that may or may not start with '---'.
func splitYAMLDocs(s string) []string {
	lines := strings.Split(s, "\n")

	var docs []string
	var cur []string

	flush := func() {
		if len(cur) == 0 {
			return
		}
		docs = append(docs, strings.Join(cur, "\n"))
		cur = nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			flush()
			continue
		}
		cur = append(cur, line)
	}
	flush()

	return docs
}

func extractQACommands(rawDoc string) (pass []string, fail []string) {
	lines := strings.Split(rawDoc, "\n")

	inQA := false
	mode := "" // "pass" or "fail"
	skippingHeredoc := false

	appendCmd := func(cmd string) {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			return
		}

		// Drop "(optional)" prefix but keep the command.
		cmd = strings.TrimPrefix(cmd, "(optional)")
		cmd = strings.TrimSpace(cmd)

		// Skip heredoc starts and everything until EOF line.
		if strings.Contains(cmd, "cat <<EOF") || strings.Contains(cmd, "<<EOF") {
			skippingHeredoc = true
			return
		}
		if skippingHeredoc {
			// End heredoc when we see an EOF line (commented or not).
			trim := strings.TrimSpace(cmd)
			trim = strings.Trim(trim, `"'`)
			if trim == "EOF" || strings.HasPrefix(trim, "EOF") {
				skippingHeredoc = false
			}
			return
		}

		// We only want actual runnable commands (no bullets, no notes).
		if strings.HasPrefix(cmd, "Where:") || strings.HasPrefix(cmd, "Expected:") {
			return
		}

		switch mode {
		case "pass":
			pass = append(pass, cmd)
		case "fail":
			fail = append(fail, cmd)
		}
	}

	for _, line := range lines {
		trim := strings.TrimSpace(line)

		// QA block marker
		if strings.HasPrefix(trim, "# QA Validation:") {
			inQA = true
			mode = ""
			skippingHeredoc = false
			continue
		}
		if !inQA {
			continue
		}

		// We only parse comment lines once we're in QA
		if !strings.HasPrefix(trim, "#") {
			// If we reach non-comment YAML content after QA started,
			// keep scanning; QA is typically contiguous comment lines,
			// but we won't assume it must be.
			continue
		}

		// Normalize comment text
		c := strings.TrimSpace(strings.TrimPrefix(trim, "#"))

		// Switch modes
		if strings.HasPrefix(c, "- PASS (run):") {
			mode = "pass"
			skippingHeredoc = false
			continue
		}
		if strings.HasPrefix(c, "- FAIL (run):") {
			mode = "fail"
			skippingHeredoc = false
			continue
		}
		if strings.HasPrefix(c, "- Expected:") {
			mode = ""
			skippingHeredoc = false
			continue
		}

		// Command lines are typically "- <cmd>" or "-   - <cmd>"
		// After stripping '#', we accept lines beginning with '-' then keep text after it.
		c = strings.TrimSpace(c)
		if !strings.HasPrefix(c, "-") {
			continue
		}
		c = strings.TrimSpace(strings.TrimPrefix(c, "-"))

		// Some QA blocks nest bullets: "-   - cmd"
		if strings.HasPrefix(c, "-") {
			c = strings.TrimSpace(strings.TrimPrefix(c, "-"))
		}

		// If we are not in pass/fail mode, ignore bullets (Where/Expected/etc.)
		if mode != "pass" && mode != "fail" {
			continue
		}

		appendCmd(c)
	}

	// Final cleanup: remove any accidental YAML comment fragments like "# ..."
	pass = cleanupCmdList(pass)
	fail = cleanupCmdList(fail)

	return pass, fail
}

func cleanupCmdList(cmds []string) []string {
	var out []string
	for _, c := range cmds {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		// Remove any leading "#" that slipped through
		c = strings.TrimSpace(strings.TrimPrefix(c, "#"))
		// Collapse internal CRLF artifacts if any
		c = strings.ReplaceAll(c, "\r", "")
		out = append(out, c)
	}
	// De-dupe consecutive duplicates (rare)
	var dedup []string
	var last string
	for _, c := range out {
		if c == last {
			continue
		}
		dedup = append(dedup, c)
		last = c
	}
	return dedup
}

// Optional helper: for debugging raw-doc splitting
func _debugDoc(doc string) string {
	var buf bytes.Buffer
	buf.WriteString("---- DOC START ----\n")
	buf.WriteString(doc)
	buf.WriteString("\n---- DOC END ----\n")
	return buf.String()
}
