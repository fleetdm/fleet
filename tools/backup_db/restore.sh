#!/usr/bin/env bash
set -euo pipefail

docker run --rm -i --network fleet_default mysql:5.7 bash -c 'gzip -kdc - | mysql -hmysql -uroot -ptoor fleet'  < backup.sql.gz

