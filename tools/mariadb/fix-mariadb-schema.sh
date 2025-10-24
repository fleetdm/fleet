#!/bin/bash
# Script to convert schema.sql to MariaDB-compatible format

set -e

INPUT="server/datastore/mysql/schema.sql"
OUTPUT="server/datastore/mysql/schema-mariadb.sql"

echo "Converting schema.sql to MariaDB-compatible format..."

# Start by creating the file with MariaDB settings
cat > "$OUTPUT" << 'EOF'
SET FOREIGN_KEY_CHECKS=0;
SET SESSION sql_mode='';

EOF

# Append the schema content
cat "$INPUT" >> "$OUTPUT"

# Append the closing statement
cat >> "$OUTPUT" << 'EOF'

SET FOREIGN_KEY_CHECKS=1;
EOF

# Detect OS for sed -i syntax (macOS vs Linux)
if [[ "$OSTYPE" == "darwin"* ]]; then
  SED_INPLACE="sed -i ''"
else
  SED_INPLACE="sed -i"
fi

# Fix 1: Remove the functional index - MariaDB doesn't support this MySQL 8.0 syntax
# We'll just remove this index as it's an optimization, not critical
$SED_INPLACE '/KEY `idx_host_vpp_software_installs_verification`/d' "$OUTPUT"

# Fix 2: Remove TABLESPACE directives - MariaDB handles these differently
$SED_INPLACE 's/\/\*!50100 TABLESPACE `innodb_system` \*\/ //g' "$OUTPUT"

# Fix 3: Fix STORED NOT NULL -> just STORED (MariaDB doesn't like NOT NULL after STORED for generated columns)
# Line 2013 has: STORED NOT NULL
$SED_INPLACE 's/) STORED NOT NULL/) STORED/g' "$OUTPUT"

echo "✓ MariaDB-compatible schema created at: $OUTPUT"
echo ""
echo "To import into MariaDB, run:"
echo "mysql --host localhost --user root --protocol=tcp -P 3306 --password fleet < $OUTPUT"
