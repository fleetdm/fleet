# Fleet Playwright Test Suite

Automated browser tests for Fleet, split into two suites: **e2e** for
functional coverage and **loadtest** for performance measurement against
high-scale instances.

---

## Setup

**1. Install dependencies**

```bash
cd qa/playwright
npm install
npx playwright install --with-deps
```

**2. Create your local env files**

```bash
cp .env.example .env
cp .env.loadtest.example .env.loadtest
```

Fill in both files with the appropriate Fleet URL and credentials. These
files are gitignored and never committed.

---

## Running tests

From `qa/playwright/`:

| Command | What it runs |
|---|---|
| `npm run test:e2e` | All e2e tests, headless |
| `npm run test:e2e:headed` | All e2e tests, browser visible |
| `npm run test:e2e:ui` | All e2e tests, Playwright UI |
| `npm run test:loadtest` | All loadtest tests, headless |
| `npm run test:loadtest:headed` | All loadtest tests, browser visible |
| `npm run test:loadtest:ui` | All loadtest tests, Playwright UI |

---

## Structure

```
qa/playwright/
├── tests/                        # All test specs — single source of truth
│   ├── auth/                     # Authentication tests
│   ├── hosts/                    # Hosts page tests
│   └── performance/              # Page load timing tests (one file per section)
├── setup/
│   ├── e2e.setup.ts              # Logs into e2e instance, saves session
│   └── loadtest.setup.ts         # Logs into loadtest instance, saves session
├── helpers/
│   ├── auth.ts                   # Shared login helper
│   ├── perf.ts                   # Timing measurement and result writing
│   └── perf-teardown.ts          # Aggregates timing files into final summary table
├── .env                          # e2e credentials (gitignored)
├── .env.loadtest                 # loadtest credentials (gitignored)
├── .env.example                  # Template for .env
└── .env.loadtest.example         # Template for .env.loadtest
```

---

## How the two suites differ

| | e2e | loadtest |
|---|---|---|
| Target | Regular Fleet instance | High-scale / loadtest instance |
| Tests | All tests not tagged `@loadtest` or `@perf` | Tests tagged `@loadtest` |
| Includes perf tests | No | Yes (`@perf` implies `@loadtest`) |
| Retries on failure | Yes (in CI) | No — a slow run is a slow run |
| Auth state | `.auth/e2e-admin.json` | `.auth/loadtest-admin.json` |

---

## Adding tests

**New functional test (e2e only):**
Add a spec to `tests/` with no tag. It will run in e2e automatically.

**New test that should also run on loadtest:**
Add `{ tag: '@loadtest' }` to the test:
```ts
test('my test', { tag: '@loadtest' }, async ({ page }) => { ... });
```

**New performance test:**
Add `{ tag: ['@loadtest', '@perf'] }` and use `measureNav` from `helpers/perf.ts`.
The result is automatically included in the summary table at the end of the loadtest run.

---

## Performance summary

At the end of every loadtest run, a timing table is printed:

```
─────────────────────────────────────────
 Performance Summary
─────────────────────────────────────────
 Section    Page              Load Time
─────────────────────────────────────────
 Dashboard  Dashboard         1.878s
 Hosts      Hosts list        1.611s
            Specific host     1.378s
 Controls   OS Updates        1.865s
            ...
─────────────────────────────────────────
```

Load times are color-coded in terminal output:
- **Orange** — over 5s
- **Red** — over 15s

---

## CI

Tests run via GitHub Actions (`workflow_dispatch`) with a suite dropdown.
Required secrets per suite:

| Secret | Used by |
|---|---|
| `FLEET_E2E_URL` | e2e |
| `FLEET_E2E_ADMIN_EMAIL` | e2e |
| `FLEET_E2E_ADMIN_PASSWORD` | e2e |
| `FLEET_LOADTEST_URL` | loadtest |
| `FLEET_LOADTEST_ADMIN_EMAIL` | loadtest |
| `FLEET_LOADTEST_ADMIN_PASSWORD` | loadtest |
