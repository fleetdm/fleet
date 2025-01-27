#!/usr/bin/env bash
set -euo pipefail
BACKUP_NAME="${1:-backup.sql.gz}"
docker run --rm --network fleet_default ${FLEET_MYSQL_IMAGE:-mysql:8.0.36} bash -c 'mysqldump -hmysql -uroot -ptoor --default-character-set=utf8mb4 fleet | gzip -' > $BACKUP_NAME