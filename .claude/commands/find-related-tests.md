Look at my recent git changes (`git diff` and `git diff --cached`) and find all related test files.

For each modified file, find:
1. The `_test.go` file in the same package
2. Integration tests that exercise the modified code (check `server/service/integration_*_test.go` files)
3. Any test helpers or fixtures that may need updating

List the test files and suggest specific test functions to run with the exact `go test` commands, including the right env vars (MYSQL_TEST, REDIS_TEST, etc.).
