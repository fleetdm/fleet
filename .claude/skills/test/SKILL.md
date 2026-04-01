---
name: test
description: Run Go tests related to recent changes with appropriate environment variables. Use when asked to "run tests", "test my changes", or "test this".
allowed-tools: Bash(go test *), Bash(MYSQL_TEST*), Bash(MYSQL_TEST=1 *), Bash(MYSQL_TEST=1 REDIS_TEST=1 *), Bash(FLEET_INTEGRATION_TESTS_DISABLE_LOG=1 *), Read, Grep, Glob
---

Run Go tests related to my recent changes. Look at `git diff` and `git diff --cached` to determine which packages were modified.

For each modified package, run the tests with appropriate env vars:
- If the package is under `server/datastore/mysql`: use `MYSQL_TEST=1`
- If the package is under `server/service`: use `MYSQL_TEST=1 REDIS_TEST=1`
- Otherwise: run without special env vars

If an argument is provided, use it as a `-run` filter: $ARGUMENTS

Show a summary of results: which packages passed, which failed, and any failure details.
