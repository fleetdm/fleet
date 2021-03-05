# Testing
- [Full test suite](#full-test-suite)
  - [Database tests](#database-tests)
  - [Email tests](#email-tests)
- [Integration tests](#integration-tests)
  - [Email](#email)

## Full test suite

To execute all of the tests that CI will execute, run the following from the root of the repository:

```
make test
```

It is a good idea to run `make test` before submitting a Pull Request.

#### Go unit tests

To run all Go unit tests, run the following:

```
make test-go
```

#### JavaScript unit tests

To run all JavaScript unit tests, run the following:

```
make test-js
```

#### Go linters

To run all Go linters and static analyzers, run the following:

```
make lint-go
```

### Database tests

To run database tests set environment variables as follows.

```
export MYSQL_PORT_3306_TCP_ADDR=192.168.99.100
export MYSQL_TEST=1
```

### Email tests

To run email related unit tests using MailHog set the following environment
variable.

```
export MAIL_TEST=1
```

## Integration tests

By default, tests that require external dependecies like Mysql or Redis are skipped. The tests can be enabled by setting `MYSQL_TEST=true` and `REDIS_TEST=true` environment variables. MYSQL will try to connect with the following credentials.
```
user        = "kolide"
password    = "kolide"
database    = "kolide"
host        = "127.0.0.1"
```
Redis tests expect a redis instance at `127.0.0.1:6379`.

#### JavaScript linters

To run all JavaScript linters and static analyzers, run the following:

```
make lint-js
```

#### Viewing test coverage

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

### Email

#### Testing email using MailHog

To intercept sent emails while running a Fleet development environment, make sure that you've set the SMTP address to `<docker host ip>:1025` and leave the username and password blank. Then, visit `<docker host ip>:8025` in a web browser to view the [MailHog](https://github.com/mailhog/MailHog) UI.

For example, if docker is running natively on your `localhost`, then your mail settings should look something like:

```yaml
mail:
  address: localhost:1025
```

`localhost:1025` is the default configuration. You can use `fleet config_dump` to see the values which Fleet is using given your configuration.
