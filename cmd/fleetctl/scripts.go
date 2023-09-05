package main

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

func runScriptCommand() *cli.Command {
	return &cli.Command{
		Name:      "run-script",
		Aliases:   []string{"run_script"},
		Usage:     `Run a script on one host.`,
		UsageText: `fleetctl run-script [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "script-path",
				Usage:    "The path to the script.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "host",
				Usage:    "A host, specified by hostname, UUID, osquery host ID, or node key.",
				Required: true,
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			path := c.String("script-path")
			if err := validateScriptPath(path); err != nil {
				return err
			}

			ident := c.String("host")
			h, err := client.HostByIdentifier(ident)
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					return errors.New(fleet.RunScriptHostNotFoundErrMsg)
				}
				var sce service.StatusCodeErr
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusForbidden {
						return errors.New(fleet.RunScriptForbiddenErrMsg)
					}
				}
				return err
			}

			if h.Status != fleet.StatusOnline {
				return errors.New(fleet.RunScriptHostOfflineErrMsg)
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if err := fleet.ValidateHostScriptContents(string(b)); err != nil {
				return err
			}

			res, err := client.RunHostScriptSync(h.ID, b)
			if err != nil {
				return err
			}

			if err := renderScriptResult(c, res); err != nil {
				return err
			}

			return nil
		},
	}
}

func renderScriptResult(c *cli.Context, res *fleet.HostScriptResult) error {
	tmpl := template.Must(template.New("").Parse(`
{{ if .ErrorMsg -}}
Error: {{ .ErrorMsg }}
{{- else -}}
Exit code: {{ .ExitCode }} ({{ .ExitMessage }})
{{- end }}
{{ if .ShowOutput }}
Output {{- if .ExecTimeout }} before timeout {{- end }}:

-------------------------------------------------------------------------------------

{{ .Output }}

-------------------------------------------------------------------------------------
{{- end }}
`))

	data := struct {
		ExecTimeout bool
		ErrorMsg    string
		ExitCode    *int64
		ExitMessage string
		Output      string
		ShowOutput  bool
	}{
		ExitCode:    res.ExitCode,
		ExitMessage: "Script failed.",
		ShowOutput:  true,
	}

	switch {
	case res.ExitCode == nil:
		data.ErrorMsg = res.Message
	case *res.ExitCode == -2:
		data.ShowOutput = false
		data.ErrorMsg = res.Message
	case *res.ExitCode == -1:
		data.ExecTimeout = true
		data.ErrorMsg = res.Message
	case *res.ExitCode == 0:
		data.ExitMessage = "Script ran successfully."
	}

	if len(res.Output) >= fleet.MaxScriptRuneLen && utf8.RuneCountInString(res.Output) >= fleet.MaxScriptRuneLen {
		data.Output = "Fleet records the last 10,000 characters to prevent downtime.\n\n" + res.Output
	} else {
		data.Output = res.Output
	}

	return tmpl.Execute(c.App.Writer, data)
}

func validateScriptPath(path string) error {
	extension := filepath.Ext(path)
	if extension == ".sh" || extension == ".ps1" {
		return nil
	}
	return errors.New(fleet.RunScriptInvalidTypeErrMsg)
}
