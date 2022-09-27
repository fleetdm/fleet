package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
)

var (
	rxOption = regexp.MustCompile(`\-\-(\w+)\s`)

	structTpl = template.Must(template.New("struct").Funcs(template.FuncMap{
		"camelCase": camelCaseOptionName,
	}).Parse(`
// NOTE: generate automatically with ` + "`go run ./tools/osquery-agent-options/main.go`" + `
type osqueryOptions struct { {{ range $name, $type := .Options }}
	{{camelCase $name}} {{$type}} ` + "`json:\"{{$name}}\"`" + `{{end}}
}

// NOTE: generate automatically with ` + "`go run ./tools/osquery-agent-options/main.go`" + `
type osqueryCommandLineFlags struct { {{ range $name, $type := .Flags }}
	{{camelCase $name}} {{$type}} ` + "`json:\"{{$name}}\"`" + `{{end}}
}
`))
)

type templateData struct {
	Options map[string]string
	Flags   map[string]string
}

func main() {
	// get the list of flags that are valid as configuration options
	b, err := exec.Command("osqueryd", "--help").Output()
	if err != nil {
		log.Fatalf("failed to run osqueryd --help: %v", err)
	}

	var optionsStarted, optionsSeen, optionsDone bool
	var optionNames, allNames []string

	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()

		if !optionsStarted {
			if strings.Contains(line, "osquery configuration options") {
				optionsStarted = true
				continue
			}
		}

		if line == "" {
			if optionsSeen {
				// we're done for options, empty line after an option has been seen
				optionsDone = true
			}
			continue
		}

		matches := rxOption.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		if optionsStarted && !optionsDone {
			optionsSeen = true
			optionNames = append(optionNames, matches[1])
		}
		allNames = append(allNames, matches[1])
	}
	if err := s.Err(); err != nil {
		log.Fatalf("failed to read osqueryd --help output: %v", err)
	}

	// find the data type for each option
	var optionTypes []struct {
		Name string
		Type string
	}
	b, err = exec.Command("osqueryi", "--json", "SELECT name, type FROM osquery_flags").Output()
	if err != nil {
		log.Fatalf("failed to run osqueryi query: %v", err)
	}
	if err := json.Unmarshal(b, &optionTypes); err != nil {
		log.Fatalf("failed to unmarshal osqueryi query output: %v", err)
	}

	// index the results by name
	allOptions := make(map[string]string, len(optionTypes))
	for _, nt := range optionTypes {
		allOptions[nt.Name] = nt.Type
	}

	// identify the valid config options
	validOptions := make(map[string]string, len(optionNames))
	for _, nm := range optionNames {
		ot, ok := allOptions[nm]
		if ok {
			validOptions[nm] = ot
		}
	}
	// identify the valid command-line flags
	validFlags := make(map[string]string, len(allNames))
	for _, nm := range allNames {
		ot, ok := allOptions[nm]
		if ok {
			validFlags[nm] = ot
		}
	}

	if err := structTpl.Execute(os.Stdout, templateData{Options: validOptions, Flags: validFlags}); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}
}

func camelCaseOptionName(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		parts[i] = strings.Title(p)
	}
	return strings.Join(parts, "")
}
