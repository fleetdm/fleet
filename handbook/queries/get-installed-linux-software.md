# Get installed Linux software

Get all software installed on a Linux computer, including browser plugins and installed packages.

> This does not included other running processes in the `processes` table.

### Support
Linux

### Query
```sql
SELECT
  name AS name,
  version AS version
  'apt_sources' AS source
FROM apt_sources
UNION
SELECT
  name AS name,
  version AS version
  'deb_packages' AS source
FROM deb_packages
UNION
SELECT
  package AS name,
  version AS version
  'portage_packages' AS source
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version
  'rpm_packages' AS source
FROM rpm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'yum_sources' AS source
FROM yum_sources
UNION
SELECT
  name AS name,
  version AS version,
  'opera_extensions' AS source
FROM opera_extensions
UNION
SELECT
  name AS name,
  version AS version,
  'npm_packages' AS source
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  'atom_packages' AS source
FROM atom_packages
UNION
SELECT
  name AS name,
  version AS version,
  'python_packages' AS source
FROM python_packages;
```

### Purpose

Informational

### Remediation

N/A