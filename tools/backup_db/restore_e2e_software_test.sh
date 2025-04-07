#!/usr/bin/env bash
set -euo pipefail

docker run --rm -i --network fleet_default mysql:8.0.36 bash -c 'gzip -kdc - | mysql -hmysql_test -uroot -ptoor e2e'  < tools/testdata/e2e_software_test.sql.gz
