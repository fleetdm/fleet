# Get mounts

Shows system mounted devices and filesystems (not process specific).

### Support
macOS, Linux

### Query
```sql
SELECT device, device_alias, path, type, blocks_size FROM mounts;
```
### Purpose
Informational

### Remediation
N/A