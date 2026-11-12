# Percona Monitoring and Management (PMM)

Local MySQL performance monitoring using [PMM](https://docs.percona.com/percona-monitoring-and-management/). Tracks query analytics, InnoDB metrics, connection counts, and slow query analysis.

## Prerequisites

- Docker
- Assumes fleet dev environment running the `fleet-mysql-1` container on the `fleet_default` network.

## Setup

### 1. Start PMM server and client

```sh
docker compose up -d
```

### 2. Create the PMM monitoring user in MySQL

```sh
docker exec -it fleet-mysql-1 mysql -uroot -ptoor -e "
  CREATE USER IF NOT EXISTS 'pmm'@'%' IDENTIFIED BY 'pmm';
  GRANT SELECT, PROCESS, REPLICATION CLIENT, RELOAD, BACKUP_ADMIN ON *.* TO 'pmm'@'%';
"
```

### 3. Register the Fleet MySQL instance

```sh
docker exec pmm-client pmm-admin add mysql \
  --username=pmm \
  --password=pmm \
  --host=mysql \
  --port=3306 \
  --query-source=perfschema \
  --service-name=fleet-mysql
```

`--host=mysql` refers to the `mysql` service name from the Fleet `docker-compose.yml`, which resolves within the shared `fleet_default` network.

## Usage

Open https://localhost in your browser.

Note: this local PMM setup uses self-signed TLS, so your browser will typically show a certificate warning for `https://localhost`. This is expected for local development.

Default credentials: `admin` / `admin`. These are development defaults only, not production-grade credentials. Change the default admin password in the PMM UI after signing in if you need stronger local access control.

## Troubleshooting

### PMM stops collecting metrics / MySQL becomes unresponsive

PMM may lose its connection to the service. Remove and re-add the service to restore monitoring:

```sh
docker exec pmm-client pmm-admin remove mysql fleet-mysql
docker exec pmm-client pmm-admin add mysql \
  --username=pmm \
  --password=pmm \
  --host=mysql \
  --port=3306 \
  --query-source=perfschema \
  --service-name=fleet-mysql
```

## Teardown

```sh
docker compose down
```

To also remove all collected data:

```sh
docker compose down -v
```
