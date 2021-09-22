# Osquery Server Performance Tester

> **TODO: Archive this repo and move its contents inline into https://github.com/fleetdm/fleet**

This repository provides a tool to generate realistic traffic to an osquery
management server (primarily, [Fleet](https://github.com/fleetdm/fleet)). With
this tool, many thousands of hosts can be simulated from a single host.

## Requirements

The only requirement for running this tool is a working installation of
[Go](https://golang.org/doc/install).

## Usage

Typically `go run` is used:

```
go run agent.go --help
Usage of agent.go:
  -config_interval duration
    	Interval for config requests (default 1m0s)
  -enroll_secret string
    	Enroll secret to authenticate enrollment
  -host_count int
    	Number of hosts to start (default 10) (default 10)
  -query_interval duration
    	Interval for live query requests (default 10s)
  -seed int
    	Seed for random generator (default current time) (default 1586310930917739000)
  -server_url string
    	URL (with protocol and port of osquery server) (default "https://localhost:8080")
  -start_period duration
    	Duration to spread start of hosts over (default 10s)
exit status 2
```

The tool should be invoked with the appropriate enroll secret. A typical
invocation looks like:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf
```

When starting many hosts, it is a good idea to extend the intervals, and also
the period over which the hosts are started:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf --host_count 5000 --start_period 5m --query_interval 60s --config_interval 5m
```

This will start 5,000 hosts over a period of 5 minutes. Each host will check in
for live queries at a 1 minute interval, and for configuration at a 5 minute
interval. Starting over a 5 minute period ensures that the configuration
requests are spread evenly over the 5 minute interval. 

It can be useful to start the "same" hosts. This can be achieved with the
`--seed` parameter:

```
go run agent.go --enroll_secret hgh4hk3434l2jjf --seed 0
```

By using the same seed, along with other values, we usually get hosts that look
the same to the server. This is not guaranteed, but it is a useful technique.

### Resource Limits

On many systems, trying to simulate a large number of hosts will result in hitting system resource limits (such as number of open file descriptors).

If you see errors such as `dial tcp: lookup localhost: no such host` or `read: connection reset by peer`, try increasing these limits.

#### macOS

Run the following command in the shell before running the Fleet server _and_ before running `agent.go` (run it once in each shell):

``` sh
ulimit -n 64000
```

## Bugs
To report a bug, [click here](https://github.com/fleetdm/fleet).
