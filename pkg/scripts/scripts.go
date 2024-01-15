// package scripts contains constants used by fleetd and the server to
// coordinate script execution timeouts.
package scripts

import "time"

const (
	// MaxHostExecutionTime is the maximum time allowed for a script to run in a
	// host before is terminated.
	MaxHostExecutionTime = 5 * time.Minute
	// MaxServerWaitTime is the maximum time allowed for the server to wait for
	// hosts to run a script during syncronous execution. We add an extra buffer
	// to account for the notification system used to deliver scripts to the
	// host.
	MaxServerWaitTime = MaxHostExecutionTime + 1*time.Minute
)
