# Get authorizes keys

List authorized_keys for each user on the system.

### Support
macOS, Linux

### Query
```sql
SELECT * FROM users CROSS JOIN authorized_keys USING (uid);
```
### Purpose
Informational

### Remediation
N/A
