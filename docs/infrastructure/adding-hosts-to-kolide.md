# Adding Hosts To Kolide

To use Kolide, you must install the open source osquery tool on the hosts which you wish to monitor. You can find various ways to install osquery on a variety of platforms at https://osquery.io/downloads.

Once you have installed osquery, you need to do two things:

#### Set an environment variable with an agent enrollment secret

The enrollment secret is a value that osquery uses to ensure a level of confidence that the host running osquery is actually a host that you would like to hear from. There are a few ways you can set the enrollment secret on the hosts which you control. You can either set the value as:

- an value of an environment variable (a common name is `OSQUERY_ENROLL_SECRET`)
- the content of a local file (a common path is `/etc/osquery/enrollment_secret`)

The value of the environment variable or content of the file should be a secret shared between the osqueryd client and the Kolide server. This is basically osqueryd's passphrase which it uses to authenticate with Kolide, convincing Kolide that it is actually one of your hosts. The passphrase could be whatever you'd like, but it would be prudent to have the passphrase long, complex, mixed-case, etc. When you launch the Kolide server, you should specify this same value.

If you use an environment variable for this, you can specify it with the `--enroll_secret_env` flag when you launch osqueryd. If you use a local file for this, you can specify it's path with the `--enroll_secret_path` flag.

If your organization has a robust internal public key infrastructure (PKI) and you already deploy TLS client certificates to each host to uniquely identify them, then osquery supports an advanced authentication mechanism which takes advantage of this. Please contact [help@kolide.co](mailto:help@kolide.co) for assistance with this option.

#### Deploy the TLS certificate that osquery will use to communicate with Kolide

When Kolide uses a self-signed certificate, osquery agents will need a copy of that certificate in order to authenticate the Kolide server. If clients connect directly to the Kolide server, you can download the certificate through the Kolide UI. From the main dashboard (`/hosts/manage`), click "Add New Host" and "Fetch Kolide Certificate". If Kolide is running behind a load-balancer that terminates TLS, you will have to talk to your system administrator about where to find this certificate.

It is important that the CN of this certificate matches the hostname or IP that osqueryd clients will use to connect.

Specify the path to this certificate with the `--tls_server_certs` flag when you launch osqueryd.

## Launching osqueryd

Assuming that you are deploying your enrollment secret as the environment variable `OSQUERY_ENROLL_SECRET` and your osquery server certificate is at `/etc/osquery/kolide.crt`, you could copy and paste the following command with the following flags (be sure to replace acme.kolide.co with the hostname or IP of your Kolide installation):

```
osqueryd
 --enroll_secret_env=OSQUERY_ENROLL_SECRET
 --tls_server_certs=/etc/osquery/kolide.crt
 --tls_hostname=acme.kolide.co
 --host_identifier=hostname
 --enroll_tls_endpoint=/api/v1/osquery/enroll
 --config_plugin=tls
 --config_tls_endpoint=/api/v1/osquery/config
 --config_tls_refresh=10
 --disable_distributed=false
 --distributed_plugin=tls
 --distributed_interval=10
 --distributed_tls_max_attempts=3
 --distributed_tls_read_endpoint=/api/v1/osquery/distributed/read
 --distributed_tls_write_endpoint=/api/v1/osquery/distributed/write
 --logger_plugin=tls
 --logger_tls_endpoint=/api/v1/osquery/log
 --logger_tls_period=10
```

If your osquery server certificate is deployed to a path that is not `/etc/osquery/kolide.crt`, be sure to update the `--tls_server_certs` flag. Similarly, if your enrollment secret is in an environment variable that is not called `OSQUERY_ENROLL_SECRET`, then be sure to update the `--enroll_secret_env` environment variable. If your enroll secret is defined in a local file, specify the file's path with the `--enroll_secret_path` flag instead of using the `--enroll_secret_env` flag.

### Using a flag file to manage flags

For your convenience, osqueryd supports putting all of your flags into a single file. We suggest deploying this file to `/etc/osquery/kolide.flags`. If you've deployed the appropriate osquery flags to that path, you could simply launch osquery via:

```
osqueryd --flagfile=/etc/osquery/kolide.flags
```

## Enrolling multiple macOS hosts

If you're managing an enterprise environment with multiple Mac devices, you likely have an enterprise deployment tool like [Munki](https://www.munki.org/munki/) or [Jamf Pro](https://www.jamf.com/products/jamf-pro/) to deliver software to your mac fleet. You can deploy osqueryd and enroll all your macs into Kolide using your software management tool of choice.

First, [download](https://osquery.io/downloads/) and import the osquery package into your software management repository. You can also use the community supported [autopkg recipe](https://github.com/autopkg/keeleysam-recipes/tree/master/osquery)
to keep osqueryd updated.

Next, you will have to create an enrollment package to get osqueryd running and talking to Kolide. Specifically, you'll have to create a custom package because you have to provide specific information about your Kolide deployment. To make this as easy as possible, we've created a Makefile to help you build a macOS enrollment package.

First, download the Kolide repository from GitHub and navigate to the `tools/mac` directory of the repository.

Next, you'll have to edit the `config.mk` file. You'll find all of the necessary information by clicking "Add New Host" in your kolide server.

 - Set the `KOLIDE_HOSTNAME` variable to the FQDN of your Kolide server.
 - Set the `ENROLL_SECRET` variable to the enroll secret you got from Kolide.
 - Paste the contents of the Kolide TLS certificate after the following line:
      ```
      define KOLIDE_TLS_CERTIFICATE
      ```

Note that osqueryd requires a full certificate chain, even for certificates which might be trusted by your keychain. The "Fetch Kolide Certificate" button in the Add New Host screen will attempt to fetch the full chain for you.

Once you've configured the `config.mk` file with the correct variables, you can run `make` in the `tools/mac` directory. Running `make` will create a new `kolide-enroll.pkg` file which you can import into your software repository and deploy to your mac fleet.

The enrollment package must installed after the osqueryd package, and will install a LaunchDaemon to keep the osqueryd process running.
