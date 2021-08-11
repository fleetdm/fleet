# Testing & Local Development

- [License key](#license-key)
- [Simulated hosts](#hosts)
- [Test suite](#test-suite)
- [End-to-end tests](#end-to-end-tests)
- [Test hosts](#test-hosts)
- [Email](#email)
- [Database backup/restore](#database-backuprestore)
- [Teams seed data](#teams-seed-data)
- [MySQL shell](#mysql-shell)
- [Testing SSO](#testing-sso)

## License key

Need to test Fleet Basic features locally?

Use the `--dev_license` flag to use the default development license key.

For example:

```
./build/fleet serve --dev --dev_license
```

## Simulated hosts

It can be helpful to quickly populate the UI with simulated hosts when developing or testing features that require host information.

Check out [the instructions in the `/tools/osquery` directory](../../tools/osquery/README.md#testing-with-containerized-osqueryd) for starting up simulated hosts in your development environment.

## Test suite

To execute the basic unit and integration tests, run the following from the root of the repository:

```
MYSQL_TEST=1 make test
```

It is a good idea to run `make test` before submitting a Pull Request.

#### Go unit tests

To run all Go unit tests, run the following:

```
make test-go
```

#### Go linters

To run all Go linters and static analyzers, run the following:

```
make lint-go
```

#### Javascript unit tests

To run all JS unit tests, run the following:

```
make test-js
```

or

```
yarn test
```

#### Javascript linters

To run all JS linters and static analyzers, run the following:

```
make lint-js
```

or

```
yarn lint
```

#### MySQL tests

To run MySQL integration tests set environment variables as follows:

```
MYSQL_TEST=1 make test-go
```

#### Email tests

To run email related integration tests using MailHog set environment as follows:

```
MAIL_TEST=1 make test-go
```

### Viewing test coverage

When you run `make test` or `make test-go` from the root of the repository, test coverage reports are generated in every subpackage. For example, the `server` subpackage will have a coverage report generated in `./server/server.cover`

To explore a test coverage report on a line-by-line basis in the browser, run the following:

```bash
# substitute ./datastore/datastore.cover, etc
go tool cover -html=./server/server.cover
```

To view test a test coverage report in a terminal, run the following:

```bash
# substitute ./datastore/datastore.cover, etc
go tool cover -func=./server/server.cover
```

## End-to-end tests

E2E tests are run using Docker and Cypress.

#### Preparation

Make sure dependencies are up to date and the [Fleet binaries are built locally](./1-Building-Fleet.md).

For Fleet Core tests:

```
make e2e-reset-db
make e2e-serve-core
```

For Fleet Basic tests:

```
make e2e-reset-db
make e2e-serve-basic
```

This will start a local Fleet server connected to the E2E database. Leave this server running for the duration of end-to-end testing.

```
make e2e-setup
```

This will initialize the E2E instance with a user.

#### Run tests

Tests can be run in interactive mode, or from the command line.

#### Interactive

For Fleet Core tests:

```
yarn e2e-browser:core
```

For Fleet Basic tests:

```
yarn e2e-browser:basic
```

Use the graphical UI controls to run and view tests.

#### Command line

For Fleet Core tests:

```
yarn e2e-cli:core
```

For Fleet Basic tests:

```
yarn e2e-cli:basic
```

Tests will run automatically and results are reported to the shell.

## Test hosts

The Fleet repo includes tools to start test osquery hosts. Please see the documentation in [/tools/osquery](../../tools/osquery) for more information.

## Email

#### Manually testing email with MailHog

To intercept sent emails while running a Fleet development environment, first, in the Fleet UI, navigate to the Organization settings page under Admin.

Then, in the "SMTP Options" section, enter any email address in the "Sender Address" field, set the "SMTP Server" to `localhost` on port `1025`, and set "Authentication Type" to `None`. Note that you may use any active or inactive sender address.

Visit [locahost:8025](http://localhost:8025) to view Mailhog's admin interface which will display all emails sent using the simulated mail server.

## Database Backup/Restore

In the course of development (particularly when crafting database migrations), it may be useful to backup and restore the MySQL database. This can be achieved with the following commands:

Backup:

```
./tools/backup_db/backup.sh
```

The database dump is stored in `backup.sql.gz`.

Restore:

```
./tools/backup_db/restore.sh
```

Note that a "restore" will replace the state of the development database with the state from the backup.

## Teams seed data

When developing on both the `master` and `teams` branches, it may be useful to create seed data that includes users and teams.

Check out this Loom demo video that walks through creating teams seed data:
https://www.loom.com/share/1c41a1540e8f41328a7a6cfc56ad0a01

For a text-based walkthrough, check out the following steps:

First, create a `env` file with the following contents:

```
export SERVER_URL=https://localhost:8080 # your fleet server url and port
export CURL_FLAGS='-k -s' # set insecure flag
export TOKEN=eyJhbGciOi... # your login token
```

Next, set the `FLEET_ENV_PATH` to point to the `env` file. This will let the scripts in the `fleet/` folder source the env file.

```
export FLEET_ENV_PATH=/Users/victor/fleet_env
```

Finally run one of the bash scripts located in the [/tools/api](../../tools/api/README.md) directory.

The `fleet/create_core` script will generate an environment to roughly reflect an installation of Fleet Core. The script creates 3 users with different roles.

```
./tools/api/fleet/teams/create_core
```

The `fleet/create_basic` script will generate an environment to roughly reflect an installation of Fleet Basic. The script will create 2 teams 4 users with different roles.

```
./tools/api/fleet/teams/create_basic
```

The `fleet/create_figma` script will generate an environment to reflect the mockups in the Fleet EE (current) Figma file. The script creates 3 teams and 12 users with different roles.

```
./tools/api/fleet/teams/create_figma
```

Each user generated by the script has their password set to `user123#`.

## MySQL shell

Connect to the MySQL shell to view and interact directly with the contents of the development database.

To connect via Docker:

```
docker-compose exec mysql mysql -uroot -ptoor -Dfleet
```

## Testing SSO

Fleet's `docker-compose` file includes a SAML identity provider (IdP) for testing SAML-based SSO locally.

### Configuration

Configure SSO on the Organization Settings page with the following:

```
Identity Provider Name: SimpleSAML
Entity ID: https://localhost:8080
Issuer URI: http://localhost:8080/simplesaml/saml2/idp/SSOService.php
Metadata URL: http://localhost:9080/simplesaml/saml2/idp/metadata.php
```

The identity provider is configured with one user:

```
Username: sso_user
Email: sso_user@example.com
Password: user123#
```

Use the Fleet UI to invite one of these users with the associated email. Be sure the "Enable Single Sign On" box is checked for that user. Now after accepting the invitation, you should be able to log in as that user by clicking "Sign On with SimpleSAML" on the login page.

To add additional users, modify [tools/saml/users.php](../../tools/saml/users.php) and restart the `simplesaml` container.
