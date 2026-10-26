# Build DEX dashboards with Fleet and Claude

Fleet already knows what hardware your computers run, which apps people open, and which versions are installed. This guide shows you how to turn that data into eight digital employee experience (DEX) dashboards that rebuild themselves every day:

- Hardware lifecycle: which computers to replace, repair, or leave alone.
- Software usage: which installed apps nobody opens.
- Patch velocity: how fast new versions reach every computer.
- AI governance: which AI tools people use, how they're set up, and which settings put data at risk.
- Connectivity: which computers stopped checking in, and which Macs have weak Wi-Fi or no network path.
- Analytics: whether the data behind the dashboards is complete and fresh, plus security posture and Fleet agent versions.
- Sustainability: how much energy your Macs use, which computers never sleep, and which laptop batteries need service.
- Policies: which Fleet policies your computers pass and fail, on the latest run or on any day in the past year, with patch policies on their own or combined with the rest.

An overview page sums up all eight, with today's key numbers and a 30-day trend for each, and links to every dashboard.

A daily [Claude Code routine](https://code.claude.com/docs/en/routines) reads the Fleet API and publishes each dashboard as a private Claude artifact. You don't run a server or a database. Every dashboard covers macOS, Windows, and Linux, with a toggle to show one platform at a time and a light or dark theme that follows your system setting until you pick one.

## How it works

- A cloud environment holds a read-only Fleet API token and can reach only your Fleet server.
- Once a day, the routine runs a set of Python scripts from Fleet's repository. They read your hosts, their software, and 13 DEX reports, then build the eight pages and the overview.
- Each dashboard stores a small history file next to its page. Every run reads the history back and adds a day. Patch velocity uses it to time how long each new version takes to reach every computer, Hardware lifecycle uses it to tell sustained memory strain on a Mac from a single bad day, and Sustainability uses it to turn each Mac's running energy total into kWh per day. Each page also charts its key numbers over time, for up to a year, and the overview reads the same daily counts, so its numbers match each dashboard's. All of this gets more useful over the first few weeks.

## Tradeoffs

These dashboards replace a business intelligence (BI) tool with a scheduled Claude session and a few scripts. Weigh what that gets you against what it doesn't before you set them up.

### Strengths

- There's nothing to host. You don't run or patch a server, database, or BI tool. Each dashboard keeps its own history in a file next to its page.
- The scripts compute every number. Claude reads the artifacts, checks the code's hash, runs the scripts, and publishes the pages. It doesn't calculate, summarize, or write anything that appears on a dashboard.
- Access is read-only. The API-only user can call five read endpoints and nothing else. The token stays in the cloud environment, which can reach only your Fleet server.
- The dashboards are easy to change. The scripts are plain Python and HTML with no dependencies. Ask Claude Code to add a chart or a whole dashboard, review the change, then ship it by updating the hash.
- Nothing new runs on your computers. The dashboards use data Fleet already collects, plus the reports in step 1.

### Limitations

- The data is a snapshot, not live. Each dashboard is only as fresh as its last run. The Fleet API returns current state, not history, so trend charts and Patch velocity's timing start on the day you set this up.
- Fleet data passes through Claude. The routine reads each page, host names included, and the pages are stored as Claude artifacts. Check that this fits your organization's data-handling commitments.
- Viewers see every fleet. Anyone who can open a dashboard sees all the computers the API-only user can read. The fleet filter changes the view, not who can see what.
- Size has limits. The scripts are tested with a simulated fleet of 10,000 computers. Each run downloads every computer's software list from Fleet, about 0.6 MB per computer, and requests each computer's details, so bigger fleets mean longer runs and more load on your Fleet server. Past 10,000 computers, use a data pipeline instead (see "Choose how often it runs").
- Runs depend on Claude. Routines, cloud environments, and artifacts are Claude features, and each run uses Claude usage on your plan. The prompt tells the routine to stop and report a failure instead of working around it. You find out from the Slack alert or the routine's run log.
- You maintain it. The dashboards aren't part of Fleet. When the scripts change, you review the update and change the routine's hash yourself (see "Update the dashboards").
- Some signals are estimates. For example, a computer's age comes from its model or CPU release year, Windows app usage from prefetch records, Linux app usage from file access times, and carbon from each country's average grid intensity. Only Macs report an energy total, so Windows and Linux show single power readings instead of kWh. Each page's **How this is measured** section explains its rules.

## Prerequisites

- Fleet Premium 4.85.0 or later, which can limit an API-only user to specific endpoints.
- For more than 1,000 computers, Fleet 4.93.0 or later. Earlier versions keep at most 1,000 rows per report, so reports leave computers out.
- A global admin account in Fleet, to add the reports and create the API-only user.
- A Claude account that can use Claude Code [routines](https://code.claude.com/docs/en/routines), [cloud environments](https://code.claude.com/docs/en/cloud-environments), and artifacts.
- Claude Code on your computer, for the one-time setup.
- fleetd 1.59.0 or later on your computers, for the AI governance dashboard. It reads fleetd's `ai_tools` table.
- fleetd on your computers, for the Sustainability dashboard. Its reports read fleetd's `ioreg`, `pmset`, `file_lines`, and `dconf_read` tables.
- Optional: the Slack connector in Claude, if you want a Slack message when a run fails.
- Optional: [GeoIP](https://fleetdm.com/docs/configuration/fleet-server-configuration#geoip) on your Fleet server, so Sustainability can estimate carbon from each computer's country. Without it, set `DEX_GRID_INTENSITY` in step 3.
- Optional: warranty end dates in a [custom host vital](https://fleetdm.com/guides/custom-host-vitals) (Fleet 4.90.0 or later), formatted `YYYY-MM-DD`. Hardware lifecycle shows a warranty chart when it finds one.

> **Note:** Software usage lists, per computer, when each app was last opened, and AI governance lists the AI tools, MCP servers, and agent instruction files on each computer. Before you share them, check that your organization's privacy commitments allow showing that to the people you share them with. AI governance never shows command lines, launch arguments, environment values, or usernames.

## Step 1: Add the DEX reports

The dashboards read 13 reports, all in [`dex-reports.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/dex-dashboards/dex-reports.yml):

| Report | Platforms | What reads it |
|:--|:--|:--|
| `DEX - Hardware experience - Device health` | macOS | Hardware lifecycle, for memory pressure and battery health. |
| `DEX - System experience - OS health` | macOS | Patch velocity, for macOS build numbers, so it can leave betas out. |
| `DEX - AI governance - AI tools per host` | macOS, Windows, Linux | AI governance. It packs every AI tool fleetd finds into one row per computer, so a report row limit doesn't cut computers off. |
| `DEX - Network experience - Wi-Fi signal quality` | macOS | Connectivity, for signal strength, noise, and band. |
| `DEX - Network experience - VPN gate` | macOS | Connectivity, for whether a Mac's traffic goes through a VPN tunnel, goes direct, or has no network. |
| `DEX - System experience - Security posture` | macOS | Analytics, for FileVault, System Integrity Protection, Gatekeeper, and the firewall. |
| `DEX - Security compliance - Windows firewall status` | Windows | Analytics, for the firewall, antivirus, and other Security Center checks. |
| `DEX - Security compliance - Secure Boot status (Windows)` | Windows | Analytics. |
| `DEX - Security compliance - TPM status (Windows)` | Windows | Analytics. |
| `DEX - Sustainability - Mac energy` | macOS | Sustainability, for the energy total Apple silicon Macs keep since boot and their power draw right now. |
| `DEX - Sustainability - Mac sleep settings` | macOS | Sustainability, for sleep and display sleep timers, what's keeping a Mac awake, and how often it sleeps. |
| `DEX - Sustainability - Windows power` | Windows | Sustainability, for battery draw while unplugged, the power plan's sleep and display-off timeouts, Fast Startup, and sleep events. |
| `DEX - Sustainability - Linux power` | Linux | Sustainability, for systemd and GNOME sleep settings, batteries, and power sensors. |

Analytics also checks every report for missing or stale results. Where no report covers a platform, the dashboards use Fleet's built-in host data, such as check-in times, disk encryption, and policy results.

- **GitOps:** copy `dex-reports.yml` into your GitOps repository, for example to `lib/all/reports/dex-reports.yml`, and add it under `reports:` in `default.yml` to cover every host, or in each fleet file whose computers you want on the dashboards:

```yaml
reports:
  - path: ../lib/all/reports/dex-reports.yml
```

The path is relative to the file it's in. This example is for a file in `fleets/`. In `default.yml`, use `./lib/all/reports/dex-reports.yml`.

- **UI:** create the 13 reports from `dex-reports.yml` in the Fleet UI, with the same names, SQL, and platforms.

Keep the report names exactly as written. The scripts find each report by name, in any fleet or in **All fleets**.

> **Note:** The dashboards read the results Fleet stores for each report. If `discard_reports_data` is `true` in your `server_settings`, Fleet doesn't store them, and Hardware lifecycle has no memory or battery data. The reports run every 10 minutes to every four hours, so give them a few hours before the first run.

## Step 2: Create a read-only API-only user

The scripts only read data. Create an API-only user that can read only what they need:

1. Go to **Settings > Users**, click **Create user**, and select **API-only**.
2. Assign the global **Observer** role.
3. Under API access, select **Specific API endpoints**, then select only these five:

| Endpoint | Why the dashboards need it |
|:--|:--|
| [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts) (`GET /api/v1/fleet/hosts`) | Hardware, OS version, check-in times, disk encryption, MDM enrollment, failing policies, and each host's software with its last-opened time. |
| [Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host) (`GET /api/v1/fleet/hosts/:id`) | Battery health (macOS and Windows), custom host vitals such as warranty dates, each computer's policy results, and its country from GeoIP. |
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
4. Optional: to estimate carbon for computers with no GeoIP country, add `DEX_GRID_INTENSITY=<grams of CO2 per kWh>`. Without it, Sustainability uses the figure for your most common country.

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
2. Publish nine new private artifacts from placeholder-overview.html, placeholder-lifecycle.html, placeholder-software.html, placeholder-patch.html, placeholder-ai.html, placeholder-connectivity.html, placeholder-analytics.html, placeholder-sustainability.html, and placeholder-policies.html in that output directory. On the Hardware lifecycle artifact, also publish dex-dashboards.json as src/dex-dashboards.json with content type application/json. Publish the AI governance artifact with these capabilities, so its policy decisions are saved and only editors can change them: {"db": {"rules": [{"path": "governance", "read": "view", "write": "admin"}, {"path": "governance-log", "read": "view", "write": "admin"}]}, "user": {"scopes": ["profile"]}, "downloads": true}. Publish the other eight with these capabilities, so each viewer's theme choice is saved: {"db": {}, "user": {}}. Note the nine artifact URLs.
3. Read routine-prompt.md. Take the text between "PROMPT START" and "PROMPT END", and fill in {{LIFECYCLE_URL}}, {{SOFTWARE_URL}}, {{PATCH_URL}}, {{AI_URL}}, {{CONNECTIVITY_URL}}, {{ANALYTICS_URL}}, {{SUSTAINABILITY_URL}}, {{POLICIES_URL}}, {{OVERVIEW_URL}}, and {{BUNDLE_SHA256}}. Replace {{IF_THE_RUN_FAILS}} with the Slack block (and my member ID) if I asked for Slack alerts above, otherwise with the block for no Slack.
4. Create a disabled routine named "DEX dashboards (daily rebuild)" with that prompt: the environment above, a daily schedule at the run time above converted to UTC, the Sonnet model, no repository, and the tools Bash, Read, Write, ToolSearch, and Artifact. A new routine gets every connector on my account by default, so remove them all. If I asked for Slack alerts, keep only the Slack connector and allow its slack_send_message tool. If you can't find my Slack connector, create the routine without it and tell me.
5. Run the routine once. When it finishes, read its run log. If it published all nine artifacts, enable the routine and send me the overview's link. If it didn't, show me the error and leave the routine disabled.

Never ask me for the Fleet API token. It lives only in the cloud environment.
```

Claude may ask you to confirm the schedule, model, or repository before it creates the routine. Confirm the values from the prompt. The first run starts each dashboard's history. From then on, the routine runs once a day.

> **Note:** Routine schedules run in UTC. When daylight saving time starts or ends, the routine runs an hour earlier or later in your local time.

> **Warning:** Check the routine's connectors after setup. A routine with connectors it doesn't need, like mail or file storage, can reach data that has nothing to do with these dashboards.

### Choose how often it runs

The routine runs once a day by default. You can run it more often on a smaller fleet. Each run reads every computer from Fleet: its software list (about 0.6 MB per computer) and one details request per computer. Pick the most frequent schedule your fleet size allows:

| Computers | How often | Cron expression |
|:--|:--|:--|
| Up to 2,000 | Every 4 hours | `7 */4 * * *` |
| 2,001 to 5,000 | Every 12 hours | `7 */12 * * *` |
| 5,001 to 10,000 | Once a day | `7 14 * * *`, or any one time a day |
| More than 10,000 | Use a data pipeline instead | |

Each row keeps the load on your Fleet server near the same daily total: about 10,000 to 12,000 computer reads and 6 to 7 GB of software data a day. Claude usage is roughly the same for each run whatever the fleet size, so running every 4 hours uses about six times the Claude usage of running once a day.

To change the schedule:

1. Open Claude Code and run `/schedule update`.
2. Ask Claude to change the "DEX dashboards (daily rebuild)" routine to the cron expression from the table. A run scheduled exactly on the hour can start several minutes late, which is why the expressions start at 7 minutes past.
3. Optional: ask Claude to rename the routine to match its new schedule, for example "DEX dashboards (every 4 hours)."

Routines can't run more than once an hour. Trend charts and day-by-day signals, such as memory strain and weak Wi-Fi, keep one value per UTC day, from that day's last run.

On a large fleet, keep these in mind:

- A run that takes longer than one 10-minute command saves what it collected and continues in the next command. The routine prompt handles this, up to 10 commands per run.
- A page or history that's bigger than about 8 MB is split into part files, because an artifact file can't be larger than 16 MB. Pages load their parts before they draw.
- The scripts send up to 8 requests to Fleet at once. If your Fleet server is busy during runs, add `DEX_WORKERS=4` to the cloud environment's variables.
- Tables show 200 rows at a time, with a button to show more.

## Verify

1. Open the overview. Each card shows today's numbers and opens its dashboard in a new tab. Each dashboard shows today's date next to **Updated**, and **All dashboards** takes you back to the overview.
2. On Patch velocity, the summary says "Timing starts with the next release." Until a new version ships and computers pick it up, it shows the share of installs on the newest version instead.
3. Under **Over time**, each chart says "Not enough history yet," and the overview's cards say the trend starts after a second day. The charts fill in after the second daily run.
4. On AI governance, pick a posture under **AI policy**. The change log shows your name and the time. People who can only view the dashboard see the policy but can't change it.
5. On Sustainability, the summary shows each Mac's average power since boot. Energy in kWh starts after the second run, once there are two readings to compare.
6. On Policies, the date picker starts at today's run. Each run adds a day, so you can go back to any day since setup, for up to a year. Pick **Patch policies** or **Other policies** to see the policies Fleet keeps for Fleet-maintained apps apart from the rest, or **All policies** for both. Click a policy to see which computers fail it.
7. On Analytics, check the **Data sources** table. Each report shows how many computers sent results in the last 7 days. A report marked **Not set up** isn't in Fleet yet, or its name doesn't match `dex-reports.yml`.
8. Open the routine at [claude.ai/code/routines](https://claude.ai/code/routines). Check that it's enabled, runs daily, and lists no connectors, or only Slack if you asked for alerts.

## Share the dashboards on your own domain

Claude artifacts are hosted on claude.ai, so their links are claude.ai addresses. There are two ways to give people an address on your own domain.

### Option 1: Link to the overview from your domain

Point an address on your domain, or a short link in your organization's link shortener, at the overview artifact's URL, for example with a redirect rule at your DNS or web host. People still sign in to Claude, and the artifacts' sharing settings still decide who can open them. Nothing about the routine changes.

### Option 2: Host the pages yourself

The routine can also write a copy of every page that works on any web server, and upload it to yours. Before you choose this, weigh what changes:

- You own access control. The pages show computer names, which apps each person opens, and their AI tools, so put the site behind your single sign-on. Never make it public.
- Features that need claude.ai don't work there. AI governance shows the inventory and risk findings, but not the policy decisions saved on the artifact, and nobody can change the policy or see its change log. Each viewer's theme choice stays in their browser only.
- The upload credential lives in the cloud environment, next to the Fleet token, so give it access to that one site and nothing else.

To set it up:

1. In the cloud environment, add your web host's upload domain to **Allowed domains**, and add its credential to the environment's variables, or as a [network secret](https://code.claude.com/docs/en/cloud-environments#add-network-secrets) if your host takes a token in a request header. If the upload needs a command-line tool, install it in the environment's [setup script](https://code.claude.com/docs/en/cloud-environments#setup-scripts).
2. In Claude Code, ask Claude to edit the routine's prompt: add `--site "$W/site"` to the run.py command in step 4, and add a step after step 6 that uploads `$W/site` with your host's tool. For example, for an Amazon S3 bucket behind CloudFront: `aws s3 sync "$W/site" s3://<bucket>/dex --delete`.
3. Run the routine once and open the site. `index.html` is the overview, and each dashboard is in its own folder.

The site holds no history files. Each run's history stays on the artifacts, so the artifacts keep working, and you can stop uploading at any time.

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

**The run log shows "CONTINUE"**

This isn't an error. On a large fleet, a run takes longer than one 10-minute command, so run.py saves what it collected and the routine runs the same command again. If a run still stops after 10 commands, your Fleet server answers slowly for the fleet's size. Move to a less frequent schedule (see "Choose how often it runs"), or raise `DEX_WORKERS` if your Fleet server has room.

**A dashboard says "This dashboard's data didn't load"**

The page's part files are missing from the artifact. Run the routine again. If it keeps happening, check that the routine's prompt matches the current `routine-prompt.md`, which publishes every file run.py lists.

**"FATAL: ... lists data/history-2.json, which is missing"**

The routine read a history that's split into parts, but not the parts. Run it again. If it keeps happening, update the routine's prompt from the current `routine-prompt.md`.

**"FATAL: ... over the 16 MB artifact file limit" or "over what one publish carries"**

The fleet is too large for these dashboards. Use a data pipeline instead (see "Choose how often it runs").

**AI governance says "No AI tool inventory yet"**

The AI tools report has no stored results. Check that it's assigned to your fleets and that your computers run fleetd 1.59.0 or later. The report runs hourly.

**Connectivity says "Wi-Fi report not set up" or "VPN gate report not set up"**

The scripts didn't find a report with that exact name in any fleet. Add it from `dex-reports.yml` (step 1) without renaming it.

**Connectivity shows Macs on Wi-Fi as not on Wi-Fi**

Recent macOS versions hide Wi-Fi signal details from apps that don't have location access, and report a signal of 0 dBm instead. Connectivity doesn't count those Macs as on Wi-Fi. The band can read "unknown" for the same reason.

**Hardware lifecycle shows no battery or memory data for Macs**

The Device health report has no stored results. Check that it's assigned to the fleets your Macs are in, and that Fleet stores report results (step 1).

**Sustainability shows no energy for a Mac**

The Mac energy report reads fleetd's `ioreg` table. Check that the Mac runs fleetd and that the report is assigned to its fleet. Macs without the energy total show their power draw right now instead.

**Sustainability shows no readings for Windows laptops**

A Windows reading counts only while a laptop runs on battery, so plugged-in laptops have none. Readings build up over the first few weeks.

**Windows apps all show as idle on some computers**

Windows records recently run programs only while its prefetch feature (the SysMain service) is on. Software usage leaves out a Windows computer with no record for any program. If many computers are missing, check whether SysMain is turned off.

## Update the dashboards

When Fleet updates these scripts, or you change them:

1. Run `git pull` in `fleet-dex`, and review what changed.
2. Ask Claude Code to run `bundle.py` again, publish the new `dex-dashboards.json` to the Hardware lifecycle artifact as `src/dex-dashboards.json`, and replace the hash in the routine's prompt with the new one.
3. If `routine-prompt.md` changed, ask Claude Code to fill it in again with your artifact URLs, the new hash, and your failure block, and to replace the routine's prompt with it.
4. If the update adds a dashboard, also ask Claude Code to publish it from its placeholder page with `empty-history.json` as `data/history.json`, and to update the routine's prompt from the new `routine-prompt.md`. The routine stops when some dashboards have history and others don't, so the new one needs an empty history file to start from. The overview is the exception: it keeps no history, so publish it from `placeholder-overview.html` without one.

The history files stay where they are, so the dashboards keep their history.

## Further reading

- [Claude Code routines](https://code.claude.com/docs/en/routines)
- [Claude Code cloud environments](https://code.claude.com/docs/en/cloud-environments)
- [Create an API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user)
- [Use custom host vitals in scripts and configuration profiles](https://fleetdm.com/guides/custom-host-vitals)
- [Carbon intensity of electricity generation (Our World in Data, from Ember)](https://ourworldindata.org/grapher/carbon-intensity-electricity)

<meta name="articleTitle" value="Build DEX dashboards with Fleet and Claude">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-10-16">
<meta name="category" value="guides">
<meta name="description" value="Build eight daily DEX dashboards from Fleet data with a Claude Code routine, from hardware lifecycle and patching to AI governance and policies.">
