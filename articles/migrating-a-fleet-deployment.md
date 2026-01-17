# Migrate Fleet server to a new deployment

This guide covers migrating your Fleet server from one deployment to another. Every environment is different, so this guide focuses on the essential steps rather than trying to cover every possible scenario.


## Before you begin

Before starting the migration, take time to prepare:

- **Back up your database.** Create a backup of your MySQL database and verify you can restore it. This is your safety net.
- **Plan for downtime.** Your Fleet instance will be unavailable during the migration. The process typically takes 5-10 minutes but could be longer depending on your database size.
- **Lower your DNS TTL.** Check the TTL (Time To Live) on your Fleet DNS record. If it's set to a high value (like 24 hours), lower it to 5 minutes at least 24-48 hours before the migration. This ensures the DNS change propagates quickly to all hosts. You can raise it back after the migration is stable.
- **Check version compatibility.** Your new Fleet instance should run the same version or a compatible version of Fleet. Don't try to upgrade and migrate at the same time.
- **Gather your configuration.** You'll need to recreate your Fleet server configuration on the new instance. This includes environment variables, TLS certificates, and any custom settings.


## Stop the Fleet server

Before migrating the database, shut down all Fleet instances to prevent data corruption during the migration.

If you're using systemd:

```bash
sudo systemctl stop fleet
```

If you're running Fleet in Docker:

```bash
docker stop fleet
```

Make sure all Fleet instances are completely stopped before proceeding.


## Back up the MySQL database

Create a backup of your Fleet database using `mysqldump`:

```bash
mysqldump -u fleet_user -p --single-transaction fleet > fleet_backup.sql
```

The `--single-transaction` flag ensures a consistent backup without locking tables.

Verify your backup file exists and has content:

```bash
ls -lh fleet_backup.sql
```


## Set up the new Fleet instance

On your new server or deployment:

1. Install Fleet following the [deployment guide](https://fleetdm.com/docs/deploy/deploy-fleet)
2. Install and configure MySQL
3. Install and configure Redis
4. Configure your TLS certificates
5. Set up your Fleet server configuration (environment variables, config file, or command-line flags)

Don't start Fleet yet - you'll import the database first.


## Import the database

Transfer your backup file to the new server, then import it:

```bash
mysql -u fleet_user -p fleet < fleet_backup.sql
```

This creates all the necessary tables and imports your data.


## Configure S3 storage (if applicable)

If you're using S3 for software installers, carves, or other file storage, add your S3 credentials to the new Fleet instance configuration.

Your S3 bucket and its contents don't need to be migrated - just point the new instance to the same bucket by configuring these environment variables:

```bash
FLEET_S3_BUCKET=your-bucket-name
FLEET_S3_REGION=us-east-1
FLEET_S3_ACCESS_KEY_ID=your-access-key
FLEET_S3_SECRET_ACCESS_KEY=your-secret-key
```


## Prepare the database

Run the database migration preparation to ensure your database schema is up to date:

```bash
fleet prepare db \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=fleet_user \
  --mysql_password=your-password
```

This command is safe to run even if your schema is already current.


## Start Fleet on the new instance

Start the Fleet server on your new deployment:

```bash
fleet serve \
  --mysql_address=127.0.0.1:3306 \
  --mysql_database=fleet \
  --mysql_username=fleet_user \
  --mysql_password=your-password \
  --redis_address=127.0.0.1:6379 \
  --server_cert=/path/to/server.cert \
  --server_key=/path/to/server.key \
  --logging_json
```

Or if you're using systemd:

```bash
sudo systemctl start fleet
```

Check that Fleet started successfully:

```bash
curl -k https://localhost:8080/healthz
```


## Update DNS

Update your DNS records to point to the new Fleet instance. This is the critical step that switches traffic from the old server to the new one.

If you lowered your DNS TTL earlier, the change should propagate within that time window. Hosts will start connecting to the new server as their cached DNS entries expire.

If you're using a load balancer, update the backend pool to point to the new instance instead of updating DNS directly.


## Verify the migration

After DNS propagates:

1. **Check the Fleet UI.** Log in and verify your hosts, queries, policies, and settings are present.
2. **Monitor host check-ins.** Hosts should automatically reconnect to the new server as DNS propagates. The time this takes depends on your DNS TTL - if you set it to 5 minutes, most hosts should reconnect within 5-10 minutes. Check the *Hosts* page to watch them come online.
3. **Test integrations.** If you have log forwarding, webhooks, or other integrations configured, verify they're working.
4. **Run a live query.** Execute a simple query to confirm the query system is working.

Once the migration is stable and all hosts have reconnected, you can raise your DNS TTL back to its previous value.


## Additional notes

**Redis doesn't need migration.** Redis stores ephemeral data (live query results, short-term caches), so you don't need to migrate Redis data. The new instance can start with a fresh Redis.

**Secrets are in the database.** Your API tokens, enroll secrets, and other configuration are stored in MySQL, so they'll carry over automatically.

**Consider using MySQL replication.** If you're using MySQL replication, you can set up a replica on the new deployment before migrating. When you're ready, promote the replica to primary and point Fleet to it. This can significantly reduce downtime.

**Advanced load balancing scenarios.** If you're running Fleet behind a load balancer with multiple instances, you can migrate one instance at a time. Update the load balancer to route traffic to new instances as you bring them online.


## Troubleshooting

**Hosts aren't checking in**

Hosts will reconnect as their DNS cache expires, which depends on your DNS TTL setting. If you didn't lower the TTL before migration and it's set to a high value (like 24 hours), hosts may take up to that long to start connecting to the new server.

If hosts still aren't reconnecting after the TTL period has passed:

- Verify DNS has propagated using `dig` or `nslookup` to check the DNS record
- Check that the new instance is accessible from your network
- Review Fleet server logs for connection errors

**Database connection errors**

If Fleet can't connect to the database:

- Verify MySQL is running on the new server
- Check database credentials in your Fleet configuration
- Confirm the database user has the necessary permissions
- Test the connection manually with the `mysql` command-line client

**Can't access the Fleet UI**

If you can't access the web interface:

- Verify Fleet is running (`systemctl status fleet` or check Docker container status)
- Check that your TLS certificates are configured correctly
- Review firewall rules to ensure port 8080 (or your configured port) is accessible

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="authorFullName" value="Kitzy">
<meta name="publishedOn" value="2026-01-16">
<meta name="articleTitle" value="Migrate Fleet server to a new deployment">
<meta name="description" value="Migrate your Fleet server to a new deployment with this step-by-step guide. Includes database migration, DNS configuration, and host reconnection tips.">