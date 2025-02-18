// Command appmanifest takes a .pkg file and outputs an XML manifest for it
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/appmanifest"
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

	m, err := appmanifest.NewPlist(fp, *pkgURL)
	if err != nil {
		log.Fatal(err) //nolint:gocritic // ignoring exitAfterDefer
	}

	fmt.Println(string(m))
}
