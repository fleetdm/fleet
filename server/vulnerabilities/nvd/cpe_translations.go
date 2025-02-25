package nvd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/go-github/v37/github"
)

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

// DownloadCPETranslationsFromGithub downloads the CPE translations to the given vulnPath. If cpeTranslationsURL is empty, attempts to download it
// from the latest release of github.com/fleetdm/nvd. Skips downloading if CPE translations is newer than the release.
func DownloadCPETranslationsFromGithub(vulnPath string, cpeTranslationsURL string) error {
	path := filepath.Join(vulnPath, cpeTranslationsFilename)

	if cpeTranslationsURL == "" {
		stat, err := os.Stat(path)
		switch {
		case errors.Is(err, os.ErrNotExist):
			// okay
		case err != nil:
			return err
		case stat.ModTime().Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour)):
			// Vulnerability assets are published once per day - if the asset in question has a
			// mod date of 'today', then we can assume that is already up to day.
			return nil
		}

		release, asset, err := GetGithubNVDAsset(func(asset *github.ReleaseAsset) bool {
			return cpeTranslationsFilename == asset.GetName()
		})
		if err != nil {
			return err
		}
		if asset == nil {
			return errors.New("failed to find cpe translations in nvd release")
		}
		if stat != nil && stat.ModTime().After(release.CreatedAt.Time) {
			// file is newer than release, do nothing
			return nil
		}
		cpeTranslationsURL = asset.GetBrowserDownloadURL()
	}

	u, err := url.Parse(cpeTranslationsURL)
	if err != nil {
		return err
	}
	client := fleethttp.NewGithubClient()
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
//	[
//	  {
//	    "software": {
//	      "bundle_identifier": ["com.1password.1password"]
//	    },
//	    "filter": {
//	      "product": ["1password"],
//	      "vendor": ["agilebits"]
//	    }
//	  }
//	]
type CPETranslations []CPETranslationItem

func (c CPETranslations) Translate(reCache *regexpCache, s *fleet.Software) (CPETranslation, bool, error) {
	for _, item := range c {
		match, err := item.Software.Matches(reCache, s)
		if err != nil {
			return CPETranslation{}, false, err
		}
		if match {
			return item.Filter, true, nil
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
	Part     string   `json:"part"`
	// If Skip is set, no NVD vulnerabilities will be reported for the matching software.
	Skip bool `json:"skip"`
}
