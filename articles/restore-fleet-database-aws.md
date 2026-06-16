# Restore a Fleet database on AWS

This guide provides the steps for restoring the Aurora database for a [self-hosted Fleet deployment on AWS using Terraform](https://github.com/fleetdm/fleet-terraform/tree/main). Please contact Fleet support before performing this action, or proceed at your own risk.

The `db-restore.sh` script creates a new database cluster from a point-in-time recovery or snapshot. It updates Terraform configuration and brings services back online.

This guide covers deployments created with the [fleet-terraform `example` module](https://github.com/fleetdm/fleet-terraform/tree/main/example).

All commands in this guide use `example` as the environment name. Replace `example` with your environment directory name. Replace `fleet-terraform/example` with the path to your Terraform checkout.

> **Note:** Fleet built and tested `db-restore.sh` against the [`fleet-terraform/example`](https://github.com/fleetdm/fleet-terraform/tree/main/example) (Standard) deployment layout. If your Fleet deployment is not based on `fleet-terraform/example`, this guide will not support your restore.

## Prerequisites

- Your Terraform environment directory checked out locally
- `db-restore.sh` available somewhere on disk. Download it from `fleet-terraform/tools/rds-db-restore/db-restore.sh` ([link](https://github.com/fleetdm/fleet-terraform/blob/main/tools/rds-db-restore/db-restore.sh)), or check it out as part of the `fleet-terraform` repo. The script does not need to live inside your Terraform environment directory. `cd` into your environment directory and invoke the script via its absolute path (shown as `/path/to/db-restore.sh` in the examples below). Remember to `chmod +x` it after downloading.
- AWS credentials with permissions to manage RDS, ECS, IAM, Secrets Manager, and EC2 security groups
- `terraform`, `aws` CLI, `jq`, `perl`, and `python3` available on your `PATH`

## Quick start

Every restore command needs exactly one restore source. Use `--restore-time <iso-8601>` for point-in-time recovery (PITR), or `--restore-snapshot <id|arn>` to restore from an RDS DB cluster snapshot. The two flags are mutually exclusive but interchangeable. Each step below shows both forms.

1. **List available restore points:**
   ```bash
   cd fleet-terraform/example
   AWS_PROFILE=<profile> /path/to/db-restore.sh --list
   ```

2. **Dry-run the restore:**

   PITR:
   ```bash
   AWS_PROFILE=<profile> /path/to/db-restore.sh \
     --restore-time 2026-05-05T11:00:00Z \
     --dry-run
   ```
   Snapshot:
   ```bash
   AWS_PROFILE=<profile> /path/to/db-restore.sh \
     --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
     --dry-run
   ```

3. **Execute the restore:**

   PITR:
   ```bash
   AWS_PROFILE=<profile> /path/to/db-restore.sh \
     --restore-time 2026-05-05T11:00:00Z \
     --confirm
   ```
   Snapshot:
   ```bash
   AWS_PROFILE=<profile> /path/to/db-restore.sh \
     --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
     --confirm
   ```

## What happens during restore

The script performs the following steps in order:

1. Captures a full copy of Terraform state and resource metadata into `.db-restore-<timestamp>/`.
2. Optionally updates `fleet_config.image` if `--rollback` and `--fleet-image` are provided. Supports literal values and `local.*` references whose definitions are literals in the same file. `var.*` expressions are rejected.
3. Optionally adds/updates `rds_config.master_username` if `--master-username` is provided.
4. Updates the `rds_config` block to point to the restored database.
5. Scales Fleet ECS services to `0` so no tasks connect during the restore.
6. Removes old RDS resources from Terraform state.
7. Creates a new Aurora cluster from the chosen restore point.
8. Restores the original `rds_config` and re-applies monitoring/observability settings.
9. Applies ECS services and runs database migrations (`module.migrations`).
10. Scales ECS services back up.
11. Keeps old DB resources intact for safe cleanup later.

The script keeps the old database resources. Clean them up only after you validate the restore.

## Choose your restore method

Use point-in-time recovery (PITR) for recent incidents. Use a snapshot when the PITR window has expired or the restore point predates your backup window.

- **PITR**: Use `--restore-time` with an ISO-8601 timestamp within your backup window.
- **Snapshot**: Use `--restore-snapshot` with the snapshot ID or ARN.

`--restore-time` and `--restore-snapshot` are mutually exclusive but interchangeable. Once you pick a restore source, the remaining operational flags (`--no-ecs-apply`, `--rollback`, `--dry-run`, etc.) apply the same way to either source.

**Recommended pre-upgrade pattern.** Before any Fleet version upgrade, take a manual RDS DB cluster snapshot of the current database. If the upgrade misbehaves, you can roll the database and image back in one command using `--restore-snapshot <pre-upgrade-arn> --rollback --fleet-image <pre-upgrade-version>`. This gives you a defined, tested rollback point that doesn't depend on the PITR window still covering the incident.

## Before you restore

Complete these steps before running the restore.

### Step 1: List available restore points

```bash
cd fleet-terraform/example
AWS_PROFILE=<profile> /path/to/db-restore.sh --list
```

Example output:

```
PITR window:
  earliest: 2026-04-05T02:08:19.591000+00:00
  latest:   2026-05-05T11:18:46.739000+00:00

RDS DB cluster snapshots:
  2026-04-06T02:08:18.170000+00:00 automated available rds:<cluster>-2026-04-06-02-08 arn:aws:rds:...
```

Use a timestamp from the PITR window for `--restore-time`, or a snapshot identifier from the list for `--restore-snapshot`.

### Step 2: Verify the restore plan

Run a dry run with your intended flags before committing to the restore.

PITR:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-time 2026-05-05T11:00:00Z \
  --dry-run
```

Snapshot:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
  --dry-run
```

The dry run prints the execution path and planned changes. Verify the destination name, restore method, and target modules are correct.

## Restore the database

Run the restore with the flags you confirmed during verification.

PITR:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-time 2026-05-05T11:00:00Z \
  --confirm
```

Snapshot:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
  --confirm
```

### Restore with image rollback

PITR:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-time 2026-05-05T10:45:00Z \
  --rollback \
  --fleet-image v4.84.0 \
  --confirm
```

Snapshot:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
  --rollback \
  --fleet-image v4.84.0 \
  --confirm
```

Pass a bare tag like `v4.84.0` to update only the version. Pass a full image URI to replace the registry as well.

**Manual alternative.** The script can only rewrite `fleet_config.image` sources that are literals or `local.*` references. For other cases (for example, a `var.*` reference), edit `fleet_config.image` in `main.tf` to the rollback target by hand. Then run the restore without `--rollback` or `--fleet-image`. The restore picks up your manual edit.

### Restore without restarting services

PITR:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-time 2026-05-05T11:00:00Z \
  --no-ecs-apply \
  --confirm
```

Snapshot:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --restore-snapshot arn:aws:rds:us-east-2:123456789012:cluster-snapshot:fleet-prod-manual-2026-04-06 \
  --no-ecs-apply \
  --confirm
```

ECS services remain at `0`. Validate the database manually. Then apply the ECS targets with Terraform to bring Fleet back online.

## Clean up old resources

The restore keeps the old database cluster for safe rollback. After you confirm Fleet is running on the new cluster, clean up the old resources:

```bash
AWS_PROFILE=<profile> /path/to/db-restore.sh \
  --cleanup-only \
  --manifest .db-restore-<timestamp>/manifest.json \
  --confirm
```

`--manifest` accepts an absolute or relative path. Absolute paths are convenient when invoking the script from outside the environment directory.

Run with `--dry-run` first to preview what will be deleted. Cleanup removes the old cluster, instances, secrets, parameter groups, subnet groups, security groups, and automated snapshots.

## Troubleshooting

**Rollback fails before any AWS calls**

`--rollback --fleet-image` requires `fleet_config.image` to already be present in `main.tf` and to resolve from a source the script can rewrite. The expected behavior:

| `fleet_config.image` state | `--rollback --fleet-image` outcome |
|---|---|
| Not defined in `main.tf` | Fails. Define `fleet_config.image` before retrying. |
| Set to `var.<name>` | Fails. `var.*` references are rejected. Use the manual alternative described in the "Restore with image rollback" section. |
| Set to `local.<name>` where the local is a literal in the same file | Succeeds. |
| Set to a literal string | Succeeds. |

**Auto-detect fails for module address**

Pass `--module-address` explicitly. For the `fleet-terraform/example` (Standard) layout the value is `module.fleet.module.byo-vpc`.

**Snapshot not found or fails to restore**

Run `--list` to verify the snapshot ID or ARN. Snapshots must exist in the same AWS region and account as your environment.

**PITR timestamp outside restore window**

Run `--list` to see the valid PITR window. Choose a timestamp 5-10 minutes before the incident occurred.

**Timestamp format is invalid**

Use UTC ISO-8601 format. The script accepts `2026-05-05T11:00:00Z` or `2026-05-05T11:00:00.000Z`.

**Missing script options**

Run `db-restore.sh --help` to print the full list of flags.

<meta name="articleTitle" value="Restore a Fleet database on AWS">
<meta name="authorGitHubUsername" value="BCTBB">
<meta name="authorFullName" value="Jorge Falcon">
<meta name="publishedOn" value="2026-05-12">
<meta name="category" value="guides">
<meta name="description" value="Step-by-step guide to restoring and rolling back a Fleet environment on AWS using Terraform.">
