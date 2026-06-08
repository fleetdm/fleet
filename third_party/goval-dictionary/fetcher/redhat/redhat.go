package redhat

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/inconshreveable/log15"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/fetcher/util"
)

// FetchFiles fetch OVAL from RedHat
func FetchFiles(versions []string) (map[string][]util.FetchResult, error) {
	results := map[string][]util.FetchResult{}
	for _, v := range versions {
		switch v {
		case "1", "2", "3":
			log15.Warn("Skip redhat because no vulnerability information provided.", "version", v)
		case "4":
			rs, err := fetchOVALv1([]string{fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv1. err: %w", err)
			}
			results[v] = rs
		case "5":
			rs, err := fetchOVALv1([]string{fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v), fmt.Sprintf("com.redhat.rhsa-RHEL%s-ELS.xml", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv1. err: %w", err)
			}
			results[v] = rs
		case "6":
			rs, err := fetchOVALv1([]string{fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv1. err: %w", err)
			}
			results[v] = rs

			rs, err = fetchOVALv2([]string{fmt.Sprintf("%s-including-unpatched", v), fmt.Sprintf("%s-extras-including-unpatched", v), fmt.Sprintf("%s-supplementary", v), fmt.Sprintf("%s-els", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv2. err: %w", err)
			}
			results[v] = append(results[v], rs...)
		case "7":
			rs, err := fetchOVALv1([]string{fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv1. err: %w", err)
			}
			results[v] = rs

			rs, err = fetchOVALv2([]string{fmt.Sprintf("%s-including-unpatched", v), fmt.Sprintf("%s-extras-including-unpatched", v), fmt.Sprintf("%s-supplementary", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv2. err: %w", err)
			}
			results[v] = append(results[v], rs...)
		case "8", "9":
			rs, err := fetchOVALv1([]string{fmt.Sprintf("com.redhat.rhsa-RHEL%s.xml", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv1. err: %w", err)
			}
			results[v] = rs

			rs, err = fetchOVALv2([]string{fmt.Sprintf("%s-including-unpatched", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv2. err: %w", err)
			}
			results[v] = append(results[v], rs...)
		default:
			if _, err := strconv.Atoi(v); err != nil {
				log15.Warn("Skip unknown redhat.", "version", v)
				break
			}

			rs, err := fetchOVALv2([]string{fmt.Sprintf("%s-including-unpatched", v)})
			if err != nil {
				return nil, xerrors.Errorf("Failed to fetch OVALv2. err: %w", err)
			}
			results[v] = rs
		}
	}

	if len(results) == 0 {
		return nil, xerrors.New("There are no versions to fetch")
	}
	return results, nil
}

func fetchOVALv1(names []string) ([]util.FetchResult, error) {
	rs, err := util.FetchFeedFiles([]util.FetchRequest{{
		Target:   "oval_v1_20230706.tar.gz",
		URL:      "https://access.redhat.com/security/data/archive/oval_v1_20230706.tar.gz",
		MIMEType: util.MIMETypeGzip,
	}})
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}

	results := make([]util.FetchResult, 0, len(names))

	tr := tar.NewReader(bytes.NewReader(rs[0].Body))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, xerrors.Errorf("Failed to next tar reader. err: %w", err)
		}

		if !slices.Contains(names, hdr.Name) {
			continue
		}

		bs, err := io.ReadAll(tr)
		if err != nil {
			return nil, xerrors.Errorf("Failed to read all %s. err: %w", hdr.Name, err)
		}
		results = append(results, util.FetchResult{
			Target: strings.TrimSuffix(strings.TrimPrefix(hdr.Name, "com.redhat.rhsa-RHEL"), ".xml"),
			URL:    fmt.Sprintf("https://access.redhat.com/security/data/archive/oval_v1_20230706.tar.gz/%s", hdr.Name),
			Body:   bs,
		})
	}

	return results, nil
}

func fetchOVALv2(names []string) ([]util.FetchResult, error) {
	reqs := make([]util.FetchRequest, 0, len(names))
	for _, n := range names {
		reqs = append(reqs, util.FetchRequest{
			Target:   n,
			URL:      fmt.Sprintf("https://access.redhat.com/security/data/oval/v2/RHEL%s/rhel-%s.oval.xml.bz2", strings.Split(n, "-")[0], n),
			MIMEType: util.MIMETypeBzip2,
		})
	}

	rs, err := util.FetchFeedFiles(reqs)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. err: %w", err)
	}
	return rs, nil
}
