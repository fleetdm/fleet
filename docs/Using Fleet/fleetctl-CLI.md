# fleetctl CLI

Fleetctl (pronounced "Fleet control") is a CLI tool for managing Fleet from the command line. Fleetctl enables a GitOps workflow with Fleet and osquery. With fleetctl, you can manage configurations, queries, generate osquery installers, etc.

Fleetctl also provides a quick way to work with all the data exposed by Fleet without having to use the Fleet UI or work directly with the Fleet API.

## Using fleetctl

To install the latest version of `fleetctl` run `npm install -g fleetctl` or download the binary from [GitHub](https://github.com/fleetdm/fleet/releases).

You can use `fleetctl` to accomplish many tasks you would typically need to do through the Fleet UI. You can even set up or apply configuration files to the Fleet server.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/ERbknt6w8eg" allowfullscreen></iframe>
</div>

### Available commands

Much of the functionality available in the Fleet UI is also available in `fleetctl`. You can run queries, add and remove users, generate agent (fleetd) installers to add new hosts, get information about existing hosts, and more!

To see the commands you can run with fleetctl, run the `fleetctl --help` command.

### Get more info about a command

Each command available to `fleetctl` has a help menu with additional information. To pull up the help menu, run `fleetctl <command> --help`, replacing `<command>` with the command you're looking up:

```
> fleetctl setup --help
```

You will see more info about the command, including the usage and information about any additional commands and options (or 'flags') that can be passed with it:

```
NAME:
   fleetctl setup - Set up a Fleet instance

USAGE:
   fleetctl setup [options]

OPTIONS:
   --email value     Email of the admin user to create (required) [$EMAIL]
   --name value      Name or nickname of the admin user to create (required) [$NAME]
   --password value  Password for the admin user (recommended to use interactive entry) [$PASSWORD]
   --org-name value  Name of the organization (required) [$ORG_NAME]
   --config value    Path to the fleetctl config file (default: "/Users/ksatter/.fleet/config") [$CONFIG]
   --context value   Name of fleetctl config context to use (default: "default") [$CONTEXT]
   --debug           Enable debug http request logging (default: false) [$DEBUG]
   --help, -h        show help (default: false)

```

## Setting up Fleet

This section walks through setting up and configuring Fleet via the CLI. If you already have a running Fleet instance, skip ahead to [Logging in to an existing Fleet instance](#logging-in-to-an-existing-fleet-instance) to configure the `fleetctl` CLI.

This guide illustrates:

- A minimal CLI workflow for managing an osquery fleet
- The set of API interactions that are required if you want to perform remote, automated management of a Fleet instance

### Running Fleet

For the sake of this tutorial, we will be using the local development Docker Compose infrastructure to run Fleet locally. This is documented in some detail in the [developer documentation](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Building-Fleet.md#development-infrastructure), but the following are the minimal set of commands that you can run from the root of the repository (assuming that you have a working Go/JavaScript toolchain installed along with Docker Compose):

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
fleetctl query --query 'SELECT * FROM osquery_info;' --labels='All Hosts' > results.json
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
      "host_display_name": "marpaia",
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

The `fleetctl get <fleet-entity-here> > <configuration-file-name-here>.yml` command allows you retrieve the current configuration and create a new file for specified Fleet entity (queries, hosts, etc.)

### Fleetctl apply

The `fleetctl apply -f <configuration-file-name-here>.yml` allows you to apply the current configuration in the specified file. 

When a new configuration is applied, agent options are validated. If any errors are found, you will receive an error message describing the issue and the new configuration will not be applied. You can also verify that your agent options are valid without applying using the `--dry-run` flag. Validation is based on the latest version of osquery. If you don't use the latest version of osquery, you can override validation using the `--force` flag. This will update agent options even if they are invalid.

Check out the [configuration files](https://fleetdm.com/docs/using-fleet/configuration-files) section of the documentation for example yaml files.

## Using fleetctl with an API-only user

When running automated workflows using the Fleet API, we recommend an API-only user's API key rather than the API key of a regular user. A regular user's API key expires frequently for security purposes, requiring routine updates. Meanwhile, an API-only user's key does not expire.
An API-only user does not have access to the Fleet UI. Instead, it's only purpose is to interact with the API programmatically or from fleetctl.

### Create an API-only user

To create your new API-only user, run `fleetctl user create` and pass values for `--name`, `--email`, and `--password`, and include the `--api-only` flag:

```
fleetctl user create --name "API User" --email api@example.com --password temp!pass --api-only
```

### Creating an API-only user
An API-only user can be given the same permissions as a regular user. The default access level is `Observer`. For more information on permissions, see the [user permissions documentation](https://fleetdm.com/docs/using-fleet/permissions#user-permissions).

If you'd like your API-only user to have a different access level than the default `Observer` role, you can specify what level of access the new user should have using the `--global-role` flag:

```
fleetctl user create --name "API User" --email api@example.com --password temp#pass --api-only --global-role admin
```

On Fleet Premium, use the `--team` flag setting `team_id:role` to create an API-only user on a team:

```
fleetctl user create --name "API Team Maintainer User" --email apimaintainer@example.com --password temp#pass --team 4:maintainer
```

Assigning the [GitOps role](https://fleetdm.com/docs/using-fleet/permissions#gitops) to a user is also completed using this method because GitOps is an API-only role.  

### Changing permissions of an API-only user

To change roles of a current user, log into the Fleet UI as an admin and navigate to **Settings > Users**.

> Suggestion: To disable/enable a user's access to the UI (converting a regular user to an API-only user or vice versa), create a new user.

### Use fleetctl as an API-only user

To use fleetctl with an API-only user, you will need to log in with `fleetctl login`. Once done, you'll be able to perform tasks using `fleetctl` as your new API-only user.

> If you are using a version of Fleet older than `4.13.0`, you will need to [reset the API-only user's password](https://github.com/fleetdm/fleet/blob/a1eba3d5b945cb3339004dd1181526c137dc901c/docs/Using-Fleet/fleetctl-CLI.md#reset-the-password) before running queries.

### Get the API token of an API-only user
To get the API key of an API-only user, you need to call the Login API with the credentials supplied during user creation.

For example, say the credentials provided were `api@example.com` for the email and `foobar12345` for the password. You may call the [Log in API](https://fleetdm.com/docs/using-fleet/rest-api#log-in) like so:

```sh
curl --location --request POST 'https://myfleetdomain.com/api/v1/fleet/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "email": "api@example.com",
    "password": "foobar12345"
}'
```

The [Log in API](https://fleetdm.com/docs/using-fleet/rest-api#log-in) will return a response similar to the one below with the API token included that will not expire.

```json
{
    "user": {
        "id": 82,
        "name": "API User",
        "email": "api@example.com",
        "global_role": "observer",
        "api_only": true
    },
    "available_teams": [],
    "token": "foo_token"
}
```

### Switching users

To use `fleetctl` with your regular user account but occasionally use your API-only user for specific cases, you can set up your `fleetctl` config with a new `context` to hold the credentials of your API-only user:

```
fleetctl config set --address https://dogfood.fleetdm.com --context api
[+] Context "api" not found, creating it with default values
[+] Set the address config key to "https://dogfood.fleetdm.com" in the "api" context
```

From there on, you can use  the `--context api` flag whenever you need to use the API-only user's identity, rather than logging in and out to switch accounts:

```
fleetctl login --context admin
Log in using the admin Fleet credentials.
Email: admin@example.com
Password:
[+] Fleet login successful and context configured!
```

Running a command with no context will use the default profile.

## MDM commands

With fleetctl, you can run MDM commands to take some action on your macOS hosts, like restart the host, remotely. Learn how [here](./MDM-commands.md). 

## File carving

Fleet supports osquery's file carving functionality as of Fleet 3.3.0. This allows the Fleet server to request files (and sets of files) from osquery agents, returning the full contents to Fleet.

File carving data can be either stored in Fleet's database or to an external S3 bucket. For information on how to configure the latter, consult the [configuration docs](https://fleetdm.com/docs/deploying/configuration#s-3-file-carving-backend).

### Configuration

Given a working flagfile for connecting osquery agents to Fleet, add the following flags to enable carving:

```
--disable_carver=false
--carver_disable_function=false
--carver_start_endpoint=/api/v1/osquery/carve/begin
--carver_continue_endpoint=/api/v1/osquery/carve/block
--carver_block_size=8000000
```

The default flagfile provided in the "Add New Host" dialog also includes this configuration.

#### Carver block size

The `carver_block_size` flag should be configured in osquery.

For the (default) MySQL Backend, the configured value must be less than the value of
`max_allowed_packet` in the MySQL connection, allowing for some overhead. The default for [MySQL 5.7](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_max_allowed_packet)
is 4MB and for [MySQL 8](https://dev.mysql.com/doc/refman/8.0/en/server-system-variables.html#sysvar_max_allowed_packet) it is 64MB.

For the S3/Minio backend, this value must be set to at least 5MiB (`5242880`) due to the
[constraints of S3's multipart
uploads](https://docs.aws.amazon.com/AmazonS3/latest/dev/qfacts.html).

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
sizes are rejected. Additionally, [S3
limits](https://docs.aws.amazon.com/AmazonS3/latest/userguide/qfacts.html) the maximum number of
parts to 10,000.

The value must be small enough that HTTP requests do not time out.

Start with a default of 2MiB for MySQL (2097152 bytes), and 5MiB for S3/Minio (5242880 bytes).

## Debugging Fleet

`fleetctl` provides debugging capabilities about the running Fleet server via the `debug` command. To see a complete list of all the options run:

```
fleetctl debug --help
```

To generate a full debugging archive, run:

```
fleetctl debug archive
```

This will generate a `tar.gz` file with:

- `prof` archives that can be inspected via `go tools pprof <archive_name_here>`.
- A file containing a set of all the errors that happened in the server during the interval of time defined by the [logging_error_retention_period](https://fleetdm.com/docs/deploying/configuration#logging-error-retention-period) configuration.
- Files containing database-specific information.

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Read about fleetctl, a CLI tool for managing Fleet and osquery configurations, running queries, generating installers, and more.">
<meta name="navSection" value="The basics">