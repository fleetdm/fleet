# Querying process_file_events on CentOS 7

This guide contains step-by-step instructions for configuring the `process_file_events` table on CentOS 7.

## Setup a CentOS 7 VM

Setup a CentOS 7 VM. (VMWare Fusion was used for this guide.)
The following kernel release was used:
```sh
$ uname --kernel-release
3.10.0-1160.83.1.el7.x86_64
```

> All commands shown in this guide were executed as `root`.

## Disable auditd

The `process_file_events` table will not work if the `auditd` daemon is running (there can only be one audit daemon).
To disable auditd run the following:
```sh
systemctl disable auditd
systemctl stop auditd

# Make sure auditd is not running by executing the following:
ps -Af | grep auditd
```

If auditd is running, osquery will log the following error:
```log
I0613 11:25:39.959703 29626 auditdnetlink.cpp:686] Failed to set the netlink owner
```

## Create test files

> The `process_file_events` table can only process events for files that existed before the osquery initialization.
> New files created after osqueryd has initialized won't be tracked by the `process_file_events` table.

Create the following test files in the CentOS VM:
```sh
mkdir /etc/foobar
echo "zoo" > /etc/foobar/zoo.txt
echo "other" > /etc/foobar/other.txt
```

## Create a test team in Fleet.

We will use a test team with special settings to avoid impacting other hosts.

## Install fleetd on the CentOS instance and enroll host

Generate fleetd rpm package (This step was executed on macOS.)
```sh
fleetctl package --type=rpm --fleet-desktop --fleet-url=https://host.docker.internal:8080 --enroll-secret=[redacted team enroll secret] --insecure --debug
```

Install fleetd package on the CentOS 7 VM:
```sh
rpm --install fleet-osquery-1.10.0.x86_64.rpm
```

## Set team agent options

Configure following settings on the team's agent options:
```sh
config:
  options:
    pack_delimiter: /
    logger_tls_period: 10
    distributed_plugin: tls
    disable_distributed: false
    logger_tls_endpoint: /api/osquery/log
    distributed_interval: 10
    distributed_tls_max_attempts: 3
  decorators:
    load:
      - SELECT uuid AS host_uuid FROM system_info;
      - SELECT hostname AS hostname FROM system_info;
  file_paths:
    etc:
      - /etc/foobar/%%
command_line_flags:
  verbose: true
  events_expiry: 3600
  disable_events: false
  disable_audit: false
  audit_persist: true
  audit_allow_fim_events: true
  audit_allow_config: true
  audit_backlog_limit: 60000
  audit_allow_process_events: false
  audit_allow_sockets: false
  audit_allow_user_events: false
  audit_allow_selinux_events: false
  audit_allow_kill_process_events: false
  audit_allow_apparmor_events: false
  audit_allow_seccomp_events: false
  enable_bpf_events: false
```

Check osquery `command_line_flags` were delivered successfully to the agent:
```sh
sudo cat /opt/orbit/osquery.flags 
--audit_allow_apparmor_events=false
--enable_bpf_events=false
--audit_allow_config=true
--audit_backlog_limit=60000
--audit_allow_user_events=false
--audit_allow_seccomp_events=false
--audit_allow_selinux_events=false
--audit_allow_sockets=false
--audit_allow_process_events=false
--audit_persist=true
--audit_allow_fim_events=true
--audit_allow_kill_process_events=false
--disable_audit=false
--verbose=true
--events_expiry=3600
--disable_events=false
```

### About the flags

- `file_paths:` We set `/etc/foobar/%%` as the path to monitor for file changes.
- `verbose: true`: We set this to `true` for troubleshooting purposes only.
- `disable_events: false`: Must be set to `false` to enable evented tables in general.
- `events_expiry: 3600`: The `events_expiry` value is the time it takes for events to be cleared from osquery local storage.
- `disable_audit: false`: Must be set to `false` to enable the audit events. 
- `audit_persist: true`: Set to `true` to attempt to retain control of audit.
- `audit_allow_fim_events: true`: Must be set to `true` to generate FIM events (otherwise the `process_file_events` will generate no events). Once this is set correctly, the user should see "Enabling audit rules for the process_file_events table" in the logs.
- `audit_allow_config: true`: Must be set to `true` to allow osquery to configure the audit service (basically set backlog limit and wait time below).
- `audit_backlog_limit: 60000`: Sets the queue length for audit events awaiting transfer to osquery audit subscriber. We set this to a high value first to make sure the table is working, then it should be modified to a better value suited for production.
- The following flags were set to `false` to avoid unnecessary load on the host: `audit_allow_process_events: false`, `audit_allow_sockets: false`, `audit_allow_user_events: false`, `audit_allow_selinux_events: false`, `audit_allow_kill_process_events: false`, `audit_allow_apparmor_events: false`, `audit_allow_seccomp_events: false`, `enable_bpf_events: false`.

## Make sure osquery audit subscriber is working

```sh
auditctl -s

enabled 1
failure 0
pid 21590
rate_limit 0
backlog_limit 60000
lost 1137311
backlog 991
loginuid_immutable 0 unlocked
```

`enabled` should be `1` and `pid`'s value should be the process ID of osquery.

## Modify the test files

```sh
echo "boo" >> /etc/foobar/zoo.txt
rm /etc/foobar/other.txt
```

> Remember: the files must exist before the osquery process is initialized.
> Creating or modifying new files won't generate `process_file_events` events.

## Query the process_file_events table

Run the following live query:
```sql
SELECT * from process_file_events;
```

It should return two events, one with `operation=write` and one with `operation=unlink`.

## Additional notes

Make sure to keep an eye on logs like the following:
```log
auditdnetlink.cpp:354 The Audit publisher has throttled reading records from Netlink for 0.2 seconds. Some events may have been lost.
```
Some events might get lost due to system load or low CPU/memory resources.

<meta name="articleTitle" value="Querying process_file_events on CentOS 7">
<meta name="description" value="Learn how to configure and query the process_file_events table on CentOS 7 with Fleet.">
<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="lucasmrod">
<meta name="authorFullName" value="Lucas Rodriguez">
<meta name="publishedOn" value="2023-07-17">