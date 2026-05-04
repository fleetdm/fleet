---
name: test
description: Run tests related to recent changes with appropriate tools and environment variables. Use when asked to "run tests", "test my changes", or "test this".
allowed-tools: Bash(go test *), Bash(MYSQL_TEST*), Bash(MYSQL_TEST=1 *), Bash(MYSQL_TEST=1 REDIS_TEST=1 *), Bash(FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 *), Bash(yarn test*), Bash(npx jest*), Bash(git diff*), Bash(git status*), Read, Grep, Glob
effort: low
---

Run tests related to my recent changes. Look at `git diff` and `git diff --cached` to determine which files were modified.

## Go tests

For each modified Go package, run the tests with appropriate env vars:
- If the package is under `server/datastore/mysql`: use `MYSQL_TEST=1`
- If the package is under `server/service`: use `MYSQL_TEST=1 REDIS_TEST=1`
- Otherwise: run without special env vars

## Frontend tests

If any files under `frontend/` were modified, run the relevant frontend tests:
- Find test files matching the changed components (e.g., `ComponentName.tests.tsx`)
- Run with: `yarn test --testPathPattern "path/to/changed/component"`
- If many files changed, run the full suite: `yarn test`

## Choosing what to run

- If only Go files changed, run Go tests only
- If only frontend files changed, run frontend tests only
- If both changed, run both
- If an argument is provided, use it as a filter: $ARGUMENTS (passed as `-run` for Go or `--testPathPattern` for frontend)

Show a summary of results: which packages/suites passed, which failed, and any failure details.
