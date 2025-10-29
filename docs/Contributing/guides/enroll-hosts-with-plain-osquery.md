# Enroll hosts with plain osquery

Osquery's [TLS API plugin](http://osquery.readthedocs.io/en/stable/deployment/remote/) lets you use the native osqueryd binaries to connect to Fleet.

You can find various ways to install osquery on your hosts at https://osquery.io/downloads. Once you have installed osquery, you need to do three things on your hosts: 

- Set up your Fleet enroll secret.
- Provide the TLS certificate that osquery will use to communicate with Fleet.
- Configure and launch osqueryd.

## Set up your Fleet enroll secret

The enroll secret is a value that osquery provides to authenticate with Fleet. There are a few ways you can set the enroll secret on the hosts that you control. You can either set the value as

- a value of an environment variable (a common name is `OSQUERY_ENROLL_SECRET`)
- the content of a local file (a common path is `/etc/osquery/enroll_secret`)

The value of the environment variable or content of the file should be a secret shared between the osqueryd client and the Fleet server. This is osqueryd's passphrase which it uses to authenticate with Fleet, convincing Fleet that it is actually one of your hosts. The passphrase could be whatever you'd like, but it would be prudent to have the passphrase long, complex, mixed-case, etc. When you launch the Fleet server, you should specify this same value.

If you use an environment variable for this, you can specify it with the `--enroll_secret_env` flag when you launch osqueryd. If you use a local file for this, you can specify its path with the `--enroll_secret_path` flag.

To retrieve the enroll secret, use the "Add New Host" dialog in the Fleet UI or
`fleetctl get enroll_secret`).

If your organization has a robust internal public key infrastructure (PKI) and you already deploy TLS client certificates to each host to uniquely identify them, then osquery supports an advanced authentication mechanism that takes advantage of this. Fleet can be fronted with a proxy to perform the TLS client authentication.

## Provide the TLS certificate that osquery will use to communicate with Fleet

When Fleet uses a self-signed certificate, osquery agents will need a copy of that certificate in order to authenticate the Fleet server. If clients connect directly to the Fleet server, you can download the certificate through the Fleet UI. From the main dashboard (`/hosts/manage`), click **Add New Host** and **Fetch Certificate**. If Fleet is running behind a load balancer that terminates TLS, you will have to talk to your system administrator about where to find this certificate.

It is important that the CN of this certificate matches the hostname or IP that osqueryd clients will use to connect.

Specify the path to this certificate with the `--tls_server_certs` flag when you launch osqueryd.

## Configure and launch osquery

For osquery to connect to the Fleet server, some flags need to be set:

```sh
 --enroll_secret_path=/etc/osquery/enroll_secret 
 --tls_server_certs=/etc/osquery/fleet.crt
 --tls_hostname=fleet.example.com 
 --host_identifier=uuid 
 --enroll_tls_endpoint=/api/osquery/enroll 
 --config_plugin=tls 
 --config_tls_endpoint=/api/osquery/config 
 --config_refresh=10 
 --disable_distributed=false
 --distributed_plugin=tls 
 --distributed_interval=10 
 --distributed_tls_max_attempts=3 
 --distributed_tls_read_endpoint=/api/osquery/distributed/read 
 --distributed_tls_write_endpoint=/api/osquery/distributed/write 
 --logger_plugin=tls 
 --logger_tls_endpoint=/api/osquery/log 
 --logger_tls_period=10
 ```
These can be specified directly in the command line or saved to a flag file. 

### Launching osqueryd using command-line flags

Assuming that you are deploying your enroll secret in the file `/etc/osquery/enroll_secret` and your osquery server certificate is at `/etc/osquery/fleet.crt`, you could copy and paste the following command with the following flags (be sure to replace `fleet.acme.net` with the hostname or IP of your Fleet installation):

```sh
sudo osqueryd \
 --enroll_secret_path=/etc/osquery/enroll_secret \
 --tls_server_certs=/etc/osquery/fleet.crt \
 --tls_hostname=fleet.example.com \
 --host_identifier=uuid \
 --enroll_tls_endpoint=/api/osquery/enroll \
 --config_plugin=tls \
 --config_tls_endpoint=/api/osquery/config \
 --config_refresh=10 \
 --disable_distributed=false \
 --distributed_plugin=tls \
 --distributed_interval=10 \
 --distributed_tls_max_attempts=3 \
 --distributed_tls_read_endpoint=/api/osquery/distributed/read \
 --distributed_tls_write_endpoint=/api/osquery/distributed/write \
 --logger_plugin=tls \
 --logger_tls_endpoint=/api/osquery/log \
 --logger_tls_period=10
```

If your osquery server certificate is deployed to a path that is not `/etc/osquery/fleet.crt`, be sure to update the `--tls_server_certs` flag. Similarly, if your enroll secret is in an environment variable that is not called `OSQUERY_ENROLL_SECRET`. Then, be sure to update the `--enroll_secret_env` environment variable.

If your enroll secret is defined in a local file, specify the file's path with the `--enroll_secret_path` flag instead of using the `--enroll_secret_env` flag.

### Launching osqueryd using a flag file

For your convenience, osqueryd supports putting all your flags into a single file. We suggest deploying this file to `/etc/osquery/fleet.flags`. If you've deployed the appropriate osquery flags to that path, you could simply launch osquery via:

```sh
osqueryd --flagfile=/etc/osquery/fleet.flags
```

When using a flag file on Windows, make sure that file paths in the flag file are absolute and not quoted. For example, in `C:\Program Files\osquery\osquery.flags`:

```sh
--tls_server_certs=C:\Program Files\osquery\fleet.pem
--enroll_secret_path=C:\Program Files\osquery\secret.txt
```

## Migrating from plain osquery to osquery installer

The following is a strategy for migrating a plain osquery deployment. Unlike plain osquery, Fleet's
osquery installer supports the automatic updating of osquery on your hosts so that you don't have to
deploy a new package for every new osquery release.

### Generate installer

```sh
fleetctl package --type [pkg|msi|deb|rpm] --fleet-url [fleet-hostname:port] --enroll-secret [secret]
```

If you currently ship a certificate (`fleet.pem`), also include this in the generated package with
`--fleet-certificate [/path/to/fleet.pem]`.

Fleet automatically manages most of the osquery flags to connect to the Fleet server. There's no
need to set any of the flags mentioned above in [Configure and launch osquery](#configure-and-launch-osquery). To
include other osquery flags, provide a flagfile when packaging with `--osquery-flagfile
[/path/to/osquery.flags]`.

Test the installers on each platform before initiating the migration.

### Migrate

Install the generated package using your standard deployment tooling (Chef, Puppet, etc.). At this
time, [uninstall the existing
osquery](https://blog.fleetdm.com/how-to-uninstall-osquery-f01cc49a37b9).

If the existing enrolled hosts use `--host_identifier=uuid` (or the `uuid` setting for Fleet's
[osquery_host_identifier](https://fleetdm.com/docs/deploying/configuration#osquery-host-identifier)), the new
installation should appear as the same host in the Fleet UI. If other settings are used, duplicate
entries will appear in the Fleet UI. The older entries can be automatically cleaned up with the host
expiration setting. To configure this setting, in the Fleet UI, head to **Settings > Organization settings > Advanced options**. 

<meta name="pageOrderInSection" value="1600">
