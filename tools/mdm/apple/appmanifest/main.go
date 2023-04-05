// Command appmanifest takes a .pkg file and outputs an XML manifest for it
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
	"github.com/groob/plist"
)

func main() {
	pkgFile := flag.String("pkg-file", "", "Path to a .pkg file")
	pkgURL := flag.String("pkg-url", "", "URL where the package will be served")
	flag.Parse()

	if *pkgFile == "" || *pkgURL == "" {
		log.Fatal("both --pkg-file and --pkg-url must be provided")
	}

	fp, err := os.Open(*pkgFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	m, err := appmanifest.Create(fp, *pkgURL)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("  ")
	if err := enc.Encode(m); err != nil {
		log.Fatal(err)
	}

	fmt.Println(buf.String())
}
