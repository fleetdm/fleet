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
	"net/http"
)

type executor interface {
	execute(func() (*http.Response, error)) (*http.Response, error)
}

type executorClient struct {
	Client
	executor
}

// Do is a part of the Client interface
func (c *executorClient) Do(req *http.Request) (*http.Response, error) {
	return c.execute(func() (*http.Response, error) {
		return c.Client.Do(req)
	})
}

// Get is a part of the Client interface
func (c *executorClient) Get(url string) (*http.Response, error) {
	return c.execute(func() (*http.Response, error) {
		return c.Client.Get(url)
	})
}
