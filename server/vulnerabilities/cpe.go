package vulnerabilities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v37/github"
	"github.com/jmoiron/sqlx"
)

const (
	owner = "fleetdm"
	repo  = "nvd"
)

type NVDRelease struct {
	Etag      string
	CreatedAt time.Time
	CPEURL    string
}

var cpeSqliteRegex = regexp.MustCompile(`^cpe-.*\.sqlite\.gz$`)

func GetLatestNVDRelease(client *http.Client) (*NVDRelease, error) {
	ghclient := github.NewClient(client)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 10})
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	cpeURL := ""

	// TODO: get not draft release

	for _, asset := range releases[0].Assets {
		if asset != nil {
			matched := cpeSqliteRegex.MatchString(asset.GetName())
			if !matched {
				continue
			}
			cpeURL = asset.GetBrowserDownloadURL()
		}
	}

	return &NVDRelease{
		Etag:      releases[0].GetName(),
		CreatedAt: releases[0].GetCreatedAt().Time,
		CPEURL:    cpeURL,
	}, nil
}

type syncOpts struct {
	url string
}

type CPESyncOption func(*syncOpts)

func WithCPEURL(url string) CPESyncOption {
	return func(o *syncOpts) {
		o.url = url
	}
}

const cpeDatabaseFilename = "cpe.sqlite"

// DownloadCPEDatabase downloads the CPE database from the
// latest release of github.com/fleetdm/nvd to the given dbPath.
// An alternative URL can be set via the WithCPEURL option.
//
// It won't download the database if the database has already been downloaded and
// has an mtime after the release date.
func DownloadCPEDatabase(
	vulnPath string,
	client *http.Client,
	opts ...CPESyncOption,
) error {
	var o syncOpts
	for _, fn := range opts {
		fn(&o)
	}

	dbPath := filepath.Join(vulnPath, cpeDatabaseFilename)

	if o.url == "" {
		nvdRelease, err := GetLatestNVDRelease(client)
		if err != nil {
			return err
		}
		stat, err := os.Stat(dbPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else if !nvdRelease.CreatedAt.After(stat.ModTime()) {
			return nil
		}
		o.url = nvdRelease.CPEURL
	}

	u, err := url.Parse(o.url)
	if err != nil {
		return err
	}
	if err := download.DownloadAndExtract(client, u, dbPath); err != nil {
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

var onlyAlphaNumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// sanitizeMatch sanitizes the search string for sqlite fts queries. Replaces all special characters with spaces.
func santizeMatch(s string) string {
	return onlyAlphaNumeric.ReplaceAllString(s, " ")
}

var sanitizeVersionRe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// sanitizeVersion attempts to sanitize versions and attempt to make it dot separated.
// Eg Zoom reports version as "5.11.1 (8356)". In the NVD CPE dictionary it should be 5.11.1.8356.
func sanitizeVersion(version string) string {
	parts := onlyAlphaNumeric.Split(version, -1)
	return strings.Trim(strings.Join(parts, "."), ".")
}

// TODO: add more vendors
var macOSVendors = map[string]string{
	"com.postmanlabs.mac":           "getpostman",
	"org.virtualbox.app.VirtualBox": "oracle",
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

func DownloadCPETranslations(vulnPath string, client *http.Client) error {
	ghClient := github.NewClient(client)

	opts := &github.RepositoryContentGetOptions{
		Ref: "michal-6628-macos-vuln",
	}
	r, resp, err := ghClient.Repositories.DownloadContents(context.Background(), "fleetdm", "nvd", cpeTranslationsFilename, opts)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github response non 200 status code: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp(vulnPath, cpeTranslationsFilename)
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer tmpFile.Close()

	// Clean up tmp file if not moved
	moved := false
	defer func() {
		if !moved {
			os.Remove(tmpFile.Name())
		}
	}()

	if _, err := io.Copy(tmpFile, r); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("write and close temporary file: %w", err)
	}

	path := filepath.Join(vulnPath, cpeTranslationsFilename)

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("rename temporary file: %w", err)
	}

	moved = true

	return nil
}

// CPETranslations include special case translations for software that fail to match entries in the NVD CPE Dictionary
// using the standard logic. This may be due to unexpected vendor or product names.
//
// Example:
//   [
//     {
//       "match": {
//         "bundle_identifier": ["com.1password.1password"]
//       },
//       "translation": {
//         "product": ["1password"],
//         "vendor": ["agilebits"]
//       }
//     }
type CPETranslations []CPETranslationEntry

func (c CPETranslations) Translate(s *fleet.Software) (CPETranslation, bool) {
	for _, entry := range c {
		if entry.Match.Matches(s) {
			return entry.Translation, true
		}
	}

	return CPETranslation{}, false
}

type CPETranslationEntry struct {
	Match       CPETranslationMatch `json:"match"`
	Translation CPETranslation      `json:"translation"`
}

// CPETranslationMatch represents match criteria for cpe translations.
type CPETranslationMatch struct {
	Name             []string `json:"name"`
	BundleIdentifier []string `json:"bundle_identifier"`
	Source           []string `json:"source"`
}

func (c CPETranslationMatch) Matches(s *fleet.Software) bool {
	for _, name := range c.Name {
		if name != s.Name {
			return false
		}
	}
	for _, bundleID := range c.BundleIdentifier {
		if bundleID != s.BundleIdentifier {
			return false
		}
	}
	for _, source := range c.Source {
		if source != s.Source {
			return false
		}
	}
	return true
}

type CPETranslation struct {
	Product  []string `json:"product"`
	Vendor   []string `json:"vendor"`
	TargetSW []string `json:"target_sw"`
}

func CPEFromSoftware(db *sqlx.DB, software *fleet.Software, translations CPETranslations) (string, error) {
	version := sanitizeVersion(software.Version)

	ds := goqu.Dialect("sqlite").From(goqu.I("cpe").As("c")).
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

	if translation, ok := translations.Translate(software); ok {
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
			targetSW = `"node.js"`
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
				goqu.I("c.target_sw").Eq(targetSW),
			)
		}

		// sanitize name for full text search on title
		nameTerms := onlyAlphaNumeric.ReplaceAllString(name, " ")
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

	// if there are any non-depecrated cpes, return the first one
	for _, item := range indexedCPEs {
		if !item.Deprecated {
			return item.CPE23, nil
		}
	}

	// try to find a non-depcrecated cpe by looking up deprecated_by
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
    cpe
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
	dbPath := filepath.Join(vulnPath, cpeDatabaseFilename)

	iterator, err := ds.AllSoftwareWithoutCPEIterator(ctx)
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
		level.Warn(logger).Log("msg", "failed to load cpe translations", "err", err)
	}

	for iterator.Next() {
		software, err := iterator.Value()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting value from iterator")
		}
		cpe, err := CPEFromSoftware(db, software, cpeTranslations)
		if err != nil {
			level.Error(logger).Log("software->cpe", "error translating to CPE, skipping...", "err", err)
			continue
		}
		if cpe == "" {
			// The schema for storing CVEs requires that a CPE for every software exists,
			// having that constraint in place works fine when the only source for vulnerabilities
			// is the NVD dataset but breaks down when we look at other sources for vulnerabilities (like OVAL) - this is
			// why we set a default value for CPEs.
			cpe = fmt.Sprintf("none:%d", software.ID)
		}
		err = ds.AddCPEForSoftware(ctx, *software, cpe)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting cpe")
		}
	}

	return nil
}
