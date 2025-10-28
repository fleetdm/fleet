#!/bin/bash
# Script to convert schema.sql to MariaDB-compatible format

set -e

INPUT="server/datastore/mysql/schema.sql"
OUTPUT="server/datastore/mysql/schema-mariadb.sql"

echo "Converting schema.sql to MariaDB-compatible format..."

cat > "$OUTPUT" << 'EOF'
SET FOREIGN_KEY_CHECKS=0;
SET SESSION sql_mode='';

EOF

cat "$INPUT" >> "$OUTPUT"

cat >> "$OUTPUT" << 'EOF'

SET FOREIGN_KEY_CHECKS=1;
EOF

# Remove the functional indexes - MariaDB doesn't support this
if [[ "$OSTYPE" == "darwin"* ]]; then
  sed -i '' '/KEY `idx_host_vpp_software_installs_verification`/d' "$OUTPUT"
  sed -i '' '/KEY `idx_host_in_house_software_installs_verification`/d' "$OUTPUT"
else
  sed -i '/KEY `idx_host_vpp_software_installs_verification`/d' "$OUTPUT"
  sed -i '/KEY `idx_host_in_house_software_installs_verification`/d' "$OUTPUT"
fi

# Remove TABLESPACE directives - MariaDB handles these differently
if [[ "$OSTYPE" == "darwin"* ]]; then
  sed -i '' 's/\/\*!50100 TABLESPACE `innodb_system` \*\/ //g' "$OUTPUT"
else
  sed -i 's/\/\*!50100 TABLESPACE `innodb_system` \*\/ //g' "$OUTPUT"
fi

# Fix STORED NOT NULL -> just STORED (MariaDB supports theses just different syntax)
if [[ "$OSTYPE" == "darwin"* ]]; then
  sed -i '' 's/) STORED NOT NULL/) STORED/g' "$OUTPUT"
else
  sed -i 's/) STORED NOT NULL/) STORED/g' "$OUTPUT"
fi

echo "✓ MariaDB-compatible schema created at: $OUTPUT"
echo ""
echo "To import into MariaDB, run:"
echo "mysql --host localhost --user root --protocol=tcp -P 3306 --password fleet < $OUTPUT"
