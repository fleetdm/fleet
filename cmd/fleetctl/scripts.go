package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/briandowns/spinner"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/urfave/cli/v2"
)

// Helper function to convert a boolean to an integer
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func runScriptCommand() *cli.Command {
	return &cli.Command{
		Name:      "run-script",
		Aliases:   []string{"run_script"},
		Usage:     `Run a script on one host and get results back.`,
		UsageText: `fleetctl run-script [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "script-path",
				Usage:    "The path to the script.",
				Required: false,
			},
			&cli.StringFlag{
				Name:     "host",
				Usage:    "The host, specified by hostname, UUID, or serial number.",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "script-name",
				Usage:    "Name of saved script to run.",
				Required: false,
			},
			&cli.UintFlag{
				Name:     "team",
				Usage:    `Available in Fleet Premium. ID of the team that the saved script belongs to. 0 targets hosts assigned to “No team” (default: 0).`,
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "async",
				Usage:    `Queue the script and don't wait for the return.`,
				Required: false,
			},
			&cli.BoolFlag{
				Name:     "quiet",
				Usage:    `Suppress messages that are not the script output / error`,
				Required: false,
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

			appCfg, err := client.GetAppConfig()
			if err != nil {
				return err
			}

			if appCfg.ServerSettings.ScriptsDisabled {
				return errors.New(fleet.RunScriptScriptsDisabledGloballyErrMsg)
			}

			async := c.Bool("async")
			quiet := c.Bool("quiet")

			// Require 1 and only 1 of these 3 options
			path := c.String("script-path")
			name := c.String("script-name")
			args := c.Args().Len()

			notEmpty := boolToInt(path != "") + boolToInt(name != "") + boolToInt(args > 0)

			if notEmpty < 1 {
				return errors.New("One of '--script-path' or '--script-name' or '-- <contents>' must be specified.")
			}

			if notEmpty > 1 {
				return errors.New("Only one of '--script-path' or '--script-name' or '-- <contents>' is allowed.")
			}

			if path != "" {
				if err := validateScriptPath(path); err != nil {
					return err
				}
			}

			ident := c.String("host")
			h, err := client.HostByIdentifier(ident)
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					return errors.New(fleet.HostNotFoundErrMsg)
				}
				var sce fleet.ErrWithStatusCode
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

			var b []byte
			if path != "" || args > 0 {
				if path != "" {
					b, err = os.ReadFile(path)
					if err != nil {
						return err
					}
				}

				if args > 0 {
					commandString := strings.Join(c.Args().Slice(), " ")
					b = []byte(commandString)
				}

				// validate script contents with isSavedScript flag set to false so that we check
				// for the shorter
				if err := fleet.ValidateHostScriptContents(string(b), false); err != nil {
					if err.Error() == fleet.RunScripUnsavedMaxLenErrMsg {
						return errors.New("Script is too large. Script referenced by '--script-path' is limited to 10,000 characters. To run larger script save it to Fleet and use '--script-name'.")
					}
					return err
				}
			}

			if async {
				res, err := client.RunHostScriptAsync(h.ID, b, name, c.Uint("team"))
				if err != nil {
					if strings.Contains(err.Error(), `Only one of 'script_contents' or 'team_id' is allowed`) {
						return errors.New("Only one of '--script-path' or '--team' is allowed.")
					}
					return err
				}
				fmt.Fprintf(c.App.Writer, "%s\n", res.ExecutionID)
				return nil
			}

			s := spinner.New(spinner.CharSets[24], 200*time.Millisecond)
			if !quiet {
				fmt.Println()
				s.Suffix = " Script is running or will run when the host comes online..."
				s.Start()
			}

			res, err := client.RunHostScriptSync(h.ID, b, name, c.Uint("team"))
			s.Stop()
			if err != nil {
				if strings.Contains(err.Error(), `Only one of 'script_contents' or 'team_id' is allowed`) {
					return errors.New("Only one of '--script-path' or '--team' is allowed.")
				}
				return err
			}

			if !quiet {
				if err := renderScriptResult(c, res); err != nil {
					return err
				}
			} else {
				fmt.Fprintf(c.App.Writer, "%s", res.Output)
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

	if len(res.Output) >= fleet.UnsavedScriptMaxRuneLen && utf8.RuneCountInString(res.Output) >= fleet.UnsavedScriptMaxRuneLen {
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
