# Get installed Chrome Extensions

List installed Chrome Extensions for all users.

### Support
macOS, Linux, Windows, FreeBSD

### Query
```sql
SELECT * FROM users CROSS JOIN chrome_extensions USING (uid);
```
### Purpose
Informational

### Remediation
N/A
