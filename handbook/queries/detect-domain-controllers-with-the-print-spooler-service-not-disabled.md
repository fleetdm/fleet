# Detect domain controllers with the print spooler service not disabled

Detect domain controllers with the print spooler service not disabled to help mitigate the Windows Print Spooler Remote Code Execution Vulnerability. CVE-2021-1675. Attribution to [@maravedi](https://github.com/maravedi).

### Platforms
Windows

### Query
```sql
SELECT CASE cnt
           WHEN 2 THEN "TRUE"
           ELSE "FALSE"
       END "Vulnerable"
FROM
  (SELECT name,
          start_type,
          COUNT(name) AS cnt
   FROM services
   WHERE name = 'NTDS' or (name = 'Spooler' and start_type <> 'DISABLED'))
WHERE Cnt = 2;
```

### Purpose

Detection

### Remediation

TODO