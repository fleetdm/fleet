# Get crashes

Retrieve application, system, and mobile app crash logs.

### Support
macOS

### Query
```sql
SELECT uid, datetime, responsible, exception_type, identifier, version, crash_path FROM users CROSS JOIN crashes USING (uid);
```
### Purpose
Informational

### Remediation
N/A