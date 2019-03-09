# Adding Hosts To Fleet

Kolide Fleet is powered by the open source osquery tool. To connect a host to Kolide Fleet, you have two general options. You can install the osquery binaries on your hosts via the packages distributed at https://osquery.io/downloads or you can use the [Kolide Osquery Launcher](https://github.com/kolide/launcher). The Launcher is a light wrapper that aims to make running and deploying osquery easier by adding a few features and minimizing the configuration interface. Some features of The Launcher are:

- Secure autoupdates to the latest stable osqueryd
- Remote communication via a strongly-typed, versioned, modern gRPC server API
- a curated `kolide_best_practices` table which includes a curated set of standards for the modern enterprise

The Launcher also contains robust tooling to help you generate packages for your environment that are designed to work together with Kolide Fleet. For specific documentation on using Launcher with Fleet, see the section below called "Kolide Osquery Launcher".

If you'd like to use the native osqueryd binaries to connect to Fleet, this is enabled by using osquery's TLS API plugins that are principally documented on the official osquery wiki: http://osquery.readthedocs.io/en/stable/deployment/remote/. These plugins are very customizable and thus have a large configuration surface. Configuring osqueryd to communicate with Fleet is documented below in the "Native Osquery TLS Plugins" section.

## Kolide Osquery Launcher

We provide compiled releases of the launcher for all supported platforms. Those can be found [here](https://github.com/kolide/launcher/releases). But if you’d like to compile from source, the instructions are [here](https://github.com/kolide/fleet/tree/master/docs/development).

#### Connecting a single Launcher to Fleet

To directly execute the launcher binary without having to mess with packages, invoke the binary with just a few flags:

- `--hostname`: the hostname of the gRPC server for your environment
- `--root_directory`: the location of the local database, pidfiles, etc.
- `--enroll_secret`: the enroll secret you generated above for your environment

```
./build/launcher \
  --hostname=fleet.acme.net:443 \
  --root_directory=$(mktemp -d) \
  --enroll_secret=32IeN3QLgckHUmMD3iW40kyLdNJcGzP5
```

You may also need to define the `--insecure` and/or `--insecure_grpc` flag. If you're running Fleet locally, include `--insecure` because your TLS certificate will not be signed by a valid CA.

#### Generating packages

The Launcher also provides easy, robust tooling for creating packages that you can distribute across your fleet:

```
$ make package-builder
$ ./build/package-builder make \
  --hostname=fleet.acme.net:443 \
  --enroll_secret=32IeN3QLgckHUmMD3iW40kyLdNJcGzP5
```

As you can see, to generate a Launcher package, you need only call `package-builder make` with two command-line arguments:

- `--hostname`: the hostname of the gRPC server for your environment
- `--enroll_secret`: the enroll secret you generated above for your environment

You can also add the `--mac_package_signing_key` flag to define the name of the macOS package signing key name that you'd like to use to sign the macOS packages. For example:

```
--mac_package_signing_key="Developer ID Installer: Acme Inc (ABCDEF123456)"
```

If you want to generate a package for local testing, you can call `package-builder make` with the `--insecure` flag as well and the auto-run command in the resultant packages will include `--insecure` as well.

## Native Osquery TLS Plugins

You can find various ways to install osquery on a variety of platforms at https://osquery.io/downloads. Once you have installed osquery, you need to do two things:

#### Set an environment variable with an agent enrollment secret

The enrollment secret is a value that osquery uses to ensure a level of confidence that the host running osquery is actually a host that you would like to hear from. There are a few ways you can set the enrollment secret on the hosts which you control. You can either set the value as:

- an value of an environment variable (a common name is `OSQUERY_ENROLL_SECRET`)
- the content of a local file (a common path is `/etc/osquery/enrollment_secret`)

The value of the environment variable or content of the file should be a secret shared between the osqueryd client and the Fleet server. This is basically osqueryd's passphrase which it uses to authenticate with Fleet, convincing Fleet that it is actually one of your hosts. The passphrase could be whatever you'd like, but it would be prudent to have the passphrase long, complex, mixed-case, etc. When you launch the Fleet server, you should specify this same value.

If you use an environment variable for this, you can specify it with the `--enroll_secret_env` flag when you launch osqueryd. If you use a local file for this, you can specify it's path with the `--enroll_secret_path` flag.
s
If your organization has a robust internal public key infrastructure (PKI) and you already deploy TLS client certificates to each host to uniquely identify them, then osquery supports an advanced authentication mechanism which takes advantage of this. For assitance, please file a [Github issue](https://github.com/kolide/fleet/issues/new) or contact us on [osquery Slack](https://osquery-slack.herokuapp.com/).

#### Deploy the TLS certificate that osquery will use to communicate with Fleet

When Fleet uses a self-signed certificate, osquery agents will need a copy of that certificate in order to authenticate the Fleet server. If clients connect directly to the Fleet server, you can download the certificate through the Fleet UI. From the main dashboard (`/hosts/manage`), click "Add New Host" and "Fetch Kolide Certificate". If Fleet is running behind a load-balancer that terminates TLS, you will have to talk to your system administrator about where to find this certificate.

It is important that the CN of this certificate matches the hostname or IP that osqueryd clients will use to connect.

Specify the path to this certificate with the `--tls_server_certs` flag when you launch osqueryd.

## Launching osqueryd

Assuming that you are deploying your enrollment secret as the environment variable `OSQUERY_ENROLL_SECRET` and your osquery server certificate is at `/etc/osquery/kolide.crt`, you could copy and paste the following command with the following flags (be sure to replace kolide.acme.net with the hostname or IP of your Fleet installation):

```
sudo osqueryd \
 --enroll_secret_env=OSQUERY_ENROLL_SECRET \
 --tls_server_certs=/etc/osquery/kolide.crt \
 --tls_hostname=kolide.acme.net \
 --host_identifier=uuid \
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

If your osquery server certificate is deployed to a path that is not `/etc/osquery/kolide.crt`, be sure to update the `--tls_server_certs` flag. Similarly, if your enrollment secret is in an environment variable that is not called `OSQUERY_ENROLL_SECRET`, then be sure to update the `--enroll_secret_env` environment variable. If your enroll secret is defined in a local file, specify the file's path with the `--enroll_secret_path` flag instead of using the `--enroll_secret_env` flag.

### Using a flag file to manage flags

For your convenience, osqueryd supports putting all of your flags into a single file. We suggest deploying this file to `/etc/osquery/kolide.flags`. If you've deployed the appropriate osquery flags to that path, you could simply launch osquery via:

```
osqueryd --flagfile=/etc/osquery/kolide.flags
```

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

Note that osqueryd requires a full certificate chain, even for certificates which might be trusted by your keychain. The "Fetch Kolide Certificate" button in the Add New Host screen will attempt to fetch the full chain for you.

Once you've configured the `config.mk` file with the correct variables, you can run `make` in the `tools/mac` directory. Running `make` will create a new `kolide-enroll.pkg` file which you can import into your software repository and deploy to your mac fleet.

The enrollment package must installed after the osqueryd package, and will install a LaunchDaemon to keep the osqueryd process running.
