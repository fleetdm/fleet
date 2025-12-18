# Deploy Fleet with Docker Compose


This guide walks you through deploying Fleet using Docker Compose. You'll have a Fleet instance running with MySQL and Redis in about 15 minutes.


## What you'll need


- [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/install/) installed
- Ports 1337, 3306, and 6379 available
- Basic command line familiarity


## Download the configuration files


Create a new directory for your Fleet deployment:

```bash
mkdir fleet-deployment
cd fleet-deployment
```

Download the Docker Compose file and environment template:

```bash
curl -O https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/docs/solutions/docker-compose/docker-compose.yml
curl -O https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/docs/solutions/docker-compose/env.example
```


## Configure your environment


Copy the `env.example` file

```bash
cp env.example .env
```

Generate a random server key with this command:

```bash
openssl rand -base64 32
```

Open the `.env` file and update these required values, using the key you generated in the last step as your `FLEET_SERVER_PRIVATE_KEY`:

```bash
# Generate a secure password for MySQL root
MYSQL_ROOT_PASSWORD=your_secure_root_password

# Generate a secure password for the Fleet database user
MYSQL_PASSWORD=your_secure_fleet_password

# Generate Fleet's server key (this encrypts session tokens)
FLEET_SERVER_PRIVATE_KEY=your_random_32_char_base64_key_here
```

Save the changes to your `.env` file.

## Configure TLS


Fleet requires HTTPS for MDM enrollment. Choose the option that fits your setup:

**Option 1: Reverse proxy or load balancer handles TLS** (recommended for production)

If you're running Fleet behind a reverse proxy (nginx, Caddy, Traefik) or load balancer that terminates TLS, set this in your `.env` file:

```bash
FLEET_SERVER_TLS=false
```

Your reverse proxy handles HTTPS, and Fleet listens on HTTP internally. Skip to "Start Fleet" below.

**Option 2: Fleet handles TLS directly**

For testing or simple deployments where Fleet serves HTTPS directly, you'll need TLS certificates.

Create a directory for your certificates:

```bash
mkdir certs
```

Generate a self-signed certificate (valid for 365 days):

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/fleet.key \
  -out certs/fleet.crt \
  -subj "/CN=localhost"
```

For production with a custom domain, replace `localhost` with your domain name:

```bash
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/fleet.key \
  -out certs/fleet.crt \
  -subj "/CN=fleet.example.com"
```

For production deployments, replace the self-signed certificate with one from a trusted certificate authority (Let's Encrypt, DigiCert, etc.).

Set this in your `.env` file:

```bash
FLEET_SERVER_TLS=true
```

The docker-compose.yml mounts your certificates from the `./certs` directory automatically.


## Optional: Add your license key


If you have a Fleet Premium license, add it to your `.env` file:

```bash
FLEET_LICENSE_KEY=your_license_key_here
```

Leave this blank to use Fleet's free tier.


## Optional: Configure S3 storage


Fleet can store software installers in S3-compatible storage. If you want to use this feature, update these values in your `.env` file:

```bash
FLEET_S3_SOFTWARE_INSTALLERS_BUCKET=your_bucket_name
FLEET_S3_SOFTWARE_INSTALLERS_ACCESS_KEY_ID=your_access_key
FLEET_S3_SOFTWARE_INSTALLERS_SECRET_ACCESS_KEY=your_secret_key
FLEET_S3_SOFTWARE_INSTALLERS_REGION=us-east-1
```

For RustFS or LocalStack, also set:

```bash
FLEET_S3_SOFTWARE_INSTALLERS_ENDPOINT_URL=http://your-s3-compatible-host:9000
FLEET_S3_SOFTWARE_INSTALLERS_FORCE_S3_PATH_STYLE=true
FLEET_S3_SOFTWARE_INSTALLERS_REGION=localhost
```


## Start Fleet


Run Docker Compose to start all services:

```bash
docker compose up -d
```

Docker will download the images and start MySQL, Redis, and Fleet. This takes 2-3 minutes on the first run.

Check the status:

```bash
docker compose ps
```

All services should show as "healthy" after about 30 seconds.


## Access Fleet


Open your browser and navigate to:

**If using TLS (FLEET_SERVER_TLS=true):**
```
https://localhost:1337
```

**If behind a reverse proxy (FLEET_SERVER_TLS=false):**
```
http://localhost:1337
```

If using a self-signed certificate, your browser will warn you about the connection. This is expected - click "Advanced" and proceed.

You'll see the Fleet setup screen. Follow the prompts to:

1. Create your first admin account
2. Configure your organization name
3. Add your first device (optional)

That's it! Fleet is running.


## What's running


This deployment includes three services and one initialization container:

**fleet-init** is a one-time setup container that fixes volume permissions before Fleet starts. Fleet runs as a non-root user (UID 100) for security, but Docker creates volumes owned by root. This container runs once, sets the correct ownership, and exits. You'll see it listed as "Exited (0)" when you run `docker compose ps -a`.

**MySQL** stores all Fleet data (devices, policies, queries, users). The database persists in a Docker volume so your data survives restarts.

**Redis** handles background jobs and caching. Fleet uses this for scheduling tasks and improving performance.

**Fleet** is the main application. It serves the web UI, API, and handles device connections.


## Ports


Devices connect to Fleet on port 1337. If needed, update your firewall to allow inbound connections on port 1337.


## View logs


Check Fleet's logs if you run into issues:

```bash
docker compose logs fleet
```

View MySQL or Redis logs:

```bash
docker compose logs mysql
docker compose logs redis
```


## Update Fleet


Pull the latest Fleet image:

```bash
docker compose pull fleet
docker compose up -d fleet
```

Fleet automatically runs database migrations on startup.


## Stop Fleet


Stop all services:

```bash
docker compose down
```

Stop and remove all data (careful - this deletes everything):

```bash
docker compose down -v
```


## Troubleshooting


**Permission denied errors on /logs**

The docker-compose file includes an initialization container that automatically fixes volume permissions. If you still see errors like `open /logs/osqueryd.status.log: permission denied`, try:

```bash
docker compose down
docker compose up -d
```

This restarts the initialization process.

**Fleet won't start**

Check that all required environment variables are set in your `.env` file. The `FLEET_SERVER_PRIVATE_KEY` must be a valid base64 string.

**Can't access the web UI**

Verify Fleet is running:

```bash
docker compose ps fleet
```

Check that port 1337 isn't blocked by your firewall.

If using TLS, verify your certificates exist in the `./certs` directory and are readable.

**Certificate errors**

If you see certificate-related errors in the logs, verify:

```bash
ls -l certs/
```

Both `fleet.crt` and `fleet.key` should exist and be readable. If using Option 1 (reverse proxy), make sure `FLEET_SERVER_TLS=false` in your `.env` file.

**Devices won't connect**

Ensure port 1337 is accessible from your devices. Check your firewall and network configuration.

If using self-signed certificates, devices need the certificate installed or TLS verification disabled (not recommended for production).


## Production considerations


This deployment works well for testing and small fleets. For production use with many devices:

- Use a managed MySQL database (AWS RDS, Google Cloud SQL)
- Use a managed Redis instance (AWS ElastiCache, Google Memorystore)
- Run multiple Fleet containers behind a load balancer
- Use certificates from a trusted certificate authority
- Pin container versions in docker-compose.yml
- Enable automatic backups
- Monitor resource usage
- Set up log forwarding to a centralized logging system

See Fleet's [Reference configuration strategies](https://fleetdm.com/docs/deploy/reference-architectures#reference-configuration-strategies) for production best practices.

<meta name="articleTitle" value="Deploy Fleet with Docker Compose">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2025-12-01">
<meta name="category" value="guides">
<meta name="description" value="Learn how to deploy Fleet using Docker Compose.">