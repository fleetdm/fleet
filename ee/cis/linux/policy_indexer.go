// policy_indexer.go
//
// Usage:
//   go run . -file cis-policy-queries.yml
//   go run . -file cis-policy-queries.yml 15 20
//
// Behavior:
// - Scans a multi-document YAML file (--- separated).
// - Counts *policies* (kind: policy) and prints: "<index>\t<name>".
// - If you pass 2 args (start end), it prints only policy indexes in [start,end] (1-based).

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type PolicyDoc struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       struct {
		Name string `yaml:"name"`
	} `yaml:"spec"`
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  %s -file <path> [start end]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  %s -file cis-policy-queries.yml\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -file cis-policy-queries.yml 15 20\n", os.Args[0])
}

func parseRange(args []string) (start, end int, ok bool, err error) {
	if len(args) == 0 {
		return 0, 0, false, nil
	}
	if len(args) == 1 {
		v, e := strconv.Atoi(args[0])
		if e != nil || v <= 0 {
			return 0, 0, false, fmt.Errorf("invalid index %q (must be positive int)", args[0])
		}
		return v, v, true, nil
	}
	if len(args) == 2 {
		a, e1 := strconv.Atoi(args[0])
		b, e2 := strconv.Atoi(args[1])
		if e1 != nil || e2 != nil || a <= 0 || b <= 0 {
			return 0, 0, false, fmt.Errorf("invalid range %q %q (must be positive ints)", args[0], args[1])
		}
		if a > b {
			a, b = b, a
		}
		return a, b, true, nil
	}
	return 0, 0, false, fmt.Errorf("too many args: expected 0, 1, or 2 indexes")
}

func main() {
	filePath := flag.String("file", "cis-policy-queries.yml", "path to YAML file (multi-doc, --- separated)")
	flag.Usage = usage
	flag.Parse()

	start, end, ranged, err := parseRange(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		usage()
		os.Exit(2)
	}

	f, err := os.Open(*filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		os.Exit(1)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)

	policyIndex := 0
	for {
		var doc PolicyDoc
		err := dec.Decode(&doc)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "YAML decode error:", err)
			os.Exit(1)
		}

		if doc.Kind != "policy" {
			continue
		}

		policyIndex++

		if ranged && (policyIndex < start || policyIndex > end) {
			continue
		}

		name := doc.Spec.Name
		if name == "" {
			name = "<missing spec.name>"
		}
		fmt.Printf("%d\t%s\n", policyIndex, name)
	}
}
