# Migrating from Bitnami Sub-charts to Fleet v7.x

## Overview

Starting with Fleet Helm chart v7.0.0, the bundled MySQL and Redis sub-charts have been replaced:

| Component | Old (v6.x) | New (v7.x) |
|-----------|-----------|------------|
| MySQL | Bitnami MySQL 9.12.5 | Minimal local chart using official `mysql:8.4` image |
| Redis | Bitnami Redis 18.1.6 | [Valkey](https://valkey.io) 0.9.3 (Redis-compatible) |

**If you are NOT using the sub-charts** (i.e., `mysql.enabled: false` and `redis.enabled: false`, which is the default), **you are not affected by this change**. No action is needed.

## Who is affected?

Only users who deploy Fleet with `mysql.enabled: true` and/or `redis.enabled: true` in their Helm values. These sub-charts are intended for dev/test convenience — production deployments should use external MySQL and Redis instances.

## Value key mapping

### MySQL

| Old (Bitnami) | New | Notes |
|---------------|-----|-------|
| `mysql.enabled` | `mysql.enabled` | Unchanged |
| `mysql.auth.username` | `mysql.auth.username` | Unchanged |
| `mysql.auth.password` | `mysql.auth.password` | Unchanged |
| `mysql.auth.database` | `mysql.auth.database` | Unchanged |
| `mysql.auth.rootPassword` | `mysql.auth.rootPassword` | Unchanged |
| `mysql.auth.existingSecret` | `mysql.auth.existingSecret` | Unchanged |
| `mysql.primary.persistence.enabled` | `mysql.primary.persistence.enabled` | Unchanged |
| `mysql.primary.persistence.size` | `mysql.primary.persistence.size` | Unchanged |
| `mysql.primary.persistence.storageClass` | `mysql.primary.persistence.storageClass` | Unchanged |
| `mysql.primary.resources` | `mysql.primary.resources` | Unchanged |
| `mysql.primary.livenessProbe.enabled` | *(removed)* | Probes are always enabled; not configurable |
| `mysql.primary.readinessProbe.enabled` | *(removed)* | Probes are always enabled; not configurable |
| `mysql.primary.startupProbe.enabled` | *(removed)* | No startup probe in new chart |
| `mysql.image.repository` | `mysql.image.repository` | Default changed from `bitnami/mysql` to `mysql` |
| `mysql.image.tag` | `mysql.image.tag` | Default: `8.4` |

### Redis → Valkey

| Old (Bitnami) | New | Notes |
|---------------|-----|-------|
| `redis.enabled` | `redis.enabled` | Unchanged — kept for backwards compatibility |
| `redis.architecture: standalone` | `valkey.replica.enabled: false` | Standalone is the default |
| `redis.auth.enabled` | `valkey.auth.enabled` | Same semantics |
| `redis.auth.password` | *(see Valkey auth docs)* | Valkey uses ACL-based auth |

## Service DNS name changes

| Old | New |
|-----|-----|
| `<release>-mysql` | `<release>-mysql` | **Unchanged** |
| `<release>-redis-master` | `<release>-valkey` | **Changed** |

Update your `cache.address` value accordingly:

```yaml
cache:
  address: <release>-valkey:6379    # was: <release>-redis-master:6379
```

## Migration scenarios

### Fresh install

No special steps. Use the new value keys documented above.

### Existing deployment — data can be reprovisioned

1. Update your values file with the new keys (see mapping above).
2. Update `cache.address` to `<release>-valkey:6379`.
3. Delete old sub-chart PVCs if they exist:
   ```sh
   kubectl delete pvc -l app.kubernetes.io/instance=<release>,app.kubernetes.io/name=mysql -n <namespace>
   kubectl delete pvc -l app.kubernetes.io/instance=<release>,app.kubernetes.io/name=redis -n <namespace>
   ```
4. Run `helm upgrade`.

### Existing deployment — MySQL data must be preserved

The Bitnami MySQL chart stores data at `/bitnami/mysql/data/`, while the official MySQL image uses `/var/lib/mysql/`. A direct upgrade will result in an empty database.

1. **Back up your MySQL data:**
   ```sh
   kubectl exec -n <namespace> <release>-mysql-0 -- mysqldump -u root -p --all-databases > backup.sql
   ```

2. **Scale Fleet to 0 and disable the old MySQL:**
   ```sh
   kubectl scale deploy/<release>-fleet -n <namespace> --replicas=0
   ```

3. **Update your Helm values** with the new keys and run `helm upgrade`.

4. **Restore data into the new MySQL:**
   ```sh
   kubectl exec -i -n <namespace> <release>-mysql-0 -- mysql -u root -p < backup.sql
   ```

5. **Scale Fleet back up:**
   ```sh
   kubectl scale deploy/<release>-fleet -n <namespace> --replicas=<desired>
   ```

### Existing deployment — switching Redis to Valkey

Fleet uses Redis as a cache only — no persistent data migration is needed. Simply:

1. Update `cache.address` to `<release>-valkey:6379`.
2. Remove any `redis.architecture` or Bitnami-specific Redis values.
3. Add `valkey.*` values as needed (see mapping above).
4. Run `helm upgrade`.
