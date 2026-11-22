# DEX dashboards

Scripts that build nine dashboards from the Fleet API for macOS, Windows, and Linux: Experience score, Hardware lifecycle, Software usage, Patch velocity, AI governance, Connectivity, Analytics, Sustainability, and Policies. An overview page sums up all nine and links to each one. A daily Claude Code routine runs them and publishes each page as a private claude.ai Artifact. To set them up, follow the guide: [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude).

| File | What it does |
|:--|:--|
| `collect.py` | Reads hosts, software, and 15 DEX reports from the Fleet API, several requests at a time. It only sends `GET` requests, and keeps only the software rows a page reads. Leaves out devices assigned to Fleet in Apple Business that haven't enrolled yet. |
| `vendor.py` | Reads the latest releases from outside Fleet: macOS from SOFA (`sofafeed.macadmins.io`), Windows builds from Microsoft's Security Update Guide (`api.msrc.microsoft.com`), and fleetd, osquery, and Fleet Desktop from Fleet's update server (`updates.fleetdm.com`). It only sends `GET` requests. When a source can't be reached, the pages that use it fall back to the newest version the fleet runs and say so. |
| `build.py` | Builds each dashboard page and updates its history, or builds the overview from the dashboards' histories. Splits a page's data, or a history, into part files when it's too big for one artifact file. |
| `run.py` | Collects, reads the latest releases, then builds all nine pages and the overview in one process, and prints the publish call for each page. The routine calls this. |
| `bundle.py` | Packages the code above into one file for the routine and prints its sha256. |
| `templates/` | Page templates. |
| `dex-reports.yml` | The 15 DEX reports the dashboards read: six for Macs, three for Windows security, two for AI governance on every platform (AI tools and agent usage), and four for Sustainability (two for Macs, one each for Windows and Linux). |
| `routine-prompt.md` | The routine's prompt, with placeholders the setup fills in. |
| `github-workflow.yml` | A GitHub Actions workflow that runs the scripts on a schedule, keeps the histories on a branch, and publishes the pages to GitHub Pages. To set it up, follow [Build DEX dashboards with GitHub Actions](https://fleetdm.com/guides/build-dex-dashboards-with-github-actions). |
| `github-update-workflow.yml` | An optional GitHub Actions workflow that opens a pull request in your repository when Fleet's copy of these scripts changes. It never runs the scripts. |

The scripts need Python 3.9 or later and no third-party packages. To try them locally, set `FLEET_URL` and `FLEET_API_TOKEN`, then run:

```sh
python3 run.py --work ./work --first-run --site ./site
```

That writes `work/out/<page>/index.html` with the page's `data/` folder (its `history.json`, and part files on a large fleet) for each dashboard and the overview. `--site` also writes a copy for your own web server: `site/index.html` is the overview, and each dashboard is in its own folder. Its pages link to each other by relative path and open in the same tab. Serve it over HTTP, because a page with part files can't fetch them from a local file. On later runs, copy `work/out` somewhere else, for example to `last`, and pass it with `--in last` so each page picks up its history.

## Large fleets

The scripts are tested with a simulated fleet of 10,000 computers: a run there takes about 90 seconds of processing and under 1 GB of memory, not counting time spent waiting on Fleet. Each page and history is split into part files of about 8 MB when it's bigger than that, because an artifact file can't be larger than 16 MB. The page fetches its parts before it draws, and `history.json` lists its parts under `__parts__`.

A run that needs more than about 9 minutes saves what it collected under `--work`, prints a line starting with `CONTINUE`, and exits with code 3. Run the same command again to pick up where it stopped. `DEX_WORKERS` sets how many requests go to Fleet at once (default 8). Lower it if your Fleet server is busy during runs.

## Environment variables

- `DEX_WARRANTY_VITAL`: the custom host vital that holds warranty end dates as `YYYY-MM-DD`. The default is `AppleCare warranty expiration`.
- `DEX_GRID_INTENSITY`: grams of CO2 per kWh for computers with no GeoIP country. Without it, Sustainability uses the figure for the fleet's most common country.
- `DEX_WORKERS`: requests to Fleet at once (default 8).
- `DEX_SCORING_FILE`: a JSON file of experience score settings, in the shape **Download settings** saves on the Experience score page. Settings saved on the published page win over it. `run.py --scoring FILE` does the same for one run.

## Experience score

Experience score is built last, from the other pages' data. Its history keeps each computer's signals, one character per signal per day, not its score, so the page scores every day with the current settings and a settings change applies to the past too. `build.py` keeps the same scoring in Python for the overview's card, so the two always agree. The history is capped at about 12 MB: a year of days on a small fleet, and about 54 days at 10,000 computers.
- `DEX_COMPANY_DOMAINS`: comma-separated email domains that count as company sign-ins for coding agents. Without it, AI governance uses the domains of end users from the identity provider, leaving out personal email providers such as gmail.com.
