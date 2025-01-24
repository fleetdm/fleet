# Backend patterns

The backend software patterns that we follow in Fleet.

> NOTE: There are always exceptions to the rules, but we try to follow these patterns as much as possible unless a specific use case calls
> for something else. These should be discussed within the team and documented before merging.

Table of Contents
- [API Inputs](#api-inputs)
- [Go](#go)
- [MySQL](#mysql)

## API Inputs

### Input preprocessing and validation

Validate API inputs and return a 4XX status code if invalid. If you did not do authorization checking before failing validation, skip the authorization check with `svc.authz.SkipAuthorization(ctx)`.

Inputs corresponding to sortable or indexed DB fields should be preprocessed (trim spaces, normalize Unicode, etc.). Use utility method `fleet.Preprocess(input string) string`. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=4055688254267958899).

Invalid inputs should NOT log a server error. Server errors should be reserved for unexpected/serious issues. [`InvalidArgumentError` implements `IsServerError`](https://github.com/fleetdm/fleet/blob/75671e406183de2484c245ba424a4cddaaf8da06/server/fleet/errors.go#L134) method to indicate that it is a client error. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=6515110653090875786&highlights=%5B%7B%22type%22%3A%22SHARE%22%2C%22from%22%3A340%2C%22to%22%3A1578%7D%5D).

### JSON unmarshaling

`PATCH` API calls often need to distinguish between a field being set to `null` and a field not being present in the JSON. Use the structs from `optjson` package to handle this. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=4055688254267958899). [JSON unmarshaling article and example](https://victoronsoftware.com/posts/go-json-unmarshal/).

## Go

### Integer number types

Use `int` number type for general integer numbers. See [Why does len() returned a signed value?](https://stackoverflow.com/questions/39088945/why-does-len-returned-a-signed-value) for some context.

Exceptions:
- Database IDs
- Extra range of unsigned needed for a specific use case
- Specific performance/memory requirements

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
Going forward, we need to keep this data in a dedicated table(s). A reference unmerged PR is [here](https://github.com/fleetdm/fleet/pull/17472/files#diff-57a635e42320a87dd15a3ae03d66834f2cbc4fcdb5f3ebb7075d966b96f760afR16).
The `id` may be the same as that of the original table. For example, if the `user` row is deleted, a new entry with the same `user.id` can be added to `user_persistent_info`.

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