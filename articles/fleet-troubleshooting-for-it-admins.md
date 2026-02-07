# Fleet troubleshooting for IT admins

Lorem ipsum...


## Finding fleetd logs

Fleetd will send stdout/stderr logs to the following directories:

- macOS: `/var/log/orbit/orbit.std{out|err}.log`.
- Windows: `C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log` (the log file is rotated).
- Linux: Orbit and osqueryd stdout/stderr output is sent to syslog (`/var/log/syslog` on Debian systems, `/var/log/messages` on CentOS, and `journalctl -u orbit` on Fedora).

If the `logger_path` agent configuration is set to `filesystem`, fleetd will send osquery's "result" and "status" logs to the following directories:
- macOS: `/opt/orbit/osquery_log`
- Windows: `C:\Program Files\Orbit\osquery_log`
- Linux: `/opt/orbit/osquery_log`

The Fleet Desktop log files can be found in the following directories depending on the platform:

- macOS: `$HOME/Library/Logs/Fleet`
- Windows: `%LocalAppData%/Fleet`
- Linux: `$XDG_STATE_HOME/Fleet` or `$HOME/.local/state/Fleet`

The log file name is `fleet-desktop.log`.


## Enabling debug mode for fleetd

Debug mode can be helpful by providing more information in the logs.

When [generating an installer package](https://fleetdm.com/guides/enroll-hosts#cli) with `fleetctl package`, add the `--debug` argument to enable debug mode for the agent installer.

If you're trying to troubleshoot macOS hosts, you can [run a script](../docs/solutions/macos/scripts/manage-orbit-debug.sh) on the host to turn on debug mode. After you're done, you can run the script again to disable debug mode on the host.

1. Run the script on the affected host.
2. Wait ~10 min.
3. Refetch the host.
4. Wait another ~10 min.
5. Run the script again to disable debug logging.
6. Grab the logs from `/var/log/orbit/orbit.stderr.log`.


## Checking MDM commands

If you suspect something went wrong with an MDM command for a device (such as locking, wiping, installing an app, etc.), you can use the UI or API to view the MDM command results.

For the UI, open the host details page and under **Activity** toggle the switch for **Show MDM commands**.

<img width="717" height="365" alt="Show MDM commands toggle" src="https://github.com/user-attachments/assets/41e7297c-efb4-4355-841e-d46296b99505" />

Hover over the command you'd like to view, and select the **"i"** button.

For the API, use the [List MDM commands](https://fleetdm.com/docs/rest-api/rest-api#list-mdm-commands) endpoint to find the `command_uuid` for the command. Use this UUID with the [Get MDM command results](https://fleetdm.com/docs/rest-api/rest-api#get-mdm-command-results) endpoint. The result of this looks like a random string of characters, but this is because it's base64 encoded. A quick way to decode this is on a Mac is to copy the long string, then decode it in the Terminal:

```bash
pbpaste | base64 -d
```


## Server-side logs

Use [fleetctl](https://fleetdm.com/guides/fleetctl) to see server logs.

```bash
fleetctl debug errors
```


<meta name="category" value="guides">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-01-08">
<meta name="articleTitle" value="Fleet troubleshooting for IT admins">
