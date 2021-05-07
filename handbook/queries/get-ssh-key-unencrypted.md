# Detect unencrypted SSH keys 

## Description
SSH keys can be used in Lateral Movement, where attackers use unencrypted SSH keys to access other systems.

### Platforms
Linux, macOS

### Query: SSH Keys for Local Accounts

```sql
SELECT uid, username, description, path, encrypted FROM users cross join user_ssh_keys using (uid) WHERE encrypted=0;
```

### Query: SSH Keys for Domain Accounts

```sql
SELECT uid,username,description,path, encrypted FROM users cross join user_ssh_keys using (uid) WHERE encrypted=0 and username in (SELECT distinct(username) FROM last);
```
### Purpose

Detection

### Remediation

- Contact the impacted user to rotate their keys and use a passphrase when they generate keys
- raise users awareness around SSH keys

