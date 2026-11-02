# Deploy a Fleet lab with Apple Container and fleetlab

Run Fleet, MySQL, and Redis on your Mac without Docker. `fleetlab` is a single script that sets everything up with Apple's open-source [container](https://github.com/apple/container) tool. It takes two commands and about 15 minutes, and deploys the same configuration as the [Deploy Fleet with Docker Compose](https://fleetdm.com/guides/deploy-fleet-on-docker-compose) guide.

> **Warning:** `fleetlab` is for local labs, testing, and learning Fleet. It isn't intended for production. For production deployments, see Fleet's [Reference configuration strategies](https://fleetdm.com/docs/deploy/reference-architectures#reference-configuration-strategies).

## What you'll need

- A Mac with Apple silicon (M1 or later)
- macOS 26 or later
- Administrator access (you'll be asked for your password during setup)
- At least 8 GB of free memory and about 6 GB of free disk space
- Port 1337 available

You don't need to install Rosetta or Apple container ahead of time. `fleetlab` checks for both and installs them if they're missing.

## Download the script

Create a new directory for your Fleet lab:

```sh
mkdir ~/fleet-lab
cd ~/fleet-lab
```

Download the script from GitHub, then make it executable:

```sh
curl -O https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/docs/solutions/apple-container/fleetlab.sh
chmod +x fleetlab.sh
```

Keep the script in this directory. It stores your configuration (`.env`) and certificates (`certs/`) alongside it.

## Set up your Mac

Run the one-time setup:

```sh
./fleetlab.sh setup
```

Setup checks your Mac, installs anything that's missing, and prepares your Fleet configuration. Enter your administrator password when prompted. If Apple container isn't installed, setup asks before downloading and installing it.

Setup is safe to run again. It skips anything that's already in place, and never overwrites an existing `.env` file.

## Optional: Add your license key

If you have a Fleet Premium license or a [30-day trial key](https://fleetdm.com/new-license), add it to the `.env` file setup created:

```sh
FLEET_LICENSE_KEY=your_license_key_here
```

Leave this blank to use Fleet's free tier.

> **Note:** If you add the key after Fleet is already running, run `./fleetlab.sh update` to apply it.

## Start Fleet

Deploy Fleet:

```sh
./fleetlab.sh up
```

`fleetlab` downloads the images, starts MySQL and Redis, starts Fleet, and waits until Fleet is ready. The first run takes 5-10 minutes, because Fleet runs its database migrations under Rosetta.

When Fleet is ready, you'll see:

```
 ok Fleet is ready
    On this Mac:       https://localhost:1337
    Server URL to use: https://<Mac's LAN IP address>:1337
```

## Access Fleet

Open your browser and navigate to:

```
https://localhost:1337
```

Your browser will warn you about the self-signed certificate. This is expected. Select **Advanced** and continue. To remove the warning, see [Manage the certificate](#manage-the-certificate).

You'll see the Fleet setup screen. Follow the prompts to:

1. Create your first admin account
2. Configure your organization name
3. Set the Fleet server URL

For the Fleet server URL, use the **Server URL to use** address `fleetlab` printed. This is your Mac's LAN IP address, which enrolled hosts can reach. Don't use `localhost`.

> **Note:** Save the Fleet server URL before you enroll hosts. Fleet builds it into the fleetd packages it generates. A package built without it can't find your Fleet server.

## Enroll hosts

To enroll a host, install fleetd on it. Because `fleetlab` uses a self-signed certificate, you'll bundle the certificate with each fleetd package so hosts can verify your Fleet server.

1. Install fleetctl, Fleet's command-line tool:

   ```sh
   npm install -g fleetctl
   ```

2. In Fleet, select **Hosts** > **Add hosts** and choose your host's platform. Copy the `fleetctl package` command and add the `--fleet-certificate` flag. Check that `--fleet-url` uses your Mac's LAN IP address. For example, for a macOS host:

   ```sh
   fleetctl package --type=pkg --fleet-desktop \
     --fleet-url=https://<Mac's LAN IP address>:1337 \
     --enroll-secret=YOUR_ENROLL_SECRET \
     --fleet-certificate=certs/fleet.crt
   ```

   Run the command from your `~/fleet-lab` directory. With `--fleet-certificate`, fleetd trusts your certificate without changes to the host's certificate store. Learn more in [Certificates in fleetd](https://fleetdm.com/guides/certificates-in-fleetd).

3. Copy the package to your host and install it. On macOS, double-click the `.pkg` file

4. Open the **Hosts** page. The host appears within a few minutes.

> **Note:** Hosts connect to Fleet on port 1337. If a host can't reach Fleet, allow inbound connections on port 1337 in your Mac's firewall.

> **Note:** Testing with a virtual machine on the same Mac? Set the VM to bridged networking, so it reaches Fleet over your network like a separate computer. Shut the VM down before restarting Apple container's services (`container system stop`), because the restart can break the VM's network connection.

## Manage your Fleet lab

Run every command from your `~/fleet-lab` directory.

| Command | What it does |
| --- | --- |
| `./fleetlab.sh setup` | One-time setup. Safe to run again |
| `./fleetlab.sh up` | Deploys Fleet for the first time and waits until it's ready |
| `./fleetlab.sh start` | Starts an existing deployment in the right order |
| `./fleetlab.sh stop` | Stops all services. Your data is kept |
| `./fleetlab.sh status` | Shows the containers and whether Fleet is answering |
| `./fleetlab.sh logs` | Follows Fleet's logs. Use `logs mysql` or `logs redis` for the other services |
| `./fleetlab.sh update` | Pulls the latest Fleet image and recreates the Fleet container |
| `./fleetlab.sh certs` | Creates a new TLS certificate and restarts Fleet with it. Use it when your IP address changes or to renew the certificate |
| `./fleetlab.sh down` | Removes the containers. Your data is kept |
| `./fleetlab.sh reset` | Removes the containers and all data (asks first) |

Apple container doesn't restart containers automatically. After you restart your Mac, run `./fleetlab.sh start`.

Fleet automatically runs database migrations when you run `update`. Your data is kept.

To start over with a fresh Fleet, run `reset`, then `up`:

```sh
./fleetlab.sh reset
./fleetlab.sh up
```

## Manage the certificate

`fleetlab` creates a self-signed certificate valid for 365 days. It covers your Mac's hostname, `localhost`, `fleet.test`, `127.0.0.1`, and your Mac's LAN IP address. Check what it covers and when it expires:

```sh
openssl x509 -in certs/fleet.crt -noout -text | grep -A1 "Subject Alternative Name"
openssl x509 -in certs/fleet.crt -noout -enddate
```

Run `./fleetlab.sh certs` to renew the certificate, or when your Mac's LAN IP address changes. Then generate new fleetd packages and reinstall them on your hosts, because fleetd only trusts the certificate bundled in its package.

If your IP address changed, also update the Fleet server URL in **Settings** > **Organization settings** before generating packages. A DHCP reservation for your Mac avoids IP changes.

To use fleetctl with your Fleet lab, point it at your certificate, then log in:

```sh
fleetctl config set --address https://localhost:1337 --rootca certs/fleet.crt
fleetctl login
```

To stop your browser warning about the certificate, trust it in your Mac's System keychain:

```sh
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain certs/fleet.crt
```

> **Note:** This self-signed certificate works for enrolling hosts with fleetd. Fleet's MDM features, like turning on Apple MDM, have additional certificate requirements and aren't covered in this lab.

## What's running

Each service runs in its own lightweight Linux virtual machine on a private network called `fleet-net`. Containers find each other through a local DNS domain (`test`) that `setup` creates.

| Container | Role |
| --- | --- |
| `fleet-mysql` | Stores all Fleet data (hosts, policies, reports, users). Runs natively on Apple silicon |
| `fleet-redis` | Handles background jobs and caching. Runs natively on Apple silicon |
| `fleet` | The main application. Serves the web UI and API, and handles host connections. Fleet publishes only an amd64 image, so it runs under Rosetta with 4 GB of memory |

Fleet's data lives in five named volumes, which are kept when you stop or remove the containers.

## Configuration options

All Fleet settings live in `.env`, and work the same as in the [Docker Compose guide](https://fleetdm.com/guides/deploy-fleet-on-docker-compose). After changing a Fleet setting, recreate the Fleet container with `./fleetlab.sh update`.

These additional settings control `fleetlab` itself. Add them to `.env` if needed:

| Setting | Default | Description |
| --- | --- | --- |
| `FLEET_IMAGE` | `fleetdm/fleet:latest` | The Fleet image to run. Pin a version for repeatable labs |
| `FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS` | `yes` | Set to `no` to turn off vulnerability processing and reduce memory use |

## Troubleshooting

**"MySQL rejected the password in .env"**: an earlier deployment created the `fleet-mysql` volume with different passwords. Use that deployment's `.env`, or run `./fleetlab.sh reset`.

**Hosts stop checking in after your IP address changes**: run `./fleetlab.sh certs`, update the Fleet server URL, and reinstall new fleetd packages. See [Manage the certificate](#manage-the-certificate).

<meta name="articleTitle" value="Deploy a Fleet lab with Apple Container and fleetlab">
<meta name="authorGitHubUsername" value="adriant172">
<meta name="authorFullName" value="Matthew Thomas">
<meta name="publishedOn" value="2026-10-22">
<meta name="category" value="guides">
<meta name="description" value="Run a local Fleet lab with MySQL and Redis on an Apple silicon Mac using Apple container and the fleetlab script. No Docker required.">
