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
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
```

Make sure it is available in your `PATH`. To execute the basic unit and integration tests, run the following from the root of the repository:

```sh
REDIS_TEST=1 MYSQL_TEST=1 make test
```

The integration tests in the `server/service` package can generate a lot of logs mixed with the test results output. To make it easier to identify a failing test in this package, you can set the `FLEET_INTEGRATION_TESTS_DISABLE_LOG=1` environment variable so that logging is disabled.

Note that on a Linux system, the Redis tests will include running in cluster mode, so the docker Redis Cluster setup must be running. This implies starting the docker dependencies as follows:

```sh
# start both the default docker-compose.yml and the redis cluster-specific
# docker-compose-redis-cluster.yml
$ docker-compose -f docker-compose.yml -f docker-compose-redis-cluster.yml up
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

Then, in the "SMTP options" section, enter any email address in the "Sender address" field, set the "SMTP server" to `localhost` on port `1025`, and set "Authentication type" to `None`. Note that you may use any active or inactive sender address.

Visit [localhost:8025](http://localhost:8025) to view MailHog's admin interface displaying all emails sent using the simulated mail server.

#### Mailpit SMTP server with plain authentication

Alternatively, if you need to test a SMTP server with plain basic authentication enabled, set:
- "SMTP server" to `localhost` on port `1026`
- "Authentication type" to `Plain`.
- "SMTP username" to `mailpit-username`.
- "SMTP password" to `mailpit-password`.
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

Configure SSO on the Organization Settings page with the following:

```
Identity Provider Name: SimpleSAML
Entity ID: https://localhost:8080
Metadata URL: http://localhost:9080/simplesaml/saml2/idp/metadata.php
```

The identity provider is configured with four users:

```
Username: sso_user
Email: sso_user@example.com
Password: user123#

Username: sso_user2
Email: sso_user2@example.com
Password: user123#

# sso_user_3_global_admin is automatically added as Global admin.
Username: sso_user_3_global_admin
Email: sso_user_3_global_admin@example.com
Password: user123#

# sso_user_4_team_maintainer is automatically added as maintainer of Team with ID = 1.
# If a team with ID 1 doesn't exist then the login with this user will fail.
Username: sso_user_4_team_maintainer
Email: sso_user_4_team_maintainer@example.com
Password: user123#
```

Use the Fleet UI to invite one of these users with the associated email. Be sure the "Enable single sign-on" box is checked for that user. Now, after accepting the invitation, you should be able to log in as that user by clicking "Sign on with SimpleSAML" on the login page.

To add additional users, modify [tools/saml/users.php](https://github.com/fleetdm/fleet/tree/main/tools/saml/users.php) and restart the `simplesaml` container.

<meta name="pageOrderInSection" value="200">

## Testing Kinesis Logging

Tip: Install [AwsLocal](https://github.com/localstack/awscli-local) to ease interaction with
[LocalStack](https://github.com/localstack/localstack). Alternatively, you can use the `aws` client
and use `--endpoint-url=http://localhost:4566` on all invocations.

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

## Testing pre-built installers

Pre-built installers are kept in a blob storage like AWS S3. As part of your your local development there's a [MinIO](https://min.io/) instance running on http://localhost:9000. To test the pre-built installers functionality locally:

1. Build the installers you want using `fleetctl package`. Be sure to include the `--insecure` flag
   for local testing.
2. Use the [installerstore](https://github.com/fleetdm/fleet/tree/97b4d1f3fb30f7b25991412c0b40327f93cb118c/tools/installerstore) tool to upload them to your MinIO instance.
3. Configure your fleet server setting `FLEET_PACKAGING_GLOBAL_ENROLL_SECRET` to match your global enroll secret.
4. Set `FLEET_SERVER_SANDBOX_ENABLED=1`, as the endpoint to retrieve the installer is only available in the sandbox.

```sh
FLEET_SERVER_SANDBOX_ENABLED=1 FLEET_PACKAGING_GLOBAL_ENROLL_SECRET=xyz  ./build/fleet serve --dev
```

Be sure to replace the `FLEET_PACKAGING_GLOBAL_ENROLL_SECRET` value above with the global enroll
secret from the `fleetctl package` command used to build the installers.

MinIO also offers a web interface at http://localhost:9001. Credentials are `minio` / `minio123!`.

## Telemetry

You can configure the server to record and report trace data using OpenTelemetry or Elastic APM and use a tracing system like [Jaeger](https://www.jaegertracing.io/) to consume this data and inspect the traces locally.

Please refer to [tools/telemetry](https://github.com/fleetdm/fleet/tree/main/tools/telemetry/README.md) for instructions.

## Fleetd Chrome extension

### Debugging the service Worker

View service worker logs in chrome://serviceworker-internals/?devtools (in production), or in chrome://extensions (only during development).

## MDM setup and testing

To run your local server with the MDM features enabled, you need to get certificates and keys.

- [ABM setup](#abm-setup)
- [APNs and SCEP setup](#apns-and-scep-setup)
- [Running the server](#running-the-server)
- [Testing MDM](#testing-mdm)

### ABM setup

To enable the [DEP](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#dep-device-enrollment-program) enrollment flow, the Fleet server needs three things:

1. A private key.
1. A certificate.
1. An encrypted token generated by Apple.

#### Private key, certificate, and encrypted token

First ask @lukeheath to create an account for you in [ABM](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#abm-apple-business-manager). You'll need an account to generate an encrypted token.

Once you have access to ABM, follow [these guided instructions](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#apple-business-manager-abm) in the user facing docs to generate the private key, certificate, and encrypted token.

### APNs and SCEP setup

The server also needs a private key + certificate to identify with Apple's [APNs](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#apns-apple-push-notification-service) servers, and another for [SCEP](https://github.com/fleetdm/fleet/blob/main/tools/mdm/apple/glossary-and-protocols.md#scep-simple-certificate-enrollment-protocol).

To generate both, follow [these guided instructions](https://fleetdm.com/docs/using-fleet/mdm-macos-setup#apple-push-notification-service-apns).

Note that:

1. Fleet must be running to generate the certificates and keys.
2. You must be logged in to Fleet as a global admin. See [Building Fleet](./Building-Fleet.md) for details on getting Fleet setup locally.
3. To login into https://identity.apple.com/pushcert you can use your ABM account generated in the previous step.
4. Save all the certificates and keys in a safe place.

Internally, the certificates are generated using this flow. Note that the fleet sails API base url can be changed using the `TEST_FLEETDM_API_URL` environment variable.

```mermaid
sequenceDiagram
    participant user as user email
    participant fleetctl as fleetctl
    participant server as fleet server
    participant fleetdm as fleetdm.com sails app
    participant apple as identity.apple.com
    link apple: PushCert @ https://identity.apple.com/pushcert

    note over fleetctl: fleetctl login
    fleetctl->>+server: login
    server-->>-fleetctl: token
    note over fleetctl: fleetctl generate mdm_apple
    fleetctl->>+server: generate certificates
    server->>server: generate self-signed SCEP cert & key
    server->>server: generate APNs key
    server->>server: generate APNs CSR
    server-)+fleetdm: request vendor signature on APNs CSR
    server-->>-fleetctl: SCEP cert, SCEP key, APNs key
    note over fleetdm: calls /ee/tools/mdm/cert
    fleetdm--)-user: vendor-signed APNs CSR
    user->>+apple: vendor-signed APNs CSR
    note right of apple: managed through web ui
    apple-->>-user: Apple-signed APNs certificate
```

Another option, if for some reason, generating the certificates and keys fails or you don't have a supported email address handy is to use `openssl` to generate your SCEP key pair:

```sh
$ openssl genrsa -out fleet-mdm-apple-scep.key 4096

$ openssl req -x509 -new -nodes -key fleet-mdm-apple-scep.key -sha256 -days 1826 -out fleet-mdm-apple-scep.crt -subj '/CN=Fleet Root CA/C=US/O=Fleet DM.'
```

### Running the server

Try to store all the certificates and tokens you generated in the earlier steps together in a safe place outside of the repo, then start the server with:

```sh
FLEET_MDM_APPLE_SCEP_CHALLENGE=scepchallenge \
FLEET_MDM_APPLE_SCEP_CERT=/path/to/fleet-mdm-apple-scep.crt \
FLEET_MDM_APPLE_SCEP_KEY=/path/to/fleet-mdm-apple-scep.key \
FLEET_MDM_APPLE_BM_SERVER_TOKEN=/path/to/dep_encrypted_token.p7m \
FLEET_MDM_APPLE_BM_CERT=/path/to/fleet-apple-mdm-bm-public-key.crt \
FLEET_MDM_APPLE_BM_KEY=/path/to/fleet-apple-mdm-bm-private.key \
FLEET_MDM_APPLE_APNS_CERT=/path/to/mdmcert.download.push.pem \
FLEET_MDM_APPLE_APNS_KEY=/path/to/mdmcert.download.push.key \
 ./build/fleet serve --dev --dev_license --logging_debug
```

Note: if you need to enroll VMs using MDM, the server needs to run behind TLS with a valid certificate. In a separate terminal window/tab, create a local tunnel to your server using `ngrok` (`brew install ngrok/ngrok/ngrok` if you don't have it.)

```sh
ngrok http https://localhost:8080
```

> NOTE: If this is your first time using ngrok this command will fail and you will see a message
> about signing up. Open the sign up link and complete the sign up flow. You can rerun the same command
> and ngrok should work this time. After this open the forwarding link, you will be asked to confirm that you'd like
> to be forwarded to your local server and should accept.

Don't forget to edit your Fleet server settings (through the UI or `fleetctl`) to use the URL `ngrok` provides to you. You need to do this whenever you restart `ngrok`.

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

A bootstrap package is a `pkg` file that gets automatically installed on hosts when they enroll via DEP.

The `pkg` file needs to be a signed "distribution package", you can find a dummy file that meets all the requirements [in Drive](https://drive.google.com/file/d/1adwAOTD5G6D4WzWvJeMId6mDhyeFy-lm/view). We have instructions in [the docs](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#bootstrap-package) to upload a new bootstrap package to your Fleet instance.

The dummy package linked above adds a Fleet logo in `/Library/FleetDM/fleet-logo.png`. To verify if the package was installed, you can open that folder and verify that the logo is there.

### Puppet module

Instructions to develop and test the module can be found in the [`CONTRIBUTING.md` file](https://github.com/fleetdm/fleet/blob/main/ee/tools/puppet/fleetdm/CONTRIBUTING.md) that sits alongside the module code.

### Testing the end user flow for MDM migrations

The [end user flow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide#end-user-workflow) requires you to have a webserver running to receive a webhook from the Fleet server and perform an unenrollment.

We have a few servers in `tools/mdm/migration` that you can use. Follow the instructions on their README and configure your Fleet server to point to them.

<meta name="pageOrderInSection" value="1500">
<meta name="description" value="An overview of Fleet's full test suite and integration tests.">
