package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

type listBucketResult struct {
	XMLName     xml.Name  `xml:"ListBucketResult"`
	Text        string    `xml:",chardata"`
	Xmlns       string    `xml:"xmlns,attr"`
	Name        string    `xml:"Name"`
	Prefix      string    `xml:"Prefix"`
	Marker      string    `xml:"Marker"`
	MaxKeys     string    `xml:"MaxKeys"`
	IsTruncated string    `xml:"IsTruncated"`
	Contents    []content `xml:"Contents"`
}

type content struct {
	Text         string `xml:",chardata"`
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

func main() {
	app := cli.NewApp()
	app.Name = "tuf-status"
	app.Usage = "CLI to query a Fleet TUF repository hosted on AWS S3"
	app.Commands = []*cli.Command{
		keyFilterCommand(),
		channelVersionCommand(),
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
	}
}

func keyFilterCommand() *cli.Command {
	var (
		filter string
		tufURL string
	)
	return &cli.Command{
		Name:  "key-filter",
		Usage: "Fetch and filter the entries by the given value",
		UsageText: `- To filter all items on the edge channel use -filter="edge"
- To filter all items on version 1.3 including patches that run on Linux use -filter="linux/1.3.*"
- To filter Fleet Desktop items on 1.3.*, stable and edge that run on macOS use -filter="desktop/*.*/macos/(1.3.*|stable|edge)"
`,
		Flags: []cli.Flag{
			urlFlag(&tufURL),
			&cli.StringFlag{
				Name:        "filter",
				EnvVars:     []string{"VALUE"},
				Value:       "stable",
				Destination: &filter,
				Usage:       "Filter string value",
			},
		},
		Action: func(c *cli.Context) error {
			res, err := http.Get(tufURL) //nolint
			if err != nil {
				return err
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			var list listBucketResult
			if err := xml.Unmarshal(body, &list); err != nil {
				return err
			}

			if err := printTable(list.Contents, filter); err != nil {
				return err
			}
			return nil
		},
	}
}

func printTable(contents []content, filter string) error {
	data := [][]string{}
	regFilter, err := regexp.Compile(filter)
	if err != nil {
		return err
	}

	for _, content := range contents {
		if regFilter.MatchString(content.Key) {
			r := strings.Split(content.Key, "/")
			platform, version := r[2], r[3]
			data = append(data, []string{version, platform, content.Key, content.LastModified, byteCountSI(content.Size), content.ETag})
		}
	}

	// sort by version, platform, key
	sort.Slice(data, func(i, j int) bool {
		if data[i][0] != data[j][0] {
			return data[i][0] < data[j][0]
		}

		if data[i][1] != data[j][1] {
			return data[i][1] < data[j][1]
		}

		return data[i][2] < data[j][2]
	})

	fmt.Printf("\nResults filtered by \"%s\" and sorted by version, platform and key.\n\n", filter)
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"version", "platform", "key", "last modified", "size", "etag"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data)
	table.Render()
	return nil
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func channelVersionCommand() *cli.Command {
	componentFileMap := map[string]map[string]string{
		"orbit": {
			"linux":       "orbit",
			"linux-arm64": "orbit",
			"macos":       "orbit",
			"windows":     "orbit.exe",
		},
		"desktop": {
			"linux":       "desktop.tar.gz",
			"linux-arm64": "desktop.tar.gz",
			"macos":       "desktop.app.tar.gz",
			"windows":     "fleet-desktop.exe",
		},
		"osqueryd": {
			"linux":       "osqueryd",
			"linux-arm64": "osqueryd",
			"macos-app":   "osqueryd.app.tar.gz",
			"windows":     "osqueryd.exe",
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
			urlFlag(&tufURL),
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

			res, err := http.Get(tufURL) //nolint
			if err != nil {
				return err
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			selectedComponents := make(map[string]struct{})
			for _, component := range components.Value() {
				selectedComponents[component] = struct{}{}
			}

			var list listBucketResult
			if err := xml.Unmarshal(body, &list); err != nil {
				return err
			}

			eTagMap := make(map[string][]string)
			foundComponents := make(map[string]map[string]string) // component -> OS -> eTag
			for _, content := range list.Contents {
				parts := strings.Split(content.Key, "/")
				if len(parts) != 5 {
					continue
				}
				componentPart := parts[1]
				if _, ok := selectedComponents[componentPart]; !ok {
					continue
				}
				osPart := parts[2]
				channelPart := parts[3]
				itemPart := parts[4]
				if validVersion(channelPart) {
					eTagMap[content.ETag] = append(eTagMap[content.ETag], channelPart)
				}
				if channelPart != channel {
					continue
				}
				m, ok := componentFileMap[componentPart]
				if !ok {
					continue
				}
				if v, ok := m[osPart]; !ok || v != itemPart {
					continue
				}

				osMap := foundComponents[componentPart]
				if osMap == nil {
					osMap = make(map[string]string)
				}
				osMap[osPart] = content.ETag
				foundComponents[componentPart] = osMap
			}

			outputMap := make(map[string]map[string]string) // component -> OS -> version
			for component, osMap := range foundComponents {
				outputMap[component] = make(map[string]string)
				for os, eTag := range osMap {
					versions := eTagMap[eTag]
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
				table.SetHeader([]string{"Component\\OS", "macOS", "Linux", "Windows", "Linux (arm64)"})
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
					for _, os := range []string{"macos", "linux", "windows", "linux-arm64"} {
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

func urlFlag(url *string) *cli.StringFlag {
	return &cli.StringFlag{
		Name:        "url",
		EnvVars:     []string{"URL"},
		Value:       "https://tuf.fleetctl.com",
		Destination: url,
		Usage:       "URL of the TUF repository",
	}
}
