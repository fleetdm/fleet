package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"time"

	"github.com/facebookincubator/flog"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	_ "github.com/mattn/go-sqlite3"
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

	cve := nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()
	localdir, err := os.Getwd()
	panicif(err)

	if err := nvd.SetUserAgent(nvd.UserAgent()); err != nil {
		flog.Warningf("could not set User-Agent HTTP header, using default: %v", err)
	}
	flog.Infof("Using http User-Agent: %s", nvd.UserAgent())

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cve},
		Source:   source,
		LocalDir: localdir,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := dfs.Do(ctx); err != nil {
		flog.Fatal(err)
	}

	var files []string
	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if match, err := regexp.MatchString("nvdcve-1.1-.*.gz$", path); !match || err != nil {
			return nil
		}
		files = append(files, path)
		return nil
	})
	panicif(err)

	var readers []io.Reader
	for _, path := range files {
		reader, err := os.Open(path)
		panicif(err)
		readers = append(readers, reader)
		defer reader.Close()
	}

	fmt.Println("Generating DB...")
	//dbPath := path.Join(cwd, "cve.sqlite")
	//err = vulnerabilities.GenerateCVEDB(dbPath, readers...)
	//panicif(err)

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

	////////////////////////////////////////////////////////////

	//db, err := sqlx.Open("sqlite3", "./cpe.sqlite")
	//panicif(err)
	//var cpeList []string
	//err = db.Select(&cpeList, `SELECT cpe23 FROM cpe limit 500000`)
	//panicif(err)
	//
	//start := time.Now()
	//
	//f, err := os.Create("trace.out")
	//panicif(err)
	//defer f.Close()
	//err = trace.Start(f)
	//panicif(err)
	//defer trace.Stop()
	////f, err := os.Create("cpu.profile")
	////panicif(err)
	////defer f.Close()
	////err = pprof.StartCPUProfile(f)
	////panicif(err)
	////defer pprof.StopCPUProfile()
	//
	//cpes := make([]*wfn.Attributes, 0, len(cpeList))
	//for _, uri := range cpeList {
	//	attr, err := wfn.Parse(uri)
	//	panicif(err)
	//	cpes = append(cpes, attr)
	//}
	//
	//dict, err := cvefeed.LoadJSONDictionary(files...)
	//panicif(err)
	//cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(0)
	//cache.Idx = cvefeed.NewIndex(dict)
	//
	//cveMatches := make(map[string][]string)
	//type cveMatch struct {
	//	cve     string
	//	matches []string
	//}
	//
	//cancelCtx, cancelFunc := context.WithCancel(context.Background())
	//cpeCh := make(chan *wfn.Attributes)
	//matchesCh := make(chan cveMatch)
	//
	//cpeCount := 0
	//mu := sync.Mutex{}
	//total := 500000
	//
	//for i := 0; i < 4; i++ {
	//	go func() {
	//		for {
	//			select {
	//			case cpe := <-cpeCh:
	//
	//				for _, matches := range cache.Get([]*wfn.Attributes{cpe}) {
	//					ml := len(matches.CPEs)
	//					matchingCPEs := make([]string, ml)
	//					for i, attr := range matches.CPEs {
	//						if attr == nil {
	//							flog.Errorf("%s matches nil CPE", matches.CVE.ID())
	//							continue
	//						}
	//						matchingCPEs[i] = (*wfn.Attributes)(attr).BindToURI()
	//					}
	//					matchesCh <- cveMatch{cve: matches.CVE.ID(), matches: matchingCPEs}
	//				}
	//				mu.Lock()
	//				cpeCount++
	//				if cpeCount >= total {
	//					cancelFunc()
	//				}
	//				mu.Unlock()
	//			case <-cancelCtx.Done():
	//				fmt.Println("Done wfn processing, returning...")
	//				return
	//			}
	//		}
	//	}()
	//}
	//go func() {
	//	for {
	//		select {
	//		case match := <-matchesCh:
	//			cveMatches[match.cve] = match.matches
	//		case <-cancelCtx.Done():
	//			fmt.Println("Done collecting matches, returning...")
	//			return
	//		}
	//	}
	//}()
	//
	//fmt.Println("Starting pushing cpes")
	//
	//for _, cpe := range cpes {
	//	cpeCh <- cpe
	//}
	//
	//fmt.Println("Done pushing cpes")
	//
	//<-cancelCtx.Done()
	//
	////for _, matches := range cache.Get(cpes) {
	////	ml := len(matches.CPEs)
	////	matchingCPEs := make([]string, ml)
	////	for i, attr := range matches.CPEs {
	////		if attr == nil {
	////			flog.Errorf("%s matches nil CPE", matches.CVE.ID())
	////			continue
	////		}
	////		matchingCPEs[i] = (*wfn.Attributes)(attr).BindToURI()
	////	}
	////	cveMatches[matches.CVE.ID()] = matchingCPEs
	////}
	//took := time.Since(start)
	//
	////fmt.Println(cveMatches)
	//
	////mf, err := os.Create("mem.profile")
	////panicif(err)
	////defer mf.Close()
	////runtime.GC()
	////err = pprof.WriteHeapProfile(mf)
	////panicif(err)
	//
	//fmt.Println("Done. Took", took, len(cveMatches))
}
