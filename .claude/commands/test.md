Run Go tests related to my recent changes. Look at `git diff` and `git diff --cached` to determine which packages were modified.

For each modified package, run the tests with appropriate env vars:
- If the package is under `server/datastore/mysql`: use `MYSQL_TEST=1`
- If the package is under `server/service`: use `MYSQL_TEST=1 REDIS_TEST=1`
- Otherwise: run without special env vars

If an argument is provided, use it as a `-run` filter: $ARGUMENTS

Show a summary of results: which packages passed, which failed, and any failure details.
