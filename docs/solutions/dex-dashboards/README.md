# DEX dashboards

Scripts that build seven dashboards from the Fleet API for macOS, Windows, and Linux: Hardware lifecycle, Software usage, Patch velocity, AI governance, Connectivity, Analytics, and Sustainability. A daily Claude Code routine runs them and publishes each dashboard as a private claude.ai Artifact. To set them up, follow the guide: [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude).

| File | What it does |
|:--|:--|
| `collect.py` | Reads hosts, software, and 13 DEX reports from the Fleet API. It only sends `GET` requests. |
| `build.py` | Builds one dashboard page and updates that page's history file. |
| `run.py` | Runs `collect.py`, then builds all seven pages. The routine calls this. |
| `bundle.py` | Packages the code above into one file for the routine and prints its sha256. |
| `templates/` | Page templates. |
| `dex-reports.yml` | The 13 DEX reports the dashboards read: five for Macs, three for Windows security, one for AI tools on every platform, and four for Sustainability (two for Macs, one each for Windows and Linux). |
| `routine-prompt.md` | The routine's prompt, with placeholders the setup fills in. |

The scripts need Python 3.9 or later and no third-party packages. To try them locally, set `FLEET_URL` and `FLEET_API_TOKEN`, then run:

```sh
python3 run.py --work ./out --first-run
```

That writes `out/out/<page>.html` and `out/out/<page>-history.json`. Optional: set `DEX_WARRANTY_VITAL` to the name of a custom host vital that holds warranty end dates as `YYYY-MM-DD`. The default is `AppleCare warranty expiration`. Sustainability estimates carbon from each computer's GeoIP country. For computers with no country, set `DEX_GRID_INTENSITY` to grams of CO2 per kWh. Without it, the page uses the figure for the fleet's most common country.
