package vulnerabilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v37/github"
	"github.com/jmoiron/sqlx"
)

const (
	owner = "fleetdm"
	repo  = "nvd"
)

func GetLatestNVDRelease(client *http.Client) (*github.RepositoryRelease, error) {
	ghclient := github.NewClient(client)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 10})
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		// skip draft releases
		if !release.GetDraft() {
			return release, nil
		}
	}

	return nil, errors.New("no nvd release found")
}

const cpeDBFilename = "cpe.sqlite"

var cpeDBRegex = regexp.MustCompile(`^cpe-.*\.sqlite\.gz$`)

// DownloadCPEDB downloads the CPE database to the given vulnPath. If cpeDBURL is empty, attempts to download it
// from the latest release of github.com/fleetdm/nvd. Skips downloading if CPE database is newer than the release.
func DownloadCPEDB(
	vulnPath string,
	client *http.Client,
	cpeDBURL string,
) error {
	path := filepath.Join(vulnPath, cpeDBFilename)

	if cpeDBURL == "" {
		release, err := GetLatestNVDRelease(client)
		if err != nil {
			return err
		}
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		default:
			if stat.ModTime().After(release.CreatedAt.Time) {
				// file is newer than release, do nothing
				return nil
			}
		}

		for _, asset := range release.Assets {
			if cpeDBRegex.MatchString(asset.GetName()) {
				cpeDBURL = asset.GetBrowserDownloadURL()
				break
			}
		}
		if cpeDBURL == "" {
			return errors.New("failed to find cpe database in nvd release")

		}
	}

	u, err := url.Parse(cpeDBURL)
	if err != nil {
		return err
	}
	if err := download.DownloadAndExtract(client, u, path); err != nil {
		return err
	}

	return nil
}

type IndexedCPEItem struct {
	ID         int     `json:"id" db:"rowid"`
	Title      string  `json:"title" db:"title"`
	Product    string  `json:"product" db:"product"`
	Vendor     string  `json:"vendor" db:"vendor"`
	Version    *string `json:"version" db:"version"`
	TargetSW   *string `json:"target_sw" db:"target_sw"`
	CPE23      string  `json:"cpe23" db:"cpe23"`
	Deprecated bool    `json:"deprecated" db:"deprecated"`
}

func cleanAppName(appName string) string {
	return strings.TrimSuffix(appName, ".app")
}

var nonAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// sanitizeMatch sanitizes the search string for sqlite fts queries. Replaces all non alpha numeric characters with spaces.
func sanitizeMatch(s string) string {
	return nonAlphaNumeric.ReplaceAllString(s, " ")
}

var sanitizeVersionRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// sanitizeVersion attempts to sanitize versions and attempt to make it dot separated.
// Eg Zoom reports version as "5.11.1 (8356)". In the NVD CPE dictionary it should be 5.11.1.8356.
func sanitizeVersion(version string) string {
	parts := sanitizeVersionRe.Split(version, -1)
	return strings.Trim(strings.Join(parts, "."), ".")
}

const cpeTranslationsFilename = "cpe_translations.json"

func loadCPETranslations(path string) (CPETranslations, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var translations CPETranslations
	if err := json.NewDecoder(f).Decode(&translations); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	return translations, nil
}

// DownloadCPETranslations downloads the CPE translations to the given vulnPath. If cpeTranslationsURL is empty, attempts to download it
// from the latest release of github.com/fleetdm/nvd. Skips downloading if CPE translations is newer than the release.
func DownloadCPETranslations(vulnPath string, client *http.Client, cpeTranslationsURL string) error {
	path := filepath.Join(vulnPath, cpeTranslationsFilename)

	if cpeTranslationsURL == "" {
		release, err := GetLatestNVDRelease(client)
		if err != nil {
			return err
		}
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		default:
			if stat.ModTime().After(release.CreatedAt.Time) {
				// file is newer than release, do nothing
				return nil
			}
		}

		for _, asset := range release.Assets {
			if cpeTranslationsFilename == asset.GetName() {
				cpeTranslationsURL = asset.GetBrowserDownloadURL()
				break
			}
		}
		if cpeTranslationsURL == "" {
			return errors.New("failed to find cpe translations in nvd release")

		}
	}

	u, err := url.Parse(cpeTranslationsURL)
	if err != nil {
		return err
	}
	if err := download.Download(client, u, path); err != nil {
		return err
	}

	return nil
}

// regexpCache caches compiled regular expressions. Not safe for concurrent use.
type regexpCache struct {
	re map[string]*regexp.Regexp
}

func newRegexpCache() *regexpCache {
	return &regexpCache{re: make(map[string]*regexp.Regexp)}
}

func (r *regexpCache) Get(pattern string) (*regexp.Regexp, error) {
	if re, ok := r.re[pattern]; ok {
		return re, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	r.re[pattern] = re
	return re, nil
}

// CPETranslations include special case translations for software that fail to match entries in the NVD CPE Dictionary
// using the standard logic. This may be due to unexpected vendor or product names.
//
// Example:
//
//     [
//       {
//         "match": {
//           "bundle_identifier": ["com.1password.1password"]
//         },
//         "translation": {
//           "product": ["1password"],
//           "vendor": ["agilebits"]
//         }
//       }
//     ]
type CPETranslations []CPETranslationItem

func (c CPETranslations) Translate(reCache *regexpCache, s *fleet.Software) (CPETranslation, bool, error) {
	for _, entry := range c {
		match, err := entry.Software.Matches(reCache, s)
		if err != nil {
			return CPETranslation{}, false, err
		}
		if match {
			return entry.Filter, true, nil
		}
	}

	return CPETranslation{}, false, nil
}

type CPETranslationItem struct {
	Software CPETranslationSoftware `json:"software"`
	Filter   CPETranslation         `json:"filter"`
}

// CPETranslationSoftware represents software match criteria for cpe translations.
type CPETranslationSoftware struct {
	Name             []string `json:"name"`
	BundleIdentifier []string `json:"bundle_identifier"`
	Source           []string `json:"source"`
}

// Matches returns true if the software satifies all the match criteria.
func (c CPETranslationSoftware) Matches(reCache *regexpCache, s *fleet.Software) (bool, error) {
	matches := func(a, b string) (bool, error) {
		// check if its a regular expression enclosed in '/'
		if len(a) > 2 && a[0] == '/' && a[len(a)-1] == '/' {
			pattern := a[1 : len(a)-1]
			re, err := reCache.Get(pattern)
			if err != nil {
				return false, err
			}
			return re.MatchString(b), nil
		}
		return a == b, nil
	}

	if len(c.Name) > 0 {
		found := false
		for _, name := range c.Name {
			match, err := matches(name, s.Name)
			if err != nil {
				return false, err
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	if len(c.BundleIdentifier) > 0 {
		found := false
		for _, bundleID := range c.BundleIdentifier {
			match, err := matches(bundleID, s.BundleIdentifier)
			if err != nil {
				return false, err
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	if len(c.Source) > 0 {
		found := false
		for _, source := range c.Source {
			match, err := matches(source, s.Source)
			if err != nil {
				return false, err
			}
			if match {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

type CPETranslation struct {
	Product  []string `json:"product"`
	Vendor   []string `json:"vendor"`
	TargetSW []string `json:"target_sw"`
}

// CPEFromSoftware attempts to find a matching cpe entry for the given software in the NVD CPE dictionary. `db` contains data from the NVD CPE dictionary
// and is optimized for lookups, see `GenerateCPEDB`. `translations` are used to aid in cpe matching. When searching for cpes, we first check if it matches
// any translations, and then lookup in the cpe database based on the title, product, vendor, target_sw, and version.
func CPEFromSoftware(db *sqlx.DB, software *fleet.Software, translations CPETranslations, reCache *regexpCache) (string, error) {
	version := sanitizeVersion(software.Version)

	ds := goqu.Dialect("sqlite").From(goqu.I("cpe_2").As("c")).
		Select(
			"c.rowid",
			"c.title",
			"c.product",
			"c.vendor",
			"c.version",
			"c.target_sw",
			"c.cpe23",
			"c.deprecated",
		).
		Join(
			goqu.I("cpe_search").As("cs"),
			goqu.On(goqu.I("cs.rowid").Eq(goqu.I("c.rowid"))),
		).
		Where(
			goqu.I("c.version").Eq(version),
		)

	translation, match, err := translations.Translate(reCache, software)
	if err != nil {
		return "", fmt.Errorf("translate software: %w", err)
	}
	if match {
		if len(translation.Product) > 0 {
			var exps []goqu.Expression
			for _, product := range translation.Product {
				exps = append(exps, goqu.I("c.product").Eq(product))
			}
			ds = ds.Where(goqu.Or(exps...))
		}
		if len(translation.Vendor) > 0 {
			var exps []goqu.Expression
			for _, vendor := range translation.Vendor {
				exps = append(exps, goqu.I("c.vendor").Eq(vendor))
			}
			ds = ds.Where(goqu.Or(exps...))
		}
		if len(translation.TargetSW) > 0 {
			var exps []goqu.Expression
			for _, targetSW := range translation.TargetSW {
				exps = append(exps, goqu.I("c.target_sw").Eq(targetSW))
			}
			ds = ds.Where(goqu.Or(exps...))
		}
	} else {

		name := software.Name
		var targetSW string

		switch software.Source {
		case "apps":
			name = cleanAppName(software.Name)

			// match on bundle identifier to reduce false positives for software with short names eg notes,
			// printer, calculator.
			// match the following target_sw
			// - mac
			// - mac_os
			// - mac_os_x
			// - macos
			ds = ds.Where(
				goqu.L("? LIKE '%' || c.vendor || '%'", software.BundleIdentifier),
				goqu.Or(
					goqu.I("c.target_sw").Eq(""),
					goqu.I("c.target_sw").Like("mac%"),
				),
			)
		case "python_packages":
			targetSW = "python"
		case "chrome_extensions":
			targetSW = "chrome"
		case "firefox_addons":
			targetSW = "firefox"
		case "safari_extensions":
			targetSW = "safari"
		case "npm_packages":
			targetSW = `node.js`
		case "programs":

			// match the following target_sw
			// - windows
			// - windows_10
			// - windows_7
			// - windows_8
			// - windows_8.1
			// - windows_ce
			// - windows_communication_foundation
			// - windows_integrated_security
			// - windows_mobile
			// - windows_phone
			// - windows_server
			// - windows_server_2003
			// - windows_server_2008
			// - windows_vista
			// - windows_xp
			ds = ds.Where(
				goqu.Or(
					goqu.I("c.target_sw").Like("windows%"),
				),
			)
		}
		if targetSW != "" {
			ds = ds.Where(
				goqu.L("c.target_sw").Eq(targetSW),
			)
		}

		// sanitize name for full text search on title
		nameTerms := sanitizeMatch(name)
		ds = ds.Where(
			goqu.L("cs.title MATCH ?", nameTerms),
		)
	}

	sql, args, err := ds.ToSQL()
	if err != nil {
		return "", fmt.Errorf("sql: %w", err)
	}

	var indexedCPEs []IndexedCPEItem
	err = db.Select(&indexedCPEs, sql, args...)
	if err != nil {
		return "", fmt.Errorf("getting cpes for: %s: %w", software.Name, err)
	}

	// if there are any non-deprecated cpes, return the first one
	for _, item := range indexedCPEs {
		if !item.Deprecated {
			return item.CPE23, nil
		}
	}

	// try to find a non-deprecated cpe by looking up deprecated_by
	for _, item := range indexedCPEs {
		deprecatedItem := item
		for {
			var deprecation IndexedCPEItem

			err = db.Get(
				&deprecation,
				`
SELECT
    rowid,
    title,
    product,
    vendor,
    version,
    target_sw,
    cpe23,
    deprecated
FROM
    cpe_2
WHERE
    cpe23 IN (
        SELECT cpe23 FROM deprecated_by d WHERE d.cpe_id = ?
    )
`,
				deprecatedItem.ID,
			)
			if err != nil {
				return "", fmt.Errorf("getting deprecation: %w", err)
			}
			if deprecation.Deprecated {
				deprecatedItem = deprecation
				continue
			}

			return deprecation.CPE23, nil
		}
	}

	return "", nil
}

func TranslateSoftwareToCPE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
) error {
	dbPath := filepath.Join(vulnPath, cpeDBFilename)

	// Skip software from platforms for which we will be using OVAL for vulnerability detection.
	iterator, err := ds.AllSoftwareWithoutCPEIterator(ctx, oval.SupportedHostPlatforms)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "all software iterator")
	}
	defer iterator.Close()

	db, err := sqliteDB(dbPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "opening the cpe db")
	}
	defer db.Close()

	cpeTranslationsPath := filepath.Join(vulnPath, cpeTranslationsFilename)
	cpeTranslations, err := loadCPETranslations(cpeTranslationsPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to load cpe translations", "err", err)
	}

	reCache := newRegexpCache()

	for iterator.Next() {
		software, err := iterator.Value()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting value from iterator")
		}
		cpe, err := CPEFromSoftware(db, software, cpeTranslations, reCache)
		if err != nil {
			level.Error(logger).Log("software->cpe", "error translating to CPE, skipping...", "err", err)
			continue
		}
		if cpe == "" {
			continue
		}
		err = ds.AddCPEForSoftware(ctx, *software, cpe)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting cpe")
		}
	}

	return nil
}
