# Count Apple applications installed

Count the number of Apple applications installed on the machine.

### Support
macOS

### Query
```sql
SELECT
  COUNT(*)
FROM
  apps
WHERE
  bundle_identifier LIKE 'com.apple.%';
```
### Purpose
Informational

### Remediation
N/A
