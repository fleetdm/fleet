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

package rate

import (
	"time"
)

// Limiter provides only one function: Allow. it blocks until routine can proceed
type Limiter interface {
	Allow()
}

type token struct{}

type limiter struct {
	tokens chan token
}

// BurstyLimiter will create a limiter which allows bursts of maximum requestsPerPeriod
// and otherwise allows requests with period/requestsPerPeriod gap in between
func BurstyLimiter(period time.Duration, requestsPerPeriod int) Limiter {
	l := &limiter{
		tokens: make(chan token, requestsPerPeriod),
	}

	// start filling indefinitely
	go func() {
		for range time.Tick(period / time.Duration(requestsPerPeriod)) {
			l.tokens <- token{}
		}
	}()

	return l
}

func (l *limiter) Allow() {
	<-l.tokens
}
