package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "tuf-status"
	app.Usage = "CLI to query a Fleet TUF repository"
	app.Commands = []*cli.Command{
		channelVersionCommand(),
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
	}
}

var componentFileMap = map[string]map[string]string{
	"orbit": {
		"linux":         "orbit",
		"linux-arm64":   "orbit",
		"macos":         "orbit",
		"windows":       "orbit.exe",
		"windows-arm64": "orbit.exe",
	},
	"desktop": {
		"linux":         "desktop.tar.gz",
		"linux-arm64":   "desktop.tar.gz",
		"macos":         "desktop.app.tar.gz",
		"windows":       "fleet-desktop.exe",
		"windows-arm64": "fleet-desktop.exe",
	},
	"osqueryd": {
		"linux":         "osqueryd",
		"linux-arm64":   "osqueryd",
		"macos-app":     "osqueryd.app.tar.gz",
		"windows":       "osqueryd.exe",
		"windows-arm64": "osqueryd.exe",
	},
	"nudge": {
		"macos": "nudge.app.tar.gz",
	},
	"swiftDialog": {
		"macos": "swiftDialog.app.tar.gz",
	},
	"escrowBuddy": {
		"macos": "escrowBuddy.pkg",
	},
}

func channelVersionCommand() *cli.Command {
	var (
		channel    string
		tufURL     string
		components cli.StringSlice
		format     string
	)
	return &cli.Command{
		Name:  "channel-version",
		Usage: "Fetch display the version of components on a channel (JSON output)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "url",
				EnvVars:     []string{"URL"},
				Value:       "https://updates.fleetdm.com",
				Destination: &tufURL,
				Usage:       "URL of the TUF repository",
			},
			&cli.StringFlag{
				Name:        "channel",
				EnvVars:     []string{"TUF_STATUS_CHANNEL"},
				Value:       "stable",
				Destination: &channel,
				Usage:       "Channel name",
			},
			&cli.StringSliceFlag{
				Name:        "components",
				EnvVars:     []string{"TUF_STATUS_COMPONENTS"},
				Value:       cli.NewStringSlice("orbit", "desktop", "osqueryd", "nudge", "swiftDialog", "escrowBuddy"),
				Destination: &components,
				Usage:       "List of components",
			},
			&cli.StringFlag{
				Name:        "format",
				EnvVars:     []string{"TUF_STATUS_FORMAT"},
				Value:       "json",
				Destination: &format,
				Usage:       "Output format (json, markdown)",
			},
		},
		Action: func(c *cli.Context) error {
			if format != "json" && format != "markdown" {
				return errors.New("supported formats are: json, markdown")
			}

			var (
				foundComponents map[string]map[string]string // component -> OS -> sha512
				sha512Map       map[string][]string
				err             error
			)
			foundComponents, sha512Map, err = getComponents(tufURL, components.Value(), channel)
			if err != nil {
				return fmt.Errorf("get components: %w", err)
			}

			outputMap := make(map[string]map[string]string) // component -> OS -> version
			for component, osMap := range foundComponents {
				outputMap[component] = make(map[string]string)
				for os, sha512 := range osMap {
					versions := sha512Map[sha512]
					var maxPartsVersion string
					for _, version := range versions {
						if len(strings.Split(version, ".")) > len(strings.Split(maxPartsVersion, ".")) {
							maxPartsVersion = version
						}
					}
					if os == "macos-app" {
						os = "macos" // this is an implementation detail in TUF.
					}
					outputMap[component][os] = maxPartsVersion
				}
			}

			if format == "json" {
				b, err := json.MarshalIndent(outputMap, "", "  ")
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", b)
			} else if format == "markdown" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Component\\OS", "macOS", "Linux", "Windows", "Linux (arm64)", "Windows (arm64)"})
				table.SetAutoFormatHeaders(false)
				table.SetCenterSeparator("|")
				table.SetHeaderLine(true)
				table.SetRowLine(false)
				table.SetTablePadding("\t")
				table.SetColumnSeparator("|")
				table.SetNoWhiteSpace(false)
				table.SetAutoWrapText(true)
				table.SetBorders(tablewriter.Border{
					Left:   true,
					Top:    false,
					Bottom: false,
					Right:  true,
				})
				var rows [][]string
				componentsInOrder := []string{"orbit", "desktop", "osqueryd", "nudge", "swiftDialog", "escrowBuddy"}
				setIfEmpty := func(m map[string]string, k string) string {
					v := m[k]
					if v == "" {
						v = "-"
					}
					return v
				}
				for _, component := range componentsInOrder {
					oss := outputMap[component]
					row := []string{component}
					for _, os := range []string{"macos", "linux", "windows", "linux-arm64", "windows-arm64"} {
						row = append(row, setIfEmpty(oss, os))
					}
					rows = append(rows, row)
				}
				table.AppendBulk(rows)
				table.Render()
			}

			return nil
		},
	}
}

// validVersion performs a loose parsing of a version string, because
// some components like nudge do not use semantic versioning (e.g. 1.1.10.81462).
func validVersion(version string) bool {
	parts := strings.Split(version, ".")
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func getComponents(tufURL string, components []string, channel string) (foundComponents map[string]map[string]string, sha512Map map[string][]string, err error) {
	res, err := http.Get(tufURL + "/targets.json") //nolint
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get /targets.json: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read /targets.json response: %w", err)
	}

	selectedComponents := make(map[string]struct{})
	for _, component := range components {
		selectedComponents[component] = struct{}{}
	}

	var targetsJSON map[string]interface{}
	if err := json.Unmarshal(body, &targetsJSON); err != nil {
		return nil, nil, fmt.Errorf("failed to parse the source targets.json file: %w", err)
	}

	signed_ := targetsJSON["signed"]
	if signed_ == nil {
		return nil, nil, errors.New("missing signed key in targets.json file")
	}
	signed, ok := signed_.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid signed key in targets.json file: %T, expected map", signed_)
	}
	targets_ := signed["targets"]
	if targets_ == nil {
		return nil, nil, errors.New("missing signed.targets key in targets.json file")
	}
	targets, ok := targets_.(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid signed.targets key in targets.json file: %T, expected map", targets_)
	}

	sha512Map = make(map[string][]string)
	foundComponents = make(map[string]map[string]string) // component -> OS -> sha512
	for target, metadata_ := range targets {
		parts := strings.Split(target, "/")
		if len(parts) != 4 {
			return nil, nil, fmt.Errorf("target %q: invalid number of parts, expected 4", target)
		}

		targetName := parts[0]
		platformPart := parts[1]
		channelPart := parts[2]
		executablePart := parts[3]

		if _, ok := selectedComponents[targetName]; !ok {
			continue
		}

		metadata, ok := metadata_.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("target: %q: invalid metadata field: %T, expected map", target, metadata_)
		}
		custom_ := metadata["custom"]
		if custom_ == nil {
			return nil, nil, fmt.Errorf("target: %q: missing custom field", target)
		}
		hashes_ := metadata["hashes"]
		if hashes_ == nil {
			return nil, nil, fmt.Errorf("target: %q: missing hashes field", target)
		}
		hashes, ok := hashes_.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("target: %q: invalid hashes field: %T", target, hashes_)
		}
		sha512_ := hashes["sha512"]
		if sha512_ == nil {
			return nil, nil, fmt.Errorf("target: %q: missing hashes.sha512 field", target)
		}
		hashSHA512, ok := sha512_.(string)
		if !ok {
			return nil, nil, fmt.Errorf("target: %q: invalid hashes.sha512 field: %T", target, sha512_)
		}
		if validVersion(channelPart) {
			sha512Map[hashSHA512] = append(sha512Map[hashSHA512], channelPart)
		}
		if channelPart != channel {
			continue
		}
		m, ok := componentFileMap[targetName]
		if !ok {
			continue
		}
		if v, ok := m[platformPart]; !ok || v != executablePart {
			continue
		}
		osMap := foundComponents[targetName]
		if osMap == nil {
			osMap = make(map[string]string)
		}
		osMap[platformPart] = hashSHA512
		foundComponents[targetName] = osMap
	}
	return foundComponents, sha512Map, nil
}
