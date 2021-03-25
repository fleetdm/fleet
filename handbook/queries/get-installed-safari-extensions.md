# Get installed Safari Extensions

Retreives the list of installed Safari Extensions for all users in the target system.

### Support
macOS

### Query
```sql
SELECT safari_extensions.* FROM users join safari_extensions USING (uid);
```
### Purpose
Informational

### Remediation
N/A