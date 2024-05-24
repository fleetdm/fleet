[![Deploy to Render](https://render.com/images/deploy-to-render-button.svg)](https://render.com/deploy?repo=https://github.com/fleetdm/fleet)

# Fleet Deployment Guide

This guide outlines the services configured in the Render blueprint for deploying the Fleet system, which includes a web service, a MySQL database, and a Redis server.

## Services Overview

### 1. Fleet Web Service
- **Type:** Web
- **Runtime:** Image
- **Image:** `fleetdm/fleet:latest` 
- **Description:** Main web service running the Fleet application, which is deployed using the latest Fleet Docker image. Configured to prepare the database before deployment.
- **Health Check Path:** `/healthz`
- **Environment Variables:** Connects to MySQL and Redis using service-bound environment variables.

### 2. Fleet MySQL Database
- **Type:** Private Service (pserv)
- **Runtime:** Docker
- **Repository:** [MySQL Example on Render](https://github.com/render-examples/mysql)
- **Disk:** 10 GB mounted at `/var/lib/mysql`
- **Description:** MySQL database used by the Fleet web service. Environment variables for database credentials are managed within the service and some are automatically generated.

### 3. Fleet Redis Service
- **Type:** Private Service (pserv)
- **Runtime:** Image
- **Repository:** [Redis Docker image](https://hub.docker.com/_/redis)
- **Description:** Redis service for caching and other in-memory data storage needs of the Fleet web service.

## Deployment Guide

### Prerequisites
- You need an account on [Render](https://render.com).
- Familiarity with Render's dashboard and deployment concepts.

### Steps to Deploy

Click the deploy on render button or import the blueprint from the Render service deployment dashboard.

### Post-Deployment

Navigate to the generated URL and run through the initial setup.
