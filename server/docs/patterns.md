# Backend patterns

The backend software patterns that we follow in Fleet.

> NOTE: There are always exceptions to the rules, but we try to follow these patterns as much as possible unless a specific use case calls
> for something else. These should be discussed within the team and documented before merging.

## MySQL

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

Do not use [goqu](https://github.com/doug-martin/goqu); use MySQL queries directly. Searching for, understanding, and debugging direct MySQL
queries is easier. If needing to modify an existing `goqu` query, try to rewrite it in
MySQL. [Backend sync where discussed](https://us-65885.app.gong.io/call?id=8041045095900447703).

### Data retention

Sometimes we need data from rows that have been deleted from DB. For example, the activity feed may be retained forever, and it needs user info (or host info) that may not exist anymore.
Going forward, we need to keep this data in a dedicated table(s). A reference unmerged PR is [here](https://github.com/fleetdm/fleet/pull/17472/files#diff-57a635e42320a87dd15a3ae03d66834f2cbc4fcdb5f3ebb7075d966b96f760afR16).
The `id` may be the same as that of the original table. For example, if the `user` row is deleted, a new entry with the same `user.id` can be added to `user_persistent_info`.
