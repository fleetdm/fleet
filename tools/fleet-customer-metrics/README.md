# Fleet Customer Metrics

Imports Fleet usage statistics from a Heroku dataclip JSON export into PostgreSQL and visualizes them with Grafana.

## Setup

1. Download the dataclip JSON from Heroku and save it to `~/Downloads/dataclips_ygqmhevpsmobjacgeafofmdpvaoz.json`.

2. Start everything:

```sh
docker compose up --build
```

This starts:
- **metrics-db** — PostgreSQL database for usage data (port 5432)
- **grafana-db** — PostgreSQL database for Grafana config
- **grafana** — Grafana dashboard (http://localhost:3000, login: admin/admin)
- **importer** — Loads the JSON data into metrics-db, then exits

The importer mounts `~/Downloads` into the container at `/data` to access the JSON file.

## Re-importing data

To re-import with fresh data, download a new JSON export and run:

```sh
docker compose run --build importer
```

Note: the importer appends rows — you may want to truncate the table first if re-importing the full dataset.
