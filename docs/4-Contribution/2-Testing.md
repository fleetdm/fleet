# Testing & Local Development
- [Test suite](#test-suite)
- [End-to-end tests](#end-to-end-tests)
- [Email](#email)
- [Database backup/restore](#database-backuprestore)
- [MySQL shell](#mysql-shell)
- [Testing SSO](#testing-sso)

## Test suite

To execute the basic unit tests, run the following from the root of the repository:

```
MYSQL_TEST=1 REDIS_TEST=1 make test
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

#### Redis tests

To run Redis integration tests set environment variables as follows:

```
REDIS_TEST=1 make test-go
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

```
make e2e-reset-db
make e2e-serve
```

This will start a local Fleet server connected to the E2E database. Leave this server running for the duration of end-to-end testing.

```
make e2e-setup
```

This will initialize the E2E instance with a user.

#### Run tests

Tests can be run in interactive mode, or from the command line.

#### Interactive

```
yarn cypress open
```

Use the graphical UI controls to run and view tests.

#### Command line

```
yarn cypress run
```

Tests will run automatically and results are reported to the shell.


## Email

#### Manually testing email with MailHog

To intercept sent emails while running a Fleet development environment, make sure that you've set the SMTP address to `localhost:1025` and leave the username and password blank. Then, visit http://localhost:8025 in a web browser to view the [MailHog](https://github.com/mailhog/MailHog) UI.

When Fleet sends emails, the contents of the messages are available in the MailHog UI.

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

The identity provider is configured with 2 users:

```
Username: user1
Email: user1@example.com
Password: user1pass
```

and

```
Username: user2
Email: user2@example.com
Password: user2pass
```

Use the Fleet UI to invite one of these users with the associated email. Be sure the "Enable Single Sign On" box is checked for that user. Now after accepting the invitation, you should be able to log in as that user by clicking "Sign On with SimpleSAML" on the login page.
