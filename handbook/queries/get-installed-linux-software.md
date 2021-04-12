# Get installed Linux software

Get all software installed on a Linux computer, including browser plugins and installed packages.

> This does not included other running processes in the `processes` table.

### Support
Linux

### Query
```sql
SELECT
  name AS name,
  version AS version,
  architectures AS 'architecture',
  '' AS 'install_time',
  'apt_sources' AS source
FROM apt_sources
UNION
SELECT
  name AS name,
  version AS version,
  arch AS 'architecture',
  '' AS 'install_time',
  'deb_packages' AS source
FROM deb_packages
UNION
SELECT
  package AS name,
  version AS version,
  '' AS 'architecture',
  '' AS 'install_time',
  'portage_packages' AS source
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  arch AS 'architecture',
  install_time AS 'install_time',
  'rpm_packages' AS source
FROM rpm_packages
UNION
SELECT
  name AS name,
  '' AS version,
  '' AS 'architecture',
  '' AS 'install_time',
  'yum_sources' AS source
FROM yum_sources
UNION
SELECT
  name AS name,
  version AS version,
  '' AS 'architecture',
  '' AS 'install_time',
  'npm_packages' AS source
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS 'architecture',
  '' AS 'install_time',
  'atom_packages' AS source
FROM atom_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS 'architecture',
  '' AS 'install_time',
  'python_packages' AS source
FROM python_packages;
```

### Purpose

Informational

### Remediation

N/A
