package nanopush

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"golang.org/x/net/http2"
)

// Doer is ostensibly an *http.Client
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

const (
	Development     = "https://api.development.push.apple.com"
	Development2197 = "https://api.development.push.apple.com:2197"
	Production      = "https://api.push.apple.com"
	Production2197  = "https://api.push.apple.com:2197"
)

// Provider sends pushes to Apple's APNs servers.
type Provider struct {
	client     Doer
	expiration time.Duration
	workers    int
	baseURL    string
}

// JSONPushError is a JSON error returned from the APNs service.
type JSONPushError struct {
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

func (e *JSONPushError) Error() string {
	s := "APNs push error"
	if e == nil {
		return s + ": nil"
	}
	if e.Reason != "" {
		s += ": " + e.Reason
	}
	if e.Timestamp > 0 {
		s += ": timestamp " + strconv.FormatInt(e.Timestamp, 10)
	}
	return s
}

func newError(body io.Reader, statusCode int) error {
	var err error = new(JSONPushError)
	if decodeErr := json.NewDecoder(body).Decode(err); decodeErr != nil {
		err = fmt.Errorf("decoding JSON push error: %w", decodeErr)
	}
	return fmt.Errorf("push HTTP status: %d: %w", statusCode, err)
}

// do performs the HTTP push request
func (p *Provider) do(ctx context.Context, pushInfo *mdm.Push) *push.Response {
	jsonPayload := []byte(`{"mdm":"` + pushInfo.PushMagic + `"}`)

	url := p.baseURL + "/3/device/" + pushInfo.Token.String()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonPayload))

	if err != nil {
		return &push.Response{Err: err}
	}

	req.Header.Set("Content-Type", "application/json")
	if p.expiration > 0 {
		exp := time.Now().Add(p.expiration)
		req.Header.Set("apns-expiration", strconv.FormatInt(exp.Unix(), 10))
	}
	r, err := p.client.Do(req)
	var goAwayErr http2.GoAwayError
	if errors.As(err, &goAwayErr) {
		body := strings.NewReader(goAwayErr.DebugData)
		return &push.Response{Err: newError(body, r.StatusCode)}
	} else if err != nil {
		return &push.Response{Err: err}
	}

	defer r.Body.Close()
	response := &push.Response{Id: r.Header.Get("apns-id")}
	if r.StatusCode != http.StatusOK {
		response.Err = newError(r.Body, r.StatusCode)
	}
	return response
}

// pushSerial performs APNs pushes serially.
func (p *Provider) pushSerial(ctx context.Context, pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	ret := make(map[string]*push.Response)
	for _, pushInfo := range pushInfos {
		if pushInfo == nil {
			continue
		}
		ret[pushInfo.Token.String()] = p.do(ctx, pushInfo)
	}
	return ret, nil
}

// pushConcurrent performs APNs pushes concurrently.
// It spawns worker goroutines and feeds them from the list of pushInfos.
func (p *Provider) pushConcurrent(ctx context.Context, pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	// don't start more workers than we have pushes to send
	workers := p.workers
	if len(pushInfos) > workers {
		workers = len(pushInfos)
	}

	// response associates push.Response with token
	type response struct {
		token    string
		response *push.Response
	}

	jobs := make(chan *mdm.Push)
	results := make(chan response)
	var wg sync.WaitGroup

	// start our workers
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for pushInfo := range jobs {
				results <- response{
					token:    pushInfo.Token.String(),
					response: p.do(ctx, pushInfo),
				}
			}
		}()
	}

	// start the "feeder" (queue source)
	go func() {
		for _, pushInfo := range pushInfos {
			jobs <- pushInfo
		}
		close(jobs)
	}()

	// watch for our workers finishing (they should after feeding is done)
	// stop the collector when the workers have finished.
	go func() {
		wg.Wait()
		close(results)
	}()

	// collect our results
	ret := make(map[string]*push.Response)
	for r := range results {
		ret[r.token] = r.response
	}

	return ret, nil
}

// Push sends APNs pushes to MDM enrollments.
func (p *Provider) Push(ctx context.Context, pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	if len(pushInfos) < 1 {
		return nil, errors.New("no push data provided")
	} else if len(pushInfos) == 1 {
		return p.pushSerial(ctx, pushInfos)
	}
	return p.pushConcurrent(ctx, pushInfos)
}
