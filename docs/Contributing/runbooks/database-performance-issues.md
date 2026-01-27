## Database performance issues

### Use this runbook if

1. A customer environment is experiencing elevated error rate, outages, slow load times, or timeouts/502s.
2. Database load has not been eliminated as a cause of the issues.

This runbook is written for an engineering audience; if you're on the infrastructure team, you'll have access to these tools directly rather than needing to ask for them.

### Process

#### 1. Check RDS insights

If available (e.g. on managed cloud customers, or self-hosted customers running on RDS), check active queries in AWS RDS insights. For managed cloud environments, ask infrastructure for this information. For self-hosted environments, ask the customer.

#### 2. If locks are the problem, check them

If transaction locks are the source of issues, run [troubleshoot_locks.sql](https://github.com/fleetdm/confidential/blob/main/infrastructure/cloud/scripts/sql/troubleshoot_locks.sql) on the database to find which locks are causing the issue.

#### 3. Check table row counts

As of Fleet 4.81, managed cloud environments include table row counts as part of logs generated post-database-migration, with DB migrations happening on each deploy. Compare these row counts with load test info below to see if we're dealing with an environment that is shaped differently than we've load tested.

##### For Fleet < 4.81

The query used for this check is

```sql
SELECT table_name, COALESCE(table_rows, 0) table_rows
FROM information_schema.tables
WHERE table_schema = (SELECT DATABASE());
```

which can be run directly on a MySQL reader, in case a self-hosted customer wants to pull this data without running the migration command, or if they are using a Fleet version prior to 4.81.

##### Cloud environments

For cloud environments on >= 4.81, you can scan CloudWatch Logs for the appropriate row counts line:

```shell
TODO
```

##### Self-hosted

For self-hosted environments on >= 4.81, running `fleet prepare` with the `--with-table-stats` will provide this information in real time.

This command is safe to run without taking systems offline as the migrations themselves are a no-op in those cases and we pull approximate row counts from MySQL's `information_schema` table to get close-enough numbers with minimal overhead.

##### Compare with load test benchmark data

Here's an example of a load test envvironment's row counts by table, updated 2026-XX-YY:

```
TODO
```

##### 4. TODO
