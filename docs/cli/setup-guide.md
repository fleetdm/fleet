# Setting Up Fleet via the CLI

In this document, I'm going to walk through how to setup and configure Kolide Fleet using just the CLI (which in-turn uses the Go API client). This document will hopefully illustrate:

- A minimal CLI workflow for managing an osquery fleet
- The set of API interactions that are required if you want to perform remote, automated management of a Fleet instance

## Running Fleet

For the sake of this tutorial, I will be using the local development Docker Compose infrastructure to run Fleet locally. This is documented in some detail in the [developer documentation](../development/development-infrastructure.md), but the following are the minimal set of commands that you can run from the root of the repository (assuming that you have a working Go/JavaScript toolchain installed along with Docker Compose):

```
docker-compose up -d
make deps
make generate
make
./build/fleet prepare db
./build/fleet serve --auth_jwt_key="insecure"
```

The `fleet serve` command will be the long running command that runs the Fleet server.

## `fleetctl config`

At this point, the MySQL database doesn't have any users in it. Because of this, Fleet is exposing a one-time setup endpoint. Before we can hit that endpoint (by running `fleetctl setup`), we have to first configure the local `fleetctl` context.

Now, since our Fleet instance is local in this tutorial, we didn't get a valid TLS certificate, so we need to run the following to configure our Fleet context:

```
$ fleetctl config set --address https://localhost:8080 --tls-skip-verify
[+] Set the address config key to "https://localhost:8080" in the "default" context
[+] Set the tls-skip-verify config key to "true" in the "default" context
```

Now, if you were connecting to a Fleet instance for real, you wouldn't want to skip TLS certificate verification, so you might run something like:

```
$ fleetctl config set --address https://fleet.osquery.tools
[+] Set the address config key to "https://fleet.osquery.tools" in the "default" context
```

## `fleetctl setup`

Now that we've configured our local CLI context, lets go ahead and create our admin account:

```
$ fleetctl setup --email mike@arpaia.co
Password:
[+] Fleet setup successful and context configured!
```

It's possible to specify the password via the `--password` flag or the `$PASSWORD` environment variable, but be cautious of the security implications of such an action. For local use, the interactive mode above is the most secure.

## Connecting a Host

For the sake of this tutorial, I'm going to be using Kolide's osquery launcher to start osquery locally and connect it to Fleet. To learn more about connecting osquery to Fleet, see the [Adding Hosts to Fleet](../infrastructure/adding-hosts-to-fleet.md) documentation.

To get your osquery enroll secret, run the following:

```
$ fleetctl get enroll-secret
E7P6zs9D0mvY7ct08weZ7xvLtQfGYrdC
```

You need to use this secret to connect a host. If you're running Fleet locally, you'd run:

```
launcher \
  --hostname localhost:8080 \
  --enroll_secret E7P6zs9D0mvY7ct08weZ7xvLtQfGYrdC \
  --root_directory=$(mktemp -d) \
  --insecure
```

## Query Hosts

To run a simple query against all hosts, you might run something like the following:

```
$ fleetctl query --query 'select * from osquery_info;' --labels='All Hosts' > results.json
⠂  100% responded (100% online) | 1/1 targeted hosts (1/1 online)
^C
```

When the query is done (or you have enough results), CTRL-C and look at the `results.json` file:

```json
{
  "host": "marpaia",
  "rows": [
    {
      "build_distro": "10.13",
      "build_platform": "darwin",
      "config_hash": "d7cafcd183cc50c686b4c128263bd4eace5d89e1",
      "config_valid": "1",
      "extensions": "active",
      "host_hostname": "marpaia",
      "instance_id": "37840766-7182-4a68-a204-c7f577bd71e1",
      "pid": "22984",
      "start_time": "1527031727",
      "uuid": "B312055D-9209-5C89-9DDB-987299518FF7",
      "version": "3.2.3",
      "watcher": "-1"
    }
  ]
}
```

## Update Osquery Options

By default, each osquery node will check in with Fleet every 10 seconds. Let's say, for testing, you want to increase this to every 2 seconds. If this is the first time you've ever modified osquery options, let's download them locally:

```
fleetctl get options > options.yaml
```

The `options.yaml` file will look something like this:

```yaml
apiVersion: v1
kind: options
spec:
  config:
    decorators:
      load:
      - SELECT uuid AS host_uuid FROM system_info;
      - SELECT hostname AS hostname FROM system_info;
    options:
      disable_distributed: false
      distributed_interval: 10
      distributed_plugin: tls
      distributed_tls_max_attempts: 3
      distributed_tls_read_endpoint: /api/v1/osquery/distributed/read
      distributed_tls_write_endpoint: /api/v1/osquery/distributed/write
      logger_plugin: tls
      logger_tls_endpoint: /api/v1/osquery/log
      logger_tls_period: 10
      pack_delimiter: /
  overrides: {}
```

Let's edit the file so that the `distributed_interval` option is 2 instead of 10. Save the file and run:

```
fleetctl apply -f ./options.yaml
```

Now run a live query again. You should notice results coming back more quickly.

## Logging In To An Existing Fleet Instance

If you have an existing Fleet instance (version 2.0.0 or above), then simply run `fleet login` (after configuring your local CLI context):

```
$ fleetctl config set --address https://fleet.osquery.tools
[+] Set the address config key to "https://fleet.osquery.tools" in the "default" context

$ fleetctl login
Log in using the standard Fleet credentials.
Email: mike@arpaia.co
Password:
[+] Fleet login successful and context configured!
```

Once your local context is configured, you can use the above `fleetctl` normally. See `fleetctl --help` for more information.
