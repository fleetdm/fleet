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

package nvd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// http test helpers

func httptestNewServer(f http.Handler) (*httptest.Server, SourceConfig) {
	ts := httptest.NewServer(f)

	tsurl, _ := url.Parse(ts.URL)
	src := SourceConfig{
		Scheme:      tsurl.Scheme,
		Host:        tsurl.Host,
		CVEFeedPath: "/",
		CPEFeedPath: "/",
	}
	return ts, src
}

func TestResponseNotOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "hello world")
	}))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	err = httpResponseNotOK(resp)
	if err == nil {
		t.Fatal("unexpected response OK")
	}

	if !strings.Contains(err.Error(), "hello world") {
		t.Fatalf("unexpected response: %q", err)
	}
}
