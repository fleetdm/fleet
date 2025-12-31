# Fleet troubleshooting for IT admins

Lorem ipsum...


## Finding fleetd logs

Fleetd will send stdout/stderr logs to the following directories:

- macOS: `/private/var/log/orbit/orbit.std{out|err}.log`.
- Windows: `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` (the log file is rotated).
- Linux: Orbit and osqueryd stdout/stderr output is sent to syslog (`/var/log/syslog` on Debian systems, `/var/log/messages` on CentOS, and `journalctl -u orbit` on Fedora).

If the `logger_path` agent configuration is set to `filesystem`, fleetd will send osquery's "result" and "status" logs to the following directories:
- Windows: `C:\Program Files\Orbit\osquery_log`
- macOS: `/opt/orbit/osquery_log`
- Linux: `/opt/orbit/osquery_log`

The Fleet Desktop log files can be found in the following directories depending on the platform:

- Linux: `$XDG_STATE_HOME/Fleet` or `$HOME/.local/state/Fleet`
- macOS: `$HOME/Library/Logs/Fleet`
- Windows: `%LocalAppData%/Fleet`

The log file name is `fleet-desktop.log`.


## Enabling debug mode for fleetd

Debug mode can be helpful by providing more information in the logs.

When [generating an installer package](https://fleetdm.com/guides/enroll-hosts#cli) with `fleetctl package`, add the `--debug` argument to enable debug mode for the agent installer.

If you're trying to troubleshoot macOS hosts, you can [run a script](TOOD) on the host to turn on debug mode. After you're done, you can run the script again to disable debug mode on the host.

1. Run the script on the affected host.
2. Wait ~10 min.
3. Refetch the host.
4. Wait another ~10 min.
5. Run the script again to disable debug logging.
6. Grab the logs from `/var/log/orbit/orbit.stderr.log`.


<meta name="category" value="guides">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-01-08">
<meta name="articleTitle" value="Fleet troubleshooting for IT admins">
