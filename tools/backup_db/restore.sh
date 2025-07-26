#!/usr/bin/env bash
set -euo pipefail
BACKUP_NAME="${1:-backup.sql.gz}"
docker run --rm -i --network fleet_default ${FLEET_MYSQL_IMAGE:-mysql:8.0.40} bash -c 'gzip -dc - | MYSQL_PWD=toor mysql -hmysql -uroot fleet' < ${BACKUP_NAME}
