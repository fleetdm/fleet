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
	"time"

	"github.com/facebookincubator/nvdtools/providers/lib/rate"
)

// WithThrottling creates a rate limitted client - all requests are throttled
func WithThrottling(c Client, period time.Duration, requestsPerPeriod int) Client {
	limiter := rate.BurstyLimiter(period, requestsPerPeriod)
	return &executorClient{c, &rateLimitedExecutor{limiter}}
}

type rateLimitedExecutor struct {
	rate.Limiter
}

func (e *rateLimitedExecutor) execute(f func() (*http.Response, error)) (*http.Response, error) {
	e.Limiter.Allow() // block until we can make another request
	return f()
}
