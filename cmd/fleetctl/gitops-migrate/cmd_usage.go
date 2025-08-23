package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
)

const cmdUsage = "usage"

func cmdUsageExec(ctx context.Context, _ Args) error {
	log := LoggerFromContext(ctx)

	text, err := usageText()
	if err != nil {
		log.Error("encountered error in usage text generation", "error", err)
		os.Exit(1)
	}

	fmt.Println(text)

	return nil
}

var appDetails = struct {
	Name    string
	Command string
}{
	Name:    "Fleet GitOps Migration Tool",
	Command: "fm",
}

const (
	ansiGreen  = "\x1b[32m"
	ansiPurple = "\x1b[35m"
	ansiReset  = "\x1b[0m"
)

func colorizer(c string) func(string) string {
	return func(s string) string {
		return c + s + ansiReset
	}
}

var tmplFuncs = template.FuncMap{
	"green":  colorizer(ansiGreen),
	"purple": colorizer(ansiPurple),
}

var tmplUsageText = `
Welcome to the {{ green "Fleet GitOps" }} migration utility!

The purpose of this package is to assist with automated GitOps YAML file
transformations during the {{ green "4.74" }}(?) release.

{{ purple ">> Commands" }}

   backup   Perform a backup of a target directory, output as a gzipped tarball.
   restore  Restore* a backup produced by the 'backup' command.
   migrate  Migrates a provided directory's {{ green "Fleet GitOps" }} files.

   * NOTE: Restore _will_ overwrite any existing files in the specified output
           directory!

{{ purple ">> Flags" }}

   --from, -f  The _input_ directory, or archive, path for all commands.
   --to,   -t  The _output_ directory path for all commands.
   --debug     Enables additional log output during any command executions.

{{ purple ">> Examples" }}

   To perform a {{ green "Fleet GitOps" }} migration where:
     - {{ green "Fleet GitOps" }} files reside in directory: './gitops'.

     $ {{ .Command }} migrate -f ./gitops
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
	err = tmpl.Execute(sb, appDetails)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}
