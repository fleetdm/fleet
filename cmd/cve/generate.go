package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
)

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println("Starting CVE sqlite generation")

	cwd, err := os.Getwd()
	panicif(err)

	fmt.Println("CWD:", cwd)

	//resp, err := http.Get("https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-2020.json.gz")
	//panicif(err)
	//defer resp.Body.Close()
	//
	//// TODO: only download if changed
	//
	//fmt.Println("Needs updating. Generating...")

	//gr, err := gzip.NewReader(resp.Body)
	//panicif(err)
	//defer gr.Close()

	fmt.Println("Downloading feeds...")

	var cvefeed nvd.CVE
	cvefeed = nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()
	localdir, err := os.Getwd()
	panicif(err)

	if err := nvd.SetUserAgent(nvd.UserAgent()); err != nil {
		flog.Warningf("could not set User-Agent HTTP header, using default: %v", err)
	}
	flog.Infof("Using http User-Agent: %s", nvd.UserAgent())

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cvefeed},
		Source:   source,
		LocalDir: localdir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := dfs.Do(ctx); err != nil {
		flog.Fatal(err)
	}

	var readers []io.Reader
	var files []string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if match, err := regexp.MatchString("nvdcve-1.1-.*.gz$", path); !match || err != nil {
			return nil
		}
		files = append(files, path)
		return nil
	})
	panicif(err)

	for _, path := range files {
		reader, err := os.Open(path)
		panicif(err)
		readers = append(readers, reader)
		defer reader.Close()
	}

	fmt.Println("Generating DB...")
	dbPath := path.Join(cwd, "cve.sqlite")
	err = vulnerabilities.GenerateCVEDB(dbPath, readers...)
	panicif(err)

	// build an Index which is cpe.Product->CVEs, then match with that dict
	// so maybe we can build a sql db that reflects this index for all years
	// and check against that? we need the whole entry for the json so the
	// matcher works

	// so for each cpe string, we transform into wfn.Attributes, then search all the CVEs for the product and load it into memory
	// throw that to one of N goroutines that run match with the original CPE and the vulns from ^
	// the above is written to a chan that gathers N and writes in bulk

	//fmt.Println("Compressing db...")
	//compressedDB, err := os.Create(fmt.Sprintf("%s.gz", dbPath))
	//panicif(err)
	//
	//db, err := os.Open(dbPath)
	//w := gzip.NewWriter(compressedDB)
	//
	//_, err = io.Copy(w, db)
	//panicif(err)
	//w.Close()
	//compressedDB.Close()

	//file, err := os.Create(path.Join(cwd, "etagenv"))
	//panicif(err)
	//file.WriteString(fmt.Sprintf(`ETAG=%s`, remoteEtag))
	//file.Close()

	fmt.Println("Done.")
}
