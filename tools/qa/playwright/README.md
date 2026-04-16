# Fleet Playwright Test Suite

Automated browser tests for Fleet, split into two suites: **e2e** for
functional coverage and **loadtest** for performance measurement against
high-scale instances.

---

## Setup

**1. Install dependencies**

```bash
cd tools/qa/playwright
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

From `tools/qa/playwright/`:

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
tools/qa/playwright/
├── tests/
│   ├── auth/                     # Login, logout, forgot-password tests
│   ├── hosts/                    # Hosts page tests
│   ├── smoke/                    # End-to-end smoke tests by area
│   │   ├── security-and-compliance/  # Vulnerability smoke tests
│   │   ├── mdm/
│   │   ├── orchestration/
│   │   └── software/
│   └── performance/              # Page load timing tests (one file per section)
├── setup/
│   ├── e2e.setup.ts              # Logs into e2e instance, saves session
│   └── loadtest.setup.ts         # Logs into loadtest instance, saves session
├── helpers/
│   ├── auth.ts                   # Shared login helper
│   ├── nav.ts                    # Table locators, dropdown interactions
│   ├── vuln.ts                   # Vulnerability test helpers (API, filters, assertions)
│   ├── perf.ts                   # Timing measurement and result writing
│   └── perf-teardown.ts          # Aggregates timing files into summary table with history
├── .env                          # e2e credentials (gitignored)
├── .env.loadtest                 # loadtest credentials (gitignored)
├── .env.example                  # Template for .env
├── .env.loadtest.example         # Template for .env.loadtest
├── .perf-history/                # Stored performance run history (gitignored)
└── playwright.config.ts
```

---

## How the two suites differ

| | e2e | loadtest |
|---|---|---|
| Target | Regular Fleet instance | High-scale / loadtest instance |
| Tests | All tests not tagged `@loadtest` or `@perf` | Tests tagged `@loadtest` |
| Includes perf tests | No | Yes (`@perf` implies `@loadtest`) |
| Retries on failure | Yes (in CI) | No — a slow run is a slow run |
| Timeouts | 30s test / 5s expect | 60s test / 30s expect |
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

**New smoke test:**
Add a spec to the appropriate `tests/smoke/<area>/` directory. Smoke tests
run in the e2e suite and follow real user flows (click-through navigation,
no direct URL access for internal pages).

---

## Performance summary

At the end of every loadtest run, a timing table is printed comparing the
current run against up to 3 previous runs:

```
───────────────────────────────────────────────────────────────────────
 Performance Summary
───────────────────────────────────────────────────────────────────────
 Section    Page              Current       prev-1      prev-2      prev-3
───────────────────────────────────────────────────────────────────────
 Dashboard  Platform cards    1.539s        1.655s      1.602s      1.580s
            Software block    0.873s        0.881s      0.884s      0.880s
            Activity block    1.400s        0.367s      0.370s      0.365s
 Hosts      Hosts list        1.436s        1.420s      1.415s      1.430s
            ...
───────────────────────────────────────────────────────────────────────
 3 previous run(s) | green = current faster | yellow = current slower
```

- Previous times in **green** where the current run is faster
- Previous times in **yellow** where the current run is slower
- Previous times in **gray** when the difference is negligible (<200ms)
- Current times in **yellow** when over 5s, **red** when over 15s

Run history is stored in `.perf-history/` (max 10 runs, oldest pruned automatically).

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
