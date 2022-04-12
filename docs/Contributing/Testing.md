# Testing & local development

- [License key](#license-key)
- [Simulated hosts](#hosts)
- [Test suite](#test-suite)
- [End-to-end tests](#end-to-end-tests)
- [Test hosts](#test-hosts)
- [Email](#email)
- [Database backup/restore](#database-backuprestore)
- [Seeding Data](./Seeding-Data.md)
- [MySQL shell](#mysql-shell)
- [Testing SSO](#testing-sso)
- [Testing Kinesis Logging](#testing-kinesis-logging)

## License key

Need to test Fleet Premium features locally?

Use the `--dev_license` flag to use the default development license key.

For example:

```
./build/fleet serve --dev --dev_license
```

## Simulated hosts

It can be helpful to quickly populate the UI with simulated hosts when developing or testing features that require host information.

Check out [the instructions in the `/tools/osquery` directory](https://github.com/fleetdm/fleet/tree/main/tools/osquery) for starting up simulated hosts in your development environment.

## Test suite

You must install the [`golangci-lint`](https://golangci-lint.run/) command to run `make test[-go]` or `make lint[-go]`, using:

```
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.0
```

Make sure it is available in your `PATH`. To execute the basic unit and integration tests, run the following from the root of the repository:

```
REDIS_TEST=1 MYSQL_TEST=1 make test
```

### Go unit tests

To run all Go unit tests, run the following:

```
REDIS_TEST=1 MYSQL_TEST=1 make test-go
```

### Go linters

To run all Go linters and static analyzers, run the following:

```
make lint-go
```

### Javascript unit tests

To run all JS unit tests, run the following:

```
make test-js
```

or

```
yarn test
```

### Javascript linters

To run all JS linters and static analyzers, run the following:

```
make lint-js
```

or

```
yarn lint
```

### MySQL tests

To run MySQL integration tests set environment variables as follows:

```
MYSQL_TEST=1 make test-go
```

### Email tests

To run email related integration tests using MailHog set environment as follows:

```
MAIL_TEST=1 make test-go
```

### Network tests

A few tests require network access as they make requests to external hosts. Given that the network is unreliable, may not be available, and those hosts may also not be unavailable, those tests are skipped by default and are opt-in via the `NETWORK_TEST` environment variable. To run them:

```
NETWORK_TEST=1 make test-go
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

E2E tests are constantly evolving and running them or examining CI results is the best way to understand what they cover, but at a high level they cover:
1. Setup
2. Login/out flows
3. Host page
    add hosts
    label flows
4. Queries flows
5. Policies flows
6. Schedule flows
    scheduling
    packs
6. Permissions
    Admin
    Observer (global and team)
    Maintainer
7. Organizational Settings
    Settings adjustments
    Users

### Preparation

Make sure dependencies are up to date and the [Fleet binaries are built locally](./Building-Fleet.md).

For Fleet Free tests:

```
make e2e-reset-db
make e2e-serve-free
```

For Fleet Premium tests:

```
make e2e-reset-db
make e2e-serve-premium
```

This will start a local Fleet server connected to the E2E database. Leave this server running for the duration of end-to-end testing.

```
make e2e-setup
```

This will initialize the E2E instance with a user.

### Run tests

Tests can be run in interactive mode, or from the command line.

### Interactive

For Fleet Free tests:

```
yarn e2e-browser:free
```

For Fleet Premium tests:

```
yarn e2e-browser:premium
```

Use the graphical UI controls to run and view tests.

### Command line

For Fleet Free tests:

```
yarn e2e-cli:free
```

For Fleet Premium tests:

```
yarn e2e-cli:premium
```

Tests will run automatically and results are reported to the shell.

## Test hosts

The Fleet repo includes tools to start test osquery hosts. Please see the documentation in [/tools/osquery](https://github.com/fleetdm/fleet/tree/main/tools/osquery) for more information.

## Email

### Manually testing email with MailHog

To intercept sent emails while running a Fleet development environment, first, in the Fleet UI, navigate to the Organization settings page under Admin.

Then, in the "SMTP options" section, enter any email address in the "Sender address" field, set the "SMTP server" to `localhost` on port `1025`, and set "Authentication type" to `None`. Note that you may use any active or inactive sender address.

Visit [localhost:8025](http://localhost:8025) to view Mailhog's admin interface which will display all emails sent using the simulated mail server.

## Development database management

In the course of development (particularly when crafting database migrations), it may be useful to
backup, restore, and reset the MySQL database. This can be achieved with the following commands:

Backup:

```
make db-backup
```

The database dump is stored in `backup.sql.gz`.

Restore:

```
make db-restore
```

Note that a "restore" will replace the state of the development database with the state from the backup.

Reset:

```
make db-reset
```


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

The identity provider is configured with two users:

```
Username: sso_user
Email: sso_user@example.com
Password: user123#

Username: sso_user2
Email: sso_user2@example.com
Password: user123#
```

Use the Fleet UI to invite one of these users with the associated email. Be sure the "Enable single sign on" box is checked for that user. Now after accepting the invitation, you should be able to log in as that user by clicking "Sign on with SimpleSAML" on the login page.

To add additional users, modify [tools/saml/users.php](https://github.com/fleetdm/fleet/tree/main/tools/saml/users.php) and restart the `simplesaml` container.

<meta name="pageOrderInSection" value="200">

## Testing Kinesis Logging

Tip: Install [awslocal](https://github.com/localstack/awscli-local) to ease interaction with
[localstack](https://github.com/localstack/localstack). Alternatively you can use the `aws` client
and use `--endpoint-url=http://localhost:4566` on all invocations.

The following guide assumes you have server dependencies running:
```sh
docker-compose up
#
# (Starts localstack with kinesis enabled.)
#
```

First, create one stream for "status" logs and one for "result" logs (see
https://osquery.readthedocs.io/en/stable/deployment/logging/ for more information around the two
types of logs):

```
$ awslocal kinesis create-stream --stream-name "sample_status" --shard-count 1
$ awslocal kinesis create-stream --stream-name "sample_result" --shard-count 1
$ awslocal kinesis list-streams
{
    "StreamNames": [
        "sample_result",
        "sample_status"
    ]
}
```

Use the following configuration to run fleet:
```
FLEET_OSQUERY_RESULT_LOG_PLUGIN=kinesis
FLEET_OSQUERY_STATUS_LOG_PLUGIN=kinesis
FLEET_KINESIS_REGION=us-east-1
FLEET_KINESIS_ENDPOINT_URL=http://localhost:4566
FLEET_KINESIS_ACCESS_KEY_ID=default
FLEET_KINESIS_SECRET_ACCESS_KEY=default
FLEET_KINESIS_STATUS_STREAM=sample_status
FLEET_KINESIS_RESULT_STREAM=sample_result 
```

Here's a sample command for running `fleet serve`:
```
make fleet && FLEET_OSQUERY_RESULT_LOG_PLUGIN=kinesis FLEET_OSQUERY_STATUS_LOG_PLUGIN=kinesis FLEET_KINESIS_REGION=us-east-1 FLEET_KINESIS_ENDPOINT_URL=http://localhost:4566 FLEET_KINESIS_ACCESS_KEY_ID=default FLEET_KINESIS_SECRET_ACCESS_KEY=default FLEET_KINESIS_STATUS_STREAM=sample_status FLEET_KINESIS_RESULT_STREAM=sample_result ./build/fleet serve --dev --dev_license --logging_debug
```
Fleet will now be relaying "status" and "result" logs from osquery agents to the localstack's
kinesis.

Let's work on inspecting "status" logs received by Kinesis ("status" logs are easier to verify, to generate "result" logs you need to configure "schedule queries").

Get "status" logging stream shard ID:
```
$ awslocal kinesis list-shards --stream-name sample_status

{
    "Shards": [
        {
            "ShardId": "shardId-000000000000",
            "HashKeyRange": {
                "StartingHashKey": "0",
                "EndingHashKey": "340282366920938463463374607431768211455"
            },
            "SequenceNumberRange": {
                "StartingSequenceNumber": "49627262640659126499334026974892685537161954570981605378"
            }
        }
    ]
}
```

Get the shard-iterator for the status logging stream:
```
awslocal kinesis get-shard-iterator --shard-id shardId-000000000000 --shard-iterator-type TRIM_HORIZON --stream-name sample_status

{
    "ShardIterator": "AAAAAAAAAAERtiUrWGI0sq99TtpKnmDu6haj/80llVpP80D4A5XSUBFqWqcUvlwWPsTAiGin/pDYt0qJ683PeuSFP0gkNISIkGZVcW3cLvTYtERGh2QYVv+TrAlCs6cMpNvPuW0LwILTJDFlwWXdkcRaFMjtFUwikuOmWX7N4hIJA+1VsTx4A0kHfcDxHkjYi1WDe+8VMfYau+fB1XTEJx9AerfxdTBm"
}
```

Finally, you can use the above `ShardIterator` to get "status" log records:
```
awslocal kinesis get-records --shard-iterator AAAAAAAAAAERtiUrWGI0sq99TtpKnmDu6haj/80llVpP80D4A5XSUBFqWqcUvlwWPsTAiGin/pDYt0qJ683PeuSFP0gkNISIkGZVcW3cLvTYtERGh2QYVv+TrAlCs6cMpNvPuW0LwILTJDFlwWXdkcRaFMjtFUwikuOmWX7N4hIJA+1VsTx4A0kHfcDxHkjYi1WDe+8VMfYau+fB1XTEJx9AerfxdTBm
[...]
        {
            "SequenceNumber": "49627262640659126499334026986980734807488684740304699394",
            "ApproximateArrivalTimestamp": "2022-03-02T19:45:54-03:00",
            "Data": "eyJob3N0SWRlbnRpZmllciI6Ijg3OGE2ZWRmLTcxMzEtNGUyOC05NWEyLWQzNDQ5MDVjYWNhYiIsImNhbGVuZGFyVGltZSI6IldlZCBNYXIgIDIgMjI6MDI6NTQgMjAyMiBVVEMiLCJ1bml4VGltZSI6IjE2NDYyNTg1NzQiLCJzZXZlcml0eSI6IjAiLCJmaWxlbmFtZSI6Imdsb2dfbG9nZ2VyLmNwcCIsImxpbmUiOiI0OSIsIm1lc3NhZ2UiOiJDb3VsZCBub3QgZ2V0IFJQTSBoZWFkZXIgZmxhZy4iLCJ2ZXJzaW9uIjoiNC45LjAiLCJkZWNvcmF0aW9ucyI6eyJob3N0X3V1aWQiOiJlYjM5NDZiMi0wMDAwLTAwMDAtYjg4OC0yNTkxYTFiNjY2ZTkiLCJob3N0bmFtZSI6ImUwMDg4ZDI4YTYzZiJ9fQo=",
            "PartitionKey": "149",
            "EncryptionType": "NONE"
        }
    ],
[...]
```