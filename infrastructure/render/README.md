[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/fleetdm/fleet)

# Fleet deployment guide

This guide outlines the services configured in the Render blueprint for deploying the Fleet system, which includes a web service, a MySQL database, and a Redis server.

## Services overview

### 1. Fleet web service
- **Type:** Web
- **Runtime:** Image
- **Image:** `fleetdm/fleet:latest` 
- **Description:** Main web service running the Fleet application, which is deployed using the latest Fleet Docker image. Configured to prepare the database before deployment.
- **Health check path:** `/healthz`
- **Environment variables:** Connects to MySQL and Redis using service-bound environment variables.

### 2. Fleet MySQL database
- **Type:** Private service (pserv)
- **Runtime:** Docker
- **Repository:** [MySQL Example on Render](https://github.com/render-examples/mysql)
- **Disk:** 10 GB mounted at `/var/lib/mysql`
- **Description:** MySQL database used by the Fleet web service. Environment variables for database credentials are managed within the service and some are automatically generated.

### 3. Fleet Redis service
- **Type:** Private service (pserv)
- **Runtime:** Image
- **Repository:** [Redis Docker image](https://hub.docker.com/_/redis)
- **Description:** Redis service for caching and other in-memory data storage needs of the Fleet web service.

## Deployment guide

### Prerequisites
- You need an account on [Render](https://render.com).
- Familiarity with Render's dashboard and deployment concepts.

### Steps to deploy

Click the deploy on render button or import the blueprint from the Render service deployment dashboard.

### Post-deployment

Navigate to the generated URL and run through the initial setup. If you have a license key you can add it post-deploy as
an environment variable `FLEET_LICENSE_KEY=value` in the Fleet service configuration.
