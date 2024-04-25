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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
)

var userAgent = "nvdsync-" + Version

// http helpers

func httpNewRequestContext(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent())
	return req.WithContext(ctx), nil
}

func httpResponseNotOK(resp *http.Response) error {
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if err != nil {
		return err
	}
	return fmt.Errorf("unexpected http response from %q (%q): %q",
		resp.Request.URL.String(), resp.Status, string(body))
}

// SetUserAgent sets the value of User-Agent HTTP header for the client
func SetUserAgent(ua string) error {
	if !regexp.MustCompile("^[[:ascii:]]+$").MatchString(ua) {
		return fmt.Errorf("non-ascii character in User-Agent header: %q", ua)
	}
	userAgent = ua
	return nil
}

// UserAgent returns the value of User-Agent HTTP header used by the client
func UserAgent() string {
	return userAgent
}
