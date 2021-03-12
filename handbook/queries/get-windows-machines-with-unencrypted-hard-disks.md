# Get Windows machines with unencrypted hard disks

### Platforms
Windows

### Query
```sql
SELECT * FROM bitlocker_info WHERE protection_status = 0;
```

### Purpose

Informational

### Remediation

N/A