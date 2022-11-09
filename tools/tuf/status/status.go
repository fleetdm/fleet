package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
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
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `
This is a CLI utility to fetch and filter the entries posted by a TUF repository.

`)
		flag.CommandLine.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), `

Examples

- To filter all items on the edge channel use --key-filter="edge"
- To filter all items on version 1.3 including patches that run on Linux use --key-filter="linux/1.3.*"
- To filter Fleet Desktop items on 1.3.*, stable and edge that run on macOS use --key-filter="desktop/*.*/macos/(1.3.*|stable|edge)"

`)
	}
	filter := flag.String("key-filter", "stable", "filter keys using a regular expression")
	url := flag.String("url", "https://tuf.fleetctl.com", "URL of the TUF repository")
	flag.Parse()

	res, err := http.Get(*url)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var list listBucketResult
	if err := xml.Unmarshal(body, &list); err != nil {
		panic(err)
	}

	if err := printTable(list.Contents, *filter); err != nil {
		panic(err)
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
