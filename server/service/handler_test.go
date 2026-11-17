package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

func TestAPIRoutesConflicts(t *testing.T) {
	ds := new(mock.Store)

	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	cfg := config.TestConfig()
	h, err := MakeHandler(svc, cfg, slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)
	router := h.(interface{ Router() *endpointer.Router }).Router()

	type testCase struct {
		name string
		path string
		verb string
		want int
	}
	var cases []testCase

	// Build the test cases: for each route, generate a request designed to match
	// it, and override its handler to return a unique status code. If the
	// request doesn't result in that status code, then some other route
	// conflicts with it and took precedence - a route conflict. The route's name
	// is used to name the sub-test for that route.
	status := 200
	router.WrapHandlers(func(rt endpointer.Route, h http.Handler) http.Handler {
		_, path, err := routeTestRequest(rt)
		require.NoError(t, err)
		if path == "" {
			return h
		}
		for _, meth := range rt.Methods {
			cases = append(cases, testCase{name: rt.Name, path: path, verb: meth, want: status})
		}
		want := status
		status++
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(want) })
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Log(c.verb, c.path)
			req := httptest.NewRequest(c.verb, c.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			require.Equal(t, c.want, rr.Code)
		})
	}
}

func TestAPIRoutesMetrics(t *testing.T) {
	ds := new(mock.Store)

	svc, _ := newTestService(t, ds, nil, nil)
	limitStore, _ := memstore.New(0)
	h, err := MakeHandler(svc, config.TestConfig(), slog.New(slog.DiscardHandler), limitStore, nil, nil, nil)
	require.NoError(t, err)
	router := h.(interface{ Router() *endpointer.Router }).Router()

	// collect the requests to make to each route, then replace all handlers with mocks
	var reqs []*http.Request
	for _, rt := range router.Routes() {
		verb, path, err := routeTestRequest(rt)
		require.NoError(t, err)
		if path != "" {
			reqs = append(reqs, httptest.NewRequest(verb, path, nil))
		}
	}
	router.WrapHandlers(func(endpointer.Route, http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	})

	// wrap the handlers with the metric handlers
	addMetrics(router)

	// add the handler that returns the metrics, itself instrumented
	router.Handle(endpointer.Route{Name: "metrics", Path: "/metrics"}, promhttp.Handler())

	// collect the route names
	routeNames := make(map[string]bool)
	for _, rt := range router.Routes() {
		if routeNames[rt.Name] {
			t.Errorf("duplicate route name: %s (%s)", rt.Name, rt.Template())
		}
		routeNames[rt.Name] = true
	}

	// make the requests to each route
	for _, req := range reqs {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// get the metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rxMetric := regexp.MustCompile(`^([\w_]+)({[^}]+})? (.+)$`)
	rxHandler := regexp.MustCompile(`handler="([\w_]+)"`)

	// expected metric names and their counts
	metricCounts := map[string]int{
		"go_gc_duration_seconds":                     0,
		"go_gc_duration_seconds_sum":                 0,
		"go_gc_duration_seconds_count":               0,
		"go_gc_gogc_percent":                         0,
		"go_gc_gomemlimit_bytes":                     0,
		"go_sched_gomaxprocs_threads":                0,
		"go_goroutines":                              0,
		"go_info":                                    0,
		"go_memstats_alloc_bytes":                    0,
		"go_memstats_alloc_bytes_total":              0,
		"go_memstats_buck_hash_sys_bytes":            0,
		"go_memstats_frees_total":                    0,
		"go_memstats_gc_cpu_fraction":                0,
		"go_memstats_gc_sys_bytes":                   0,
		"go_memstats_heap_alloc_bytes":               0,
		"go_memstats_heap_idle_bytes":                0,
		"go_memstats_heap_inuse_bytes":               0,
		"go_memstats_heap_objects":                   0,
		"go_memstats_heap_released_bytes":            0,
		"go_memstats_heap_sys_bytes":                 0,
		"go_memstats_last_gc_time_seconds":           0,
		"go_memstats_lookups_total":                  0,
		"go_memstats_mallocs_total":                  0,
		"go_memstats_mcache_inuse_bytes":             0,
		"go_memstats_mcache_sys_bytes":               0,
		"go_memstats_mspan_inuse_bytes":              0,
		"go_memstats_mspan_sys_bytes":                0,
		"go_memstats_next_gc_bytes":                  0,
		"go_memstats_other_sys_bytes":                0,
		"go_memstats_stack_inuse_bytes":              0,
		"go_memstats_stack_sys_bytes":                0,
		"go_memstats_sys_bytes":                      0,
		"go_threads":                                 0,
		"http_request_duration_seconds_bucket":       0,
		"http_request_duration_seconds_sum":          0,
		"http_request_duration_seconds_count":        0,
		"http_request_size_bytes_bucket":             0,
		"http_request_size_bytes_sum":                0,
		"http_request_size_bytes_count":              0,
		"http_requests_total":                        0,
		"http_response_size_bytes_bucket":            0,
		"http_response_size_bytes_sum":               0,
		"http_response_size_bytes_count":             0,
		"process_cpu_seconds_total":                  0,
		"process_max_fds":                            0,
		"process_open_fds":                           0,
		"process_resident_memory_bytes":              0,
		"process_start_time_seconds":                 0,
		"process_virtual_memory_bytes":               0,
		"process_virtual_memory_max_bytes":           0,
		"promhttp_metric_handler_requests_in_flight": 0,
		"promhttp_metric_handler_requests_total":     0,
	}

	wantCounts := map[string]int{
		"go_gc_duration_seconds":                     5, // quantiles 0, .25, .5, .75 and 1
		"go_gc_duration_seconds_sum":                 1,
		"go_gc_duration_seconds_count":               1,
		"go_goroutines":                              1,
		"go_gc_gogc_percent":                         1,
		"go_gc_gomemlimit_bytes":                     1,
		"go_sched_gomaxprocs_threads":                1,
		"go_info":                                    1,
		"go_memstats_alloc_bytes":                    1,
		"go_memstats_alloc_bytes_total":              1,
		"go_memstats_buck_hash_sys_bytes":            1,
		"go_memstats_frees_total":                    1,
		"go_memstats_gc_cpu_fraction":                0, // does not appear to be reported anymore
		"go_memstats_gc_sys_bytes":                   1,
		"go_memstats_heap_alloc_bytes":               1,
		"go_memstats_heap_idle_bytes":                1,
		"go_memstats_heap_inuse_bytes":               1,
		"go_memstats_heap_objects":                   1,
		"go_memstats_heap_released_bytes":            1,
		"go_memstats_heap_sys_bytes":                 1,
		"go_memstats_last_gc_time_seconds":           1,
		"go_memstats_lookups_total":                  0, // does not appear to be reported anymore
		"go_memstats_mallocs_total":                  1,
		"go_memstats_mcache_inuse_bytes":             1,
		"go_memstats_mcache_sys_bytes":               1,
		"go_memstats_mspan_inuse_bytes":              1,
		"go_memstats_mspan_sys_bytes":                1,
		"go_memstats_next_gc_bytes":                  1,
		"go_memstats_other_sys_bytes":                1,
		"go_memstats_stack_inuse_bytes":              1,
		"go_memstats_stack_sys_bytes":                1,
		"go_memstats_sys_bytes":                      1,
		"go_threads":                                 1,
		"http_request_duration_seconds_bucket":       len(reqs) * (len(prometheus.DefBuckets) + 1), // +1 for the last bucket, ending at +Inf
		"http_request_duration_seconds_sum":          len(reqs),
		"http_request_duration_seconds_count":        len(reqs),
		"http_request_size_bytes_bucket":             len(reqs) * 6, // size of req size buckets
		"http_request_size_bytes_sum":                len(reqs),
		"http_request_size_bytes_count":              len(reqs),
		"http_requests_total":                        len(reqs),
		"http_response_size_bytes_bucket":            len(reqs) * 6, // size of res size buckets
		"http_response_size_bytes_sum":               len(reqs),
		"http_response_size_bytes_count":             len(reqs),
		"process_cpu_seconds_total":                  1,
		"process_max_fds":                            1,
		"process_open_fds":                           1,
		"process_resident_memory_bytes":              1,
		"process_start_time_seconds":                 1,
		"process_virtual_memory_bytes":               1,
		"process_virtual_memory_max_bytes":           1,
		"promhttp_metric_handler_requests_in_flight": 1,
		"promhttp_metric_handler_requests_total":     3, // status codes 200, 500, 503
	}

	s := bufio.NewScanner(rr.Body)
	for s.Scan() {
		line := s.Text()

		// line must be one of those options, which is the prometheus format
		matches := rxMetric.FindStringSubmatch(line)
		switch {
		case strings.HasPrefix(line, "# TYPE "),
			strings.HasPrefix(line, "# HELP "):
			// that's fine, metadata about the metric

		case len(matches) > 0:
			_, ok := metricCounts[matches[1]]
			if !ok {
				// Some metrics may be environment-specific.
				// If it's something we want to track, we can add it to the
				// metricCounts map, but for now, we just ignore it.
				continue
			}
			metricCounts[matches[1]]++

			// if there are dimensions or labels associated with the metric, check
			// if there is a handler name.
			if len(matches) > 3 {
				labels := matches[2]
				if handlerMatches := rxHandler.FindStringSubmatch(labels); len(handlerMatches) > 0 {
					require.True(t, routeNames[handlerMatches[1]], "unexpected handler route name: %s: %s", matches[1], handlerMatches[1])
				}
			}

			// the last capture is the value, which must be parsable as a float
			val := matches[len(matches)-1]
			_, err := strconv.ParseFloat(val, 64)
			require.NoError(t, err, "value must be a valid float: %s", matches[1])

		default:
			require.Fail(t, "invalid line", line)
		}
	}
	require.NoError(t, s.Err())

	for name, got := range metricCounts {
		want, ok := wantCounts[name]
		require.True(t, ok, "unexpected metric: %s", name)
		require.Equal(t, want, got, name)
	}
}

var reSimpleVar, reNumVar = regexp.MustCompile(`\{(\w+)\}`), regexp.MustCompile(`\{\w+:[^\}]+\}`)

// routeTestRequest returns a method and a concrete path that match rt. The path is empty for a route without methods.
func routeTestRequest(rt endpointer.Route) (verb, path string, err error) {
	if len(rt.Methods) == 0 {
		// only route without method is distributed_query_results (websocket)
		if rt.Name != "distributed_query_results" {
			return "", "", fmt.Errorf("%s %s: route has no methods", rt.Name, rt.Path)
		}
		return "", "", nil
	}

	path = reSimpleVar.ReplaceAllString(rt.Template(), "$1")
	// for now at least, the only times we use regexp-constrained vars is
	// for numeric arguments or the fleetversion specifier.
	path = reNumVar.ReplaceAllStringFunc(path, func(s string) string {
		if strings.Contains(s, "fleetversion") {
			parts := strings.Split(strings.TrimPrefix(s, "{fleetversion:(?:"), "|")
			// test with "latest" if not deprecated, or last supported version for that route
			// (for either case, this will be in the last part)
			return strings.TrimSuffix(parts[len(parts)-1], ")}")
		}
		return "1"
	})

	return rt.Methods[0], path, nil
}

func TestGzipResponses(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	gzipTestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Write enough data to trigger gzip (default threshold is 1500 bytes)
		data := make([]byte, 2000)
		for i := range data {
			data[i] = 'a'
		}
		// The response body is what the assertions inspect; a write error surfaces there.
		_, _ = w.Write(data)
	})
	testRoute := func(r *endpointer.Router, opts []kithttp.ServerOption) {
		r.Handle(endpointer.Route{Name: "test_gzip", Path: "/api/test-gzip"}, gzipTestHandler)
		// A method-scoped route is served by the stdlib fast path rather than gorilla, which applies the router middleware
		// itself, so gzip has to be exercised on both.
		r.Handle(endpointer.Route{Name: "test_gzip_fast", Methods: []string{"GET"}, Path: "/api/test-gzip-fast"}, gzipTestHandler)
	}

	t.Run("Enabled", func(t *testing.T) {
		cfg := config.TestConfig()
		cfg.Server.GzipResponses = true

		_, server := RunServerForTestsWithDS(t, ds, &TestServerOpts{
			FleetConfig:         &cfg,
			FeatureRoutes:       []endpointer.HandlerRoutesFunc{testRoute},
			SkipCreateTestUsers: true,
		})
		defer server.Close()

		for _, path := range []string{"/api/test-gzip", "/api/test-gzip-fast"} {
			t.Run("WithAcceptEncoding "+path, func(t *testing.T) {
				req, err := http.NewRequest("GET", server.URL+path, nil)
				require.NoError(t, err)
				req.Header.Set("Accept-Encoding", "gzip")
				resp, err := fleethttp.NewClient().Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"), "Expected gzip Content-Encoding when enabled")
			})
		}

		t.Run("WithoutAcceptEncoding", func(t *testing.T) {
			req, err := http.NewRequest("GET", server.URL+"/api/test-gzip", nil)
			require.NoError(t, err)
			// Do NOT set Accept-Encoding header
			transport := fleethttp.NewTransport()
			transport.DisableCompression = true // Prevents automatic addition of Accept-Encoding: gzip
			client := fleethttp.NewClient()
			client.Transport = transport
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Empty(t, resp.Header.Get("Content-Encoding"), "Expected no gzip Content-Encoding when Accept-Encoding not set")
		})
	})

	t.Run("Disabled", func(t *testing.T) {
		t.Parallel()
		cfg := config.TestConfig()
		cfg.Server.GzipResponses = false
		_, server := RunServerForTestsWithDS(t, ds, &TestServerOpts{
			FleetConfig:         &cfg,
			FeatureRoutes:       []endpointer.HandlerRoutesFunc{testRoute},
			SkipCreateTestUsers: true,
		})
		defer server.Close()

		req, err := http.NewRequest("GET", server.URL+"/api/test-gzip", nil)
		require.NoError(t, err)
		req.Header.Set("Accept-Encoding", "gzip")
		resp, err := fleethttp.NewClient().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Empty(t, resp.Header.Get("Content-Encoding"), "Expected no gzip Content-Encoding when disabled")
	})
}
