# Osquery 5.8.1 | Process auditing, stats, and additional tables.

![osquery 5.8.1](../website/assets/images/articles/osquery-5.8.1-cover-1600x900@2x.png)

Osquery 5.8 introduces process auditing on Windows, statistics for live queries, and additional tables.  🟣 Openness is a key Fleet [value](https://fleetdm.com/handbook/company#values). We welcome contributions to Fleet and find ways to contribute to other open-source projects. When you support Fleet, you are also contributing to projects like osquery. Let’s take a look at the changes in this latest release.

Please note that osquery 5.8 has already been pushed to Fleet’s stable and edge auto-update channels.


## Windows `process_etw_events` table

This PR introduces POTE ([Programmable OS Tracing Engine](https://github.com/osquery/osquery/issues/7826)) framework + a new windows `evented` table called `etw_process_events` which is built on top of POTE. The primary purpose of this new `evented` table is to audit process creation and termination on Windows. Having POTE in place will simplify the addition of future `evented` tables as POTE provides a simplified mechanism to create ETW-based Event publishers. 

The Windows `process_etw_events` table brings osquery towards parity with System Monitor (Sysmon). Sysmon is a common add-on for Windows logging. With Sysmon, you can detect malicious activity by tracking code behavior and network traffic. Sysmon is part of the Sysinternals package and is owned by Microsoft.

_Fleetie, Marcos contributed this [pull request](https://github.com/osquery/osquery/pull/7821) to the osquery project._


## Live query statistics

This PR creates a new top-level `stats` key when writing a distributed query response. This includes the data in `QueryPerformance` class, indexed by the query ID in the server's read endpoint. A new stats JSON subkey exposes the `stats` key in the distributed query response. Performance stats are not stored. When a query executes, the stats for that execution are returned.

The addition of `stats` unlocks future work in Fleet that will enable performance stats for live queries and policies.

_Fleetie, Artemis contributed this [pull request](https://github.com/osquery/osquery/pull/7920) to the osquery project._

## Add `pid_with_namespace` for `yara` table

On October 25, the OpenSSL project team [announced](https://mta.openssl.org/pipermail/openssl-announce/2022-October/000238.html) a security fix for a critical vulnerability in OpenSSL version 3.x. The patch was released on November 1, 2022. Akamai released a [blog post](http://akamai.com/blog/security-research/openssl-vulnerability-how-to-effectively-prepare#query) with a [YARA](https://github.com/VirusTotal/yara)-based rule, helping Sysadmins find processes running with vulnerable OpenSSL versions. OpenSSL process identification works well for processes on the host OS but breaks down for processes inside containers.

This change adds the `pid_with_namespace` column to the YARA table in osquery, allowing for querying within containers using the `yara `table.

## `Unit_file_state` column in `systemd_units` table

This change adds a new column to the `systemd_units` table to determine if a `systemd` service is in one of several enabled states, such as `enabled` or `masked`. This allows for discovering running processes that could have potential security implications. Previously, determining if a service was enabled was not possible in osquery.

_Fleetie, Artemis contributed this [pull request](https://github.com/osquery/osquery/pull/7895) to the osquery project._


## `Bpf_process_events_v2` table

An initial experiment has been included, called `linuxevents`. This PR  adds a new `bpf_process_events_v2` table, a better, container-aware version of the built-in `bpf_process_events`. The new functionality is considered experimental and must be explicitly enabled with `--experiment_list=linuxevents`.

Key features:
1. The table now traces internal kernel structures (i.e., task_struct) to capture all the data. We no longer need to trace system calls and keep track of file descriptors.
2. Significantly lower memory and CPU usage.
3. Container aware: contains both the container ID and container backend name (currently only supports podman).
4. Uses the BTF kernel debug symbols: no kernel headers required!

## macOS `secureboot` table

This PR adds support for macOS (Intel-based) hardware that have a secure enclave and support secure boot. This PR extends the secureboot schema from boolean to the following: Secure mode for Intel-based macOS: 0 disabled, 1 full security, 2 medium security.

## Linux `kernel_keys` table

This PR adds a new table called `kernel_keys` for Linux. This table exposes the content of the file /proc/keys,

this file exposes a list of the keys for which the reading thread has view permission, providing various information about each key.

## Cached_memory column in `docker_container_stats`

The docker container memory usage is not in sync with docker CLI which subtracts the cached memory from the used memory. A new `cached_memory` column has been added to `docker_container_stats` to retrieve the cached container memory to provide more detailed information about container memory usage.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-03-14">
<meta name="articleTitle" value="osquery 5.8.1 | Process auditing, stats, and additional tables">
<meta name="articleImageUrl" value="../website/assets/images/articles/osquery-5.8.1-cover-1600x900@2x.png">
