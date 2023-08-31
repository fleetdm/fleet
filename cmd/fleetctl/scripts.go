package main

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"strings"

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
					return errors.New(fleet.ErrRunScriptHostNotFound)
				}
				var sce service.StatusCodeErr
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusForbidden {
						return errors.New(fleet.ErrRunScriptForbidden)
					}
				}
				return err
			}

			if h.Status != fleet.StatusOnline {
				return errors.New(fleet.ErrRunScriptHostOffline)
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
{{ if .ErrorMsg }}Error: {{ .ErrorMsg }}{{else}}Exit code: {{ .ExitCode }} ({{ .ExitMessage }}){{end}}
{{ if (not .HideOutput) }}
Output{{ .BeforeTimeout }}:

-------------------------------------------------------------------------------------

{{ .Output }}

-------------------------------------------------------------------------------------{{end}}
`))

	data := struct {
		BeforeTimeout string
		ErrorMsg      string
		ExitCode      int64
		ExitMessage   string
		HideOutput    bool
		Output        string
	}{
		ExitCode:    res.ExitCode.Int64,
		ExitMessage: "Script failed.",
	}

	switch res.ExitCode.Int64 {
	case -2:
		data.ErrorMsg = res.Message
		data.HideOutput = true
	case -1:
		data.BeforeTimeout = " before timeout"
		data.ErrorMsg = res.Message
	case 0:
		if res.ExitCode.Valid {
			data.ExitMessage = "Script ran successfully."
		}
	}

	if len(res.Output) >= 10000 {
		data.Output = "Fleet records the last 10,000 characters to prevent downtime.\n\n" + res.Output
	} else {
		data.Output = res.Output
	}

	return tmpl.Execute(c.App.Writer, data)
}

func validateScriptPath(path string) error {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return errors.New(fleet.ErrRunScriptInvalidType)
	}
	extension := parts[len(parts)-1]
	switch extension {
	case "sh", "ps1":
		return nil
	default:
		return errors.New(fleet.ErrRunScriptInvalidType)
	}
}
