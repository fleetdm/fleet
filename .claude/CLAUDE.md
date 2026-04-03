## Running Tests

```bash
# Quick Go tests (no external deps)
go test ./server/fleet/...

# Integration tests
MYSQL_TEST=1 go test ./server/datastore/mysql/...
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...

# Run a specific test
MYSQL_TEST=1 go test -run TestFunctionName ./server/datastore/mysql/...

# Generate boilerplate for a new frontend component, including associated stylesheet, tests, and storybook
./frontend/components/generate -n RequiredPascalCaseNameOfTheComponent -p optional/path/to/desired/parent/directory
```

## Go code style

- Prefer `map[T]struct{}` over `map[T]bool` when the map represents a set.
- Convert a map's keys to a slice with `slices.Collect(maps.Keys(m))` instead of manually appending in a loop.
- Avoid `time.Sleep` in tests. Prefer `testing/synctest` to run code in a fake-clock bubble, or use polling helpers, channels, or `require.Eventually`.
- Use `require` and `assert` from `github.com/stretchr/testify` in tests.
- Use `t.Context()` in tests instead of `context.Background()`.
- Use `any` instead of `interface{}`
- Use `math/rand/v2` instead of `math/rand`.
- Use `for i := range 10` style loops instead of `for i := 0; i < 10; i++` to loop over integers

## DB Migrations

Use the following command to create a new migration

```bash
# Make a new migration named MyMigration (add your actual migration name)
make migration name=MyMigration
```

Your new migration will be created under server/datastore/mysql/migrations/tables/[timestamp]_MyMigration.go along with a corresponding _test.go file
Add your migration and test code to those files, then regenerate the DB schema file (used for tests):
```bash
make test-schema
```

