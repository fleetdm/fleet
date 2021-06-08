# Get laptops with failing batteries

### Platforms
macOS

### Query
```sql
SELECT * FROM battery WHERE health != 'Good' AND condition NOT IN ('', 'Normal');
```

### Purpose

Informational

### Remediation

N/A