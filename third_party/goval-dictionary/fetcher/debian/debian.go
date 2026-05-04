package debian

import (
	"fmt"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/fetcher/util"
)

// https://www.debian.org/security/oval/
func newFetchRequests(target []string) (reqs []util.FetchRequest) {
	const t = "https://www.debian.org/security/oval/oval-definitions-%s.xml.bz2"
	for _, v := range target {
		var name string
		if name = debianName(v); name == "unknown" {
			log15.Warn("Skip unknown debian.", "version", v)
			continue
		}
		reqs = append(reqs, util.FetchRequest{
			Target:   v,
			URL:      fmt.Sprintf(t, name),
			MIMEType: util.MIMETypeBzip2,
		})
	}
	return
}

func debianName(major string) string {
	switch major {
	case "7":
		return config.Debian7
	case "8":
		return config.Debian8
	case "9":
		return config.Debian9
	case "10":
		return config.Debian10
	case "11":
		return config.Debian11
	case "12":
		return config.Debian12
	case "13":
		return config.Debian13
	default:
		return "unknown"
	}
}

// FetchFiles fetch OVAL from Debian
func FetchFiles(versions []string) ([]util.FetchResult, error) {
	reqs := newFetchRequests(versions)
	if len(reqs) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}
	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return results, nil
}
