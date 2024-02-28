# fleetctl CLI

fleetctl (pronounced "Fleet control") is a CLI tool for managing Fleet from the command line. fleetctl enables a GitOps workflow with Fleet.

fleetctl also provides a quick way to work with all the data exposed by Fleet without having to use the Fleet UI or work directly with the Fleet API.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/ERbknt6w8eg" allowfullscreen></iframe>
</div>

## Installing fleetctl

Install fleetctl with npm or download the binary from [GitHub](https://github.com/fleetdm/fleet/releases).

```sh
npm install -g fleetctl
```

### Upgrading fleetctl

The easiest way to update fleetctl is by running the installation command again.

```sh
npm install -g fleetctl@latest
```

## Usage


### Available commands


Much of the functionality available in the Fleet UI is also available in `fleetctl`. You can run queries, add and remove users, generate agent (fleetd) installers to add new hosts, get information about existing hosts, and more!

To see the available commands you can run:

```sh
> fleetctl --help
```

### Get more info about a command

Each command has a help menu with additional information. To pull up the help menu, run `fleetctl <command> --help`, replacing `<command>` with the command you're looking up:

```sh
> fleetctl setup --help
```

You will see more info about the command, including the usage and information about any additional commands and options (or 'flags'):

```sh
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

## Authentication

This section walks you through authentication, assuming you already have a running Fleet instance. To learn how to set up new Fleet instance, check out the [Deploy](https://fleetdm.com/docs/deploy/introduction) section or [Building Fleet locally](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/Building-Fleet.md) docs. 

### Login

To log in to your Fleet instance, run following commands:

1. Set the Fleet instance address

```sh
> fleetctl config set --address https://fleet.example.com
[+] Set the address config key to "https://fleet.example.com" in the "default" context
```

2. Log in with your credentials

```sh
> fleetctl login
Log in using the standard Fleet credentials.
Email: mike@arpaia.co
Password:
[+] Fleet login successful and context configured!
```

Once your local context is configured, you can use `fleetctl` normally.

### Log in with SAML (SSO) authentication

Users that authenticate to Fleet via SSO should retrieve their API token from the UI and set it manually in their `fleetctl` configuration (instead of logging in via `fleetctl login`).

**Fleet UI:**
1. Go to the **My account** page (https://fleet.example.com/profile)
2. Select the **Get API token** button to bring up a modal with the API token.
3. Set the API token in the `~/.fleet/config` file. 

```yaml
contexts:
  default:
    address: https://fleet.corp.example.com
    email: example@example.com
    token: your_token_here
```

The token can also be set with `fleetctl config set --token`, but this may leak the token into a user's shell history.

## Using fleetctl with an API-only user

When running automated workflows using the Fleet API, we recommend an API-only user's API key rather than the API key of a regular user. A regular user's API key expires frequently for security purposes, requiring routine updates. Meanwhile, an API-only user's key does not expire.   

An API-only user does not have access to the Fleet UI. Instead, it's only purpose is to interact with the API programmatically or from fleetctl.

### Create API-only user

To create your new API-only user, use `fleetctl user create`:

```sh
fleetctl user create --name "API User" --email api@example.com --password temp@pass123 --api-only
```


To use fleetctl with an API-only user, you will need to log in via `fleetctl`.  See [authentication](https://#authentication) above for details.

#### Permissions

An API-only user can be given the same permissions as a regular user. The default access level is **Observer**. You can specify what level of access the new user should have using the `--global-role` flag:

```sh
fleetctl user create --name "API User" --email api@example.com --password temp@pass123 --api-only --global-role admin
```

On Fleet Premium, use the `--team <team_id>:<role>` to create an API-only user on a team:

```sh
fleetctl user create --name "API User" --email api@example.com --password temp@pass123 --api-only --team 4: gitops
```

#### Changing permissions

To change roles of a current user, log into the Fleet UI as an admin and navigate to **Settings > Users**.
> Suggestion: To disable/enable a user's access to the UI (converting a regular user to an API-only user or vice versa), create a new user.

### Get API token for API-only user

To get the API key of an API-only user, you need to call the [login API](https://fleetdm.com/docs/rest-api/rest-api#log-in) with the credentials supplied during user creation.

```sh
curl --location --request POST 'https://fleet.example.com/api/v1/fleet/login' \
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

```sh
fleetctl config set --address https://dogfood.fleetdm.com --context api
[+] Context "api" not found, creating it with default values
[+] Set the address config key to "https://dogfood.fleetdm.com" in the "api" context
```

From there on, you can use  the `--context api` flag whenever you need to use the API-only user's identity, rather than logging in and out to switch accounts:

```sh
fleetctl login --context admin
Log in using the admin Fleet credentials.
Email: admin@example.com
Password:
[+] Fleet login successful and context configured!
```

Running a command with no context will use the default profile.

## Debugging Fleet

`fleetctl` provides debugging capabilities about the running Fleet server via the `debug` command. To see a complete list of all the options run:

```sh
fleetctl debug --help
```

To generate a full debugging archive, run:

```sh
fleetctl debug archive
```

This will generate a `tar.gz` file with:

- `prof` archives that can be inspected via `go tools pprof <archive_name_here>`.
- A file containing a set of all the errors that happened in the server during the interval of time defined by the [logging_error_retention_period](https://fleetdm.com/docs/deploying/configuration#logging-error-retention-period) configuration.
- Files containing database-specific information.

<meta name="pageOrderInSection" value="300">
<meta name="description" value="Read about fleetctl, a CLI tool for managing Fleet and osquery configurations, running queries, generating installers, and more.">
<meta name="navSection" value="The basics">
