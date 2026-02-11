package amazon

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/fetcher/util"
	models "github.com/vulsio/goval-dictionary/models/amazon"
)

// updateinfo for x86_64 also contains information for aarch64

type mirror struct {
	core      string
	extra     string
	livepatch string
}

var mirrors = map[string]mirror{
	"1": {core: "http://repo.us-west-2.amazonaws.com/2018.03/updates/x86_64/mirror.list"},
	"2": {
		core:  "https://cdn.amazonlinux.com/2/core/latest/x86_64/mirror.list",
		extra: "http://amazonlinux.default.amazonaws.com/2/extras-catalog.json",
	},
	"2022": {
		core: "https://cdn.amazonlinux.com/al2022/core/mirrors/latest/x86_64/mirror.list",
	},
	"2023": {
		core:      "https://cdn.amazonlinux.com/al2023/core/mirrors/latest/x86_64/mirror.list",
		livepatch: "https://cdn.amazonlinux.com/al2023/kernel-livepatch/mirrors/latest/x86_64/mirror.list",
	},
}

var errNoUpdateInfo = xerrors.New("No updateinfo field in the repomd")

// FetchFiles fetch from Amazon ALAS
func FetchFiles(versions []string) (map[string]*models.Updates, error) {
	m := map[string]*models.Updates{}
	for _, v := range versions {
		switch v {
		case "1", "2022":
			us, err := fetchUpdateInfoAmazonLinux(mirrors[v].core)
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch Amazon Linux %s UpdateInfo. err: %w", v, err)
			}
			m[v] = us
		case "2":
			updates, err := fetchUpdateInfoAmazonLinux(mirrors[v].core)
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch Amazon Linux %s UpdateInfo. err: %w", v, err)
			}

			rs, err := util.FetchFeedFiles([]util.FetchRequest{{URL: mirrors[v].extra, MIMEType: util.MIMETypeJSON}})
			if err != nil || len(rs) != 1 {
				return nil, xerrors.Errorf("Failed to fetch extras-catalog.json for Amazon Linux 2. url: %s, err: %w", mirrors[v].extra, err)
			}

			var catalog extrasCatalog
			if err := json.Unmarshal(rs[0].Body, &catalog); err != nil {
				return nil, xerrors.Errorf("Failed to unmarshal extras-catalog.json for Amazon Linux 2. err: %w", err)
			}

			for _, t := range catalog.Topics {
				us, err := fetchUpdateInfoAmazonLinux(fmt.Sprintf("https://cdn.amazonlinux.com/2/extras/%s/latest/x86_64/mirror.list", t.N))
				if err != nil {
					if errors.Is(err, errNoUpdateInfo) {
						continue
					}
					return nil, xerrors.Errorf("Failed to fetch Amazon Linux 2 %s updateinfo. err: %w", t.N, err)
				}
				for _, u := range us.UpdateList {
					u.Repository = fmt.Sprintf("amzn2extra-%s", t.N)
					updates.UpdateList = append(updates.UpdateList, u)
				}
			}

			m[v] = updates
		case "2023":
			updates, err := fetchUpdateInfoAmazonLinux(mirrors[v].core)
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch Amazon Linux %s UpdateInfo. err: %w", v, err)
			}

			us, err := fetchUpdateInfoAmazonLinux(mirrors[v].livepatch)
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch Amazon Linux %s Kernel Livepatch UpdateInfo. err: %w", v, err)
			}

			for _, u := range us.UpdateList {
				u.Repository = "kernel-livepatch"
				updates.UpdateList = append(updates.UpdateList, u)
			}

			m[v] = updates
		default:
			log15.Warn("Skip unknown amazon.", "version", v)
		}
	}
	return m, nil
}

func fetchUpdateInfoAmazonLinux(mirrorListURL string) (uinfo *models.Updates, err error) {
	results, err := util.FetchFeedFiles([]util.FetchRequest{{URL: mirrorListURL, MIMEType: util.MIMETypeXML}})
	if err != nil || len(results) != 1 {
		return nil, xerrors.Errorf("Failed to fetch mirror list files. err: %w", err)
	}

	mirrors := []string{}
	for _, r := range results {
		scanner := bufio.NewScanner(bytes.NewReader(r.Body))
		for scanner.Scan() {
			mirrors = append(mirrors, scanner.Text())
		}
	}

	uinfoURLs, err := fetchUpdateInfoURL(mirrors)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch updateInfo URL. err: %w", err)
	}
	for _, url := range uinfoURLs {
		uinfo, err = fetchUpdateInfo(url)
		if err != nil {
			log15.Warn("Failed to fetch updateinfo. continue with other mirror", "err", err)
			continue
		}
		return uinfo, nil
	}
	return nil, xerrors.New("Failed to fetch updateinfo")
}

// FetchUpdateInfoURL fetches update info urls for AmazonLinux1 ,Amazon Linux2 and Amazon Linux2022.
func fetchUpdateInfoURL(mirrors []string) (updateInfoURLs []string, err error) {
	reqs := []util.FetchRequest{}
	for _, mirror := range mirrors {
		u, err := url.Parse(mirror)
		if err != nil {
			return nil, err
		}
		u.Path = path.Join(u.Path, "/repodata/repomd.xml")
		reqs = append(reqs, util.FetchRequest{
			Target:   mirror, // base URL of the mirror site
			URL:      u.String(),
			MIMEType: util.MIMETypeXML,
		})
	}

	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		log15.Warn("Some errors occurred while fetching repomd", "err", err)
	}
	if len(results) == 0 {
		return nil, xerrors.Errorf("Failed to fetch repomd.xml. URLs: %s", mirrors)
	}

	for _, r := range results {
		var repoMd repoMd
		if err := xml.NewDecoder(bytes.NewBuffer(r.Body)).Decode(&repoMd); err != nil {
			log15.Warn("Failed to decode repomd. Trying another mirror", "err", err)
			continue
		}

		for _, repo := range repoMd.RepoList {
			if repo.Type == "updateinfo" {
				u, err := url.Parse(r.Target)
				if err != nil {
					return nil, err
				}
				u.Path = path.Join(u.Path, repo.Location.Href)
				updateInfoURLs = append(updateInfoURLs, u.String())
				break
			}
		}
	}
	if len(updateInfoURLs) == 0 {
		return nil, errNoUpdateInfo
	}
	return updateInfoURLs, nil
}

func fetchUpdateInfo(url string) (*models.Updates, error) {
	results, err := util.FetchFeedFiles([]util.FetchRequest{{URL: url, MIMEType: util.MIMETypeXML}})
	if err != nil || len(results) != 1 {
		return nil, xerrors.Errorf("Failed to fetch updateInfo. err: %w", err)
	}
	r, err := gzip.NewReader(bytes.NewBuffer(results[0].Body))
	if err != nil {
		return nil, xerrors.Errorf("Failed to decompress updateInfo. err: %w", err)
	}
	defer r.Close()

	var updateInfo models.Updates
	if err := xml.NewDecoder(r).Decode(&updateInfo); err != nil {
		return nil, err
	}
	for i, alas := range updateInfo.UpdateList {
		cveIDs := []string{}
		for _, ref := range alas.References {
			if ref.Type == "cve" {
				cveIDs = append(cveIDs, ref.ID)
			}
		}
		updateInfo.UpdateList[i].CVEIDs = cveIDs
	}
	return &updateInfo, nil
}
