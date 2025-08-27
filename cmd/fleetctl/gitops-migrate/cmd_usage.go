package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/ansi"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/gitops-migrate/log"
)

const (
	cmdUsage = "usage"
	cmdHelp  = "help"
)

func cmdUsageExec(ctx context.Context, _ Args) error {
	showUsageAndExit(0, "")
	return nil
}

func showUsageAndExit(exitCode int, msg string, values ...any) {
	// Init the 'strings.Builder' to compose our error message and usage text.
	sb := new(strings.Builder)

	// Format and print the provided message, if we received one.
	if msg != "" {
		fmt.Fprint(sb, ansi.Red, "ERROR: ", ansi.Reset, msg, "\n")
	}

	// Generate the usage text.
	err := usageText(sb)
	if err != nil {
		log.Fatalf("Failed to build usage text: %s.", err)
	}

	// Print the usage text and exit with the provided exit code.
	fmt.Println(sb.String())
	os.Exit(exitCode)
}

type AppDetails struct {
	Name    string
	Command string
}

var appDetails = AppDetails{
	Name:    "Fleet GitOps Migration Tool",
	Command: "fm",
}

func colorizer(c string) func(string) string {
	return func(s string) string {
		return c + s + ansi.Reset
	}
}

var tmplFuncs = template.FuncMap{
	"green":   colorizer(ansi.BoldGreen),
	"magenta": colorizer(ansi.BoldMagenta),
	"cyan":    colorizer(ansi.BoldCyan),
	"yellow":  colorizer(ansi.BoldYellow),
	"pathJoin": func(parts ...string) string {
		return filepath.Join(parts...)
	},
}

var tmplUsageText = /* TODO: confirm this is going in 4.74 */ `
{{ $gitops := (green "Fleet GitOps") -}}
{{- $changeVer := (green "4.74(?)") -}}
Welcome to the {{ $gitops }} migration utility!

The purpose of this package is to assist with automated GitOps YAML file
transformations during the {{ $changeVer }} release.

{{ magenta ">> Commands" }}

   backup   Perform a backup of a target directory, output as a gzipped tarball.
   restore  Restore* a backup produced by the {{ green "backup" }} command.
   usage    Show this help text!

   {{ yellow "* NOTE:" }} Restore _will_ overwrite any existing files in the specified output
           directory!

{{ magenta ">> Flags" }}

   --from, -f  The {{ yellow "input" }} directory, or archive, path for all commands.
   --to,   -t  The {{ yellow "output" }} directory path for all commands.
   --debug     Enables additional log output during any command executions.
   --help      Show this help text!

{{ magenta ">> Examples" }}

   {{ $sampleSrc := (pathJoin "." "fleet" "gitops") -}}
   {{- $sampleDst := (pathJoin "." "fleet" "backups") -}}
   To perform a {{ $gitops }} migration where:
     - {{ $gitops }} files reside in directory: '{{ $sampleSrc }}'.

     {{ cyan "$" }} {{ magenta .Command }} migrate -f {{ $sampleSrc }}

   To perform a backup of {{ $gitops }} files, where:
     - {{ $gitops }} files reside in directory: '{{ $sampleSrc }}'.
     - We want to produce a backup gzipped tarball to '{{ $sampleDst }}'.

     {{ cyan "$" }} {{ magenta .Command }} backup -f {{ $sampleSrc }} -t {{ $sampleDst }}
` // TODO: More examples.

func usageText(w io.Writer) error {
	// Init the template, apply custom template functions.
	tmpl := template.New("usage")
	tmpl.Funcs(tmplFuncs)

	// Parse the template string.
	var err error
	tmpl, err = tmpl.Parse(tmplUsageText)
	if err != nil {
		return fmt.Errorf("failed to parse 'text/template' template: %w", err)
	}

	// Exec the template.
	sb := new(strings.Builder)
	sb.Grow(len(tmplUsageText))
	err = tmpl.Execute(sb, appDetails)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
