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

To ensure that it is especially difficult to compromise the TLS communication that occurs between the osqueryd agent and the Kolide server, osqueryd requires you to explicitly define the root certificate authority of the Kolide server (PEM-encoded) in the content of a local file. If you are running osqueryd behind a load-balancer which does TLS termination, then you will have to talk to your system administrator about where to find this certificate. If your browser is directly connected to the same web server which your osqueryd clients will be, you can download the certificate [here](http://66.media.tumblr.com/tumblr_lhkx3nKGK71qgzsew.jpg).

You can specify the path to this certificate with the `--tls_server_certs` flag when you launch osqueryd.

## Launching osqueryd

Assuming that you arere deploying your enrollment secret as the environment variable `OSQUERY_ENROLL_SECRET` and your osquery server certificate is at `/etc/osquery/kolide.crt`, you could copy and paste the following command with the following flags (be sure to replace acme.kolide.co with the hostname for your Kolide installation):

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

If your osquery server certificate is deployed to a path that is not `/etc/osquery/kolide.crt`, then be sure to update the `--tls_server_certs` flag. Similarly, if your enrollment secret is in an environment variable that is not called `OSQUERY_ENROLL_SECRET`, then be sure to update the `--enroll_secret_env` environment variable. If your enroll secret is defined in a local file, specify the file's path with the `--enroll_secret_path` flag instead of using the `--enroll_secret_env` flag.

### Using a flag file to manage flags

For your convenience, osqueryd supports putting all of your flags into a single file. This file is commonly deployed to `/etc/osquery/osquery.flags`. If you've deployed the appropriate osquery flags to that path, you could simply launch osquery via:

```
osqueryd --flagfile=/etc/osquery/osquery.flags
```

## Configuration Management

We recommend that you use an infrastructure configuration management tool to manage these osquery configurations consistently across your environment. If you're unsure about what configuration management tools your organization uses, contact your company's system administrators. If you are evaluating new solutions for this problem, the founders of Kolide have successfully managed configurations in large production environments using [Chef](https://www.chef.io/chef/) and [Puppet](https://puppet.com/).
