package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
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
	app.Usage = "CLI to query a Fleet TUF repository"
	app.Commands = []*cli.Command{
		channelVersionCommand(),
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stdout, "Error: %+v\n", err)
		os.Exit(1)
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

var componentFileMap = map[string]map[string]string{
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

func channelVersionCommand() *cli.Command {
	var (
		channel    string
		tufURL     string
		components cli.StringSlice
		format     string
		vendor     string
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
				Name:        "s3-vendor",
				EnvVars:     []string{"S3_VENDOR"},
				Value:       "cloudflare",
				Destination: &vendor,
				Usage:       "Vendor that hosts the TUF repository, currently one of 'cloudflare' or 'amazon'",
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
			if vendor != "cloudflare" && vendor != "amazon" {
				return errors.New("supported vendors are: cloudflare, amazon")
			}

			var (
				foundComponents map[string]map[string]string // component -> OS -> eTag
				eTagMap         map[string][]string
				err             error
			)
			if vendor == "cloudflare" {
				foundComponents, eTagMap, err = getComponentsCloudflare(tufURL, components.Value(), channel)
			} else {
				foundComponents, eTagMap, err = getComponentsAmazon(tufURL, components.Value(), channel)
			}
			if err != nil {
				return fmt.Errorf("get components: %w", err)
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

func getComponentsAmazon(tufURL string, components []string, channel string) (foundComponents map[string]map[string]string, eTagMap map[string][]string, err error) {
	res, err := http.Get(tufURL) //nolint
	if err != nil {
		return nil, nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	selectedComponents := make(map[string]struct{})
	for _, component := range components {
		selectedComponents[component] = struct{}{}
	}

	var list listBucketResult
	if err := xml.Unmarshal(body, &list); err != nil {
		return nil, nil, err
	}

	eTagMap = make(map[string][]string)
	foundComponents = make(map[string]map[string]string) // component -> OS -> eTag
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
	return foundComponents, eTagMap, nil
}

func getComponentsCloudflare(tufURL string, components []string, channel string) (foundComponents map[string]map[string]string, eTagMap map[string][]string, err error) {
	res, err := http.Get(tufURL + "/targets.json") //nolint
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	selectedComponents := make(map[string]struct{})
	for _, component := range components {
		selectedComponents[component] = struct{}{}
	}

	var targetsJSON map[string]interface{}
	if err := json.Unmarshal(body, &targetsJSON); err != nil {
		log.Fatal("failed to parse the source targets.json file")
	}

	signed_ := targetsJSON["signed"]
	if signed_ == nil {
		log.Fatal("missing signed key in targets.json file")
	}
	signed, ok := signed_.(map[string]interface{})
	if !ok {
		log.Fatalf("invalid signed key in targets.json file: %T, expected map", signed_)
	}
	targets_ := signed["targets"]
	if targets_ == nil {
		log.Fatal("missing signed.targets key in targets.json file")
	}
	targets, ok := targets_.(map[string]interface{})
	if !ok {
		log.Fatalf("invalid signed.targets key in targets.json file: %T, expected map", targets_)
	}

	eTagMap = make(map[string][]string)
	foundComponents = make(map[string]map[string]string) // component -> OS -> eTag
	for target, metadata_ := range targets {
		parts := strings.Split(target, "/")
		if len(parts) != 4 {
			log.Fatalf("target %q: invalid number of parts, expected 4", target)
		}

		targetName := parts[0]
		platformPart := parts[1]
		channelPart := parts[2]
		executablePart := parts[3]

		metadata, ok := metadata_.(map[string]interface{})
		if !ok {
			log.Fatalf("target: %q: invalid metadata field: %T, expected map", target, metadata_)
		}
		custom_ := metadata["custom"]
		if custom_ == nil {
			log.Fatalf("target: %q: missing custom field", target)
		}
		hashes_ := metadata["hashes"]
		if hashes_ == nil {
			log.Fatalf("target: %q: missing hashes field", target)
		}
		hashes, ok := hashes_.(map[string]interface{})
		if !ok {
			log.Fatalf("target: %q: invalid hashes field: %T", target, hashes_)
		}
		sha512_ := hashes["sha512"]
		if sha512_ == nil {
			log.Fatalf("target: %q: missing hashes.sha512 field", target)
		}
		hashSHA512, ok := sha512_.(string)
		if !ok {
			log.Fatalf("target: %q: invalid hashes.sha512 field: %T", target, sha512_)
		}
		if validVersion(channelPart) {
			eTagMap[hashSHA512] = append(eTagMap[hashSHA512], channelPart)
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
	return foundComponents, eTagMap, nil
}
