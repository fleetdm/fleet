// Package vuln_centos contains a ParseCentOSRepository method to parse the CentOS repository
// to look out for CentOS releases that patch CVEs. It parses the changelogs from the metadata.
//
// It also contains a LoadCentOSFixedCVEs to load the results of the parsing.
//
// Both the parsing and loading of results use sqlite3 as backend storage.
package vuln_centos

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	kitlog "github.com/go-kit/kit/log"
	"github.com/gocolly/colly"
	_ "github.com/mattn/go-sqlite3"
)

// CentOSPkg holds data to identify a CentOS package.
type CentOSPkg struct {
	Name    string
	Version string
	Release string
	Arch    string
}

// String implements fmt.Stringer.
func (p CentOSPkg) String() string {
	return p.Name + "-" + p.Version + "-" + p.Release + "." + p.Arch
}

// FixedCVESet is a set of fixed CVEs.
type FixedCVESet map[string]struct{}

// CentOSPkgSet is a set of CentOS packages and their fixed CVEs.
type CentOSPkgSet map[CentOSPkg]FixedCVESet

// Add adds the given package and CVE/s to the set.
func (p CentOSPkgSet) Add(pkg CentOSPkg, fixedCVEs ...string) {
	s := p[pkg]
	if s == nil {
		s = make(FixedCVESet)
	}
	for _, fixedCVE := range fixedCVEs {
		s[fixedCVE] = struct{}{}
	}
	p[pkg] = s
}

const centOSPkgsCVEsTable = "centos_pkgs_fixed_cves"

// LoadCentOSFixedCVEs loads the CentOS packages with known fixed CVEs from the given sqlite3 db.
func LoadCentOSFixedCVEs(ctx context.Context, db *sql.DB, logger kitlog.Logger) (CentOSPkgSet, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`SELECT name, version, release, arch, cves FROM %s`, centOSPkgsCVEsTable))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch packages: %w", err)
	}
	defer rows.Close()

	pkgs := make(CentOSPkgSet)
	for rows.Next() {
		var pkg CentOSPkg
		var cves string
		if err := rows.Scan(&pkg.Name, &pkg.Version, &pkg.Release, &pkg.Arch, &cves); err != nil {
			return nil, err
		}
		for _, cve := range strings.Split(cves, ",") {
			pkgs.Add(pkg, "CVE-"+cve)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to traverse packages: %w", err)
	}
	return pkgs, nil
}

type centOSOpts struct {
	noCrawl  bool
	verbose  bool
	localDir string
	root     string
}

type CentOSOption func(*centOSOpts)

func WithLocalDir(dir string) CentOSOption {
	return func(o *centOSOpts) {
		o.localDir = dir
	}
}

func NoCrawl() CentOSOption {
	return func(o *centOSOpts) {
		o.noCrawl = true
	}
}

func WithVerbose(v bool) CentOSOption {
	return func(o *centOSOpts) {
		o.verbose = v
	}
}

func WithRoot(root string) CentOSOption {
	return func(o *centOSOpts) {
		o.root = root
	}
}

const (
	repositoryDomain = "mirror.centos.org"
	repositoryURL    = "http://" + repositoryDomain
	defaultRoot      = "/centos/"
)

var (
	// Only parse the repository metadata for CentOS 6, 7 and 8.
	//
	// CentOS 6 maintenance updates ended in 2020-11-30, but we will still
	// fetch metadata for CentOS 6 because it's considered as of 2022-02-02
	// a "recent" release.
	//
	// See https://en.wikipedia.org/wiki/CentOS#CentOS_releases.
	recentCentOSPathRegex = regexp.MustCompile(`/centos/[678]\S*`)
	// nonReleasePathRegex is used to skip non-package centos directories/files.
	nonReleasePathRegex = regexp.MustCompile(`/centos/[^0-9]`)
)

// ParseCentOSRepository performs the following operations:
// 	- Crawls the CentOS repository website. To find all the sqlite3 files with
//	the packages metadata.
//	- Processes all the found sqlite3 files to find all fixed CVEs in each package version.
//	It parses the changelogs for each package release and looks for the "CVE-" string.
//
// It writes progress messages to stdout.
func ParseCentOSRepository(opts ...CentOSOption) (CentOSPkgSet, error) {
	var opts_ centOSOpts
	for _, fn := range opts {
		fn(&opts_)
	}

	if opts_.localDir == "" && opts_.noCrawl {
		return nil, errors.New("invalid options: if no crawl is set, local dir must be set")
	}

	if opts_.localDir == "" {
		localDir, err := os.MkdirTemp("", "centos*")
		if err != nil {
			return nil, err
		}
		opts_.localDir = localDir
	}

	fmt.Printf("Using local directory: %s\n", opts_.localDir)
	if !opts_.noCrawl {
		if err := crawl(opts_.root, opts_.localDir, opts_.verbose); err != nil {
			return nil, err
		}
	}

	pkgs, err := parse(opts_.localDir)
	if err != nil {
		return nil, err
	}
	if opts_.verbose {
		for pkg, cves := range pkgs {
			var cveList []string
			for cve := range cves {
				cveList = append(cveList, cve)
			}
			if opts_.verbose {
				fmt.Printf("%s: %v\n", pkg, cveList)
			}
		}
	}

	return pkgs, nil
}

func crawl(root string, localDir string, verbose bool) error {
	fmt.Println("Crawling CentOS repository...")
	c := colly.NewCollector()

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return err
	}

	var repoMDs []url.URL
	c.OnHTML("#indexlist .indexcolname a[href]", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		// Skip going to parent directory.
		if strings.HasPrefix(root, href) {
			return
		}
		if nonReleasePathRegex.MatchString(path.Join(e.Request.URL.Path, href)) {
			return
		}
		if !recentCentOSPathRegex.MatchString(path.Join(e.Request.URL.Path, href)) {
			if verbose {
				fmt.Printf("Ignoring old release: %s\n", path.Join(e.Request.URL.Path, href))
			}
			return
		}
		if href == "repomd.xml" {
			u := *e.Request.URL
			u.Path = path.Join(u.Path, href)
			repoMDs = append(repoMDs, u)
			if verbose {
				fmt.Printf("%s\n", u.Path)
			}
			return
		}
		if !strings.Contains(href, "/") {
			return
		}
		e.Request.Visit(href)
	})

	c.AllowedDomains = append(c.AllowedDomains, repositoryDomain)

	if root == "" {
		root = defaultRoot
	}
	if err := c.Visit(repositoryURL + root); err != nil {
		return err
	}

	for _, u := range repoMDs {
		if err := processRepoMD(u, localDir, verbose); err != nil {
			return err
		}
	}

	return nil
}

type dbs struct {
	primary, other string
}

func parse(localDir string) (CentOSPkgSet, error) {
	fmt.Println("Processing sqlite files...")

	dbPaths := make(map[string]dbs)
	if err := filepath.WalkDir(localDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".sqlite") {
			return nil
		}
		dbp := dbPaths[filepath.Dir(path)]
		if strings.HasSuffix(path, "-primary.sqlite") {
			dbp.primary = path
		} else if strings.HasSuffix(path, "-other.sqlite") {
			dbp.other = path
		}
		dbPaths[filepath.Dir(path)] = dbp
		return nil
	}); err != nil {
		return nil, err
	}

	allPkgs := make(CentOSPkgSet)
	for _, db := range dbPaths {
		pkgs, err := processSqlites(db)
		if err != nil {
			return nil, err
		}
		for pkg, cves := range pkgs {
			for cve := range cves {
				allPkgs.Add(pkg, cve)
			}
		}
	}

	return allPkgs, nil
}

func processRepoMD(mdURL url.URL, localDir string, verbose bool) error {
	resp, err := http.Get(mdURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	type location struct {
		Href string `xml:"href,attr"`
	}
	type repoDataItem struct {
		Type     string   `xml:"type,attr"`
		Location location `xml:"location"`
	}
	type repoMetadata struct {
		XMLName xml.Name       `xml:"repomd"`
		Datas   []repoDataItem `xml:"data"`
	}
	var md repoMetadata
	if err := xml.Unmarshal(b, &md); err != nil {
		return err
	}
	for _, data := range md.Datas {
		if data.Type != "primary_db" && data.Type != "other_db" {
			continue
		}
		sqliteURL := mdURL
		sqliteURL.Path = strings.TrimSuffix(sqliteURL.Path, "repomd.xml") + strings.TrimPrefix(data.Location.Href, "repodata/")
		if verbose {
			fmt.Printf("%s\n", sqliteURL.Path)
		}
		filePath := filePathfromURL(localDir, sqliteURL)
		_, err := os.Stat(filePath)
		switch {
		case err == nil:
			// File already exists, nothing to do.
		case errors.Is(err, os.ErrNotExist):
			if err := download.Decompressed(fleethttp.NewClient(), sqliteURL, filePath); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func filePathfromURL(dir string, url url.URL) string {
	filePath := filepath.Join(dir, url.Path)
	filePath = strings.TrimSuffix(filePath, ".bz2")
	filePath = strings.TrimSuffix(filePath, ".xz")
	filePath = strings.TrimSuffix(filePath, ".gz")
	return filePath
}

func processSqlites(dbPaths dbs) (CentOSPkgSet, error) {
	db, err := sql.Open("sqlite3", dbPaths.primary)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if _, err := db.Exec(fmt.Sprintf("ATTACH DATABASE '%s' as other;", dbPaths.other)); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	rows, err := db.Query(`SELECT
		p.name, p.version, p.release, p.arch, c.changelog
		FROM packages p
		JOIN other.changelog c ON (p.pkgKey=c.pkgKey)
		WHERE c.changelog LIKE '%CVE-%-%';`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pkgs := make(CentOSPkgSet)
	for rows.Next() {
		var p CentOSPkg
		var changelog string
		if err := rows.Scan(&p.Name, &p.Version, &p.Release, &p.Arch, &changelog); err != nil {
			return nil, err
		}
		cves := parseCVEs(changelog)
		for _, cve := range cves {
			pkgs.Add(p, cve)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pkgs, nil
}

var cveRegex = regexp.MustCompile(`CVE\-[0-9]+\-[0-9]+`)

func parseCVEs(changelog string) []string {
	return cveRegex.FindAllString(changelog, -1)
}

// GenCentOSSqlite will store the CentOS package set in the given sqlite handle.
func GenCentOSSqlite(db *sql.DB, pkgs CentOSPkgSet) error {
	if err := createTable(db); err != nil {
		return err
	}
	type pkgWithCVEs struct {
		pkg  CentOSPkg
		cves string
	}
	var pkgsWithCVEs []pkgWithCVEs
	for pkg, cves := range pkgs {
		var cveList []string
		for cve := range cves {
			cveList = append(cveList, strings.TrimPrefix(cve, "CVE-"))
		}
		sort.Slice(cveList, func(i, j int) bool {
			return cveList[i] < cveList[j]
		})
		pkgsWithCVEs = append(pkgsWithCVEs, pkgWithCVEs{
			pkg:  pkg,
			cves: strings.Join(cveList, ","),
		})
	}
	for _, pkgWithCVEs := range pkgsWithCVEs {
		if _, err := db.Exec(
			fmt.Sprintf("REPLACE INTO %s (name, version, release, arch, cves) VALUES (?, ?, ?, ?, ?)", centOSPkgsCVEsTable),
			pkgWithCVEs.pkg.Name,
			pkgWithCVEs.pkg.Version,
			pkgWithCVEs.pkg.Release,
			pkgWithCVEs.pkg.Arch,
			pkgWithCVEs.cves,
		); err != nil {
			return err
		}
	}
	return nil
}

func createTable(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		name TEXT,
		version TEXT,
		release TEXT,
		arch TEXT,
		cves TEXT,

		UNIQUE (name, version, release, arch)
	);`, centOSPkgsCVEsTable))
	return err
}
