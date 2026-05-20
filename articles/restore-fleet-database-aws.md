# Restore a Fleet Database on AWS

This guide provides the steps for restoring the Aurora database for a [self-hosted Fleet deployment on AWS using Terraform](https://github.com/fleetdm/fleet-terraform/tree/main).

The `db-restore.sh` script creates a new database cluster from a point-in-time recovery or snapshot. It updates Terraform configuration and brings services back online.

This guide covers deployments created with the [fleet-terraform `example` module](https://github.com/fleetdm/fleet-terraform/tree/main/example) and the [BYO-VPC `example` module](https://github.com/fleetdm/fleet-terraform/tree/main/byo-vpc/example).

All commands in this guide use `example` as the environment name. Replace `example` with your environment directory name. Replace `fleet-terraform/example` with the path to your Terraform checkout.

## Prerequisites

- Download the `db-restore.sh` script from [fleet-terraform/tools/aws-rds-restore](https://github.com/fleetdm/fleet-terraform/tree/main/tools/aws-rds-restore)
- Your Terraform environment directory checked out locally
- AWS credentials with permissions to manage RDS, ECS, IAM, Secrets Manager, and EC2 security groups
- `terraform`, `aws` CLI, `jq`, `perl`, and `python3` available on your `PATH`

## What happens during restore

The script performs the following steps in order:

1. Copies Terraform state and resource metadata to a `.db-restore-<timestamp>/` directory.
2. Ensures `fleet_image` is defined in `main.tf`. Extracts the value from ECS state if the field is missing.
3. Scales Fleet ECS services to `0` so no tasks connect during the restore.
4. Removes the old cluster, secrets, and parameter groups from Terraform state.
5. Creates a new Aurora cluster from the restore point.
6. Restores `rds_config` and reapplies monitoring settings.
7. Applies ECS services and runs database migrations.
8. Scales ECS services back to the previous count.

The script keeps the old database resources. Clean them up only after you validate the restore.

## Choose your restore method

Use point-in-time recovery (PITR) for recent incidents. Use a snapshot when the PITR window has expired or the restore point predates your backup window.

- **PITR**: Use `--restore-time` with an ISO-8601 timestamp within your backup window.
- **Snapshot**: Use `--restore-snapshot` with the snapshot ID or ARN.

## Before you restore

Complete these steps before running the restore.

### Step 1: List available restore points

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --list
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

Run a dry run with your intended flags before committing to the restore:

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --restore-time 2026-05-05T11:00:00Z \
  --dry-run
```

The dry run prints the execution path and planned changes. Verify the destination name, restore method, and target modules are correct.

### Step 3: Decide your restore options

Determine which additional flags your restore requires:

- **Inspect first**: Add `--no-ecs-apply` to leave ECS services at `0`. Apply ECS targets manually after validating the database.
- **Skip migrations**: Add `--skip-migrations` when the Fleet version has not changed since the restore point.
- **Roll back Fleet image**: Add `--rollback --fleet-image <version>` when restoring to a point before a Fleet upgrade. Choose the image version that matched the database at the restore time.

## Restore the database

Run the restore with the flags you confirmed during verification:

### PITR restore

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --restore-time 2026-05-05T11:00:00Z \
  --confirm
```

### Snapshot restore

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --restore-snapshot rds:<your-cluster>-2026-04-06-02-08 \
  --confirm
```

### Restore with image rollback

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --restore-time 2026-05-05T10:45:00Z \
  --rollback \
  --fleet-image v4.84.0 \
  --confirm
```

Pass a bare tag like `v4.84.0` to update only the version. Pass a full image URI to replace the registry as well.

### Restore without restarting services

```bash
AWS_PROFILE=<profile> db-restore.sh example \
  --env-dir fleet-terraform/example \
  --restore-time 2026-05-05T11:00:00Z \
  --no-ecs-apply \
  --confirm
```

ECS services remain at `0`. Validate the database manually. Then apply the ECS targets with Terraform to bring Fleet back online.

## Clean up old resources

The restore keeps the old database cluster for safe rollback. After you confirm Fleet is running on the new cluster, clean up the old resources:

```bash
AWS_PROFILE=<profile> db-restore.sh \
  --cleanup-only \
  --manifest example/.db-restore-<timestamp>/manifest.json \
  --confirm
```

Run with `--dry-run` first to preview what will be deleted. Cleanup removes the old cluster, instances, secrets, parameter groups, subnet groups, security groups, and automated snapshots.

## Troubleshooting

**Auto-detect fails for module address**

Pass `--module-address` explicitly. The value is `module.fleet.module.byo-vpc` for a standard layout or `module.byo-vpc.module.byo-db` for a BYO-VPC layout.

**Snapshot not found or fails to restore**

Run `--list` to verify the snapshot ID or ARN. Snapshots must exist in the same AWS region and account as your environment.

**PITR timestamp outside restore window**

Run `--list` to see the valid PITR window. Choose a timestamp 5-10 minutes before the incident occurred.

**Timestamp format is invalid**

Use UTC ISO-8601 format. The script accepts `2026-05-05T11:00:00Z` or `2026-05-05T11:00:00.000Z`.

**Missing script options**

Run `db-restore.sh --help` to print the full list of flags.

<meta name="articleTitle" value="Rollback and Restore Fleet Database on AWS">
<meta name="authorGitHubUsername" value="BCTBB">
<meta name="authorFullName" value="Jorge Falcon">
<meta name="publishedOn" value="2026-05-12">
<meta name="category" value="guides">
<meta name="description" value="Step-by-step guide to restoring and rolling back a Fleet environment on AWS using Terraform.">
