package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

/*
Usage:

go run . -file cis-policy-queries.fixed.yml -ssh-user sharon -ssh-host 172.16.196.132
go run . -file cis-policy-queries.fixed.yml -ssh-user sharon -ssh-host 172.16.196.132 15
go run . -file cis-policy-queries.fixed.yml -ssh-user sharon -ssh-host 172.16.196.132 15 20
*/

type Policy struct {
	Index       int
	Name        string
	Description string
	Purpose     string
	Resolution  string
	Query       string
	Comments    string
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

	policies, err := loadPolicies(*filePath)
	exitIfErr(err)

	toRun := applyRange(policies, start, end, ranged)

	if len(toRun) == 0 {
		fmt.Println("No policies matched.")
		return
	}

	for _, p := range toRun {

		fmt.Printf("[%d] %s\n", p.Index, p.Name)

		passLines := extractSection(p.Comments, "pass (run):")
		passCmds := distillCommands(passLines)

		if isManual(p) || len(passCmds) == 0 {
			fmt.Println("i couldn't check this policy, continuing\n")
			continue
		}

		msg := fmt.Sprintf("Policy %d: i am running the pass command", p.Index)

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

/* ---------------- YAML ---------------- */

func loadPolicies(path string) ([]Policy, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)

	var out []Policy
	index := 0

	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if getMapString(&doc, "kind") != "policy" {
			continue
		}

		index++

		spec := getMapNode(&doc, "spec")

		p := Policy{
			Index:       index,
			Name:        getMapString(spec, "name"),
			Description: getMapString(spec, "description"),
			Purpose:     getMapString(spec, "purpose"),
			Resolution:  getMapString(spec, "resolution"),
			Query:       getMapString(spec, "query"),
			Comments:    gatherComments(&doc),
		}

		out = append(out, p)
	}

	return out, nil
}

func getMapNode(n *yaml.Node, key string) *yaml.Node {
	root := docRoot(n)
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

func getMapString(n *yaml.Node, key string) string {
	v := getMapNode(n, key)
	if v != nil && v.Kind == yaml.ScalarNode {
		return v.Value
	}
	return ""
}

func docRoot(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

func gatherComments(n *yaml.Node) string {
	var b strings.Builder
	var walk func(*yaml.Node)
	walk = func(x *yaml.Node) {
		if x == nil {
			return
		}
		root := docRoot(x)
		for _, c := range []string{root.HeadComment, root.LineComment, root.FootComment} {
			if strings.TrimSpace(c) != "" {
				b.WriteString(c + "\n")
			}
		}
		for _, child := range root.Content {
			walk(child)
		}
	}
	walk(n)
	return b.String()
}

/* ---------------- PASS extraction ---------------- */

func extractSection(commentText, marker string) []string {
	lines := splitLines(commentText)
	marker = strings.ToLower(marker)

	start := -1
	for i, l := range lines {
		if strings.Contains(strings.ToLower(l), marker) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return nil
	}

	var out []string
	for i := start; i < len(lines); i++ {
		l := lines[i]
		low := strings.ToLower(strings.TrimSpace(l))
		if strings.Contains(low, "fail (run):") ||
			strings.Contains(low, "expected:") ||
			strings.Contains(low, "where:") {
			break
		}
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func distillCommands(lines []string) []string {
	var cmds []string
	for _, l := range lines {
		t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "-"))
		if t == "" {
			continue
		}
		lc := strings.ToLower(t)

		if strings.HasPrefix(lc, "sudo ") ||
			strings.HasPrefix(lc, "modprobe ") ||
			strings.HasPrefix(lc, "rm ") ||
			strings.HasPrefix(lc, "bash ") ||
			strings.Contains(t, "2>/dev/null") ||
			strings.Contains(t, " >/") {
			cmds = append(cmds, t)
		}
	}
	return cmds
}

func isManual(p Policy) bool {
	all := strings.ToLower(p.Name + p.Description + p.Purpose + p.Resolution)
	return strings.Contains(all, "manual")
}

/* ---------------- Range ---------------- */

func parseRange(args []string) (int, int, bool, error) {
	if len(args) == 0 {
		return 0, 0, false, nil
	}
	if len(args) == 1 {
		v, err := strconv.Atoi(args[0])
		return v, v, true, err
	}
	a, err1 := strconv.Atoi(args[0])
	b, err2 := strconv.Atoi(args[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false, errors.New("invalid range")
	}
	if a > b {
		a, b = b, a
	}
	return a, b, true, nil
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

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func exitIfErr(err error) {
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
