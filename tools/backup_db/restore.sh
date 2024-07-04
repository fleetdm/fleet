#!/usr/bin/env bash
set -euo pipefail

docker run --rm -i --network fleet_default ${FLEET_MYSQL_IMAGE:-mysql:5.7} bash -c 'gzip -dc - | mysql -hmysql -uroot -ptoor fleet' < backup.sql.gz
