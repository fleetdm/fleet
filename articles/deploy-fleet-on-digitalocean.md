# Deploy Fleet on DigitalOcean

Get Fleet running on DigitalOcean with HTTPS, a database, and storage for software installers. The first path takes a few clicks and one paste. You don't need Terraform or Docker.

Pick the path that fits:

- **Single Droplet**: everything runs on one server that you create in the DigitalOcean control panel. Best for trials and small deployments.
- **App Platform with Terraform**: DigitalOcean runs Fleet and a managed MySQL database for you, with nothing to patch. Best for production. It needs Terraform.

## Deploy on a single Droplet

The script installs Fleet, MySQL, Redis, and a web server that gets a free HTTPS certificate. Your data lives on the Droplet's disk.

### Prerequisites

- A [DigitalOcean account](https://cloud.digitalocean.com/registrations/new).
- (Optional) A domain you can add a DNS record to. Without one, the script uses a free `sslip.io` address, which is fine for trials.

### Create the Droplet

1. In the DigitalOcean control panel, select **Create > Droplets**.
2. Choose a region, then choose the **Ubuntu 24.04 (LTS) x64** image.
3. Choose a size with at least 2 vCPUs and 4 GB of memory (`s-2vcpu-4gb`).
4. Select **Enable backups**. Your Fleet data lives on this Droplet's disk.
5. Under **Authentication**, choose an SSH key or a password. You don't need to log in to the Droplet to finish setup.
6. Expand **Advanced Options**, select **Add initialization scripts (free)**, and paste in the contents of [`cloud-init.yml`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/digitalocean/cloud-init.yml).
7. Select **Create Droplet**.

To use your own domain, change the `FLEET_DOMAIN=` line near the top of the script to `FLEET_DOMAIN=fleet.example.com` before you create the Droplet. Then create an A record for that name that points at the Droplet's IP address.

> **Note:** Caddy requests the certificate as soon as the name resolves to the Droplet. If your DNS record isn't live yet, Caddy retries until it is. You don't need to restart anything.

### Open Fleet

Wait a few minutes for the script to install and start everything. Then open `https://fleet.example.com`.

Without a domain, your address is the Droplet's IP address with dashes in place of dots, followed by `.sslip.io`. For a Droplet at `203.0.113.10`, that's `https://203-0-113-10.sslip.io`.

Fleet shows the setup screen. Create your first admin account.

### Software installers

Fleet stores software installers and bootstrap packages on the Droplet's disk, in a Docker volume. Droplet backups include them. This storage works for a single Fleet server only. If you outgrow one server, use the App Platform path below.

### Maintain your Droplet

Open the Droplet's console from the control panel (**Access > Launch Droplet Console**), or connect with SSH.

**Update Fleet.** Fleet runs database migrations when it starts.

```bash
cd /opt/fleet
docker compose pull fleet
docker compose up -d fleet
```

**Add a Fleet Premium license key.** Put your key on the `FLEET_LICENSE_KEY=` line in `/opt/fleet/.env`, then restart Fleet:

```bash
cd /opt/fleet
docker compose up -d fleet
```

**Back up your database.** [Droplet backups](https://docs.digitalocean.com/products/backups/) cover the whole server. For a consistent copy of the database, dump it:

```bash
cd /opt/fleet
docker compose exec -T mysql sh -c 'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" fleet' > fleet-backup.sql
```

## Deploy on App Platform with Terraform

This path runs Fleet on App Platform with one `terraform apply`. You get managed MySQL 8.4 with daily backups that only the Fleet app can reach, an internal Redis service, and a Spaces bucket for software installers with a key limited to that bucket. A pre-deploy job runs `fleet prepare db` before every deploy. A failed migration stops the deploy and the previous version keeps serving.

### Prerequisites

- A [DigitalOcean account](https://cloud.digitalocean.com/registrations/new) and a [personal access token](https://docs.digitalocean.com/reference/api/create-personal-access-token/) with read and write scope.
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.5 or later.
- A [Spaces access key](https://docs.digitalocean.com/products/spaces/how-to/manage-access/) with access to all buckets. Terraform uses it to create the installers bucket. To skip the bucket, set `create_installers_bucket = false` and you don't need the key.
- (Optional) A domain you can add a DNS record to.

> **Warning:** Terraform's state file contains your database password and Fleet's private key. Store it in a [remote backend](https://developer.hashicorp.com/terraform/language/backend) with restricted access, and never commit it.

### Deploy

1. Download the module's files into a new folder.

   ```bash
   mkdir fleet-do && cd fleet-do
   base=https://raw.githubusercontent.com/fleetdm/fleet/main/docs/solutions/digitalocean/terraform
   for f in main.tf variables.tf outputs.tf terraform.tfvars.example; do curl -fsSLO "$base/$f"; done
   mv terraform.tfvars.example terraform.tfvars
   ```

2. Set your API token and Spaces key, then apply.

   ```bash
   export DIGITALOCEAN_TOKEN=your_api_token
   export SPACES_ACCESS_KEY_ID=your_spaces_access_key_id
   export SPACES_SECRET_ACCESS_KEY=your_spaces_secret_key
   terraform init
   terraform apply
   ```

   Review the plan and enter `yes`. The first apply takes several minutes while DigitalOcean creates the database cluster and deploys the app.

3. Open the `fleet_url` address that Terraform prints, then create your first admin account.

The defaults deploy to New York. To use another region, set `app_region`, `db_region`, and `installers_region` in `terraform.tfvars`. Choose the database datacenter that matches your app region, such as `nyc3` for `nyc` or `fra1` for `fra`.

### Use your own domain

1. Add your domain to `terraform.tfvars`:

   ```hcl
   domain = "fleet.example.com"
   ```

2. Run `terraform apply`.
3. Create a CNAME record for your domain that points at the `dns_target` output.
4. In Fleet, go to **Settings > Organization settings > Fleet web address** and enter your new address.

DigitalOcean issues the certificate once the record resolves.

> **Note:** MDM and device enrollment need a stable address with a trusted certificate. Pick your final domain before you enroll devices.

### Add a Fleet Premium license key

Add your key to `terraform.tfvars` before the first apply:

```hcl
fleet_license_key = "your_license_key"
```

> **Note:** The module sets secrets only when it creates the app, so Terraform doesn't redeploy on every apply. To add or rotate a secret later, remove `spec[0].env` from the `ignore_changes` list in `main.tf`, apply, then put it back.

### Choose where software installers are stored

The module creates a private Spaces bucket in `nyc3` and gives Fleet a key limited to it. To use another Spaces region, set `installers_region`. To skip the bucket, for example if you don't deploy software, set:

```hcl
create_installers_bucket = false
```

### Upgrade Fleet

Set the new version in `terraform.tfvars`, then apply:

```hcl
fleet_version = "v4.96.0"
```

```bash
terraform apply
```

Replace the example version with the release you want. App Platform runs the migration job first, then replaces the running version.

### Back up and restore

DigitalOcean backs up managed MySQL daily and keeps seven days of point-in-time recovery. To restore, follow DigitalOcean's [restore instructions](https://docs.digitalocean.com/products/databases/mysql/how-to/restore-from-backups/) for your cluster. Database backups don't include the Spaces bucket.

### Remove the deployment

`terraform destroy` deletes the app and the database cluster, including all Fleet data. Take a manual backup first if you need to keep anything. Terraform won't delete the installers bucket while it holds files, so empty it first.

## Verify the deployment

1. Open your Fleet address. The browser shows a valid HTTPS certificate with no warnings.
2. Log in and go to **Hosts**. The page loads with no hosts yet.
3. Select **Add hosts** and follow the prompts to enroll your first device.

## Troubleshoot

**The browser shows a certificate error.** The certificate isn't issued yet. Confirm your DNS record points at the Droplet's IP address (or the `dns_target` output on App Platform), then wait for DNS to propagate. On a Droplet, check Caddy's logs with `docker compose logs caddy` in `/opt/fleet`. Let's Encrypt limits repeated failed requests, so fix DNS before retrying.

**The site doesn't load after several minutes on a Droplet.** In the Droplet console, check the setup script's output with `tail -n 50 /var/log/cloud-init-output.log`. Then check the containers with `docker compose ps` in `/opt/fleet`.

**Fleet restarts in a loop on a Droplet.** Run `docker compose logs fleet` in `/opt/fleet` and look for the first error. MySQL can take a minute to initialize on first boot, and Fleet waits for it.

**The first `terraform apply` fails during the deploy.** Open your app in the DigitalOcean control panel, then select **Activity** and the failed deployment to read the build and job logs. Fix the cause, then run `terraform apply` again.

**The deploy fails at `fleet-migrate` with a primary key or SQL mode error.** DigitalOcean can reset the MySQL settings when it creates a cluster. Run `terraform apply` again so Terraform reapplies `sql_require_primary_key` and `sql_mode`, then redeploy the app.

**Fleet on App Platform restarts and its logs show it can't reach the database.** Open the database cluster in the control panel and check its trusted sources. They should list only your Fleet app. If you edited the list by hand, run `terraform apply` to restore it.

**Fleet on App Platform restarts and runs out of memory.** The vulnerability database loads into memory. Set `fleet_instance_size` to a larger size slug, such as `apps-s-2vcpu-4gb`, and apply.

## Further reading

- [Fleet reference architectures](https://fleetdm.com/docs/deploy/reference-architectures)
- [Fleet server configuration](https://fleetdm.com/docs/configuration/fleet-server-configuration)
- [App Platform documentation](https://docs.digitalocean.com/products/app-platform/)
- [Enroll a DigitalOcean Droplet in Fleet](https://fleetdm.com/guides/enrolling-a-digital-ocean-droplet-on-a-fleet-instance)

<meta name="articleTitle" value="Deploy Fleet on DigitalOcean">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-18">
<meta name="category" value="guides">
<meta name="description" value="Deploy Fleet on DigitalOcean by pasting one script into a Droplet, or use Terraform and App Platform for production.">
