# Build DEX dashboards with Fleet and Claude

Fleet already knows what hardware your computers run, which apps people open, and which versions are installed. This guide shows you how to turn that data into three digital employee experience (DEX) dashboards that rebuild themselves every day: Hardware lifecycle (which computers to replace, repair, or leave alone), Software usage (which installed apps nobody opens), and Patch velocity (how fast new versions reach every computer). A daily [Claude Code routine](https://code.claude.com/docs/en/routines) reads the Fleet API and publishes each dashboard as a private Claude artifact. You don't run a server or a database. Hardware lifecycle and Software usage cover macOS, Windows, and Linux. Patch velocity covers macOS and Windows. Each dashboard has a toggle to show one platform at a time, and a light or dark theme that follows your system setting.

## How it works

- A cloud environment holds a read-only Fleet API token and can reach only your Fleet server.
- Once a day, the routine runs a set of Python scripts from Fleet's repository. They read your hosts, their software, and two DEX reports, then build the three pages.
- Each dashboard stores a small history file next to its page. Every run reads the history back and adds a day. Patch velocity uses it to time how long each new version takes to reach every computer, and Hardware lifecycle uses it to tell sustained memory strain on a Mac from a single bad day. Each page also charts its key numbers over time, for up to a year. All of this gets more useful over the first few weeks.

## Prerequisites

- Fleet Premium 4.85.0 or later, which can limit an API-only user to specific endpoints.
- A global admin account in Fleet, to add the reports and create the API-only user.
- A Claude account that can use Claude Code [routines](https://code.claude.com/docs/en/routines), [cloud environments](https://code.claude.com/docs/en/cloud-environments), and artifacts.
- Claude Code on your computer, for the one-time setup.
- Optional: the Slack connector in Claude, if you want a Slack message when a run fails.
- Optional: warranty end dates in a [custom host vital](https://fleetdm.com/guides/custom-host-vitals) (Fleet 4.90.0 or later), formatted `YYYY-MM-DD`. Hardware lifecycle shows a warranty chart when it finds one.

> **Note:** Software usage lists, per computer, when each app was last opened. Before you share the dashboard, check that your organization's privacy commitments allow showing that to the people you share it with.

## Step 1: Add the DEX reports

Hardware lifecycle reads memory pressure and battery health from the `DEX - Hardware experience - Device health` report. Patch velocity reads macOS build numbers from the `DEX - System experience - OS health` report, so it can leave macOS betas out. Both are in [`dex-reports.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/dex-dashboards/dex-reports.yml). They run on macOS only. On Windows and Linux, the dashboards use Fleet's built-in host and software data.

- **GitOps:** copy `dex-reports.yml` into your GitOps repository, for example to `lib/macos/reports/dex-reports.yml`, and add it under `reports:` in `default.yml` to cover every host, or in each fleet file whose Macs you want on the dashboards:

```yaml
reports:
  - path: ../lib/macos/reports/dex-reports.yml
```

The path is relative to the file it's in. This example is for a file in `fleets/`. In `default.yml`, use `./lib/macos/reports/dex-reports.yml`.

- **UI:** create both reports from `dex-reports.yml` in the Fleet UI, with the same names, SQL, and macOS as the platform.

Keep the report names exactly as written. The scripts find each report by name, in any fleet or in **All fleets**.

> **Note:** The dashboards read the results Fleet stores for each report. If `discard_reports_data` is `true` in your `server_settings`, Fleet doesn't store them, and Hardware lifecycle has no memory or battery data. The reports run every one to four hours, so give them a few hours before the first run.

## Step 2: Create a read-only API-only user

The scripts only read data. Create an API-only user that can read only what they need:

1. Go to **Settings > Users**, click **Create user**, and select **API-only**.
2. Assign the global **Observer** role.
3. Under API access, select **Specific API endpoints**, then select only these five:

| Endpoint | Why the dashboards need it |
|:--|:--|
| [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) (`GET /api/v1/fleet/hosts`) | Hardware, OS version, and each host's software with its last-opened time. |
| [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) (`GET /api/v1/fleet/hosts/:id`) | Battery health (macOS and Windows) and custom host vitals, such as warranty dates. |
| [List fleets](https://fleetdm.com/docs/rest-api/rest-api#list-fleets) (`GET /api/v1/fleet/fleets`) | Finds the DEX reports in each fleet. |
| [List reports](https://fleetdm.com/docs/rest-api/rest-api#list-reports) (`GET /api/v1/fleet/reports`) | Looks up each DEX report by name. |
| [Get report data](https://fleetdm.com/docs/rest-api/rest-api#get-report-data) (`GET /api/v1/fleet/reports/:id/report`) | Reads the stored DEX report results. |

4. Save the user and copy its API token.

A global observer can read every host but can't change anything. Any request outside the five endpoints gets a `403` that says "endpoint not permitted for this API-only user."

## Step 3: Create a cloud environment

The routine runs in a [cloud environment](https://code.claude.com/docs/en/cloud-environments) that holds the token. Create one named `DEX dashboards`:

1. Set **Network access** to **Custom**, and allow only your Fleet server's hostname, for example `fleet.example.com`. Enter the hostname only, without `https://` or a path.
2. Add these environment variables, one per line, without quotes:

```
FLEET_URL=https://fleet.example.com
FLEET_API_TOKEN=<api-only-user-token>
```

3. Optional: if your warranty dates are in a custom host vital with another name, add `DEX_WARRANTY_VITAL=<vital name>`. The default is `AppleCare warranty expiration`.

> **Warning:** Anyone who can use an environment can read its environment variables. Keep this environment to yourself.

## Step 4: Download the scripts

The scripts are in [`docs/solutions/dex-dashboards`](https://github.com/fleetdm/fleet/tree/main/docs/solutions/dex-dashboards) in Fleet's repository. They're Python 3 with no third-party packages. Download a sparse copy of the repository that includes the folder. It also contains a few top-level repository files, but none of Fleet's other code:

```bash
git clone --depth 1 --filter=blob:none --sparse https://github.com/fleetdm/fleet.git fleet-dex
```

```bash
cd fleet-dex && git sparse-checkout set docs/solutions/dex-dashboards
```

Review the scripts before you run them. The routine runs them every day with your token in its environment. Its prompt carries a sha256 hash of the code, and it refuses to run code that doesn't match, so changes reach the routine only when you set it up or update it yourself.

## Step 5: Create the dashboards and the routine

Open Claude Code in `docs/solutions/dex-dashboards` inside `fleet-dex`. If you're still in `fleet-dex` from step 4, run `cd docs/solutions/dex-dashboards` first. Paste this prompt, and fill in the values in its first three lines. If you don't want Slack alerts, delete the Slack line.

```
Set up the Fleet DEX dashboards from the files in this folder.

- Cloud environment for the routine: DEX dashboards
- Daily run time: 09:00 in my time zone
- Slack alerts: send failure DMs to Slack member ID <your Slack member ID>

1. Run `python3 bundle.py --out <dir>`, where <dir> is a new temporary directory outside this folder that you choose. It prints a sha256 hash. Don't change any files in this folder.
2. Publish three new private artifacts from placeholder-lifecycle.html, placeholder-software.html, and placeholder-patch.html in that output directory. On the Hardware lifecycle artifact, also publish dex-dashboards.json as src/dex-dashboards.json with content type application/json. Note the three artifact URLs.
3. Read routine-prompt.md. Take the text between "PROMPT START" and "PROMPT END", and fill in {{LIFECYCLE_URL}}, {{SOFTWARE_URL}}, {{PATCH_URL}}, and {{BUNDLE_SHA256}}. Replace {{IF_THE_RUN_FAILS}} with the Slack block (and my member ID) if I asked for Slack alerts above, otherwise with the block for no Slack.
4. Create a disabled routine named "DEX dashboards (daily rebuild)" with that prompt: the environment above, a daily schedule at the run time above converted to UTC, the Sonnet model, no repository, and the tools Bash, Read, Write, ToolSearch, and Artifact. A new routine gets every connector on my account by default, so remove them all. If I asked for Slack alerts, keep only the Slack connector and allow its slack_send_message tool. If you can't find my Slack connector, create the routine without it and tell me.
5. Run the routine once. When it finishes, read its run log. If it published all three dashboards, enable the routine and send me the three artifact links. If it didn't, show me the error and leave the routine disabled.

Never ask me for the Fleet API token. It lives only in the cloud environment.
```

Claude may ask you to confirm the schedule, model, or repository before it creates the routine. Confirm the values from the prompt. The first run starts each dashboard's history. From then on, the routine runs once a day.

> **Note:** Routine schedules run in UTC. When daylight saving time starts or ends, the routine runs an hour earlier or later in your local time.

> **Warning:** Check the routine's connectors after setup. A routine with connectors it doesn't need, like mail or file storage, can reach data that has nothing to do with these dashboards.

## Verify

1. Open the three artifact links. Each shows today's date next to **Updated**.
2. On Patch velocity, the summary says "Timing starts with the next release." Until a new version ships and computers pick it up, it shows the share of installs on the newest version instead.
3. Under **Over time**, each chart says "Not enough history yet." The charts fill in after the second daily run.
4. Open the routine at [claude.ai/code/routines](https://claude.ai/code/routines). Check that it's enabled, runs daily, and lists no connectors, or only Slack if you asked for alerts.

## Troubleshoot

**"FATAL: FLEET_URL and FLEET_API_TOKEN must be set"**

The routine isn't using the environment from step 3, or its variables didn't save. Open the routine, check its environment, and check the environment's variables.

**"Tunnel connection failed: 403 Forbidden"**

The environment's network access doesn't allow your Fleet server. Set **Network access** to **Custom** and add the hostname from `FLEET_URL`.

**`403` with "endpoint not permitted for this API-only user"**

The API-only user is missing one of the five endpoints in step 2. Edit the user in **Settings > Users** and add it.

**"BUNDLE_HASH_MISMATCH"**

The code in `src/dex-dashboards.json` doesn't match the hash in the routine's prompt. If you didn't change the code, someone else did. Find out who before you update the hash.

**"history for lifecycle is missing"**

The routine couldn't read a history file, so it stopped instead of starting over. Run it again. If it keeps happening, check that each artifact still has `data/history.json`.

**Hardware lifecycle shows no battery or memory data for Macs**

The Device health report has no stored results. Check that it's assigned to the fleets your Macs are in, and that Fleet stores report results (step 1).

**Windows apps all show as idle on some computers**

Windows records recently run programs only while its prefetch feature (the SysMain service) is on. Software usage leaves out a Windows computer with no record for any program. If many computers are missing, check whether SysMain is turned off.

## Update the dashboards

When Fleet updates these scripts, or you change them:

1. Run `git pull` in `fleet-dex`, and review what changed.
2. Ask Claude Code to run `bundle.py` again, publish the new `dex-dashboards.json` to the Hardware lifecycle artifact as `src/dex-dashboards.json`, and replace the hash in the routine's prompt with the new one.

The history files stay where they are, so the dashboards keep their history.

## Further reading

- [Claude Code routines](https://code.claude.com/docs/en/routines)
- [Claude Code cloud environments](https://code.claude.com/docs/en/cloud-environments)
- [Create an API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user)
- [Use custom host vitals in scripts and configuration profiles](https://fleetdm.com/guides/custom-host-vitals)

<meta name="articleTitle" value="Build DEX dashboards with Fleet and Claude">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-16">
<meta name="category" value="guides">
<meta name="description" value="Build daily Hardware lifecycle, Software usage, and Patch velocity dashboards from Fleet data with a Claude Code routine.">
