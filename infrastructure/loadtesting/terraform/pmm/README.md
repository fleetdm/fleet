# PMM for Fleet loadtesting

Deploys [Percona Monitoring and Management (PMM)](https://docs.percona.com/percona-monitoring-and-management/) as an optional
add-on for a loadtest environment. PMM runs on ECS Fargate behind the internal ALB and automatically registers the loadtest's
Aurora MySQL for metrics dashboards. Adapted from the local dev setup in `tools/percona/pmm/`.

## What PMM is for (and not for)

Use PMM when chasing a MySQL-internal bottleneck. Its dashboards expose `SHOW GLOBAL STATUS` counters that neither CloudWatch
nor Performance Insights surface. These include InnoDB buffer pool hit rate and evictions, redo log and checkpoint pressure,
row lock waits and deadlocks, handler rates, temp tables on disk, thread and table cache churn, and connection errors.

Do NOT use PMM's Query Analytics (QAN) for per-query analysis. Fleet runs its parametrized DML as binary-protocol prepared
statements. MySQL never aggregates those into the `performance_schema` digest table that QAN reads, so QAN misses most of the
write path (verified on MySQL 8.0.44 and 8.4.9). For per-query analysis use one of these instead:

- **AWS Performance Insights** (enabled on loadtest instances). Server-side and protocol-agnostic. Group by `db.sql_tokenized`.
- **SigNoz**. Client-side otelsql spans with full query text and the Fleet endpoint or cron context that issued them.

## Prerequisites

- A running loadtest environment (`../infra`), deployed in the same Terraform workspace
- VPN connection (the PMM UI is internal-only)

## Usage

```sh
cd infrastructure/loadtesting/terraform/pmm
terraform init
terraform workspace select <workspace_name>   # must match your infra workspace
terraform apply
```

Outputs:

- `pmm_url` is the UI at `https://pmm-<workspace>.loadtest.fleetdm.com` (VPN required).
- `pmm_admin_password_secret` is the Secrets Manager secret holding the `admin` user's password:

```sh
aws secretsmanager get-secret-value \
  --secret-id fleet-<workspace>-pmm-admin-password \
  --query SecretString --output text
```

The MySQL metrics dashboards work out of the box and do not need `performance_schema`. That setting is OFF by default on the
loadtest Aurora cluster, and enabling it requires a parameter group change plus instance reboots.

## Teardown

```sh
terraform destroy
```

Destroy PMM before destroying the infra environment it monitors.
