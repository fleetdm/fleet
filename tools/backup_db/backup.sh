#!/usr/bin/env bash
set -euo pipefail

docker run --rm --network fleet_default mysql:5.7 bash -c 'mysqldump -hmysql -uroot -ptoor fleet | gzip -' > backup.sql.gz
