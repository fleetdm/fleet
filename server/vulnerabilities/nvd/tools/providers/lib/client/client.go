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
	"context"
	"fmt"
	"net/http"
)

// Client is an interface used for making http requests
type Client interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
}

// Default returns the default http client to use
func Default() Client {
	return http.DefaultClient
}

// Get will create a GET request with given headers and call Do on the client
func Get(ctx context.Context, c Client, url string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create http get request: %v", err)
	}
	req.Header = header

	id := traceRequestStart(req)
	if debug.failRequestNum == id {
		return nil, &Err{
			Code:   503,
			Status: "Service Unavailable",
			Body:   "Request cancelled by debug feature",
		}
	}
	resp, err := c.Do(req)
	traceRequestEnd(id, resp)

	return resp, err
}

// Err encapsulates stuff from the http.Response
type Err struct {
	Code   int
	Status string
	Body   string
}

// Error is a part of the error interface
func (e *Err) Error() string {
	return fmt.Sprintf("http error %s:\n %q", e.Status, e.Body)
}
