# Load testing of the unified queue story

This is the Go program used to run load tests for the [unified queue story](https://github.com/fleetdm/fleet/issues/22866).

It expects some software to be available for install on both macOS and Windows (including VPP apps for macOS), and some scripts too, and it enqueues installs and script execution requests on every host in the Fleet deployment for an hour.
