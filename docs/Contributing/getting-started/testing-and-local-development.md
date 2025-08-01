# Testing and local development

- [Testing and local development](#testing-and-local-development)
  - [License key](#license-key)
  - [Simulated hosts](#simulated-hosts)
  - [Test suite](#test-suite)
    - [Go unit tests](#go-unit-tests)
    - [Go linters](#go-linters)
    - [Javascript unit and integration tests](#javascript-unit-and-integration-tests)
    - [Javascript linters](#javascript-linters)
    - [MySQL tests](#mysql-tests)
    - [Email tests](#email-tests)
    - [Network tests](#network-tests)
    - [Viewing test coverage](#viewing-test-coverage)
  - [End-to-end tests](#end-to-end-tests)
    - [Preparation](#preparation)
    - [Run tests](#run-tests)
    - [Interactive](#interactive)
    - [Command line](#command-line)
  - [Test hosts](#test-hosts)
  - [Email](#email)
    - [Manually testing email with MailHog and Mailpit](#manually-testing-email-with-mailhog-and-mailpit)
      - [MailHog SMTP server without authentication](#mailhog-smtp-server-without-authentication)
      - [Mailpit SMTP server with plain authentication](#mailpit-smtp-server-with-plain-authentication)
  - [Development database management](#development-database-management)
  - [MySQL shell](#mysql-shell)
  - [Redis REPL](#redis-repl)
  - [Testing SSO](#testing-sso)
    - [Configuration](#configuration)
  - [Testing Kinesis Logging](#testing-kinesis-logging)
  - [Testing pre-built installers](#testing-pre-built-installers)
  - [Telemetry](#telemetry)
  - [Fleetd Chrome extension](#fleetd-chrome-extension)
  - [fleetd-base installers](#fleetd-base-installers)
  - [MDM setup and testing](#mdm-setup-and-testing)
    - [ABM setup](#abm-setup)
      - [Private key, certificate, and encrypted token](#private-key-certificate-and-encrypted-token)
    - [APNs and SCEP setup](#apns-and-scep-setup)
    - [Running the server](#running-the-server)
    - [Testing MDM](#testing-mdm)
      - [Testing manual enrollment](#testing-manual-enrollment)
      - [Testing DEP enrollment](#testing-dep-enrollment)
        - [Gating the DEP profile behind SSO](#gating-the-dep-profile-behind-sso)
    - [Nudge](#nudge)
      - [Debugging tips](#debugging-tips)
    - [Bootstrap package](#bootstrap-package)
    - [Puppet module](#puppet-module)
    - [Testing the end user flow for MDM migrations](#testing-the-end-user-flow-for-mdm-migrations)
  - [Software packages](#software-packages)
    - [Troubleshooting installation](#troubleshooting-installation)

## License key

Do you need to test Fleet Premium features locally?

Use the `--dev_license` flag to use the default development license key.

For example:

```sh
./build/fleet serve --dev --dev_license
```

## Simulated hosts

It can be helpful to quickly populate the UI with simulated hosts when developing or testing features that require host information.

Check out [`/tools/osquery` directory instructions](https://github.com/fleetdm/fleet/tree/main/tools/osquery) for starting up simulated hosts in your development environment.

## Test suite

You must install the [`golangci-lint`](https://golangci-lint.run/) command to run `make test[-go]` or `make lint[-go]`, using:

```sh
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

Make sure it is available in your `PATH`. To execute the basic unit and integration tests, run the following from the root of the repository:

```sh
REDIS_TEST=1 MYSQL_TEST=1 make test
```

The integration tests in the `server/service` package can generate a lot of logs mixed with the test results output. To make it easier to identify a failing test in this package, you can set the `FLEET_INTEGRATION_TESTS_DISABLE_LOG=1` environment variable so that logging is disabled.

The MDM integration tests are run with a random selection of software installer storage backends (local filesystem or S3/minio), and similar for the bootstrap packages storage (DB or S3/minio). You can force usage of the S3 backend by setting `FLEET_INTEGRATION_TESTS_SOFTWARE_INSTALLER_STORE=s3`. Note that `MINIO_STORAGE_TEST=1` must also be set for the S3 backend to be used.

When the S3 backend is used, this line will be printed in the tests' output (as this could be relevant to understand and debug the test failure):

```
    integration_mdm_test.go:196: >>> using S3/minio software installer store
```

Note that on a Linux and macOS systems, the Redis tests will include running in cluster mode, so the docker Redis Cluster setup must be running. This implies starting the docker dependencies as follows:

```sh
# start both the default docker-compose.yml and the redis cluster-specific
# docker-compose-redis-cluster.yml
$ docker-compose -f docker-compose.yml -f docker-compose-redis-cluster.yml up
```

### Redis cluster on macOS

Redis cluster mode can also be run on macOS, but requires [Docker Mac Net Connect](https://github.com/chipmk/docker-mac-net-connect) to give the local development environment access to the docker network. Run the following commands to setup the docker VPN bridge:

```sh
# Install via Homebrew
$ brew install chipmk/tap/docker-mac-net-connect

# Run the service and register it to launch at boot
$ sudo brew services start chipmk/tap/docker-mac-net-connect
```

### Go unit tests

To run all Go unit tests, run the following:

```bash
REDIS_TEST=1 MYSQL_TEST=1 MINIO_STORAGE_TEST=1 SAML_IDP_TEST=1 NETWORK_TEST=1 make test-go
```

### Go linters

To run all Go linters and static analyzers, run the following:

```sh
make lint-go
```

### Javascript unit and integration tests

To run all JS unit tests, run the following:

```sh
make test-js
```

or

```sh
yarn test
```

### Javascript linters

To run all JS linters and static analyzers, run the following:

```sh
make lint-js
```

or

```sh
yarn lint
```

### MySQL tests

To run MySQL integration tests, set environment variables as follows:

```sh
MYSQL_TEST=1 make test-go
```

### Email tests

To run email related integration tests using MailHog set environment as follows:

```sh
MAIL_TEST=1 make test-go
```

### Network tests

A few tests require network access as they make requests to external hosts. Given that the network is unreliable and may not be available. Those hosts may also be unavailable so these tests are skipped by default. They are opt-in via the `NETWORK_TEST` environment variable. To run them:

```sh
NETWORK_TEST=1 make test-go
```

### Viewing test coverage

When you run `make test` or `make test-go` from the root of the repository, a test coverage report is generated at the root of the repo in a filed named `coverage.txt`

To explore a test coverage report on a line-by-line basis in the browser, run the following:

```bash
go tool cover -html=coverage.txt
```

To view test a test coverage report in a terminal, run the following:

```bash
go tool cover -func=coverage.txt
```

## End-to-end tests

We have partnered with [QA Wolf](https://www.qawolf.com/) to help manage and maintain our E2E testing suite.
The code is deployed and tested once daily on the testing instance.

QA Wolf manages any issues found from these tests and will raise github issues. Engineers should not
have to worry about working with E2E testing code or raising issues themselves.

However, development may necessitate running E2E tests on demand. To run E2E tests live on a branch such as the `main` branch, developers can navigate to [Deploy Cloud Environments](https://github.com/fleetdm/confidential/actions/workflows/cloud-deploy.yml) in our [/confidential](https://github.com/fleetdm/confidential) repo's Actions and select "Run workflow".

For Fleet employees, if you would like access to the QA Wolf platform you can reach out in the [#help-engineering](https://fleetdm.slack.com/archives/C019WG4GH0A) slack channel.

### Preparation

Make sure dependencies are up to date and to build the [Fleet binaries locally](https://fleetdm.com/docs/contributing/building-fleet).

For Fleet Free tests:

```sh
make e2e-reset-db
make e2e-serve-free
```

For Fleet Premium tests:

```sh
make e2e-reset-db
make e2e-serve-premium
```

This will start a local Fleet server connected to the E2E database. Leave this server running for the duration of end-to-end testing.

```sh
make e2e-setup
```

This will initialize the E2E instance with a user.

### Run tests

Tests can be run in interactive mode or from the command line.

### Interactive

For Fleet Free tests:

```sh
yarn e2e-browser:free
```

For Fleet Premium tests:

```sh
yarn e2e-browser:premium
```

Use the graphical UI controls to run and view tests.

### Command line

For Fleet Free tests:

```sh
yarn e2e-cli:free
```

For Fleet Premium tests:

```sh
yarn e2e-cli:premium
```

Tests will run automatically, and results are reported to the shell.

## Test hosts

The Fleet repo includes tools to start testing osquery hosts. Please see the documentation in [/tools/osquery](https://github.com/fleetdm/fleet/tree/main/tools/osquery) for more information.

## Email

### Manually testing email with MailHog and Mailpit

#### MailHog SMTP server without authentication

To intercept sent emails while running a Fleet development environment, first, as an Admin in the Fleet UI, navigate to the Organization settings.

Then, in the "SMTP options" section, set:
- "Sender address" to any email address. Note that you may use any active or inactive sender address.
- "SMTP server" to `localhost` on port `1025`.
- "Use SSL/TLS to connect (recommended)" to unchecked. 
- "Authentication type" to `None`.

Visit [localhost:8025](http://localhost:8025) to view MailHog's admin interface displaying all emails sent using the simulated mail server.

#### Mailpit SMTP server with plain authentication

Alternatively, if you need to test a SMTP server with plain basic authentication enabled, set:
- "SMTP server" to `localhost` on port `1026`
- "Use SSL/TLS to connect (recommended)" to unchecked.
- "Authentication type" to `Username and password`
- "SMTP username" to `mailpit-username`
- "SMTP password" to `mailpit-password`
- "Auth method" to `Plain`
- Note that you may use any active or inactive sender address.

Visit [localhost:8026](http://localhost:8026) to view Mailpit's admin interface displaying all emails sent using the simulated mail server.

## Development database management

In the course of development (particularly when crafting database migrations), it may be useful to
backup, restore, and reset the MySQL database. This can be achieved with the following commands:

Backup:

```sh
make db-backup
```

The database dump is stored in `backup.sql.gz`.

Restore:

```sh
make db-restore
```

Note that a "restore" will replace the state of the development database with the state from the backup.

Reset:

```sh
make db-reset
```


## MySQL shell

Connect to the MySQL shell to view and interact directly with the contents of the development database.

To connect via Docker:

```sh
docker-compose exec mysql mysql -uroot -ptoor -Dfleet
```

## Redis REPL

Connect to the `redis-cli` in REPL mode to view and interact directly with the contents stored in Redis.

```sh
docker-compose exec redis redis-cli
```

## Testing SSO

Fleet's `docker-compose` file includes a SAML identity provider (IdP) for testing SAML-based SSO locally.

### Configuration

Configure SSO on the **Integration settings** page with the following:
```
Identity Provider Name: SimpleSAML
Entity ID: https://localhost:8080
Metadata URL: http://127.0.0.1:9080/simplesaml/saml2/idp/metadata.php
```

The identity provider is configured with these users:
```
Username: sso_user
Email: sso_user@example.com
Password: user123#
Display name: SSO User 1

Username: sso_user2
Email: sso_user2@example.com
Password: user123#
Display name: SSO User 2

# sso_user_3_global_admin is automatically added as Global admin.
Username: sso_user_3_global_admin
Email: sso_user_3_global_admin@example.com
Password: user123#
Display name: SSO User 3

# sso_user_4_team_maintainer is automatically added as maintainer of Team with ID = 1.
# If a team with ID 1 doesn't exist then the login with this user will fail.
Username: sso_user_4_team_maintainer
Email: sso_user_4_team_maintainer@example.com
Password: user123#
Display name: SSO User 4

Username: sso_user_5_team_admin
Email: sso_user_5_team_admin@example.com
Password: user123#
Display name: SSO User 5

Username: sso_user_6_global_observer
Email: sso_user_6_global_observer@example.com
Password: user123#
Display name: SSO User 6

Username: sso_user_no_displayname
Email: sso_user_no_displayname@example.com
Password: user123#
```

Use the Fleet UI to invite one of these users with the associated email. Be sure the "Enable single sign-on" box is checked for that user. Now, after accepting the invitation, you should be able to log in as that user by clicking "Sign on with SimpleSAML" on the login page.

To add additional users, modify [tools/saml/users.php](https://github.com/fleetdm/fleet/tree/main/tools/saml/users.php) and restart the `simplesaml` container.

### Testing IdP initiated login

To test the "IdP initiated flow" with SimpleSAML you can visit the following URL on your browser:
http://127.0.0.1:9080/simplesaml/saml2/idp/SSOService.php?spentityid=sso.test.com
After login, SimpleSAML should redirect the user to Fleet.

<meta name="pageOrderInSection" value="200">

## Testing Kinesis logging

Install the `aws` client: `brew install aws-cli`
Set the following alias to ease interaction with [LocalStack](https://github.com/localstack/localstack):
```sh
awslocal='AWS_ACCESS_KEY_ID=default AWS_SECRET_ACCESS_KEY=default AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566'
```

The following guide assumes you have server dependencies running:
```sh
docker-compose up
#
# (Starts LocalStack with kinesis enabled.)
#
```

First, create one stream for "status" logs and one for "result" logs (see
https://osquery.readthedocs.io/en/stable/deployment/logging/ for more information around the two
types of logs):

```sh
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
Use the following to describe the sample_status stream. 

Note: Check the `StreamARN` value to see your region. If your region is not `us-east-1` you will need to update the command for running `fleet serve` below to match your region. 
```sh
$ awslocal kinesis describe-stream --stream-name sample_status
{
    "StreamDescription": {
        "Shards": [
            {
                "ShardId": "shardId-000000000000",
                "HashKeyRange": {
                    "StartingHashKey": "0",
                    "EndingHashKey": "340282366920938463463374607431768211455"
                },
                "SequenceNumberRange": {
                    "StartingSequenceNumber": "49663516104709326340269823046323233838660186951542374402"
                }
            }
        ],
        "StreamARN": "arn:aws:kinesis:us-east-1:000000000000:stream/sample_status",
        "StreamName": "sample_status",
        "StreamStatus": "ACTIVE",
        "RetentionPeriodHours": 24,
        "EnhancedMonitoring": [
            {
                "ShardLevelMetrics": []
            }
        ],
        "EncryptionType": "NONE",
        "KeyId": null,
        "StreamCreationTimestamp": 1747864229.031
    }
}
```

Use the following configuration to run Fleet:
```sh
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
```sh
make fleet && FLEET_OSQUERY_RESULT_LOG_PLUGIN=kinesis FLEET_OSQUERY_STATUS_LOG_PLUGIN=kinesis FLEET_KINESIS_REGION=us-east-1 FLEET_KINESIS_ENDPOINT_URL=http://localhost:4566 FLEET_KINESIS_ACCESS_KEY_ID=default FLEET_KINESIS_SECRET_ACCESS_KEY=default FLEET_KINESIS_STATUS_STREAM=sample_status FLEET_KINESIS_RESULT_STREAM=sample_result ./build/fleet serve --dev --dev_license --logging_debug
```
Fleet will now be relaying "status" and "result" logs from osquery agents to the LocalStack's
kinesis.

Let's work on inspecting "status" logs received by Kinesis ("status" logs are easier to verify, to generate "result" logs so you need to configure "schedule queries").

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

The `Data` field is base64 encoded. You can use the following command to decode:
```
echo eyJob3N0SWRlbnRpZmllciI6Ijg3OGE2ZWRmLTcxMzEtNGUyOC05NWEyLWQzNDQ5MDVjYWNhYiIsImNhbGVuZGFyVGltZSI6IldlZCBNYXIgIDIgMjI6MDI6NTQgMjAyMiBVVEMiLCJ1bml4VGltZSI6IjE2NDYyNTg1NzQiLCJzZXZlcml0eSI6IjAiLCJmaWxlbmFtZSI6Imdsb2dfbG9nZ2VyLmNwcCIsImxpbmUiOiI0OSIsIm1lc3NhZ2UiOiJDb3VsZCBub3QgZ2V0IFJQTSBoZWFkZXIgZmxhZy4iLCJ2ZXJzaW9uIjoiNC45LjAiLCJkZWNvcmF0aW9ucyI6eyJob3N0X3V1aWQiOiJlYjM5NDZiMi0wMDAwLTAwMDAtYjg4OC0yNTkxYTFiNjY2ZTkiLCJob3N0bmFtZSI6ImUwMDg4ZDI4YTYzZiJ9fQo= | base64 -d
{"hostIdentifier":"878a6edf-7131-4e28-95a2-d344905cacab","calendarTime":"Wed Mar  2 22:02:54 2022 UTC","unixTime":"1646258574","severity":"0","filename":"glog_logger.cpp","line":"49","message":"Could not get RPM header flag.","version":"4.9.0","decorations":{"host_uuid":"eb3946b2-0000-0000-b888-2591a1b666e9","hostname":"e0088d28a63f"}}
```

## Testing Firehose logging

We will configure Fleet to send result and status logs to Firehose directly which will in turn stream them to S3 (`Fleet -> LocalStack Firehose -> LocalStack S3`).

Install the `aws` client: `brew install aws-cli`
Set the following alias to ease interaction with [LocalStack](https://github.com/localstack/localstack):
```sh
awslocal='AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test AWS_DEFAULT_REGION=us-east-1 aws --endpoint-url=http://localhost:4566'
```

We need to create a S3 bucket in LocalStack and make it "publicly" available (so that we can inspect it in the browser)
```sh
awslocal s3 mb s3://s3-firehose --region us-east-1
awslocal s3api put-bucket-acl --bucket s3-firehose --acl public-read
```
Check `http://localhost:4566/s3-firehose` in your browser.

Create the following `iam_policy.json` file and apply it to create a "super-role":
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1572416334166",
      "Action": "*",
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```
```sh
awslocal iam create-role --role-name super-role --assume-role-policy-document file://$(pwd)/iam_policy.json
```

After applying it, grab the "Arn" in the output (e.g. `"arn:aws:iam::000000000000:role/super-role"`)

Create the following `firehose_skeleton_result.json` file to create the delivery stream for "result" logs:
```json
{
  "DeliveryStreamName": "s3-stream-result",
  "DeliveryStreamType": "DirectPut",
  "S3DestinationConfiguration": {
    "RoleARN": "arn:aws:iam::000000000000:role/super-role",
    "BucketARN": "arn:aws:s3:::s3-firehose",
    "Prefix": "result",
    "ErrorOutputPrefix": "result-error",
    "BufferingHints": {
      "SizeInMBs": 1,
      "IntervalInSeconds": 60
    },
    "CompressionFormat": "UNCOMPRESSED",
    "CloudWatchLoggingOptions": {
      "Enabled": false,
      "LogGroupName": "",
      "LogStreamName": ""
    }
  },
  "Tags": [
    {
      "Key": "tagKey",
      "Value": "tagValue"
    }
  ]
}
```
```sh
awslocal firehose create-delivery-stream --cli-input-json file://$(pwd)/firehose_skeleton_result.json
```

Similarly, create a `firehose_skeleton_status.json` file to create the delivery stream for "status" logs:
```json
{
  "DeliveryStreamName": "s3-stream-status",
  "DeliveryStreamType": "DirectPut",
  "S3DestinationConfiguration": {
    "RoleARN": "arn:aws:iam::000000000000:role/super-role",
    "BucketARN": "arn:aws:s3:::s3-firehose",
    "Prefix": "status",
    "ErrorOutputPrefix": "status-error",
    "BufferingHints": {
      "SizeInMBs": 1,
      "IntervalInSeconds": 60
    },
    "CompressionFormat": "UNCOMPRESSED",
    "CloudWatchLoggingOptions": {
      "Enabled": false,
      "LogGroupName": "",
      "LogStreamName": ""
    }
  },
  "Tags": [
    {
      "Key": "tagKey",
      "Value": "tagValue"
    }
  ]
}
```
```sh
awslocal firehose create-delivery-stream --cli-input-json file://$(pwd)/firehose_skeleton_status.json
```

After applying such configuration, "result" logs will be stored under the `results/` prefix and "status" logs will be stored under `status/` prefix (both on the `s3-firehose` bucket).

Finally, here's the Fleet configuration:
```sh
FLEET_OSQUERY_RESULT_LOG_PLUGIN=firehose
FLEET_OSQUERY_STATUS_LOG_PLUGIN=firehose
FLEET_FIREHOSE_REGION=us-east-1
FLEET_FIREHOSE_ENDPOINT_URL=http://localhost:4566
FLEET_FIREHOSE_ACCESS_KEY_ID=default
FLEET_FIREHOSE_SECRET_ACCESS_KEY=default
FLEET_FIREHOSE_STS_ASSUME_ROLE_ARN=arn:aws:iam::000000000000:role/super-role
FLEET_FIREHOSE_RESULT_STREAM=s3-stream-result
FLEET_FIREHOSE_STATUS_STREAM=s3-stream-status
```

You can inspect logs by visiting `http://localhost:4566/s3-firehose` on your browser.

## Telemetry

You can configure the server to record and report trace data using OpenTelemetry or Elastic APM and use a tracing system like [Jaeger](https://www.jaegertracing.io/) to consume this data and inspect the traces locally.

Please refer to [tools/telemetry](https://github.com/fleetdm/fleet/tree/main/tools/telemetry/README.md) for instructions.

## Fleetd Chrome extension

### Debugging the service Worker

View service worker logs in chrome://serviceworker-internals/?devtools (in production), or in chrome://extensions (only during development).

## fleetd-base installers

"fleetd-base" installers are pre-built `pkg` and `msi` installers that do not contain hardcoded `--fleet-url` and `--enroll-secret` values.

Anyone can build a base installer, but Fleet provides a public repository of signed base installers at:

- [Production Usage](https://download.fleetdm.com)
- [Development Usage](https://download-testing.fleetdm.com)

The workflow that builds and releases the installers is defined in `.github/workflows/release-fleetd-base.yml`.

The base installers are used:

- By Fleet MDM to automatically install `fleetd` when a host enables MDM features.
- By customers deploying `fleetd` using third-party tools (e.g., Puppet or Chef).

The Fleet server uses the production server by default, but you can change this during development
using the development flag `FLEET_DEV_DOWNLOAD_FLEETDM_URL`.


### Building your own non-signed fleetd-base installer

Due to historical reasons, each type of installer has its own peculiarities:

- `pkg` installers require an extra `--use-system-configuration` flag.
- `pkg` installers read configuration values from a configuration profile.
- `msi` installers need dummy configuration values.
- `msi` installers read configuration values at installation time.

```sh
# Build a fleetd-base.pkg installer
$ fleetctl package --type=pkg --use-system-configuration

# Build a fleetd-base.msi installer, using dummy values to avoid errors
$ fleetctl package --type=msi --fleet-url=dummy --enroll-secret=dummy

# Install a fleetd-base.msi installer
$ msiexec /i fleetd-base.msi FLEET_URL="<target_url>" FLEET_SECRET="<secret_to_use>"
```

**Note:** a non-signed base installer _cannot_ be installed on a macOS host during the ADE MDM enrollment
flow. Apple requires that applications installed via an `InstallEnterpriseApplication` command be
signed with a development certificate.

### Building and serving your own signed fleetd-base.pkg installer for macOS

Only signed fleetd installers can be used during the ADE MDM enrollment flow. If you are
developing/testing logic that needs to run during that flow, you will need to build and serve a
signed fleetd-base.pkg installer.

You will also need to serve the manifest for the fleetd-base installer. This manifest is used as
part of the `InstallEnterpriseApplication` command that installs fleetd; it contains a checksum of
the fleetd-base installer file, as well as the URL at which the MDM protocol can download the actual
installer file.

#### Pre-requisites

- An ngrok URL for serving the `fleetd-base.pkg` installer and the manifest `.plist` file

#### Building a signed fleetd-base installer from `edge`

We have a [GitHub workflow](../../.github/workflows/build-fleetd-base-pkg.yml) that can build a signed
fleetd-base installer using fleetd components from any of the release channels we support. You'll
most likely use `edge` since we release fleetd components built from an RC branch to `edge` for
QA before an official release.

To use the workflow, follow these steps:

1. Trigger the build and codesign fleetd-base.pkg workflow at https://github.com/fleetdm/fleet/actions/workflows/build-fleetd-base-pkg.yml.
2. Click the run workflow drop down and fill in `"edge"` for the first 3 fields. Fill in the ngrok URL
  from the "Pre-requisites" above in the last field.
3. Click the Run workflow button. This will generate two files:
  - `fleet-base-manifest.plist`
  - `fleet-base.pkg`
4. Download them to your workstation.

#### Building a signed fleetd-base installer from `local TUF` and signing with Apple Developer Account

1. Build fleetd base pkg installer from your [local TUF](https://github.com/fleetdm/fleet/blob/HEAD/docs/Contributing/Run-Locally-Built-Fleetd.md) service by running the following command after the local TUF repository is generated `fleetctl package --type=pkg --update-roots=$(fleetctl updates roots --path ./test_tuf) --disable-open-folder --debug --update-url=$LOCAL_TUF_URL --enable-scripts --use-system-configuration`.
2. Obtain a `Developer ID Installer Certificate`:
- Sign in to your Apple Developer account.
- Navigate to "Certificates, IDs, & Profiles".
- Click on "Certificates" and then click the "+" button to create a new certificate.
- Select "Developer ID Installer" and follow the prompts to create and download the certificate.
- Install the downloaded certificate to your keychain.
- Locate the certificate in your Keychain and confirm everything looks correct. Run this command to confirm you see it listed `security find-identity -v`
  - If the security  command does not show your newly added certificate you may need to install the `Developer ID - G2 (Expiring 09/17/2031 00:00:00 UTC)` certificate from [Apple PKI](https://www.apple.com/certificateauthority/). 
3. Sign your pkg with the `productsign` command replacing the placeholders with your actual values:

`productsign --sign "Developer ID Installer: Your Apple Account Name (serial number)" <path_to_unpacked_files> <path_to_signed_package.pkg>`

Example: `productsign --sign "Developer ID Installer: PezHub (5F863R529J)" fleet-osquery.pkg signed-fleetd.pkg`

4. Check the signature by running `pkgutil --check-signature signed-fleetd.pkg`
5. Rename your signed pkg `mv signed-fleetd.pkg fleet-base.pkg`
6. Create the manifest:
- Get the SHA-256 checksum of your pkg `shasum -a 256 path/to/your.pkg`
- Create a .plist with your SHA-256 hash and the URL where you plan to host the fleet pkg and save it as `fleetd-base-manifest.plist`

Example:
```
<plist version="1.0">
  <dict>
    <key>items</key>
    <array>
      <dict>
        <key>assets</key>
        <array>
          <dict>
            <key>kind</key>
            <string>software-package</string>
            <key>sha256-size</key>
            <integer>32</integer>
            <key>sha256s</key>
            <array>
              <string>1234abcd56789</string>
            </array>
            <key>url</key>
            <string>https://tuf.pezhub.ngrok.app/fleet-base.pkg</string>
          </dict>
        </array>
      </dict>
    </array>
  </dict>
</plist>
```

7. Serve the `fleet-base.pkg` and `fleetd-base-manifest.plist`


#### Serving the signed fleetd-base.pkg installer

1. Create a directory named `fleetd-base-dir` and a subdirectory named `stable`. Tip: we have the `$FLEET_REPO_ROOT_DIR/tmp`
   directory gitignored, so that's a convenient place to create the directories:
```sh
# From the Fleet repo root dir
mkdir -p ./tmp/fleetd-base-dir/stable
```
2. Move `fleet-base.pkg` to `./tmp/fleetd-base-dir`.
3. Move `fleet-base-manifest.plist` to `./tmp/fleetd-base-dir/stable`.
4. Start up an HTTP file server from the Fleet repo root directory using the [`tools/file-server`](../../tools/file-server/README.md) tool: `go run ./tools/file-server 8085 ./tmp/fleetd-base-dir`
5. Start your second ngrok tunnel and forward to http://localhost:8085.
	- Example: `ngrok http --domain=more.pezhub.ngrok.app http://localhost:8085`
6. Start your fleet server with `FLEET_DEV_DOWNLOAD_FLEETDM_URL` to point to the ngrok URL.
	- Example: `FLEET_DEV_DOWNLOAD_FLEETDM_URL="https://more.pezhub.ngrok.app"`
7. Enroll your mac with ADE. Tip: You can watch ngrok traffic via the inspect web interface url to ensure the two hosted packages are in the correct place and successfully reached by the host.

### Building and serving your own fleetd-base.msi installer for Windows

Unlike the ADE Enrollment flow Autopilot does not require a signed installer so you may build a
signed MSI containing `edge` components or, depending on your needs, build one locally from code on
a branch you're working on. Step 1 Option A below describes the former and B describes the latter

You will also need to serve the `meta.json` for the fleetd-base.msi installer, creation of which is
described below.

For Autopilot, Azure requires the Fleet server instance to have a proper domain name with some TXT/MX records added (see `/settings/integrations/automatic-enrollment/windows` on your Fleet instance).
For that reason, currently the only way to test this flow is to use Dogfood or the QA fleet server,
which already have this configured, or to configure an alternate server for this workflow.

#### Pre-requisites

- The URL of your Fleet server
- An ngrok tunnel URL pointed at your local TUF server. In the examples below this is
  https://tuf.fleetdm-example.ngrok.app and the TUF server is running on http://localhost:8081
- An ngrok tunnel URL for serving the `fleetd-base.msi` installer and a properly formatted
  `meta.json` file under the `stable/` path. In the examples below this is
  https://installers.fleetdm-example.ngrok.app and the installers are served from http://localhost:8085
- Perform a deployment with `FLEET_DEV_DOWNLOAD_FLEETDM_URL` set to the "installers" ngrok URL.

#### Step 1 Option A: Building a signed fleetd-base.msi installer from `edge`

If you want to use a signed installer, we have a [GitHub workflow](../../.github/workflows/build-fleetd-base-msi.yml)
that can build a signed fleetd-base installer using fleetd components from any of the
releasechannels we support. You'll most likely use `edge` since we release fleetd components
built from an RC branch to `edge` for QA before an official release.

To use the workflow, follow these steps:

1. Trigger the build and codesign fleetd-base.msi workflow at https://github.com/fleetdm/fleet/actions/workflows/build-fleetd-base-msi.yml.
2. Click the run workflow drop down and fill in `"edge"` for the first 3 fields. Fill in the ngrok URL
  from the "Pre-requisites" above in the last field.
3. Click the Run workflow button. This will generate two files:
  - `meta.json`
  - `fleetd-base.msi`
4. Download them to your workstation.

#### Step 1 Option B: Building a fleetd-base.msi installer from local components

If you have changes in a branch you want to test you can build an installer locally and serve it.

1. Use the ./tools/tuf/test/main.sh script to build an MSI from your branch:
```sh
#!/bin/bash
SYSTEMS="windows" \
MSI_FLEET_URL=https://[your fleet server's name] \
MSI_TUF_URL=https://tuf.fleetdm-example.ngrok.app \
GENERATE_MSI=1 \
ENROLL_SECRET=[enroll secret] \
FLEET_DESKTOP=1 \
TUF_PORT=8081 \
DEBUG=1 \
./tools/tuf/test/main.sh
```
2. Create a meta.json with the following contents:
```
{
  "fleetd_base_msi_url": "[your ngrok "installers" URL]/stable/fleetd-base.msi",
  "fleetd_base_msi_sha256": "[sha256sum of]"
}
```
3. Rename the generated fleet-osquery.msi to fleetd-base.msi

#### Serving the fleetd-base.msi installer

1. Create a directory named `fleetd-base-dir` and a subdirectory named `stable`. Tip: we have the `$FLEET_REPO_ROOT_DIR/tmp`
   directory gitignored, so that's a convenient place to create the directories:
```sh
# From the Fleet repo root dir
mkdir -p ./tmp/fleetd-base-dir/stable
```
2. Move `fleetd-base.msi` to `./tmp/fleetd-base-dir/stable`.
3. Move `meta.json` to `./tmp/fleetd-base-dir/stable`.
4. Start up an HTTP file server from the Fleet repo root directory using the [`tools/file-server`](../../tools/file-server/README.md) tool: `go run ./tools/file-server 8085 ./tmp/fleetd-base-dir`
5. Start your "installers" ngrok tunnel and forward to http://localhost:8085.
	- Example: `ngrok http --domain=installers.fleetdm-example.ngrok.app http://localhost:8085`
6. Perform a Fleet deployment(to Dogfood, QA or your own instance) with
   `FLEET_DEV_DOWNLOAD_FLEETDM_URL` set to the "installers" ngrok URL (if using Terraform, the environment variable is set on
   `infrastructure/dogfood/terraform/aws-tf-module/main.tf`).
	- Example: `FLEET_DEV_DOWNLOAD_FLEETDM_URL="https://installers.fleetdm-example.ngrok.app"`
7. Enroll your Windows device with Autopilot. Tip: You can watch ngrok traffic via the inspect web interface url to ensure the two hosted packages are in the correct place and successfully reached by the host.

## MDM setup and testing

To run your local server with the MDM features enabled, you need to get certificates and keys.

- [ABM setup](#abm-setup)
- [APNs and SCEP setup](#apns-and-scep-setup)
- [Running the server](#running-the-server)
- [Testing MDM](#testing-mdm)

### ABM setup

To enable the [DEP](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#dep-device-enrollment-program) enrollment flow, the Fleet server needs an encrypted token generated by Apple.

First ask @lukeheath to create an account for you in [ABM](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#abm-apple-business-manager). You'll need an account to generate an encrypted token.

Once you have access to ABM, follow [these guided instructions](https://fleetdm.com/docs/using-fleet/mdm-setup#apple-business-manager-abm) to get and upload the encrypted token.

### APNs and SCEP setup

The server also needs a certificate to identify with Apple's [APNs](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#apns-apple-push-notification-service) servers.

To get a certificate and upload it, [these guided instructions](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#apple-push-notification-service-apns).

Note that:

1. Fleet must be running to generate the token and certificate.
2. You must be logged in to Fleet as a global admin. See [Building Fleet](./Building-Fleet.md) for details on getting Fleet setup locally.
3. To login into https://identity.apple.com/pushcert you can use your ABM account generated in the previous step.
4. Save the token and certificate in a safe place.

### Testing MDM

To test MDM, you'll need one or more virtual machines (VMs) that you can use to enroll to your server.

Choose and download a VM software, some options:

- VMware Fusion: https://www.vmware.com/products/fusion.html
- UTM: https://mac.getutm.app/
- QEMU, for Linux, using instructions and scripts from the following repo: https://github.com/notAperson535/OneClick-macOS-Simple-KVM

If you need a license please use your Brex card (and submit the receipt on Brex.)

With the software in place, you need to create a VM and install macOS, the steps to do this vary depending on your software of choice.


If you are using VMWare, we've used [this guide](https://travellingtechguy.blog/vmware-dep/) in the
past, please reach out in [#g-mdm](https://fleetdm.slack.com/archives/C03C41L5YEL) before starting
so you can get the right serial numbers.

If you are using UTM, you can simply click "Create a New Virtual Machine" button with the default
settings. This creates a VM running the latest macOS.

If you are using QEMU for Linux, follow the instruction guide to install a recent macOS version: https://oneclick-macos-simple-kvm.notaperson535.is-a.dev/docs/start-here. Note that only the manual enrollment was successfully tested with this setup. Once the macOS VM is installed and up and running, the rest of the steps are the same.

#### Testing manual enrollment

1. Create a fleetd package that you will install on your host machine. You can get this command from the fleet
   UI on the manage hosts page when you click the `add hosts` button. Alternatively, you can run the command:

  ```sh
  ./build/fleetctl package --type=pkg --fleet-desktop --fleet-url=<url-of-fleet-instance> --enroll-secret=<your-fleet-enroll-secret>
  ```

2. Install this package on the host. This will add fleet desktop to this machine and from there you
   can go to the My Device page and see a banner at the top of the UI to enroll in Fleet MDM.

#### Testing DEP enrollment

> NOTE: Currently this is not possible for M1 Mac machines.

1. In ABM, look for the computer with the serial number that matches the one your VM has, click on it and click on "Edit MDM Server" to assign that computer to your MDM server.

2. Boot the machine, it should automatically enroll into MDM.

##### Gating the DEP profile behind SSO

For rapid iteration during local development, you can use the same configuration values as those described in [Testing SSO](#testing-sso), and test the flow in the browser by navigating to `https://localhost:8080/mdm/sso`.

To fully test e2e during DEP enrollment however, you need:

- A local tunnel to your Fleet server (instructions to set your tunnel are in the [running the server](#running-the-server) section)
- A local tunnel to your local IdP server (or, optionally create an account in a cloud IdP like Okta)

With an accessible Fleet server and IdP server, you can configure your env:

- If you're going to use the SimpleSAML server that is automatically started in local development, edit [./tools/saml/config.php](https://github.com/fleetdm/fleet/blob/6cfef3d3478f02227677071fe3a62bada77c1139/tools/saml/config.php) and replace `https://localhost:8080` everywhere with the URL of your local tunnel.
- After saving the file, restart the SimpleSAML service (eg: `docker-compose restart saml_idp`)
- Finally, edit your app configuration:

```yaml
mdm:
  end_user_authentication:
    entity_id: <your_fleet_tunnel_url>
    idp_name: SimpleSAML
    metadata_url: <your_idp_tunnel_url>/simplesaml/saml2/idp/metadata.php
```

> Note: if you're using a cloud provider, fill in the details provided by them for the app config settings above.

The next time you go through the DEP flow, you should be prompted to authenticate before enrolling.

### Nudge

We use [Nudge](https://github.com/macadmins/nudge) to enforce macOS updates. Our integration is tightly managed by Fleetd:

1. When Orbit pings the server for a config (every 30 seconds,) we send the corresponding Nudge configuration for the host. Orbit then saves this config at `<ORBIT_ROOT_DIR>/nudge-config.json`
2. If Orbit gets a Nudge config, it downloads Nudge from TUF.
3. Periodically, Orbit runs `open` to start Nudge, this is a direct replacement of Nudge's [LaunchAgent](https://github.com/macadmins/nudge/wiki#scheduling-nudge-to-run).

#### Debugging tips

- Orbit launches Nudge using the following command, you can try and run the command yourself to see if you spot anything suspicious:

```sh
open /opt/orbit/bin/nudge/macos/stable/Nudge.app --args -json-url file:///opt/orbit/nudge-config.json
```

- Make sure that the `fleet-osquery.pkg` package you build to install `fleetd` has the `--debug` flag, there are many Nudge logs at the debug level.

- Nudge has a great [guide](https://github.com/macadmins/nudge/wiki/Logging) to stream/parse their logs, the TL;DR version is that you probably want a terminal running:

```sh
log stream --predicate 'subsystem == "com.github.macadmins.Nudge"' --info --style json --debug
```

- Nudge has a couple of flags that you can provide to see what config values are actually being used. You can try launching Nudge with `-print-json-config` or `-print-profile-config` like this:

```sh
open /opt/orbit/bin/nudge/macos/stable/Nudge.app --args -json-url file:///opt/orbit/nudge-config.json -print-json-config
```

### Bootstrap package

A bootstrap package is a `pkg` file that gets automatically installed on hosts when they enroll via ABM/DEP.

The `pkg` file needs to be a signed "distribution package", you can find a dummy file that meets all the requirements [in Drive](https://drive.google.com/file/d/1adwAOTD5G6D4WzWvJeMId6mDhyeFy-lm/view). We have instructions in [the docs](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) to upload a new bootstrap package to your Fleet instance.

The dummy package linked above adds a Fleet logo in `/Library/FleetDM/fleet-logo.png`. To verify if the package was installed, you can open that folder and verify that the logo is there.

### Puppet module

> The Puppet module is deprecated as of Fleet 4.66. It is maintained for backwards compatibility.

Instructions to develop and test the module can be found in the [`CONTRIBUTING.md` file](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/CONTRIBUTING.md) that sits alongside the module code.

### Testing the end user flow for MDM migrations

The [end user flow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow) requires you to have a webserver running to receive a webhook from the Fleet server and perform an unenrollment.

We have a few servers in `tools/mdm/migration` that you can use. Follow the instructions on their README and configure your Fleet server to point to them.

<meta name="pageOrderInSection" value="1500">
<meta name="description" value="An overview of Fleet's full test suite and integration tests.">

## Software packages

### Troubleshooting installation

- macOS: `sudo grep "runner=installer" /var/log/orbit/orbit.stderr.log`.
- Ubuntu: `sudo grep "runner=installer" /var/log/syslog` (or using `journalctl` if `syslog` is not available).
- Fedora: `sudo grep "runner=installer" /var/log/messages` (or using `journalctl` if `syslog` is not available).
- Windows: `grep "runner=installer" C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log`
