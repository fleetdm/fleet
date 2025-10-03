package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/ansi"
	"github.com/fleetdm/fleet/v4/cmd/gitops-migrate/log"
)

const (
	cmdUsage = "usage"
	cmdHelp  = "help"
)

// output is the io.Writer used by the usage commands to output the help text
// once generated.
var output = io.Writer(os.Stderr)

func cmdUsageExec(_ context.Context, _ Args) error {
	showUsageAndExit(0, "")
	return nil
}

func showUsageAndExit(exitCode int, msg string, values ...any) {
	// Init the 'strings.Builder' to compose our error message and usage text.
	sb := new(strings.Builder)

	// Format and print the provided message, if we received one.
	if msg != "" {
		fmt.Fprint(sb, ansi.Red, "ERROR: ", ansi.Reset)
		fmt.Fprintf(sb, msg, values...)
		fmt.Fprintln(sb)
	}

	// Generate the usage text.
	err := usageText(sb)
	if err != nil {
		log.Fatalf("Failed to build usage text: %s.", err)
	}

	// Print the usage text and exit with the provided exit code.
	fmt.Fprintln(output, sb.String())
	os.Exit(exitCode)
}

type AppDetails struct {
	Name    string
	Command string
}

var appDetails = map[string]string{
	"Name":    "Fleet GitOps Migration Tool",
	"Command": "gitops-migrate",
}

var tmplFuncs = template.FuncMap{
	"green":    ansi.Colorizer(ansi.BoldGreen),
	"magenta":  ansi.Colorizer(ansi.BoldMagenta),
	"cyan":     ansi.Colorizer(ansi.BoldCyan),
	"yellow":   ansi.Colorizer(ansi.BoldYellow),
	"pathJoin": filepath.Join,
}

var tmplText = `
{{ $gitops := (green "Fleet GitOps") -}}
{{- $changeVer := (green "4.74") -}}
Welcome to the {{ $gitops }} migration utility!

The purpose of this package is to assist with automated GitOps YAML file
transformations during the {{ $changeVer }} release.

{{ magenta ">> Commands" }}

   format   Formats all {{ green "Fleet GitOps" }} YAML files!
   migrate  Migrates a provided directory's {{ $gitops }} files.
   backup   Perform a backup of a target directory, output as a gzipped tarball.
   restore  Restore* a backup produced by the {{ green "backup" }} command.
   usage    Show this help text (analogous to the {{ magenta "--help" }} flag)!

   {{ yellow "* NOTE:" }}  Restore _will_ overwrite any existing files with the same name in
            the specified output directory!

{{ magenta ">> Flags" }}

   --debug     Enables additional log output.
   --help      Show this help text (analogous to the {{ magenta "usage" }} command)!

{{ magenta ">> Examples" }}

   {{ $sampleSrc := (pathJoin "." "fleet_gitops") -}}
   {{- $sampleArchivePath := (pathJoin "." "fleet_backups") -}}
   To perform a {{ $gitops }} migration where:
     - {{ $gitops }} files reside in directory: '{{ $sampleSrc }}'.

     {{ cyan "$" }} {{ magenta .Command }} migrate {{ $sampleSrc }}

   To perform a backup of {{ $gitops }} files, where:
     - {{ $gitops }} files reside in directory: '{{ $sampleSrc }}'.
     - We want to produce a backup archive to '{{ $sampleArchivePath }}'.

     {{ cyan "$" }} {{ magenta .Command }} backup {{ $sampleSrc }} {{ $sampleArchivePath }}

   To restore a backup of {{ $gitops }} files, where:
     - The backup archive is at: '{{ $sampleArchivePath }}'.
     - We want to restore the backup archive to: '{{ $sampleSrc }}'.

     {{ cyan "$" }} {{ magenta .Command }} restore {{ $sampleArchivePath }} {{ $sampleSrc }}
`

func usageText(w io.Writer) error {
	// Init the template, apply custom template functions.
	tmpl := template.New("usage")
	tmpl.Funcs(tmplFuncs)

	// Parse the template string.
	var err error
	tmpl, err = tmpl.Parse(tmplText)
	if err != nil {
		return fmt.Errorf("failed to parse 'text/template' template: %w", err)
	}

	// Exec the template.
	err = tmpl.Execute(w, appDetails)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
