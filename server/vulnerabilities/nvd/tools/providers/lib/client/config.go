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
	"flag"
	"fmt"
	"regexp"
	"time"
)

var userAgentRe = regexp.MustCompile("^[[:ascii:]]+$")

// Config is used to configure a client
type Config struct {
	UserAgent         string
	numRetries        int
	retryDelay        time.Duration
	retryPolicy       RetryPolicy
	requestsPerPeriod int
	period            time.Duration
}

// AddFlags adds flags used to configure a client
func (conf *Config) AddFlags() {
	flag.StringVar(&conf.UserAgent, "user-agent", conf.UserAgent, "which user agent to use when making requests")
	flag.IntVar(&conf.numRetries, "num-retries", 0, "how many times will specified statuses get retried. 0 means no retries")
	// TODO implement exponential backoff (for some statuses?)
	flag.DurationVar(&conf.retryDelay, "retry-delay", time.Second, "delay between each retry")
	flag.Var(&conf.retryPolicy, "retry", "which http statuses to retry. empty string means no retries, all means retry all, or provide a comma separated list of status codes")
	flag.IntVar(&conf.requestsPerPeriod, "requests-per-period", 0, "how many requests per period to make. 0 means no throttling")
	flag.DurationVar(&conf.period, "period", time.Second, "period in which requests are capped by the requests-per-period flag")
}

func (conf *Config) Validate() error {
	if conf.UserAgent == "" {
		return fmt.Errorf("need to specify user agent")
	}
	if !userAgentRe.MatchString(conf.UserAgent) {
		return fmt.Errorf("User-Agent contains non ascii characters")
	}
	return nil
}

// Configure configures the given client (add throttling, retries, ...)
func (conf *Config) Configure(c Client) Client {
	if conf.numRetries > 0 {
		c = WithRetries(c, conf.numRetries, conf.retryDelay, conf.retryPolicy)
	}
	if conf.requestsPerPeriod > 0 {
		c = WithThrottling(c, conf.period, conf.requestsPerPeriod)
	}
	if conf.UserAgent != "" {
		c = WithUserAgent(c, conf.UserAgent)
	}
	return c
}
