#!/bin/bash
# Convert schema.sql (a MySQL mysqldump) into a MariaDB-loadable schema.
#
# This produces the point-in-time baseline used to stand up a MariaDB database
# without replaying the full migration history, which does not run on MariaDB
# (see https://github.com/fleetdm/fleet/issues/34952).
#
# Transformations applied, all of them things MySQL accepts and MariaDB does not:
#
#   1. Functional (expression) indexes are dropped -- MariaDB has no equivalent.
#      They are matched structurally rather than by name, since the set of them
#      grows over time. NOTE that dropping a functional UNIQUE KEY also drops
#      the uniqueness constraint it enforced.
#   2. TABLESPACE directives are dropped; MariaDB handles them differently.
#   3. `) STORED NOT NULL` becomes `) STORED`. MariaDB accepts a generated
#      stored column but rejects a nullability clause after STORED.
#   4. Generated columns that reference a column declared later in the same
#      CREATE TABLE are moved down past that column. MySQL resolves such forward
#      references; MariaDB rejects them with error 1901.
#   5. A CHAR column referenced by an *indexed* generated column is widened to
#      VARCHAR of the same length. MariaDB refuses to index a generated column
#      whose expression mixes CHAR and VARCHAR operands, reporting error 1901
#      against the whole expression. CHAR(n) right-pads on storage and strips
#      the padding on retrieval, so for the fixed-width values these columns
#      hold the two types are interchangeable.

set -e

INPUT="server/datastore/mysql/schema.sql"
OUTPUT="server/datastore/mysql/schema-mariadb.sql"

echo "Converting schema.sql to MariaDB-compatible format..."

python3 - "$INPUT" "$OUTPUT" << 'PYEOF'
import re, sys

src, dst = sys.argv[1], sys.argv[2]
lines = open(src).readlines()

FUNCTIONAL_INDEX = re.compile(r'^\s*(?:UNIQUE |FULLTEXT |SPATIAL )?KEY `[^`]+` \(\(')
COLUMN_DEF = re.compile(r'^\s*`(?P<name>[^`]+)`\s')
GENERATED = re.compile(r'GENERATED ALWAYS AS')

dropped = 0
out = []
i = 0
while i < len(lines):
    line = lines[i]

    if not line.startswith('CREATE TABLE '):
        out.append(line)
        i += 1
        continue

    # Collect the whole CREATE TABLE statement.
    start = i
    while not lines[i].startswith(')'):
        i += 1
    end = i  # line index of the closing ") ENGINE=..." line

    body = lines[start + 1:end]
    kept = []
    for b in body:
        if FUNCTIONAL_INDEX.match(b):
            dropped += 1
            continue
        b = b.replace('/*!50100 TABLESPACE `innodb_system` */ ', '')
        b = b.replace(') STORED NOT NULL', ') STORED')
        kept.append(b)

    # Reorder generated columns that forward-reference a later column. Only
    # column definitions move; keys and constraints keep their relative order.
    positions = {}
    for idx, b in enumerate(kept):
        m = COLUMN_DEF.match(b)
        if m:
            positions.setdefault(m.group('name'), idx)

    moved = True
    guard = 0
    while moved and guard < 100:
        moved = False
        guard += 1
        for idx, b in enumerate(kept):
            m = COLUMN_DEF.match(b)
            if not m or not GENERATED.search(b):
                continue
            # every other column this expression names
            refs = [r for r in re.findall(r'`([^`]+)`', b)[1:] if r in positions]
            latest = max((positions[r] for r in refs), default=-1)
            if latest > idx:
                col = kept.pop(idx)
                kept.insert(latest, col)
                positions = {}
                for j, bb in enumerate(kept):
                    mm = COLUMN_DEF.match(bb)
                    if mm:
                        positions.setdefault(mm.group('name'), j)
                moved = True
                break

    # Widen CHAR columns feeding an indexed generated column (see header note 5).
    indexed = set()
    for b in kept:
        m = re.match(r'^\s*(?:UNIQUE |FULLTEXT |SPATIAL )?KEY `[^`]+` \((.*)\)', b)
        if m:
            indexed.update(re.findall(r'`([^`]+)`', m.group(1)))
    needs_varchar = set()
    for b in kept:
        m = COLUMN_DEF.match(b)
        if m and GENERATED.search(b) and m.group('name') in indexed:
            needs_varchar.update(re.findall(r'`([^`]+)`', b)[1:])
    for idx, b in enumerate(kept):
        m = COLUMN_DEF.match(b)
        if m and m.group('name') in needs_varchar:
            kept[idx] = re.sub(r'(`[^`]+`\s+)char\((\d+)\)', r'\1varchar(\2)', b, count=1)

    # A removed or moved line can leave a dangling comma on the final item.
    for idx in range(len(kept) - 1, -1, -1):
        if kept[idx].strip():
            kept[idx] = re.sub(r',(\s*)$', r'\1', kept[idx])
            break

    closing = lines[end].replace('/*!50100 TABLESPACE `innodb_system` */ ', '')

    out.append(lines[start])
    out.extend(kept)
    out.append(closing)
    i = end + 1

with open(dst, 'w') as f:
    f.write("SET FOREIGN_KEY_CHECKS=0;\nSET SESSION sql_mode='';\n\n")
    f.writelines(out)
    f.write("\nSET FOREIGN_KEY_CHECKS=1;\n")

print(f"removed {dropped} functional index definition(s)", file=sys.stderr)
PYEOF

echo "✓ MariaDB-compatible schema created at: $OUTPUT"
echo ""
echo "To import into MariaDB, run:"
echo "mariadb --host localhost --user root --protocol=tcp -P 3306 --password fleet < $OUTPUT"
