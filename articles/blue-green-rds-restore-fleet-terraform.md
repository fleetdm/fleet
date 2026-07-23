# Restore your Fleet database with a blue-green deployment

When your RDS Aurora cluster needs to be restored — whether from a snapshot or to a point in time — the blue-green approach keeps Fleet online while you spin up, validate, and cut over to a restored cluster. This guide covers environments deployed with [fleet-terraform](https://github.com/fleetdm/fleet-terraform/tree/main).

> **Fleet Premium customers:** Contact [Fleet support](https://fleetdm.com/support) before attempting this process. Our team will guide you through it.

## Prerequisites

- Fleet deployed with fleet-terraform at `tf-mod-root-v1.31.0` or later
- Terraform installed locally with access to your state
- AWS permissions for RDS and Secrets Manager
- A snapshot ARN **or** a UTC timestamp to restore to (e.g., `2026-07-17T06:14:01Z`)

## Instructions

### 1. Migrate to `rds_configs` with your existing cluster as `"current"`

> **Note:** Skip this step if you already use `rds_configs` in your `main.tf`.

Upgrade the module source to `v1.31.0` or later, rename `rds_config` to `rds_configs`, wrap the existing block in a `current` key, and add `active_rds_config_name`:

```diff
module "main" {
- source = "github.com/fleetdm/fleet-terraform?depth=1&ref=tf-mod-root-v1.30.0"
+ source = "github.com/fleetdm/fleet-terraform?depth=1&ref=tf-mod-root-v1.31.0"

- rds_config = {
+ rds_configs = {
+   current = {
      preferred_maintenance_window = "fri:04:00-fri:05:00"
      backup_retention_period      = 30
      skip_final_snapshot          = false
      db_parameters = {
        sort_buffer_size = 8388608
      }
      db_cluster_parameters = {
        require_secure_transport = "ON"
      }
      engine_version = "8.0.mysql_aurora.3.10.3"
      name           = local.rds_cluster_name
      instance_class = "db.t4g.medium"
      replicas       = 1
      cluster_tags = {
        VantaContainsUserData = "true"
        backup                = "true"
      }
+   }
  }
+ active_rds_config_name = "current"
}
```

Run `terraform apply`.

### 2. Add the `"next"` cluster with a restore

Add a `next` config alongside `current`. Set `monitoring_interval` to `0` and set `observability.performance_insights_enabled` to `false`. Aurora can take several minutes to make a restored cluster available, and enabling full monitoring too early causes apply errors.

Use `restore_to_point_in_time` **or** `snapshot_identifier` depending on your restore method:

**Point-in-time restore:**

```diff
  rds_configs = {
    current = { ... }
+   next = {
+     preferred_maintenance_window = "fri:04:00-fri:05:00"
+     backup_retention_period      = 30
+     skip_final_snapshot          = false
+     db_parameters = {
+       sort_buffer_size = 8388608
+     }
+     db_cluster_parameters = {
+       require_secure_transport = "ON"
+     }
+     engine_version = "8.0.mysql_aurora.3.10.3"
+     name           = "${local.rds_cluster_name}-next"
+     instance_class = "db.t4g.medium"
+     replicas       = 1
+     cluster_tags = {
+       VantaContainsUserData = "true"
+       backup                = "true"
+     }
+     monitoring_interval = 0
+     observability = {
+       performance_insights_enabled = false
+     }
+     restore_to_point_in_time = {
+       source_cluster_identifier = local.rds_cluster_name
+       restore_to_time           = "2026-07-17T06:14:01Z"
+     }
+   }
  }
  active_rds_config_name = "current"
```

**Snapshot restore:**

```diff
  rds_configs = {
    current = { ... }
+   next = {
+     preferred_maintenance_window = "fri:04:00-fri:05:00"
+     backup_retention_period      = 30
+     skip_final_snapshot          = false
+     db_parameters = {
+       sort_buffer_size = 8388608
+     }
+     db_cluster_parameters = {
+       require_secure_transport = "ON"
+     }
+     engine_version = "8.0.mysql_aurora.3.10.3"
+     name           = "${local.rds_cluster_name}-next"
+     instance_class = "db.t4g.medium"
+     replicas       = 1
+     cluster_tags = {
+       VantaContainsUserData = "true"
+       backup                = "true"
+     }
+     monitoring_interval = 0
+     observability = {
+       performance_insights_enabled = false
+     }
+     snapshot_identifier = "<your-snapshot-identifier>"
+   }
  }
  active_rds_config_name = "current"
```

Run `terraform apply`. Fleet continues serving traffic from `"current"` while the restore runs.

> **Note:** Running two clusters increases your AWS cost until you remove `"current"` in step 5.

### 3. Re-enable monitoring on `"next"`

Once the restore completes and the `"next"` cluster is available, remove the monitoring overrides so it picks up the module defaults:

```diff
    next = {
      ...
-     monitoring_interval = 0
-     observability = {
-       performance_insights_enabled = false
-     }
      restore_to_point_in_time = {
        source_cluster_identifier = local.rds_cluster_name
        restore_to_time           = "2026-07-17T06:14:01Z"
      }
    }
```

Run `terraform apply`.

### 4. Cut over to `"next"`

Switch Fleet to the restored cluster:

```diff
- active_rds_config_name = "current"
+ active_rds_config_name = "next"
```

Run `terraform apply`. Fleet now reads and writes to the restored cluster.

### 5. Remove `"current"` and update the monitoring secret

Delete the `current` block. If you use the fleet-terraform monitoring module, update it to reference `"next"`'s password secret. The module names each secret `<config.name>-database-password`:

```diff
  rds_configs = {
-   current = {
-     ...
-   }
    next = {
      ...
    }
  }
```

```diff
module "monitoring" {
  cron_monitoring = {
-   mysql_password_secret_name = "${local.rds_cluster_name}-database-password"
+   mysql_password_secret_name = "${local.rds_cluster_name}-next-database-password"
  }
}
```

Run `terraform apply`. The original cluster is deprovisioned and billing stops.

That's it! Fleet is now running against your restored database. Log in and verify that your data looks as expected.

<meta name="articleTitle" value="Restore your Fleet database with a blue-green deployment">
<meta name="authorGitHubUsername" value="BCTBB">
<meta name="authorFullName" value="Jorge Falcon">
<meta name="publishedOn" value="2026-07-23">
<meta name="category" value="guides">
<meta name="description" value="Restore Fleet's RDS database from a snapshot or point in time using blue-green deployment with fleet-terraform, with no Fleet downtime.">
