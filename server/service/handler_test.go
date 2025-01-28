package service

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/gorilla/mux"
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
	h := MakeHandler(svc, cfg, kitlog.NewNopLogger(), limitStore)
	router := h.(*mux.Router)

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
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestores []*mux.Route) error {
		_, path, err := mockRouteHandler(route, status)
		if path == "" || err != nil { // failure or no method set
			return err
		}

		meths, _ := route.GetMethods()
		for _, meth := range meths {
			cases = append(cases, testCase{
				name: route.GetName(),
				path: path,
				verb: meth,
				want: status,
			})
		}

		status++
		return nil
	})
	require.NoError(t, err)

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
	h := MakeHandler(svc, config.TestConfig(), kitlog.NewNopLogger(), limitStore)
	router := h.(*mux.Router)

	// replace all handlers with mocks, and collect the requests to make to each
	// route.
	var reqs []*http.Request
	err := router.Walk(func(route *mux.Route, router *mux.Router, ancestores []*mux.Route) error {
		verb, path, err := mockRouteHandler(route, http.StatusOK)
		if path == "" || err != nil { // failure or no method set
			return err
		}
		req := httptest.NewRequest(verb, path, nil)
		reqs = append(reqs, req)
		return nil
	})
	require.NoError(t, err)

	// wrap the handlers with the metric handlers
	addMetrics(router)

	// add the handler that returns the metrics, itself instrumented
	router.Handle("/metrics", promhttp.Handler()).Name("metrics")

	// collect the route names
	routeNames := make(map[string]bool)
	err = router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		if _, ok := routeNames[route.GetName()]; ok {
			path, _ := route.GetPathTemplate()
			t.Errorf("duplicate route name: %s (%s)", route.GetName(), path)
		}
		routeNames[route.GetName()] = true
		return nil
	})
	require.NoError(t, err)

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
		"go_memstats_lookups_total":                  1,
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
			require.True(t, ok, "unexpected metric name %s", matches[1])
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

// replaces the handler of route with one that simply responds with the status
// code. Returns a verb and path that triggers this route or an error.
func mockRouteHandler(route *mux.Route, status int) (verb, path string, err error) {
	name := route.GetName()
	path, err = route.GetPathTemplate()
	if err != nil {
		// all our routes should have paths
		return "", "", fmt.Errorf("%s: %w", name, err)
	}

	meths, err := route.GetMethods()
	if err != nil || len(meths) == 0 {
		// only route without method is distributed_query_results (websocket)
		if name != "distributed_query_results" {
			return "", "", fmt.Errorf(name+" "+path+": %w", err)
		}
		return "", "", nil
	}

	path = reSimpleVar.ReplaceAllString(path, "$1")
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

	route.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
	return meths[0], path, nil
}
