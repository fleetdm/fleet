# Detect presence of authorized SSH keys

Presence of authorized SSH keys may be unusual on laptops. Could be completely normal on servers, but may be worth auditing for unusual keys and/or changes.

### Platforms
macOS, Linux

### Query
```sql
SELECT username, authorized_keys.* 
FROM users 
CROSS JOIN authorized_keys USING (uid);
```

### Purpose

Detection

### Remediation

TODO