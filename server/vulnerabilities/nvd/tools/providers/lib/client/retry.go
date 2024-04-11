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
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// backoff means that on each retry, we will multiply the delay time by this
// maybe this could be configurable as well, but meh
const backoff = 2

// FailedRetries is an error returned when all retries have been exhausted
type FailedRetries int

// Error is a part of the error interface
func (fr FailedRetries) Error() string {
	return fmt.Sprintf("failed to fetch after %d retries", int(fr))
}

// RetryPolicy is a function which returns true if HTTP status should be retried
type RetryPolicy func(status int) bool

func (rp *RetryPolicy) String() string { return "" }
func (rp *RetryPolicy) Set(s string) error {
	switch s {
	case "":
		*rp = RetryNone
	case "all":
		*rp = RetryAll
	default:
		// treat it as a comma separated list of ints
		parts := strings.Split(s, ",")
		statuses := make([]int, len(parts))
		for _, part := range parts {
			status, err := strconv.Atoi(part)
			if err != nil {
				return err
			}
			statuses = append(statuses, status)
		}
		*rp = Retry(statuses...)
	}
	return nil
}

// RetryNone is a RetryPolicy which doesn't retry anything
func RetryNone(_ int) bool { return false }

// RetryAll is a RetryPolicy which retries all http statuses
func RetryAll(_ int) bool { return true }

// Retry will only retry given statuses
func Retry(statuses ...int) RetryPolicy {
	set := make(map[int]bool, len(statuses))
	for _, s := range statuses {
		set[s] = true
	}
	return func(status int) bool {
		return set[status]
	}
}

// WithRetries will retry all given requests for the specified number of times
//	- if status is 200, returns
//	- if status is covered by the retry policy and hasn't been retried the total number of times, retry
//	- otherwise, fail the request
func WithRetries(c Client, retries int, delay time.Duration, rp RetryPolicy) Client {
	if retries <= 0 {
		// if no retries, return the normal client
		return c
	}
	return &executorClient{c, &retryExecutor{retries, delay, rp}}
}

type retryExecutor struct {
	retries     int
	delay       time.Duration
	shouldRetry RetryPolicy
}

func (c *retryExecutor) execute(f func() (*http.Response, error)) (*http.Response, error) {
	delay := c.delay
	for retry := 0; retry <= c.retries; retry++ {
		resp, err := f()
		if err != nil {
			return resp, err
		}

		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		if !c.shouldRetry(resp.StatusCode) {
			// unknown status, read the error and return it
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1024*1024))
			if err != nil {
				return nil, fmt.Errorf("cannot read http response: %v", err)
			}
			return nil, &Err{resp.StatusCode, resp.Status, string(body)}
		}

		// retry if have more retries left

		if retry != c.retries {
			time.Sleep(delay)
		}
		delay *= backoff
	}
	// no more retries left
	return nil, FailedRetries(c.retries)
}
