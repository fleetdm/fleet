#!/bin/bash
set -euo pipefail

# Install and start Docker
dnf install -y docker
systemctl enable --now docker

# Run PMM server
docker run -d \
  --name pmm-server \
  --restart always \
  -p 443:443 \
  -v pmm-data:/srv \
  percona/pmm-server:2

# Wait for PMM server to be ready
echo "Waiting for PMM server to become ready..."
for i in $(seq 1 60); do
  if docker exec pmm-server curl -sSf -k https://localhost:443/v1/readyz > /dev/null 2>&1; then
    echo "PMM server is ready"
    break
  fi
  echo "Attempt $i/60 - PMM server not ready yet..."
  sleep 5
done

# Add the RDS MySQL instance to PMM monitoring
docker exec pmm-server pmm-admin add mysql \
  --server-url=https://admin:admin@localhost:443 \
  --server-insecure-tls \
  --username='${rds_username}' \
  --password='${rds_password}' \
  --host='${rds_endpoint}' \
  --port=3306 \
  --query-source=perfschema \
  "fleet-mysql"
