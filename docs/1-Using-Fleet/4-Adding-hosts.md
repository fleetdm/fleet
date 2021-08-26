# Adding hosts
- [Orbit for osquery](#orbit-for-osquery)
- [Native osquery TLS plugins](#native-osquery-tls-plugins)
	- [Set an environment variable with an agent enrollment secret](#set-an-environment-variable-with-an-agent-enrollment-secret)
  - [Deploy the TLS certificate that osquery will use to communicate with Fleet](#deploy-the-tls-certificate-that-osquery-will-use-to-communicate-with-fleet)
- [Launching osqueryd](#launching-osqueryd)
  - [Using a flag file to manage flags](#using-a-flag-file-to-manage-flags)
- [Kolide osquery Launcher](#kolide-osquery-launcher)
- [Enrolling multiple macOS hosts](#enrolling-multiple-macos-hosts)
- [Multiple enroll secrets](#multiple-enroll-secrets)

Fleet is powered by the open source osquery tool. To connect a host to Fleet, you have three general options: 
- You can use [Orbit for osquery](https://github.com/fleetdm/orbit)
- You can install the osquery binaries on your hosts via the packages distributed at https://osquery.io/downloads
- You can use the [Kolide Osquery Launcher](https://github.com/kolide/launcher).

## Orbit for osquery

Orbit is an [osquery](https://github.com/osquery/osquery) runtime and autoupdater. With Orbit, it's easy to deploy osquery, manage configurations, and stay up to date. Orbit eases the deployment of osquery connected with a [Fleet server](https://github.com/fleetdm/fleet), and is a (near) drop-in replacement for osquery in a variety of deployment scenarios.

Orbit is the recommended agent for Fleet. But Orbit can be used with or without Fleet, and Fleet can be used with or without Orbit.

Check out the [Orbit Github repository](https://github.com/fleetdm/fleet) for information on using and packaging Orbit for osquery.

## Native osquery TLS plugins

> If you'd like to use the native osqueryd binaries to connect to Fleet, this is enabled by using osquery's TLS API plugins that are principally documented on the official osquery wiki: http://osquery.readthedocs.io/en/stable/deployment/remote/. These plugins are very customizable and thus have a large configuration surface. Configuring osqueryd to communicate with Fleet is documented below in the "Native Osquery TLS Plugins" section.

You can find various ways to install osquery on a variety of platforms at https://osquery.io/downloads. Once you have installed osquery, you need to do two things:

### Set an environment variable with an agent enrollment secret

The enrollment secret is a value that osquery provides to authenticate with Fleet. There are a few ways you can set the enrollment secret on the hosts which you control. You can either set the value as:

- an value of an environment variable (a common name is `OSQUERY_ENROLL_SECRET`)
- the content of a local file (a common path is `/etc/osquery/enrollment_secret`)

The value of the environment variable or content of the file should be a secret shared between the osqueryd client and the Fleet server. This is basically osqueryd's passphrase which it uses to authenticate with Fleet, convincing Fleet that it is actually one of your hosts. The passphrase could be whatever you'd like, but it would be prudent to have the passphrase long, complex, mixed-case, etc. When you launch the Fleet server, you should specify this same value.

If you use an environment variable for this, you can specify it with the `--enroll_secret_env` flag when you launch osqueryd. If you use a local file for this, you can specify it's path with the `--enroll_secret_path` flag.

To retrieve the enroll secret, use the "Add New Host" dialog in the Fleet UI or
`fleetctl get enroll_secret`).

If your organization has a robust internal public key infrastructure (PKI) and you already deploy TLS client certificates to each host to uniquely identify them, then osquery supports an advanced authentication mechanism which takes advantage of this. Fleet can be fronted with a proxy that will perform the TLS client authentication.

### Deploy the TLS certificate that osquery will use to communicate with Fleet

When Fleet uses a self-signed certificate, osquery agents will need a copy of that certificate in order to authenticate the Fleet server. If clients connect directly to the Fleet server, you can download the certificate through the Fleet UI. From the main dashboard (`/hosts/manage`), click "Add New Host" and "Fetch Certificate". If Fleet is running behind a load-balancer that terminates TLS, you will have to talk to your system administrator about where to find this certificate.

It is important that the CN of this certificate matches the hostname or IP that osqueryd clients will use to connect.

Specify the path to this certificate with the `--tls_server_certs` flag when you launch osqueryd.

## Launching osqueryd

Assuming that you are deploying your enrollment secret in the file `/etc/osquery/enroll_secret` and your osquery server certificate is at `/etc/osquery/fleet.crt`, you could copy and paste the following command with the following flags (be sure to replace `fleet.acme.net` with the hostname or IP of your Fleet installation):

```
sudo osqueryd \
 --enroll_secret_path=/etc/osquery/enroll_secret \
 --tls_server_certs=/etc/osquery/fleet.crt \
 --tls_hostname=fleet.acme.net \
 --host_identifier=instance \
 --enroll_tls_endpoint=/api/v1/osquery/enroll \
 --config_plugin=tls \
 --config_tls_endpoint=/api/v1/osquery/config \
 --config_refresh=10 \
 --disable_distributed=false \
 --distributed_plugin=tls \
 --distributed_interval=10 \
 --distributed_tls_max_attempts=3 \
 --distributed_tls_read_endpoint=/api/v1/osquery/distributed/read \
 --distributed_tls_write_endpoint=/api/v1/osquery/distributed/write \
 --logger_plugin=tls \
 --logger_tls_endpoint=/api/v1/osquery/log \
 --logger_tls_period=10
```

If your osquery server certificate is deployed to a path that is not `/etc/osquery/fleet.crt`, be sure to update the `--tls_server_certs` flag. Similarly, if your enrollment secret is in an environment variable that is not called `OSQUERY_ENROLL_SECRET`, then be sure to update the `--enroll_secret_env` environment variable. If your enroll secret is defined in a local file, specify the file's path with the `--enroll_secret_path` flag instead of using the `--enroll_secret_env` flag.

### Using a flag file to manage flags

For your convenience, osqueryd supports putting all of your flags into a single file. We suggest deploying this file to `/etc/osquery/fleet.flags`. If you've deployed the appropriate osquery flags to that path, you could simply launch osquery via:

```
osqueryd --flagfile=/etc/osquery/fleet.flags
```

#### Flag file on Windows

Ensure that paths to files in the flag file are absolute, and not quoted. For example in `C:\Program Files\osquery\osquery.flags`:

```
--tls_server_certs=C:\Program Files\osquery\fleet.pem
--enroll_secret_path=C:\Program Files\osquery\secret.txt
```

## Kolide osquery Launcher

Instructions on connecting a single Launcher to Fleet can be found [here in the Launcher documentation](https://github.com/kolide/launcher/blob/master/docs/launcher.md#connecting-to-fleet).

Kolide provides compiled releases of their launcher for all supported platforms. 
Those can be found [here](https://github.com/kolide/launcher/releases), or if you’d like to compile from source, the instructions are [here](https://github.com/kolide/launcher/blob/master/docs/launcher.md#building-the-code).

## Enrolling multiple macOS hosts

If you're managing an enterprise environment with multiple Mac devices, you likely have an enterprise deployment tool like [Munki](https://www.munki.org/munki/) or [Jamf Pro](https://www.jamf.com/products/jamf-pro/) to deliver software to your mac fleet. You can deploy osqueryd and enroll all your macs into Fleet using your software management tool of choice.

First, [download](https://osquery.io/downloads/) and import the osquery package into your software management repository. You can also use the community supported [autopkg recipe](https://github.com/autopkg/keeleysam-recipes/tree/master/osquery)
to keep osqueryd updated.

Next, you will have to create an enrollment package to get osqueryd running and talking to Fleet. Specifically, you'll have to create a custom package because you have to provide specific information about your Fleet deployment. To make this as easy as possible, we've created a Makefile to help you build a macOS enrollment package.

First, download the Fleet repository from GitHub and navigate to the `tools/mac` directory of the repository.

Next, you'll have to edit the `config.mk` file. You'll find all of the necessary information by clicking "Add New Host" in your Fleet server.

 - Set the `KOLIDE_HOSTNAME` variable to the FQDN of your Fleet server.
 - Set the `ENROLL_SECRET` variable to the enroll secret you got from Fleet.
 - Paste the contents of the Fleet TLS certificate after the following line:
      ```
      define KOLIDE_TLS_CERTIFICATE
      ```

Note that osqueryd requires a full certificate chain, even for certificates which might be trusted by your keychain. The "Fetch Fleet Certificate" button in the Add New Host screen will attempt to fetch the full chain for you.

Once you've configured the `config.mk` file with the correct variables, you can run `make` in the `tools/mac` directory. Running `make` will create a new `kolide-enroll.pkg` file which you can import into your software repository and deploy to your mac fleet.

The enrollment package must installed after the osqueryd package, and will install a LaunchDaemon to keep the osqueryd process running.

## Multiple Enroll Secrets

Multiple enroll secrets can be set to allow different groups of hosts to
authenticate with Fleet. When a host enrolls, the corresponding enroll secret is
recorded and can be used to segment hosts.

To set the enroll secret, use the `fleetctl` tool to apply an [enroll secret spec](../1-Using-Fleet/2-fleetctl-CLI.md#enroll-secrets) 
