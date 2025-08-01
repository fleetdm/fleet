# Backend patterns

The backend software patterns that we follow in Fleet.

> NOTE: There are always exceptions to the rules, but we try to follow these patterns as much as possible unless a specific use case calls
> for something else. These should be discussed within the team and documented before merging.

Table of Contents
- [API Inputs](#api-inputs)
- [Go](#go)
- [MySQL](#mysql)
  - [Timestamps](#timestamps)
  - [UUIDs](#uuids)
  - [Say no to `goqu`](#say-no-to-goqu)
  - [Data retention](#data-retention)
  - [Re-usable transactionable functions](#re-usable-transactionable-functions)
- [Specific features](#specific-features)
  - [GitOps](#gitops)

## API Inputs

### Input preprocessing and validation

Validate API inputs and return a 4XX status code if invalid. If you did not do authorization checking before failing validation, skip the authorization check with `svc.authz.SkipAuthorization(ctx)`.

Inputs corresponding to sortable or indexed DB fields should be preprocessed (trim spaces, normalize Unicode, etc.). Use utility method `fleet.Preprocess(input string) string`. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=4055688254267958899).

Invalid inputs should NOT log a server error. Server errors should be reserved for unexpected/serious issues. [`InvalidArgumentError` implements `IsClientError`](https://github.com/fleetdm/fleet/blob/529f4ed725117d99d668318aad23c9e1575fa7ee/server/fleet/errors.go#L134) method to indicate that it is a client error. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=6515110653090875786&highlights=%5B%7B%22type%22%3A%22SHARE%22%2C%22from%22%3A340%2C%22to%22%3A1578%7D%5D).

### JSON unmarshaling

`PATCH` API calls often need to distinguish between a field being set to `null` and a field not being present in the JSON. Use the structs from `optjson` package to handle this. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=4055688254267958899). [JSON unmarshaling article and example](https://victoronsoftware.com/posts/go-json-unmarshal/).

## Go

### Integer number types

Use `int` number type for general integer numbers. See [Why does len() returned a signed value?](https://stackoverflow.com/questions/39088945/why-does-len-returned-a-signed-value) for some context.

Exceptions:
- Database IDs
- Extra range of unsigned needed for a specific use case
- Specific performance/memory requirements

### Unit testing

Use multiple hosts in unit tests and manual QA. For example, use a Windows VM and a Windows bare metal host when testing Windows profiles. Since our customers run Fleet on many hosts, we must be vigilant regarding multi-host use cases. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8290454302335084423).

See [the migration test in PR #28601](https://github.com/fleetdm/fleet/pull/28601/files#diff-53ce88f00ff80a0f7c0a1a2e23b14f6cb7ed5d7a7d91146236f499a756935869)
and [the test added in PR #30578](https://github.com/fleetdm/fleet/pull/30578/files#diff-124b43a1afae8960d4eb3765b2a5d5525c5ffba57c9b59ff78eb6cd222532e1c)
for examples of multi-host automated testing added to validate P0 bugfixes.

#### Assert vs require

Use the `require` package (from `testify/require`) for most test assertions. The `require` package stops test execution immediately on failure, which is usually what you want.

Use the `assert` package (from `testify/assert`) when you want to test multiple properties of a response so that you can fix all issues at once. This is particularly useful when validating multiple fields of a struct or API response.

Example of when to use `assert`:
```go
// Testing multiple response fields - use assert to see all failures at once
assert.Equal(t, http.StatusOK, resp.StatusCode)
assert.NotNil(t, resp.Body)
assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
assert.Greater(t, len(resp.Hosts), 0)
```

Example of when to use `require`:
```go
// Critical setup or early assertions - use require to fail fast
require.NoError(t, err) // Stop if there's an error
require.NotNil(t, user) // Stop if user is nil to avoid panic

// Now safe to access user fields
assert.Equal(t, "expected@example.com", user.Email)
```

General guidelines:
- Use `require` for preconditions and critical assertions that would cause panics or make subsequent tests meaningless
- Use `require` when testing a single thing or when the test should stop on the first failure
- Use `assert` when validating multiple independent properties where seeing all failures helps debugging
- In table-driven tests, prefer `require` within each test case to avoid confusion between different cases

## MySQL

### Timestamps

Use high precision for all time fields. Precise timestamps make sure that we can accurately track when records were created and updated,
keep records in order with a reliable sort, and speed up testing by not having to wait for the time to
update. [MySQL reference](https://dev.mysql.com/doc/refman/8.4/en/date-and-time-type-syntax.html). [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8041045095900447703).
Example:

```sql
CREATE TABLE `sample` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `created_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
);
```

### UUIDs

Use `binary` (or `varbinary`) data type for UUIDs. [MySQL 8 has good support for UUIDs](https://dev.mysql.com/blog-archive/mysql-8-0-uuid-support/) with `UUID_TO_BIN` and `BIN_TO_UUID` functions. If needed, add a virtual table to display UUID as string. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=5477893933055484926&highlights=%5B%7B%22type%22%3A%22SHARE%22%2C%22from%22%3A440%2C%22to%22%3A612%7D%5D).

Benefits of binary UUIDs include:
- Smaller storage size
- Faster indexing/lookup

### Say no to `goqu`

Do not use [goqu](https://github.com/doug-martin/goqu); use MySQL queries directly. Searching for, understanding, and debugging direct MySQL
queries is easier. If needing to modify an existing `goqu` query, try to rewrite it in
MySQL. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8041045095900447703).

### Data retention

Sometimes we need data from rows that have been deleted from DB. For example, the activity feed may be retained forever, and it needs user info (or host info) that may not exist anymore.
We need to keep this data in dedicated table(s). When a `user` row is deleted, a new entry with the same `user.id` is added to `users_deleted` table. The user info can be retrieved using
`ds.UserOrDeletedUserByID` method.

### Re-usable transactionable functions

Sometimes we want to encapsulate a piece of functionality in such a way that it can be use both
independently and as part of a transaction. To do so, create a private function in the following way: 

```go
func myTransactionableFunction(ctx context.Context, tx sqlx.ExtContext, yourArgsHere any) error {
  // some setup, statements, etc...

  _, err := tx.ExecContext(ctx, stmt, args)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "doing some stuff in a transaction")
	}
}
```

You can then use the function as a standalone call, like so

```go
// *sqlx.DB implements the sqlx.ExtContext interface
err := myTransactionableFunction(ctx, ds.writer(ctx), myArgs)
```

or as part of a transaction, like so

```go
func (ds *Datastore) MyDSMethodWithTransaction(ctx context.Context, yourArgsHere any) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return myTransactionableFunction(ctx, tx, yourArgsHere)
	})
}
```

See [this commit](https://github.com/fleetdm/fleet/pull/22843/files#diff-c5babdad542a72acf2ec2ecb7cb43967fc53850b6998ac629e253336b87e008bR415)
for an example of this pattern.

## Specific features

### GitOps

[GitOps documentation](https://fleetdm.com/docs/configuration/yaml-files)

`fleetctl gitops` was implemented on top of the existing `fleetctl apply` command. Now that `apply` no longer supports the newest features,
we need to separate the code for the two commands.

Common issues and gotchas:

- Removing a setting. When a setting is removed from the YAML config file, the GitOps run should remove it from the server. Sometimes, the
  removal doesn't happen since `apply` did not work like this. Also, developers/QA may forget to test this case explicitly.
- Few integration tests. GitOps is a complex feature with an extensive state space because many settings interact. At the same time, setting
  up a test environment for GitOps is difficult. As we work on GitOps, we need to add more integration tests and develop testing utilities
  to make adding future integration tests easier.
- GitOps admin can define settings in `default.yml`, `teams/team-name.yml`, or `teams/no-team.yml`. Create unit tests for all these cases
  for features that support them.
