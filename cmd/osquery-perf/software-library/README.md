# Software library for osquery-perf

This directory contains the software database and tools used by osquery-perf for load testing.

## Quick start

### Initial setup

1. Create the database:

```bash
sqlite3 software.db < software.sql
```

2. Verify the database (optional):

```bash
sqlite3 software.db "SELECT COUNT(*) FROM software;"

sqlite3 software.db "SELECT source, COUNT(*) FROM software GROUP BY source;"
# Shows distribution across sources
```

### Running osquery-perf

Once the database exists, osquery-perf will automatically use it:

```bash
cd ../..
./osquery-perf --host-count 1000
```

Each simulated host will get random platform-specific software from the database.

## Directory structure

```text
software-library/
├── README.md              # This file
├── software.db            # SQLite database (created from software.sql)
├── software.sql           # SQL dump with schema + data (source of truth)
├── tools/                 # Import and maintenance tools
│   ├── import-data/       # Import server data from CSV
│   └── generate-sql/      # Generate software.sql from database
└── source-data/           # Source CSV files (all gitignored)
    └── .gitignore

```

## Tools

### import-data

Imports software data from CSV files, validates entries, and optionally filters out internal/proprietary software.

**Usage:**
```bash
cd tools/import-data

# Import CSV file (no filtering)
go run . --input ../../source-data/server_export.csv

# Import with pattern filtering
go run . --input ../../source-data/server_export.csv --filter "numa-internal,numa-,corp-"

# Import with vendor filtering
go run . --input ../../source-data/server_export.csv --filter-vendor "numa"

# Dry run (validate without importing)
go run . --input ../../source-data/server_export.csv --dry-run

# Verbose output
go run . --input ../../source-data/server_export.csv --verbose
```

**What it does:**
- Reads software entries from CSV files
- **Optional filtering** (disabled by default):
  - `--filter`: Filter names containing specified patterns (comma-separated)
  - `--filter-vendor`: Filter software from specified vendor (except well-known public software)

### generate-sql

Generates `software.sql` file from the populated database.

**Usage:**
```bash
cd tools/generate-sql

# Generate software.sql
go run .

# Specify custom paths
go run . --db ../../software.db --output ../../software.sql

# Verbose output (shows progress)
go run . --verbose
```

**What it does:**
- Reads all data from `software.db`
- Generates SQL INSERT statements
- Includes schema definition
- Creates reproducible SQL dump

## Database setup workflow

Here's the typical workflow:

### Step 1: Initialize database from software.sql

```bash
sqlite3 software.db < software.sql
```

This creates the database with schema and initial data.

### Step 2: Export server data

Export software from Fleet's MySQL database to CSV:

```bash
mysql -h <host> -u <user> -p <database> --batch --raw -e "
SELECT
    'name', 'version', 'source', 'bundle_identifier', 'vendor', 'arch', 'release', 'extension_id', 'extension_for', 'application_id', 'upgrade_code'
UNION ALL
SELECT
    IFNULL(name, ''),
    IFNULL(version, ''),
    IFNULL(source, ''),
    IFNULL(bundle_identifier, ''),
    IFNULL(vendor, ''),
    IFNULL(arch, ''),
    IFNULL(\`release\`, ''),
    IFNULL(extension_id, ''),
    IFNULL(extension_for, ''),
    IFNULL(application_id, ''),
    IFNULL(upgrade_code, '')
FROM software
" 2>&1 | sed 's/\t/","/g' | sed 's/^/"/' | sed 's/$/"/' | tail -n +2 > source-data/server_export.csv
```

**Note:** This command properly quotes CSV fields to handle commas in values (e.g., "Red Hat, Inc."). The `tail -n +2` removes the MySQL password warning message while preserving the header row.

This creates a CSV with the following columns:
- `name`, `version`, `source` - Required fields
- `bundle_identifier` - macOS bundle ID
- `vendor` - Software vendor
- `arch` - Architecture (x86_64, arm64, etc.)
- `release` - Release info
- `extension_id` - Browser/IDE extension ID
- `extension_for` - Host software for extensions (Chrome, Firefox, VS Code, etc.)
- `application_id` - Android application ID
- `upgrade_code` - Windows upgrade GUID

**Optional filtering:**
- Add `WHERE` clause to filter by date, team, or other criteria
- Example: `WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)`

### Step 3: Import server data

```bash
cd tools/import-data

# Import with filtering for internal software
go run . --input ../../source-data/server_export.csv \
  --filter "numa-internal,numa-,corp-,internal-" \
  --filter-vendor "numa" \
  --verbose
```

This imports and validates server data, optionally filtering out internal software.

### Step 4: Generate software.sql

```bash
cd ../generate-sql

# Generate SQL dump
go run . --verbose
```

This creates `software.sql` that can recreate the entire database.

### Step 5: Verify

```bash
# Check counts by source
sqlite3 software.db "
  SELECT
    source,
    COUNT(*) as count
  FROM software
  GROUP BY source
  ORDER BY count DESC
"
```
