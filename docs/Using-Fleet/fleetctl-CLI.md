# fleetctl CLI

- [Introduction](#introduction)
- [Setting Up Fleet](#setting-up-fleet)
  - [Running Fleet](#running-fleet)
  - [`fleetctl config`](#fleetctl-config)
  - [`fleetctl setup`](#fleetctl-setup)
  - [Connecting a host](#connecting-a-host)
  - [Query hosts](#query-hosts)
- [Logging in to an existing Fleet instance](#logging-in-to-an-existing-fleet-instance)
- [Using fleetctl to configure Fleet](#using-fleetctl-to-configure-fleet)
- [Using fleetctl with an API-only user](#using-fleetctl-with-an-api-only-user)
- [File carving](#file-carving)
  - [Configuration](#configuration)
  - [Usage](#usage)
  - [Troubleshooting](#troubleshooting)

## Introduction

Fleetctl (pronounced "Fleet control") is a CLI tool for managing Fleet from the command line. Fleetctl enables a GitOps workflow with Fleet and osquery. With fleetctl, you can manage configurations, queries, packs, generate osquery installers, etc.

Fleetctl also provides a quick way to work with all the data exposed by Fleet without having to use the Fleet UI or work directly with the Fleet API.

## Setting up Fleet

This section walks through setting up and configuring Fleet via the CLI. If you already have a running fleet instance, skip ahead to [Logging in to an existing Fleet instance](#logging-in-to-an-existing-fleet-instance) to configure the `fleetctl` CLI.

This guide illustrates:

- A minimal CLI workflow for managing an osquery fleet
- The set of API interactions that are required if you want to perform remote, automated management of a Fleet instance

### Running Fleet

For the sake of this tutorial, we will be using the local development Docker Compose infrastructure to run Fleet locally. This is documented in some detail in the [developer documentation](../Contributing/Building-Fleet.md#development-infrastructure), but the following are the minimal set of commands that you can run from the root of the repository (assuming that you have a working Go/JavaScript toolchain installed along with Docker Compose):

```
docker-compose up -d
make deps
make generate
make
./build/fleet prepare db
./build/fleet serve
```

The `fleet serve` command will be the long running command that runs the Fleet server.

### Fleetctl config

At this point, the MySQL database doesn't have any users in it. Because of this, Fleet is exposing a one-time setup endpoint. Before we can hit that endpoint (by running `fleetctl setup`), we have to first configure the local `fleetctl` context.

Now, since our Fleet instance is local in this tutorial, we didn't get a valid TLS certificate, so we need to run the following to configure our Fleet context:

```
fleetctl config set --address https://localhost:8080 --tls-skip-verify
[+] Set the address config key to "https://localhost:8080" in the "default" context
[+] Set the tls-skip-verify config key to "true" in the "default" context
```

Now, if you were connecting to a Fleet instance for real, you wouldn't want to skip TLS certificate verification, so you might run something like:

```
fleetctl config set --address https://fleet.corp.example.com
[+] Set the address config key to "https://fleet.corp.example.com" in the "default" context
```

### Fleetctl setup

Now that we've configured our local CLI context, lets go ahead and create our admin account:

```
fleetctl setup --email zwass@example.com --name 'Zach' --org-name 'Fleet Test'
Password:
[+] Fleet setup successful and context configured!
```

It's possible to specify the password via the `--password` flag or the `$PASSWORD` environment variable, but be cautious of the security implications of such an action. For local use, the interactive mode above is the most secure.

### Query hosts

To run a simple query against all hosts, you might run something like the following:

```
fleetctl query --query 'select * from osquery_info;' --labels='All Hosts' > results.json
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

## Logging in to an existing Fleet instance

If you have an existing Fleet instance, run `fleetctl login` (after configuring your local CLI context):

```
fleetctl config set --address https://fleet.corp.example.com
[+] Set the address config key to "https://fleet.corp.example.com" in the "default" context

fleetctl login
Log in using the standard Fleet credentials.
Email: mike@arpaia.co
Password:
[+] Fleet login successful and context configured!
```

Once your local context is configured, you can use the above `fleetctl` normally. See `fleetctl --help` for more information.

### Logging in with SAML (SSO) authentication

Users that authenticate to Fleet via SSO should retrieve their API token from the UI and set it manually in their `fleetctl` configuration (instead of logging in via `fleetctl login`).

1. Go to the "My account" page in Fleet (https://fleet.corp.example.com/profile). Click the "Get API token" button to bring up a modal with the API token.

2. Set the API token in the `~/.fleet/config` file. The file should look like the following:

```
contexts:
  default:
    address: https://fleet.corp.example.com
    email: example@example.com
    token: your_token_here
```

Note the token can also be set with `fleetctl config set --token`, but this may leak the token into a user's shell history.

## Using fleetctl to configure Fleet

A Fleet configuration is defined using one or more declarative "messages" in yaml syntax. 

Fleet configuration can be retrieved and applied using the `fleetctl` tool.

### Fleetctl get

The `fleetctl get <fleet-entity-here> > <configuration-file-name-here>.yml` command allows you retrieve the current configuration and create a new file for specified Fleet entity (queries, packs, etc.)

### Fleetctl apply

The `fleetctl apply -f <configuration-file-name-here>.yml` allows you to apply the current configuration in the specified file.

Check out the [configuration files](./configuration-files/README.md) section of the documentation for example yaml files.

### Fleetctl convert

`fleetctl` includes easy tooling to convert osquery pack JSON into the
`fleetctl` format. Use `fleetctl convert` with a path to the pack file:

You can optionally supply `-o file_name` to output to a file destination.
```
fleetctl convert -f test.json
---
apiVersion: v1
kind: pack
spec:
  name: test
  queries:
  - description: "this is a test query"
    interval: 10
    name: processes
    query: processes
    removed: false
  targets:
    labels: null
---
apiVersion: v1
kind: query
spec:
  name: processes
  query: select * from processes
```

## Using fleetctl with an API-only user

Now that `fleetctl` and the Fleet server are configured, it can be helpful to create an API-only user to use when running automated workflows. An API-only user can be given a role based on the abilities it needs. The default access level is `Observer`. For more information on roles, see the [user permissions documentation](./Permissions.md#user-permissions).

### Create an API-only user

To create your new API-only user using `fleetctl user create` and picking a name, email, and password and setting the `api-only` flag to `true`:

```
fleetctl user create --name "API User" --email api@example.com --password temp!pass --api-only
```

### Reset the password

When a new user is created, a password reset is needed before that user will be able to perform queries. Since an API-only user cannot log in to the Fleet UI, this needs to be done through the REST API. We'll be doing this through the terminal using `curl`. 

First, log in to the new user account using `fleetctl login`. Once you're logged in successfully to the API-only user, set up a variable to hold the user's token:

```
token=$(fleetctl config get token | rev | cut -d ' ' -f 1 | rev)
```

Then use `curl` to send a required password reset request to the REST API through the terminal:

```
curl -d '{"new_password":"NewPassGoesHere"}' -H "Authorization: Bearer ${token}" -X POST https://fleet.corp.example.com/api/v1/fleet/perform_required_password_reset
```

If you see a response like this, the request was successful:

```
{
  "user": {
    "created_at": "2022-03-16T20:42:00Z",
    "updated_at": "2022-03-16T20:42:00Z",
    "id": 52,
    "name": "API User",
    "email": "api@example.com",
    "force_password_reset": false,
    "gravatar_url": "",
    "sso_enabled": false,
    "global_role": "observer",
    "api_only": true,
    "teams": []
  }
}
```

While the original token is no longer valid, it's never a bad idea to clear variables out once you're done with them:

```
unset token
```

### Use fleetctl as the new user

Now that the password has been reset, you will need to log in again using the updated password with `fleetctl login`. You'll now be able to perform tasks using `fleetctl` as your new API-only user. 

### Create a new context for your standard user (optional)

If you would like to use your API user by default for automated workflows and still use `fleetctl` with your standard user account, you can create an additional context. In the example below, a new `admin` context will be created:

```
fleetctl config set --address https://fleet.corp.example.com --context admin
[+] Context "admin" not found, creating it with default values
[+] Set the address config key to "https://dogfood.fleetdm.com" in the "admin" context

fleetctl login --context admin
Log in using the admin Fleet credentials.
Email: admin@example.com
Password:
[+] Fleet login successful and context configured!
```

Now, you can use the `context` flag to indicate which profile should be used rather than logging in and out every time you need to do something with elevated permissions. Running a command with no context will use the default profile (currently the new API-only user with `Observer` privileges):

```
fleetctl user create --email test@example.com --name "New User"
Error: Failed to create user: POST /api/v1/fleet/users/admin received status 403 forbidden: forbidden
```

The user creation failed because the API-only user doesn't have the right permissions. Running the command with the admin `context` specified will succeed:

```
fleetctl user create --email test@example.com --name "New User" --context admin
Enter password for user: 
Enter password for user (confirm): 
```

## File carving

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](../Deploying/Configuration.md#s3-file-carving-backend).

### Configuration

Given a working flagfile for connecting osquery agents to Fleet, add the following flags to enable carving:

```
--disable_carver=false
--carver_disable_function=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=2097152
```

The default flagfile provided in the "Add New Host" dialog also includes this configuration.

#### Carver block size

The `carver_block_size` flag should be configured in osquery.

For the (default) MySQL Backend, the configured value must be less than the value of
`max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for MySQL 5.7
is 4MB and for MySQL 8 it is 64MB. 2MiB (`2097152`) is a good starting value.

For the S3/Minio backend, this value must be set to at least 5MiB (`5242880`) due to the
[constraints of S3's multipart
uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

Using a smaller value for `carver_block_size` will lead to more HTTP requests during the carving
process, resulting in longer carve times and higher load on the Fleet server. If the value is too
high, HTTP requests may run long enough to cause server timeouts.

#### Compression

Compression of the carve contents can be enabled with the `carver_compression` flag in osquery. When used, the carve results will be compressed with [Zstandard](https://facebook.github.io/zstd/) compression.

### Usage

File carves are initiated with osquery queries. Issue a query to the `carves` table, providing `carve = 1` along with the desired path(s) as constraints.

For example, to extract the `/etc/hosts` file on a host with hostname `mac-workstation`:

```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path = "/etc/hosts"'
```

The standard osquery file globbing syntax is also supported to carve entire directories or more:

```
fleetctl query --hosts mac-workstation --query 'SELECT * FROM carves WHERE carve = 1 AND path LIKE "/etc/%%"'
```

#### Retrieving carves

List the non-expired (see below) carves with `fleetctl get carves`. Note that carves will not be available through this command until osquery checks in to the Fleet server with the first of the carve contents. This can take some time from initiation of the carve.

To also retrieve expired carves, use `fleetctl get carves --expired`.

Contents of carves are returned as .tar archives, and compressed if that option is configured.

To download the contents of a carve with ID 3, use

```
fleetctl get carve --outfile carve.tar 3
```

It can also be useful to pipe the results directly into the tar command for unarchiving:

```
fleetctl get carve --stdout 3 | tar -x
```

#### Expiration

Carve contents remain available for 24 hours after the first data is provided from the osquery client. After this time, the carve contents are cleaned from the database and the carve is marked as "expired".

The same is not true if S3 is used as the storage backend. In that scenario, it is suggested to setup a [bucket lifecycle configuration](https://docs.aws.amazon.com/AmazonS3/latest/dev/object-lifecycle-mgmt.html) to avoid retaining data in excess. Fleet, in an "eventual consistent" manner (i.e. by periodically performing comparisons), will keep the metadata relative to the files carves in sync with what it is actually available in the bucket.

### Alternative carving backends

#### Minio

Configure the following:
- `FLEET_S3_ENDPOINT_URL=minio_host:port`
- `FLEET_S3_BUCKET=minio_bucket_name`
- `FLEET_S3_SECRET_ACCESS_KEY=your_secret_access_key`
- `FLEET_S3_ACCESS_KEY_ID=acces_key_id`
- `FLEET_S3_FORCE_S3_PATH_STYLE=true`
- `FLEET_S3_REGION=minio` or any non-empty string otherwise Fleet will attempt to derive the region.

### Troubleshooting

#### Check carve status in osquery

Osquery can report on the status of carves through queries to the `carves` table.

The details provided by

```
fleetctl query --labels 'All Hosts' --query 'SELECT * FROM carves'
```

can be helpful to debug carving problems.

#### Ensure `carver_block_size` is set appropriately

`carver_block_size` is an osquery flag that sets the size of each part of a file carve that osquery
sends to the Fleet server.

When using the MySQL backend (default), this value must be less than the `max_allowed_packet`
setting in MySQL. If it is too large, MySQL will reject the writes.

When using S3, the value must be at least 5MiB (5242880 bytes), as smaller multipart upload
sizes are rejected. Additionally [S3
limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html) the maximum number of
parts to 10,000.

The value must be small enough that HTTP requests do not time out.

Start with a default of 2MiB for MySQL (2097152 bytes), and 5MiB for S3/Minio (5242880 bytes).

<meta name="pageOrderInSection" value="300">
