# DEX dashboards

Scripts that build eight dashboards from the Fleet API for macOS, Windows, and Linux: Hardware lifecycle, Software usage, Patch velocity, AI governance, Connectivity, Analytics, Sustainability, and Policies. An overview page sums up all eight and links to each one. A daily Claude Code routine runs them and publishes each page as a private claude.ai Artifact. To set them up, follow the guide: [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude).

| File | What it does |
|:--|:--|
| `collect.py` | Reads hosts, software, and 13 DEX reports from the Fleet API, several requests at a time. It only sends `GET` requests, and keeps only the software rows a page reads. |
| `build.py` | Builds each dashboard page and updates its history, or builds the overview from the dashboards' histories. Splits a page's data, or a history, into part files when it's too big for one artifact file. |
| `run.py` | Collects, then builds all eight pages and the overview in one process, and prints the publish call for each page. The routine calls this. |
| `bundle.py` | Packages the code above into one file for the routine and prints its sha256. |
| `templates/` | Page templates. |
| `dex-reports.yml` | The 13 DEX reports the dashboards read: five for Macs, three for Windows security, one for AI tools on every platform, and four for Sustainability (two for Macs, one each for Windows and Linux). |
| `routine-prompt.md` | The routine's prompt, with placeholders the setup fills in. |

The scripts need Python 3.9 or later and no third-party packages. To try them locally, set `FLEET_URL` and `FLEET_API_TOKEN`, then run:

```sh
python3 run.py --work ./work --first-run --site ./site
```

That writes `work/out/<page>/index.html` with the page's `data/` folder (its `history.json`, and part files on a large fleet) for each dashboard and the overview. `--site` also writes a copy for your own web server: `site/index.html` is the overview, and each dashboard is in its own folder. Serve it over HTTP, because a page with part files can't fetch them from a local file. On later runs, copy `work/out` somewhere else, for example to `last`, and pass it with `--in last` so each page picks up its history.

## Large fleets

The scripts are tested with a simulated fleet of 10,000 computers: a run there takes about 90 seconds of processing and under 1 GB of memory, not counting time spent waiting on Fleet. Each page and history is split into part files of about 8 MB when it's bigger than that, because an artifact file can't be larger than 16 MB. The page fetches its parts before it draws, and `history.json` lists its parts under `__parts__`.

A run that needs more than about 9 minutes saves what it collected under `--work`, prints a line starting with `CONTINUE`, and exits with code 3. Run the same command again to pick up where it stopped. `DEX_WORKERS` sets how many requests go to Fleet at once (default 8). Lower it if your Fleet server is busy during runs.

## Environment variables

- `DEX_WARRANTY_VITAL`: the custom host vital that holds warranty end dates as `YYYY-MM-DD`. The default is `AppleCare warranty expiration`.
- `DEX_GRID_INTENSITY`: grams of CO2 per kWh for computers with no GeoIP country. Without it, Sustainability uses the figure for the fleet's most common country.
- `DEX_WORKERS`: requests to Fleet at once (default 8).
