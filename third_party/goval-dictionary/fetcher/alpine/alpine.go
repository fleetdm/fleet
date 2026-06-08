package alpine

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/fetcher/util"
)

const community = "https://secdb.alpinelinux.org/v%s/community.yaml"
const main = "https://secdb.alpinelinux.org/v%s/main.yaml"

func newFetchRequests(target []string) (reqs []util.FetchRequest) {
	for _, v := range target {
		reqs = append(reqs, util.FetchRequest{
			Target:   v,
			URL:      fmt.Sprintf(main, v),
			MIMEType: util.MIMETypeYml,
		})

		if v != "3.2" {
			reqs = append(reqs, util.FetchRequest{
				Target:   v,
				URL:      fmt.Sprintf(community, v),
				MIMEType: util.MIMETypeYml,
			})
		}
	}
	return
}

// FetchFiles fetch from alpine secdb
// https://secdb.alpinelinux.org/
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
