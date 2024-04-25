// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/facebookincubator/flog"
)

var debug struct {
	// Print HTTP requests and responses to stderr.
	traceRequests bool
	// Print the bodies of HTTP requests and responses to stderr.
	traceRequestBodies bool
	// When tools issue concurrent GET requests, the normal behaviour is to
	// cancel pending requests as soon as one request fails. This option
	// restores the old behaviour or executing the remaning requests anyway.
	continueDownloading bool
	// When set to a number n, the n th HTTP request will fail.
	failRequestNum uint64

	requestNum uint64
}

func getBool(varName string) bool {
	v, _ := strconv.ParseBool(os.Getenv(varName))
	return v
}

func getUint(varName string, defaultValue uint64) uint64 {
	v, err := strconv.ParseUint(os.Getenv(varName), 10, 64)
	if err != nil {
		return defaultValue
	}
	return v
}

func init() {
	debug.traceRequests = getBool("NVD_TRACE_REQUESTS")
	debug.traceRequestBodies = getBool("NVD_TRACE_REQUEST_BODIES")
	debug.continueDownloading = getBool("NVD_CONTINUE_DOWNLOADING")
	debug.failRequestNum = getUint("NVD_FAIL_REQUEST", 0)
}

func obfuscateHeaders(req *http.Request) *http.Request {
	authHeaders := []string{
		"Authorization",
		// fireeye
		"X-Auth",
		"X-Auth-Hash",
		// idefense
		"Auth-Token",
	}

	headers := req.Header.Clone()
	for _, header := range authHeaders {
		if headers.Get(header) == "" {
			continue
		}
		headers.Set(header, "<obfuscated>")
	}

	// A shallow copy is enough for this usage.
	newReq := *req
	newReq.Header = headers
	return &newReq
}

func traceRequestStart(req *http.Request) uint64 {
	id := atomic.AddUint64(&debug.requestNum, 1)
	if !debug.traceRequests {
		return id
	}
	data, _ := httputil.DumpRequest(obfuscateHeaders(req), debug.traceRequestBodies)
	fmt.Fprintf(os.Stderr, "Req %d: %s", id, string(data))
	return id
}

func traceRequestEnd(id uint64, resp *http.Response) {
	if !debug.traceRequests {
		return
	}
	if resp == nil {
		return
	}
	data, _ := httputil.DumpResponse(resp, debug.traceRequestBodies)
	fmt.Fprintf(os.Stderr, "Req %d: %s", id, string(data))
}

// StopOrContinue can help controlling the behaviour of concurrent GET requests
// when using an errgroup and encountering an error. Depending on the
// NVD_CONTINUE_DOWNLOADING env variable, this function will return the passed
// error (when we want to stop pending requests) or just log the error (when we
// want the pending requests to continue being processed).
func StopOrContinue(err error) error {
	if debug.continueDownloading {
		flog.Errorln(err)
		return nil
	}
	return err
}
