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
```

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

# since MySQL doesn't have ON CONFLIC IGNORE, we craft our own, this is copyied
# from the percona toolkit
CREATE TRIGGER labels_labels_1_ins AFTER INSERT ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1146 BEGIN END;
REPLACE INTO labels_1 VALUES (
  NEW.id, NEW.created_at, NEW.updated_at, NEW.name,
  NEW.description, NEW.query, NEW.platform, NEW.label_type,
  NEW.label_membership_type
);
END;

CREATE TRIGGER labels_labels_1_del AFTER DELETE ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1146 BEGIN END;
DELETE IGNORE FROM labels_1 WHERE id = OLD.id;
END;

CREATE TRIGGER labels_labels_1_upd AFTER UPDATE ON labels
FOR EACH ROW
BEGIN
DECLARE CONTINUE HANDLER FOR 1146 BEGIN END;
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

1. Non-breaking DB changes between two immediate Fleet versions, so both Fleet
   versions can use the same database.
2. [MySQL online DDL][mysql-ddl], which lets you to write non-locking alter
   statements via `ALTER ... LOCK=NONE`. This functionality is not available
   for certain operations (notably changing a column data type) but they're
   very few, all have a workaround, and per `1` we don't want immediate
   breaking changes anyway.

Common scenarios (based on our latest 10 migrations):

- Adding a new column: as long as isn't `AUTO_INCREMENT`, `LOCK=NONE` is
  allowed
- Renaming a column: don’t rename, instead
  - `vN`: non breaking, add a new column with `LOCK=NONE`, allow `NULL` values.
  - `vN+1`: stop writing to the column.
  - `vN+2`: breaking, remove the old column, add `NOT NULL` constraints if
    necessary.
- Change the data type of a column: follow the Rename process instead.
- Index operations: create, drop, rename and change type allow `LOCK=NONE`, use
  a procedure analogous to renaming a column.
- Rename table: renaming allows `LOCK=NONE`, for backwards compatibility are
  multiple options, including using a temporary view.

For a full list of which operations are allowed without locks, check
[here][mysql-ddl-table], all operations that have "yes" under "Permits
Concurrent DM" allow the use of `LOCK=NONE`, and those with "yes" under "In
Place" allow `LOCK=SHARED` which enables you to read data from the table.

### Limitations

1. This approach needs special planning and care between versions. We have to
   look into ways to keep the house in order and make sure columns/indexes/etc
   are removed.
2. Data is replicated sequentially leading to replication lags.
3. The MySQL documentation often mentions that for some concurrent operations
   data is reorganized substantially, making it an expensive operation. We need
   to compare this with the cost of recursively creating a new table + copying
   the data for parent tables.


[migrations-proposal]: https://docs.google.com/document/d/1lv67XVhLbejgeS6Vi43C8wqvjb6wRpc07zy1Guv-3VA
[mysql-ddl]: https://dev.mysql.com/doc/refman/5.7/en/innodb-online-ddl.html
[mysql-ddl-table]: https://dev.mysql.com/doc/refman/5.7/en/innodb-online-ddl-operations.html
