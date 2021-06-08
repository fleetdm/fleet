# Get macOS disk free space percentage

Displays the percentage of free space available on the primary disk partition.

### Support
macOS

### Query
```sql
SELECT (blocks_available * 100 / blocks) AS pct, * FROM mounts WHERE path = '/';
```

### Purpose

Informational

### Remediation

N/A