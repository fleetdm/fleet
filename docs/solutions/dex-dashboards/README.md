# DEX dashboards

Scripts that build three dashboards from the Fleet API: Hardware lifecycle, Software usage, and Patch velocity. A daily Claude Code routine runs them and publishes each dashboard as a private claude.ai Artifact. To set them up, follow the guide: [Build DEX dashboards with Fleet and Claude](https://fleetdm.com/guides/build-dex-dashboards-with-fleet-and-claude).

| File | What it does |
|:--|:--|
| `collect.py` | Reads hosts, software, and two DEX reports from the Fleet API. It only sends `GET` requests. |
| `build.py` | Builds one dashboard page and updates that page's history file. |
| `run.py` | Runs `collect.py`, then builds all three pages. The routine calls this. |
| `bundle.py` | Packages the code above into one file for the routine and prints its sha256. |
| `templates/` | Page templates. |
| `dex-reports.yml` | The two DEX reports the dashboards read. |
| `routine-prompt.md` | The routine's prompt, with placeholders the setup fills in. |

The scripts need Python 3.9 or later and no third-party packages. To try them locally, set `FLEET_URL` and `FLEET_API_TOKEN`, then run:

```sh
python3 run.py --work ./out --first-run
```

That writes `out/out/<page>.html` and `out/out/<page>-history.json`. Optional: set `DEX_WARRANTY_VITAL` to the name of a custom host vital that holds warranty end dates as `YYYY-MM-DD`. The default is `AppleCare warranty expiration`.
