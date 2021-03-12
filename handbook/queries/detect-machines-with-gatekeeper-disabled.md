# Detect machines with Gatekeeper disabled

Gatekeeper tries to ensure only trusted software is run on a mac machine.

### Platforms
macOS

### Query
```sql
SELECT * FROM gatekeeper WHERE assessments_enabled = 0;
```

### Purpose

Detection

### Remediation

TODO