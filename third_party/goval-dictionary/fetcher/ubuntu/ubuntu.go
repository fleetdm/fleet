package ubuntu

import (
	"fmt"
	"strings"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/config"
	"github.com/vulsio/goval-dictionary/fetcher/util"
)

func newFetchRequests(target []string) (reqs []util.FetchRequest) {
	for _, v := range target {
		switch url := getOVALURL(v); url {
		case "unknown":
			log15.Warn("Skip unknown ubuntu.", "version", v)
		case "unsupported":
			log15.Warn("Skip unsupported ubuntu version.", "version", v)
			log15.Warn("See https://wiki.ubuntu.com/Releases for supported versions")
		default:
			reqs = append(reqs, util.FetchRequest{
				Target:   v,
				URL:      url,
				MIMEType: util.MIMETypeBzip2,
			})
		}
	}
	return
}

func getOVALURL(version string) string {
	major, minor, ok := strings.Cut(version, ".")
	if !ok {
		return "unknown"
	}

	const main = "https://security-metadata.canonical.com/oval/oci.com.ubuntu.%s.cve.oval.xml.bz2"
	switch major {
	case "4", "5", "6", "7", "8", "9", "10", "11", "12":
		return "unsupported"
	case "14":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu1404)
		case "10":
			return "unsupported"
		default:
			return "unknown"
		}
	case "16":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu1604)
		case "10":
			return "unsupported"
		default:
			return "unknown"
		}
	case "17":
		return "unsupported"
	case "18":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu1804)
		case "10":
			return "unsupported"
		default:
			return "unknown"
		}
	case "19":
		return "unsupported"
	case "20":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu2004)
		case "10":
			return "unsupported"
		default:
			return "unknown"
		}
	case "21":
		return "unsupported"
	case "22":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu2204)
		case "10":
			return "unsupported"
		default:
			return "unknown"
		}
	case "23":
		return "unsupported"
	case "24":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu2404)
		case "10":
			return fmt.Sprintf(main, config.Ubuntu2410)
		default:
			return "unknown"
		}
	case "25":
		switch minor {
		case "04":
			return fmt.Sprintf(main, config.Ubuntu2504)
		default:
			return "unknown"
		}
	default:
		return "unknown"
	}
}

// FetchFiles fetch OVAL from Ubuntu
func FetchFiles(versions []string) ([]util.FetchResult, error) {
	reqs := newFetchRequests(versions)
	if len(reqs) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}

	results := make([]util.FetchResult, 0, len(reqs))
	for _, req := range reqs {
		rs, err := util.FetchFeedFiles([]util.FetchRequest{req})
		if err != nil {
			return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
		}
		results = append(results, rs...)
	}
	return results, nil
}
