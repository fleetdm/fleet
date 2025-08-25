package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	cmdUsage = "usage"
	cmdHelp  = "help"
)

func cmdUsageExec(ctx context.Context, _ Args) error {
	showUsageAndExit(0, "")
	panic("impossible")
}

func showUsageAndExit(exitCode int, msg string, values ...any) {
	// Generate the usage text.
	text, err := usageText()
	if err != nil {
		slog.Error("failed to build usage text", "error", err)
		os.Exit(exitCode)
	}

	// Format and print the provided message, if we received one.
	if msg != "" {
		msg = fmt.Sprintf(msg, values...)
		fmt.Print(ansiRed, "ERROR: ", ansiReset, msg, "\n")
	}

	// Print the usage text and exit with the provided exit code.
	fmt.Println(text)
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

const (
	ansiGreen   = "\x1b[1;32m"
	ansiYellow  = "\x1b[1;33m"
	ansiMagenta = "\x1b[1;35m"
	ansiCyan    = "\x1b[1;36m"
	ansiRed     = "\x1b[1;31m"
	ansiReset   = "\x1b[0m"
)

func colorizer(c string) func(string) string {
	return func(s string) string {
		return c + s + ansiReset
	}
}

var tmplFuncs = template.FuncMap{
	"green":   colorizer(ansiGreen),
	"magenta": colorizer(ansiMagenta),
	"cyan":    colorizer(ansiCyan),
	"yellow":  colorizer(ansiYellow),
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
   migrate  Migrates a provided directory's {{ $gitops }} files.
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

func usageText() (string, error) {
	// Init the template, apply custom template functions.
	tmpl := template.New("usage")
	tmpl.Funcs(tmplFuncs)

	// Parse the template string.
	var err error
	tmpl, err = tmpl.Parse(tmplUsageText)
	if err != nil {
		return "", fmt.Errorf("failed to parse 'text/template' template: %w", err)
	}

	// Exec the template.
	sb := new(strings.Builder)
	sb.Grow(len(tmplUsageText))
	err = tmpl.Execute(sb, appDetails)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}
