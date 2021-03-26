# Get macos disk free space percentage

Displays the percentage of free space available on the primary disk partition

### Support
MacOS

### Query
```sql
SELECT (blocks_available * 100 / blocks) AS pct FROM mounts WHERE device='/dev/disk1s1';
```

### Purpose

Informational

### Remediation

N/A