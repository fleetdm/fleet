# Online migrations

This document has two sections:

1. [Migrations PoC](#migrations-poc) with details and takeaways of building a
   PoC of the migrations proposal as described [here][migrations-proposal].
2. [Alternative proposal](#alternative-proposal) with another possible solution
   for the same problem, but using [MySQL Online DDL][mysql-ddl].

## Migrations PoC

I was able to build a PoC using the ideas described in the original
[proposal][migrations-proposal]. The approach is effective but has certain
[complexities and limitations](#complexities-and-limitations).

### General implementation details

#### Building migrations

The meat of the proposal is to define and document a new way to construct
migrations, so two Fleet instances can use the same database concurrently,
without having to shut down the server.

Starting with a simple table, `labels`, I simulated adding a new column
`priority` of type `INT`:

First, a new table needs to be created. 

```sql
CREATE TABLE labels_1 LIKE labels;
ALTER TABLE labels_1 ADD COLUMN priority int(10) NOT NULL DEFAULT '0';
``

Data from the `labels` table needs to be copied over  `labels_1`, to do this,
we add the following:

```sql
INSERT LOW_PRIORITY IGNORE INTO labels_1 SELECT * FROM labels;
```

If the table is big, we should consider splitting the rows into chunks
and do the insert in batches, but I didn't built that for the PoC.

If the table has FKs, you need to add `ALTER` statements to set those, as they
aren't created with `CREATE TABLE ... LIKE`, taking special care on the naming
as duplicates aren't allowed. 

Tools like `pt-online-schema-change` that handle this automatically, first do a
`SHOW CREATE TABLE` to get the table structure and then manipulate the string
output.

Then, insert the triggers. For the sake of simplicity, I will  include triggers
from `labels` -> `labels_1` in this document, the PoC included triggers in the
opposite direction using the same boilerplate.

```sql
# we need to set a custom delimiter in order to execute BEGIN END; statements
# in a trigger
delimiter //

# since MySQL doesn't have ON CONFLICT IGNORE, we craft our own, this is copyied
# from the percona toolkit
CREATE TRIGGER labels_labels_1_ins AFTER INSERT ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1442 BEGIN END;
REPLACE INTO labels_1 VALUES (
  NEW.id, NEW.created_at, NEW.updated_at, NEW.name,
  NEW.description, NEW.query, NEW.platform, NEW.label_type,
  NEW.label_membership_type
);
END;

CREATE TRIGGER labels_labels_1_del AFTER DELETE ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1442 BEGIN END;
DELETE IGNORE FROM labels_1 WHERE id = OLD.id;
END;

CREATE TRIGGER labels_labels_1_upd AFTER UPDATE ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1442 BEGIN END;
DELETE IGNORE FROM labels_1 WHERE id = OLD.id;
REPLACE INTO labels_1 VALUES (
  NEW.id, NEW.created_at, NEW.updated_at, NEW.name,
  NEW.description, NEW.query, NEW.platform, NEW.label_type,
  NEW.label_membership_type
);
END;

delimiter ;
```

For the cleanup script: `DROP` the `labels` table, and run a `DROP TRIGGER` for
each trigger.

#### Cleanup

The proposal introduces a two-face database migration:
1. run `fleet prepare-db`
2. run `fleet clean-db` as a last step.

To support this, "cleanup" migrations are treated as a different set: stored a
in a different folder, and tracked using a new database table
`cleanup_migrations_status_table`. This is mostly boilerplate thanks to
`goose`.

#### Table versions

Using different table versions is easy. The intuition of using a `const` to set
the table name in a place works fine, just a bit boilerplate-y to build
queries, which shouldn't be a problem with the right abstraction.

### Complexities and limitations

The process described works for a simple table, things get more complex
with foreign keys and relationships.

**Parent tables**

A major concern with this approach is with "parent" tables, tables that are
referenced by other tables and have foreign key constraints on them.

Updating a foreign key to point to the new table requires an `ALTER` statement,
we've to do the whole versioning procedure for any "child" table referencing a
"parent" table being modified.

This problem is also recursive, because a "child" table can itself be
referenced by other tables (for example the `users` table)

I did write an SQL migration simulating this scenario, but it's long enough to
not be included in this document.

Having said that, we can mitigate this problem with two considerations:

1. As mentioned in the original document, we can use `ALTER` directly on
   eventually consistent tables.
2. If a table is small enough (we should test before rolling the change)
   `ALTER`s aren't locking.

**Child tables**

The simplest case is a "child" table that references another table and has a
foreign key constraint. Certain aspects require special care:

1. Explicitly create the foreign key constraints on the new table
2. While both tables are "live," to `DELETE` a record from the parent table we
   first have to delete all references from both child tables. This is covered
   with triggers, but is an area that we should load test and inspect for write
   races.

**Races and duplicates**

Another general concern is with data races and the fact we use `AUTO_INCREMENT`
IDs. Can we have conflicts if an insert is performed simultaneously in
`table_n` and `table_n+1`? is this realistic?

## Alternative proposal

Another way to tackle this problem could be with the combination of:

1. Ensuring that two immediate Fleet versions can use the same database without
   problems, this involves developer effort and planning. For example you can't
   drop a column between two immediate Fleet versions, to do so you need a
   release between that stops using the column altogether (more details
   below.)
2. [MySQL online DDL][mysql-ddl], which lets you to write non-locking alter
   statements via `ALTER ... LOCK=NONE`. This functionality is not available
   for certain operations (notably changing a column data type) but they're
   very few, all have a workaround, and per `1` we don't want immediate
   breaking changes anyway.

Common scenarios (based on the latest 10 migrations):

- Adding a new column: append `LOCK=NONE` to the `ALTER` statement. Two notes:
    - `AUTO_INCREMENT` are the exception as they don't support `LOCK=NONE`
    - We'll have to get rid of all `SELECT *` statements, otherwise adding a
      column is not backwards compatible.
- Renaming a column: don’t rename, instead use three different Fleet versions:
  - `vN+1`:  add a new column with `LOCK=NONE`, allow `NULL` values
    so the current running version can read and write without problems.
  - `vN+2`: change the code to stop writing to the column.
  - `vN+3`: as the old column isn't used by any running Fleet version, remove
    it, add `NOT NULL` constraints if necessary.
- Change the data type of a column: follow the Rename process instead.
- Index operations: create, drop, rename and change type allow `LOCK=NONE`, use
  a procedure analogous to renaming a column.
- Rename table: renaming allows `LOCK=NONE`, for backwards compatibility are
  multiple options, including using a temporary view.

For a full list of which operations are allowed without locks, check
[here][mysql-ddl-table], all operations that have "yes" under "Permits
Concurrent DM" allow the use of `LOCK=NONE`, and those with "yes" under "In
Place" allow `LOCK=SHARED` which enables you to read data from the table.

#### Building migrations

With this new approach, the same migration to add a new column to the `labels`
table it's a regular migration with `LOCK=NONE` and a default value:

```sql
ALTER TABLE labels ADD COLUMN priority int(10) NOT NULL DEFAULT '0' LOCK=NONE;
```

A different example will illustrate the complexities/limitations of
this approach better:

**Renaming a table**

A table rename like we did in `20220526123327_RenameCVEScoresToCVEMeta.go`
using this approach looks like:

1. With Fleet `vN` running, deploy Fleet `vN+1`, which uses the new table name.
   The migrations for `vN+1` create a view with the new table name, so both
   versions can use the database simultaneously.

```sql
CREATE VIEW cve_meta AS SELECT * FROM cve_scores;
```

2. In the next release, with `vN+1` running, deploy `vN+2`. At this point
   neither of the Fleet versions make use of `cve_scores`, so we can rename the
   table and get rid of the view in the migrations for `vN+2`:

```sql
RENAME TABLE
   cve_meta TO cve_meta_view,
   cve_scores to cve_meta;

DROP VIEW cve_meta_view;
```

> note: `RENAME TABLE` doesn't accept `LOCK=NONE`, but it's an atomic operation
> that modifies the metadata of the table.
>
> `pt-online-schema-change` ([code][pt-osc-code] [description][pt-osc-desc])
> uses this exact same approach.

#### Ensuring all migrations are performed

Given that we need multiple Fleet releases to perform a migration, we need a way to
enforce that the right migrations are performed at the right time.

This needs exploration but here are a few ideas:

1. From @chiip

> If the PR has an ddl change, then a CI could start failing once a given tag
> is created. Eg:
> 
> If we are in version 4.15, we could be forced to add a line on a file that
> reads:
> 
> ```
> fleet-4.15.0 remove column blah
> ```
> 
> So if there's a new tag for fleet, CI could fail until there's a PR that
> removed that line and adds the migration.
> 
> Sounds convoluted, so I don't know if it's the best, but the best I can come
> up with right now at least.

2. From @lucasmrod

> Also, I wonder how we can merge all the migrations steps needed for a feature
> to `main` (vN, vN+1 and vN+2), but only apply them when it's actually the
> correct Fleet version. We would need something like that right?
>
> Alternatively, we keep somehow merge those future migrations (vN+1 and vN+2)
> to main but keep them in a "limbo".

3. Write all the migrations at once but in different folders. After a release
   is completed, a script is run (ideally by a bot), which copies the
   migrations that belong to the current in-development Fleet version.

4. Write a helper:

```go
func DoBefore(msg, fleetVersion string) {
  if isCI  && fleetVersion > currentFleetVersion {
    panic("you should " + msg + " before " + fleetVersion)
  }
}

// somewhere in the code:
DoBefore("drop x column in hosts table", "4.18.0")
```

### Load testing

@juan-fdz-hawa performed load testing of this feature [in a PR][juan-pr] with
promising results. We still need to simulate a high load scenario with a
more complex migration (like the renaming a table example.)

### Contingency plan

What happens if a migration introduces an undesired change? how can we go back to a previous version?

**Current state**

Going back to a previous version using the same database isn't supported,
because [we don't do `Down` migrations][contributing-migrations].

**Possible options**

A few possible options:

1. Start writing `Down` migrations again, if something goes wrong:
  a. Stop the new Fleet version, keep the old Fleet version running
  b. Run the `Down` migration
  c. Wait until a patch version with the fixes is released.
2. Since the database can be used by two Fleet versions:
  a. Stop the new Fleet version, keep the old Fleet version running
  b. Wait for a patch version, which fixes the migration issue

`2` is a bit riskier because a bad migration can corrupt/break the database,
but that's inline with the current state and perhaps this issue should be
tackled separately.

### Limitations

1. This approach needs special planning and care between versions. We've to
   look into ways to keep the house in order and make sure columns/indexes/etc
   are removed.
2. Data is replicated sequentially leading to replication lags.
3. The MySQL documentation often mentions that for some concurrent operations
   data is reorganized substantially, making it an expensive operation. We need
   to compare this with the cost of recursively creating a new table + copying
   the data for parent tables.
4. For this to work, users that want live migrations can't skip Fleet major/minor versions
   in upgrades: they can't go from `vN` to `vN+2` without going
   through `vN+1` first.


[migrations-proposal]: https://docs.google.com/document/d/1lv67XVhLbejgeS6Vi43C8wqvjb6wRpc07zy1Guv-3VA
[mysql-ddl]: https://dev.mysql.com/doc/refman/5.7/en/innodb-online-ddl.html
[mysql-ddl-table]: https://dev.mysql.com/doc/refman/5.7/en/innodb-online-ddl-operations.html
[pt-osc-code]: https://github.com/percona/percona-toolkit/blob/896fdcede8362ea14d60feb23afa657b00803851/bin/pt-online-schema-change#L10904-L10907
[pt-osc-desc]: https://www.percona.com/doc/percona-toolkit/LATEST/pt-online-schema-change.html#description
[juan-pr]: https://github.com/fleetdm/fleet/pull/6489#discussion_r915401088
[contributing-migrations]: https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Migrations.md
