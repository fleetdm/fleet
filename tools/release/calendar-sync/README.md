# Release calendar sync

Keeps the "Fleet releases" Google Calendar in sync with the GitHub milestone
due dates on `fleetdm/fleet`. For each open milestone with a `X.Y.Z` title, the
script makes sure the calendar has:

- `Release day: minor release - X.Y.Z` on the milestone's due date
- `Release candidate (next release - X.Y.Z)` ending on the milestone's due date
- `Develop (next release - X.Y.Z)` ending ~2 weeks before the due date
  (skipped for out-of-band patch-like milestones)

The script proposes changes by default (dry-run). Pass `--apply` to write
changes back to the calendar.

## How matching works

Calendar events are matched to milestones by **date proximity**, not by title.
This means renumbering (e.g. inserting an out-of-band 4.88.0 that shifts every
later minor) is handled automatically:

- A `Release day:` event whose date matches milestone X.Y.Z's due date is
  retitled to X.Y.Z (regardless of its current version label).
- An RC event whose **end** date matches a milestone's due date is retitled
  and its end is adjusted to exactly match.
- A Develop event whose end falls ~14 days before a milestone due date is
  matched to that milestone.

A milestone is considered **out-of-band** if its due date is fewer than
14 days after the previous milestone's due date (normal cadence is 21 days).
Out-of-band milestones get a Release day + a short RC event only; they do not
get a Develop sprint.

## Local usage (OAuth user login — easiest for testing)

Log in as yourself in the browser; no service account or calendar sharing needed.
You just need an OAuth client ID (one-time, below).

```bash
cd tools/release/calendar-sync
python3 -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt

# Optional, raises GitHub rate limit:
export GITHUB_TOKEN=$(gh auth token)

# client_secret.json in this dir (see one-time setup below), then:
python sync.py --oauth          # dry-run; opens a browser the first time
python sync.py --oauth --apply  # apply changes
```

The first run opens a browser to approve access and caches a refreshable token
to `token.json`, so later runs don't prompt. Both `client_secret.json` and
`token.json` are gitignored.

### One-time: create an OAuth client ID

1. Google Cloud Console → **APIs & Services → Library** → enable **Google Calendar API**.
2. **APIs & Services → Credentials → Create credentials → OAuth client ID**.
   - Application type: **Desktop app**.
   - (If prompted to configure the consent screen, pick **Internal** for a
     Workspace org, add yourself as a test user if **External**.)
3. **Download JSON** and save it as `tools/release/calendar-sync/client_secret.json`
   (or point `GCAL_OAUTH_CLIENT_SECRET` / `--client-secret` at its path).

## Local usage (service account)

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
# or:
export GCAL_SERVICE_ACCOUNT_JSON="$(cat /path/to/service-account.json)"
export GITHUB_TOKEN=$(gh auth token)   # optional

python sync.py               # dry-run, prints proposed changes
python sync.py --apply       # apply changes
```

Auth precedence: `--oauth` forces the browser login; otherwise a service account
is used if `GCAL_SERVICE_ACCOUNT_JSON` or `GOOGLE_APPLICATION_CREDENTIALS` is set;
otherwise it falls back to the OAuth user flow.

## Service-account setup (one-time)

1. In Google Cloud Console, create (or pick) a project and enable the
   **Google Calendar API**.
2. Create a **service account**. Generate a JSON key.
3. Note the service account's email (looks like
   `release-calendar-sync@…iam.gserviceaccount.com`).
4. Open the **Fleet releases** calendar in Google Calendar, go to
   *Settings and sharing → Share with specific people or groups*, and add
   the service account email with the **Make changes to events** permission.
5. Store the JSON key contents in the GitHub secret
   `GCAL_SERVICE_ACCOUNT_JSON` on the `fleetdm/fleet` repo.

## GitHub Action

The workflow `.github/workflows/calendar-sync.yml` is triggered manually
(`workflow_dispatch`). It dry-runs by default; check the **job summary** for
the proposed change list, then re-trigger with `apply = true` to write the
changes.

## Editing defaults

Constants near the top of `sync.py`:

| Constant | Meaning |
|---|---|
| `CALENDAR_ID` | Calendar to manage. |
| `OUT_OF_BAND_GAP_DAYS` | Gap below which a milestone is treated as patch-like. |
| `PATCH_RC_DURATION_DAYS` | Length of the short RC window for out-of-band milestones. |
| `DEVELOP_END_TO_DUE_TARGET_DAYS` | Expected gap from Develop end to next release. |

The "RC ritual", "Create patch RC", and "Publish patch release" events are
intentionally **not** touched by this script.
