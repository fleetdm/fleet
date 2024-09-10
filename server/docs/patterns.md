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
