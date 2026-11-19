# Deploy Fleet on DigitalOcean

Run Fleet on DigitalOcean App Platform with one `terraform apply`. You get a managed MySQL database, automatic HTTPS, and Fleet's database migrations that run before each deploy. There are no servers to patch.

## What you get

- **Fleet** on App Platform, served over HTTPS at a free `ondigitalocean.app` address or your own domain.
- **Managed MySQL 8.4** with daily backups. Only the Fleet app can connect to it.
- **Redis** running as an internal App Platform service. Fleet uses it as a cache, so nothing in it needs to survive a restart.
- **A pre-deploy job** that runs `fleet prepare db` before every deploy. A failed migration stops the deploy and the previous version keeps serving.

## Prerequisites

- A [DigitalOcean account](https://cloud.digitalocean.com/registrations/new) and a [personal access token](https://docs.digitalocean.com/reference/api/create-personal-access-token/) with read and write scope.
- [Terraform](https://developer.hashicorp.com/terraform/install) 1.5 or later.
- (Optional) A domain you can add a DNS record to.

> **Warning:** Terraform's state file contains your database password and Fleet's private key. Store it in a [remote backend](https://developer.hashicorp.com/terraform/language/backend) with restricted access, and never commit it.

## Deploy Fleet

1. Download the module's files into a new folder.

   ```bash
   mkdir fleet-do && cd fleet-do
   base=https://raw.githubusercontent.com/fleetdm/fleet/main/docs/solutions/digitalocean/terraform
   for f in main.tf variables.tf outputs.tf terraform.tfvars.example; do curl -fsSLO "$base/$f"; done
   mv terraform.tfvars.example terraform.tfvars
   ```

2. Set your API token and apply.

   ```bash
   export DIGITALOCEAN_TOKEN=your_api_token
   terraform init
   terraform apply
   ```

   Review the plan and enter `yes`. The first apply takes several minutes while DigitalOcean creates the database cluster and deploys the app.

3. Open the `fleet_url` address that Terraform prints, then create your first admin account.

The defaults deploy to New York. To use another region, set `app_region` and `db_region` in `terraform.tfvars`. Choose the database datacenter that matches your app region, such as `nyc3` for `nyc` or `fra1` for `fra`.

## Use your own domain

1. Add your domain to `terraform.tfvars`:

   ```hcl
   domain = "fleet.example.com"
   ```

2. Run `terraform apply`.
3. Create a CNAME record for your domain that points at the `dns_target` output.
4. In Fleet, go to **Settings > Organization settings > Fleet web address** and enter your new address.

DigitalOcean issues the certificate once the record resolves.

> **Note:** MDM and device enrollment need a stable address with a trusted certificate. Pick your final domain before you enroll devices.

## Add a Fleet Premium license key

Add your key to `terraform.tfvars` before the first apply:

```hcl
fleet_license_key = "your_license_key"
```

> **Note:** The module sets secrets only when it creates the app, so Terraform doesn't redeploy on every apply. To add or rotate a secret later, remove `spec[0].env` from the `ignore_changes` list in `main.tf`, apply, then put it back.

## Store software installers in Spaces

App Platform has no persistent disk, so Fleet needs S3-compatible storage for software installers. Skip this section if you don't deploy software.

1. [Create a Spaces bucket](https://docs.digitalocean.com/products/spaces/how-to/create/) and a [limited access key](https://docs.digitalocean.com/products/spaces/how-to/manage-access/) for it.
2. Add the bucket details to `terraform.tfvars`:

   ```hcl
   installers_bucket            = "my-fleet-installers"
   installers_region            = "nyc3"
   installers_access_key_id     = "your_access_key_id"
   installers_secret_access_key = "your_secret_key"
   ```

3. Remove `spec[0].env` from the `ignore_changes` list in `main.tf`, run `terraform apply`, then put it back.

## Verify the deployment

1. Open your Fleet address. The browser shows a valid HTTPS certificate with no warnings.
2. Log in and go to **Hosts**. The page loads with no hosts yet.
3. In the DigitalOcean control panel, open your app and select the **Activity** tab. The latest deployment shows the `fleet-migrate` job as succeeded.
4. Select **Add hosts** in Fleet and follow the prompts to enroll your first device.

## Upgrade Fleet

Set the new version in `terraform.tfvars`, then apply:

```hcl
fleet_version = "v4.96.0"
```

```bash
terraform apply
```

Replace the example version with the release you want. App Platform runs the migration job first, then replaces the running version. If the migration fails, the deploy stops and your current version keeps serving.

## Back up and restore

DigitalOcean backs up managed MySQL daily and keeps seven days of point-in-time recovery. To restore, follow DigitalOcean's [restore instructions](https://docs.digitalocean.com/products/databases/mysql/how-to/restore-from-backups/) for your cluster. Software installers in Spaces aren't covered by database backups.

## Remove the deployment

`terraform destroy` deletes the app and the database cluster, including all Fleet data. Take a manual backup first if you need to keep anything.

## Troubleshoot

**The first apply fails during the deploy.** Open your app in the control panel, then select **Activity** and the failed deployment to read the build and job logs. Fix the cause, then run `terraform apply` again.

**The deploy fails at `fleet-migrate` with a primary key or SQL mode error.** DigitalOcean can reset the MySQL settings when it creates a cluster. Run `terraform apply` again so Terraform reapplies `sql_require_primary_key` and `sql_mode`, then redeploy the app.

**Fleet restarts and its logs show it can't reach the database.** Open the database cluster in the control panel and check its trusted sources. They should list only your Fleet app. If you edited the list by hand, run `terraform apply` to restore it.

**Fleet restarts and runs out of memory.** The vulnerability database loads into memory. Set `fleet_instance_size` to a larger size slug, such as `apps-s-2vcpu-4gb`, and apply.

**The browser shows a certificate error on your custom domain.** The CNAME record isn't live yet. Confirm it points at the `dns_target` output, and wait for DNS to propagate.

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
<meta name="description" value="Deploy Fleet on DigitalOcean App Platform with Terraform, managed MySQL, and automatic HTTPS. No servers to patch.">
