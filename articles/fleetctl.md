# fleetctl

fleetctl (pronounced "Fleet control") is a command line interface (CLI) tool for managing Fleet from the command line. fleetctl enables a GitOps workflow with Fleet.

fleetctl also provides a quick way to work with all the data exposed by Fleet without having to use the Fleet UI or work directly with the Fleet API.

<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/ERbknt6w8eg" allowfullscreen></iframe>
</div>

## Installing fleetctl

Download and install [Node.js](https://nodejs.org/en).

Install fleetctl with npm (included in Node.js).

```sh
sudo npm install -g fleetctl
```

Alternatively, and for Windows and Linux, you can download the fleectl binary from [GitHub](https://github.com/fleetdm/fleet/releases). 

Double-click the `tar.gz` or `zip` file to extract the binary. To run fleetctl commands, use the binary's path (`/path/to/fleetctl`). For convenience, copy or move the binary to a directory in your `$PATH` (ex: `/usr/local/bin`). This allows you to execute fleetctl without specifying its location.

> To generate `fleetd` packages to enroll hosts, you may need [additional dependencies](https://fleetdm.com/guides/enroll-hosts#cli), depending on both your operating system and the OS you're packaging `fleetd` for.

### Upgrading fleetctl

If you used npm to install fleetctl, fleetctl will update itself the next time you run it.

You can also install the latest version of the binary from [GitHub](https://github.com/fleetdm/fleet/releases).


## Usage


### Available commands


Much of the functionality available in the Fleet UI is also available in fleetctl. You can run queries, add and remove users, generate Fleet's agent (fleetd) to add new hosts, get information about existing hosts, and more!

> Note: Unless a logging infrastructure is configured on your Fleet server, osquery-related logs will be stored locally on each device. Read more [here](https://fleetdm.com/guides/log-destinations)

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

To log in to your Fleet instance, run the following commands:

1. Set the Fleet instance address

```sh
> fleetctl config set --address 'https://fleet.example.com'
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

Once your local context is configured, you can use fleetctl normally.

### Log in with SAML (SSO) authentication

Users that authenticate to Fleet via SSO should retrieve their API token from the UI and manually set it in their fleetctl configuration (instead of logging in via `fleetctl login`).

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

When running automated workflows using the Fleet API, we recommend using an API-only user's API key rather than a regular user's API key. A regular user's API key expires frequently for security purposes, requiring routine updates. Meanwhile, an API-only user's key does not expire.   

An API-only user does not have access to the Fleet UI. Instead, it's only purpose is to interact with the API programmatically or from fleetctl.

### Create API-only user

Before creating the API-only user, log in to fleetctl as an admin.  See [authentication](#authentication) above for details.

To create your new API-only user, use `fleetctl user create`:

```sh
fleetctl user create --name 'API User' --email 'api@example.com' --password 'temp@pass123' --api-only
```

You'll then receive an API token:

```sh
Success! The API token for your new user is: <TOKEN>
```

> If you need to retrieve this user's token again in the future, you can do so via the [log in API](https://fleetdm.com/docs/rest-api/rest-api#log-in).

#### Permissions

An API-only user can be given the same permissions as a regular user. The default access level is **Observer**. You can specify what level of access the new user should have using the `--global-role` flag:

```sh
fleetctl user create --name 'API User' --email 'api@example.com' --password 'temp@pass123' --api-only --global-role 'admin'
```

On Fleet Premium, use the `--team <team_id>:<role>` to create an API-only user on a team:

```sh
fleetctl user create --name 'API User' --email 'api@example.com' --password 'temp@pass123' --api-only --team 4: gitops
```

#### Changing permissions

To change the role of a current user, log into the Fleet UI as an admin and navigate to Settings > Users.
> Suggestion: Create a new user to disable/enable a user's access to the UI (converting a regular user to an API-only user or vice versa).

### Switching users

To use fleetctl with your regular user account but occasionally use your API-only user for specific cases, you can set up your fleetctl config with a new `context` to hold the credentials of your API-only user:

```sh
fleetctl config set --address 'https://dogfood.fleetdm.com' --context api
[+] Context "api" not found, creating it with default values
[+] Set the address config key to "https://dogfood.fleetdm.com" in the "api" context
```

From there on, you can use  the `--context api` flag whenever you need to use the API-only user's identity, rather than logging in and out to switch accounts:

```sh
fleetctl login --context 'admin'
Log in using the admin Fleet credentials.
Email: admin@example.com
Password:
[+] Fleet login successful and context configured!
```

Running a command with no context will use the default profile.

## Debugging Fleet

fleetctl provides debugging capabilities about the running Fleet server via the `debug` command. To see a complete list of all the options, run:

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

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-07-04">
<meta name="articleTitle" value="fleetctl">
<meta name="description" value="Read about fleetctl, a CLI tool for managing Fleet and osquery configurations, running queries, generating Fleet's agent (fleetd), and more.">
