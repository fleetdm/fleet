Fix failing tests from a CI run. The argument is a GitHub Actions run URL or run ID: $ARGUMENTS

## Step 1: Identify failing jobs

Extract the run ID from the URL (the numeric path segment after `/runs/`). Use `gh run view <run_id>` to list the jobs, then find the failing ones:

```
gh run view <run_id> --json jobs --jq '.jobs[] | select(.conclusion == "failure") | {name: .name, id: .databaseId}'
```

Group the failing jobs by **test suite** (the first parenthesized token in the job name, e.g. `integration-core`, `integration-enterprise`, `service`, `mysql`, `main`). You only need to examine **one job per unique suite** since the matrix variants (OS, MySQL version) run the same tests.

## Step 2: Find the failing tests in each suite

For each unique suite, fetch the job log and find the `FAIL: ` lines. IMPORTANT: use `gh api` (not `gh run view --log`, which may return empty):

```
gh api repos/fleetdm/fleet/actions/jobs/<job_id>/logs 2>&1 | grep -e 'FAIL: ' | head -30
```

This gives you the failing test function names and subtests. Ignore the parent test if subtests are listed (e.g. if `TestFoo` and `TestFoo/Bar` both appear, focus on `TestFoo/Bar`).

## Step 3: Get error details

For each suite, fetch the error traces:

```
gh api repos/fleetdm/fleet/actions/jobs/<job_id>/logs 2>&1 | grep -e 'FAIL: \|Error Trace\|Error:\|expected:\|actual:' | head -60
```

This tells you the exact file/line and what the assertion expected vs. what it got.

## Step 4: Diagnose each failure

For each failing test, read the test code at the indicated file and line. Determine whether the failure is:

**A) A stale test assertion** — the test expects an old string/value but the production code was intentionally changed. The test needs updating to match the new behavior. Signs:
- The expected value is an old error message string and the actual value is a new one
- The change aligns with the intent of the current branch's modifications
- The production code change looks intentional

**B) A legitimate test failure** — the test is correct but the code under test is buggy. The production code needs fixing. Signs:
- The test's expected value matches the documented/intended behavior
- The actual value indicates a regression or bug
- The test was not related to any intentional change on this branch

## Step 5: Fix stale assertions (category A)

For each stale assertion:
1. Read the test file
2. Update the assertion to match the new expected value
3. Also search for **other assertions in the same file** that check similar strings — CI only catches the first failure per test, so there may be additional stale assertions that haven't failed yet. Use Grep to find them.
4. Also check for **related assertions in other test files** for the same error message pattern

## Step 6: Report legitimate failures (category B)

For each legitimate failure, report to the user:
- The test name and file location
- What the test expects vs. what it got
- Your analysis of why the production code is producing the wrong result
- The production code file/line that likely needs fixing

Do NOT fix production code bugs without user approval — only report them.

## Step 7: Verify fixes

After fixing stale assertions, run the affected tests locally to verify they pass:

- `pkg/spec/...` and `server/fleet/...`: `go test -run 'TestName' ./pkg/spec/...`
- `server/service/...` (unit tests like devices_test.go, scripts_test.go): `go test -run 'TestName' ./server/service/`
- `ee/server/service/...`: `go test -run 'TestName' ./ee/server/service/`
- `server/datastore/mysql/...`: `MYSQL_TEST=1 go test -run 'TestName' ./server/datastore/mysql/`
- Integration tests (`integration_core_test.go`, `integration_enterprise_test.go`, `integration_live_queries_test.go`): these require `MYSQL_TEST=1 REDIS_TEST=1` and take a long time, so just verify compilation with `go build ./...`

After running tests, also do a proactive Grep scan for any remaining old assertion strings in test files that might break in CI even though they didn't show up in this run (CI stops at the first failure per test function).

## Step 8: Report summary

Present a summary to the user:
- Total failing suites and tests found
- How many were stale assertions (fixed) vs. legitimate failures (reported)
- List of files modified
- Any remaining concerns or tests that couldn't be verified locally
