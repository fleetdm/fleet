#!/usr/bin/env bash
set -euo pipefail
BACKUP_NAME="${1:-backup.sql.gz}"
docker run --rm -i --network fleet_default ${FLEET_MYSQL_IMAGE:-mysql:8.0.36} bash -c 'gzip -dc - | mysql -hmysql -uroot -ptoor fleet' < ${BACKUP_NAME}
