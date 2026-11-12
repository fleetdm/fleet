package oracle

import (
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/fetcher/util"
)

func newFetchRequests() (reqs []util.FetchRequest) {
	const t = "https://linux.oracle.com/security/oval/com.oracle.elsa-all.xml.bz2"
	reqs = append(reqs, util.FetchRequest{
		URL:      t,
		MIMEType: util.MIMETypeBzip2,
	})
	return
}

// FetchFiles fetch OVAL from Oracle
func FetchFiles() ([]util.FetchResult, error) {
	reqs := newFetchRequests()
	if len(reqs) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}
	results, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return results, nil
}
