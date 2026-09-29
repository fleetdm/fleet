package main

import (
	"net/http/httptest"
	"testing"
)

// BenchmarkDeviceGetParallel drives the device lookup route, the mock's hot path under load.
// It is a read on the store, so concurrent callers must not serialize: anything that puts a
// store-wide writer on this path costs the load test throughput.
func BenchmarkDeviceGetParallel(b *testing.B) {
	store := newDeviceStore()
	mux := newMux(store, nil, 0, 0)
	path := "/v1/enterprises/" + testEnterpriseID + "/devices/fakedevice"

	// Prime the enterprise so this measures steady state rather than first sight.
	mux.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// RunParallel calls this once per worker, so the request below belongs to one
		// goroutine. It is built outside the loop deliberately: the point of measurement is
		// lock contention on the store, not request construction.
		req := httptest.NewRequest("GET", path, nil)
		for pb.Next() {
			mux.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
