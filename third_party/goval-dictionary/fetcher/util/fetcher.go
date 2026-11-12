package util

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/viper"
	"github.com/ulikunitz/xz"
	"golang.org/x/xerrors"

	"github.com/vulsio/goval-dictionary/config"
)

// MIMEType :
type MIMEType int

const (
	// MIMETypeUnknown :
	MIMETypeUnknown MIMEType = iota
	// MIMETypeXML :
	MIMETypeXML
	// MIMETypeTxt :
	MIMETypeTxt
	// MIMETypeJSON :
	MIMETypeJSON
	// MIMETypeYml :
	MIMETypeYml
	// MIMETypeHTML :
	MIMETypeHTML
	// MIMETypeBzip2 :
	MIMETypeBzip2
	// MIMETypeXz :
	MIMETypeXz
	// MIMETypeGzip :
	MIMETypeGzip
	// MIMETypeZst :
	MIMETypeZst
)

func (m MIMEType) String() string {
	switch m {
	case MIMETypeXML:
		return "xml"
	case MIMETypeTxt:
		return "txt"
	case MIMETypeJSON:
		return "json"
	case MIMETypeYml:
		return "yml"
	case MIMETypeHTML:
		return "html"
	case MIMETypeBzip2:
		return "bzip2"
	case MIMETypeXz:
		return "xz"
	case MIMETypeGzip:
		return "gz"
	case MIMETypeZst:
		return "zst"
	default:
		return "Unknown"
	}
}

// FetchRequest has url, mimetype and fetch option
type FetchRequest struct {
	Target        string
	URL           string
	MIMEType      MIMEType
	LogSuppressed bool
}

// FetchResult has url and OVAL definitions
type FetchResult struct {
	Target        string
	URL           string
	Body          []byte
	LogSuppressed bool
}

// genWorkers generate workers
func genWorkers(num int) chan<- func() {
	tasks := make(chan func())
	for i := 0; i < num; i++ {
		go func() {
			for f := range tasks {
				f()
			}
		}()
	}
	return tasks
}

// FetchFeedFiles :
func FetchFeedFiles(reqs []FetchRequest) (results []FetchResult, err error) {
	reqChan := make(chan FetchRequest, len(reqs))
	resChan := make(chan FetchResult, len(reqs))
	errChan := make(chan error, len(reqs))
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	for _, r := range reqs {
		if !r.LogSuppressed {
			log15.Info("Fetching... ", "URL", r.URL)
		}
	}

	go func() {
		for _, r := range reqs {
			reqChan <- r
		}
	}()

	concurrency := len(reqs)
	tasks := genWorkers(concurrency)
	wg := new(sync.WaitGroup)
	for range reqs {
		wg.Add(1)
		tasks <- func() {
			req := <-reqChan
			body, err := fetchFileWithUA(req)
			wg.Done()
			if err != nil {
				errChan <- err
				return
			}
			resChan <- FetchResult{
				Target:        req.Target,
				URL:           req.URL,
				Body:          body,
				LogSuppressed: req.LogSuppressed,
			}
		}
	}
	wg.Wait()

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range reqs {
		select {
		case res := <-resChan:
			results = append(results, res)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return results, fmt.Errorf("Timeout Fetching")
		}
	}
	if 0 < len(errs) {
		return results, fmt.Errorf("%s", errs)
	}
	return results, nil
}

func fetchFileWithUA(req FetchRequest) (body []byte, err error) {
	var proxyURL *url.URL
	var resp *http.Response

	httpClient := &http.Client{}
	httpProxy := viper.GetString("http-proxy")
	if httpProxy != "" {
		if proxyURL, err = url.Parse(httpProxy); err != nil {
			return nil, xerrors.Errorf("Failed to parse proxy url. err: %w", err)
		}
		httpClient = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	}

	httpreq, err := http.NewRequest("GET", req.URL, nil)
	if err != nil {
		return nil, xerrors.Errorf("Failed to download. err: %w", err)
	}

	httpreq.Header.Set("User-Agent", fmt.Sprintf("goval-dictionary/%s.%s", config.Version, config.Revision))
	resp, err = httpClient.Do(httpreq)
	if err != nil {
		return nil, xerrors.Errorf("Failed to download. err: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to HTTP GET. url: %s, response: %+v", req.URL, resp)
	}

	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, err
	}

	var b bytes.Buffer
	switch req.MIMEType {
	case MIMETypeXML, MIMETypeTxt, MIMETypeJSON, MIMETypeYml, MIMETypeHTML:
		b = buf
	case MIMETypeBzip2:
		if _, err := b.ReadFrom(bzip2.NewReader(bytes.NewReader(buf.Bytes()))); err != nil {
			return nil, xerrors.Errorf("Failed to open bzip2 file. err: %w", err)
		}
	case MIMETypeXz:
		r, err := xz.NewReader(bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, xerrors.Errorf("Failed to open xz file. err: %w", err)
		}
		if _, err = b.ReadFrom(r); err != nil {
			return nil, xerrors.Errorf("Failed to read xz file. err: %w", err)
		}
	case MIMETypeGzip:
		r, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, xerrors.Errorf("Failed to open gzip file. err: %w", err)
		}
		if _, err = b.ReadFrom(r); err != nil {
			return nil, xerrors.Errorf("Failed to read gzip file. err: %w", err)
		}
	case MIMETypeZst:
		r, err := zstd.NewReader(bytes.NewReader(buf.Bytes()))
		if err != nil {
			return nil, xerrors.Errorf("Failed to open zstd file. err: %w", err)
		}
		if _, err = b.ReadFrom(r); err != nil {
			return nil, xerrors.Errorf("Failed to read zstd file. err: %w", err)
		}
	default:
		return nil, xerrors.Errorf("unexpected request MIME Type. expected: %q, actual: %q", []MIMEType{MIMETypeXML, MIMETypeTxt, MIMETypeJSON, MIMETypeYml, MIMETypeHTML, MIMETypeBzip2, MIMETypeXz, MIMETypeGzip, MIMETypeZst}, req.MIMEType)
	}

	return b.Bytes(), nil
}
