// Package externalsvc provides types and functions to communicate with
// external services, typically via REST APIs.
package externalsvc

import "time"

const (
	maxRetries           = 5
	retryBackoff         = 300 * time.Millisecond
	maxWaitForRetryAfter = 10 * time.Second
)
