# Kolide [![CircleCI](https://circleci.com/gh/kolide/kolide-ose.svg?style=svg&circle-token=2573c239b7f18967040d2dec95ca5f71cfc90693)](https://circleci.com/gh/kolide/kolide-ose)

### Contents

- [Development Environment](#development-environment)
  - [Installing build dependencies](#installing-build-dependencies)
  - [Building](#building)
    - [Generating the packaged JavaScript](#generating-the-packaged-javascript)
    - [Automatic rebuilding of the JavaScript bundle](#automatic-rebuilding-of-the-javascript-bundle)
    - [Compiling the Kolide binary](#compiling-the-kolide-binary)
    - [Managing Go dependencies with glide](#managing-go-dependencies-with-glide)
  - [Testing](#testing)
    - [Full test suite](#full-test-suite)
    - [Go unit tests](#go-unit-tests)
    - [JavaScript unit tests](#javascript-unit-tests)
    - [Go linters](#go-linters)
    - [JavaScript linters](#javascript-linters)
    - [Viewing test coverage](#viewing-test-coverage)
  - [Email](#email)
    - [Testing email using MailHog](#testing-email-using-mailhog)
    - [Viewing email content in the terminal](#viewing-email-content-in-the-terminal)
  - [Development Infrastructure](#development-infrastructure)
    - [Starting the local development environment](#starting-the-local-development-environment)
    - [Stopping the local development environment](#stopping-the-local-development-environment)
    - [Setting up the database tables](#setting-up-the-database-tables)
  - [Running Kolide](#running-kolide)
    - [Using Docker development infrastructure](#using-docker-development-infrastructure)
    - [Using no external dependencies](#using-no-external-dependencies)


## Development Environment

### Installing build dependencies

To setup a working local development environment, you must install the following
minimum toolset:

* [Go](https://golang.org/dl/) (1.7 or greater)
* [Node.js](https://nodejs.org/en/download/current/) (and npm)
* [GNU Make](https://www.gnu.org/software/make/)
* [Docker](https://www.docker.com/products/overview#/install_the_platform)


If you're using MacOS or Linux, you should have Make installed already. If you
are using Windows, you will need to install it separately. Additionally, if you
would only like to run an in-memory instances of Kolide (for demonstrations,
testing, etc.), then you do not need to install Docker.

Once you have those minimum requirements, you will need to install Kolide's
dependent libraries. To do this, run the following:

```
make deps
```

When pulling in new revisions to the Kolide codebase to your working source
tree, it may be necessary to re-run `make deps` if a new Go or JavaScript
dependency was added.

### Building

#### Generating the packaged JavaScript

To generate all necessary code (bundling JavaScript into Go, etc), run the
following:

```
make generate
```

#### Automatic rebuilding of the JavaScript bundle

Normally, `make generate` takes the JavaScript code, bundles it into a single
bundle via Webpack, and inlines that bundle into a generated Go source file so
that all of the frontend code can be statically compiled into the binary. When
you build the code after running `make generate`, all of that JavaScript is
included in the binary.

This makes deploying Kolide a dream, since you only have to worry about a single
static binary. If you are working on frontend code, it is likely that you don't
want to have to manually re-run `make generate` and `make build` every time you
edit JavaScript and CSS in order to see your changes in the browser. To solve
this problem, before you build the Kolide binary, run the following command
instead of `make generate`:

```
make generate-dev
```

Instead of reading the JavaScript from a inlined static bundle compiled within
the binary, `make generate-dev` will generate a Go source file which reads the
frontend code from disk and run Webpack in "watch mode".

Note that when you run `make generate-dev`, Webpack will be watching the
JavaScript files that were used to generate the bundle, so the process will be
long lived. Depending on your personal workflow, you might want to run this in a
background terminal window.

After you run `make generate-dev`, run `make build` to build the binary, launch
the binary and you'll be able to refresh the browser whenever you edit and save
frontend code.

#### Compiling the Kolide binary

Use `go build` to build the application code. For your convenience, a make
command is included which builds the code:

```
make build
```

It's not necessary to use Make to build the code, but using Make allows us to
account for cross-platform differences more effectively than the `go build` tool
when writing automated tooling. Use whichever you prefer.

#### Managing Go Dependencies with Glide

[Glide](https://github.com/Masterminds/glide#glide-vendor-package-management-for-golang)
is a package manager for third party Go libraries. See the ["How It Works"](https://github.com/Masterminds/glide#how-it-works)
section in the Glide README for full details.

##### Installing the correct versions of dependencies

To install the correct versions of third package libraries, use `glide install`.
`glide install` will  use the `glide.lock` file to pull vendored packages from
remote vcs.  `make deps` takes care of this step, as well as downloading the
latest version of glide for you.

##### Adding new dependencies

To add a new dependency, use [`glide get [package name]`](https://github.com/Masterminds/glide#glide-get-package-name)

##### Updating dependencies

To update, use [`glide up`](https://github.com/Masterminds/glide#glide-update-aliased-to-up) which will use VCS and `glide.yaml` to figure out the correct updates.

##### Testing application code with glide


### Testing

#### Full test suite

To execute all of the tests that CI will execute, run the following from the
root of the repository:

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

#### JavaScript linters

To run all JavaScript linters and static analyzers, run the following:

```
make lint-js
```

#### Viewing test coverage

When you run `make test` or `make test-go` from the root of the repository, test
coverage reports are generated in every subpackage. For example, the `server`
subpackage will have a coverage report generated in `./server/server.cover`

To explore a test coverage report on a line-by-line basis in the browser, run
the following:

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

To intercept sent emails while running a Kolide development environment, make
sure that you've set the SMTP address to `<docker host ip>:1025` and leave the
username and password blank. Then, visit `<docker host ip>:8025` in a web
browser to view the [MailHog](https://github.com/mailhog/MailHog) UI.

For example, if docker is running natively on your `localhost`, then your mail
settings should look something like:

```yaml
mail:
  address: localhost:1025
```

`localhost:1025` is the default configuration. You can use `kolide config_dump`
to see the values which Kolide is using given your configuration.

#### Viewing email content in the terminal

If you're [running Kolide in dev mode](#using-no-external-dependencies), emails
will be printed to the terminal instead of being sent via an SMTP server. This
may be useful if you want to view the content of all emails that Kolide sends.

### Development infrastructure

#### Starting the local development environment

To set up a canonical development environment via docker,
run the following from the root of the repository:

```
docker-compose up
```

This requires that you have docker installed. At this point in time,
automatic configuration tools are not included with this project.


#### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by docker, run
the following from the root of the repository:

```
docker-compose down
```

#### Setting up the database tables

Once you `docker-compose up` and are running the databases, you can build
the code and run the following command to create the database tables:

```
kolide prepare db
```

### Running Kolide

#### Using Docker development infrastructure

To start the Kolide server backed by the Docker development infrasturcture, run
the Kolide binary as follows:

```
kolide serve
```

By default, Kolide will try to connect to servers running on default ports on
localhost.

If you're using Docker via [Docker Toolbox](https://www.docker.com/products/docker-toolbox).
you may have to modify the default values use the output of `docker-machine ip`
instead of `localhost`.There is an example configuration file included in this
repository to make this process easier for you.  Use the `--config` flag of the
Kolide binary to specify the path to your config. See `kolide --help` for more
options.

#### Using no external dependencies

If you'd like to launch the Kolide server with no external dependencies, run
the following:

```
kolide serve --dev
```

This will use in-memory mocks for the database, print emails to the console,
etc. If you're demo-ing Kolide or testing a quick feature, dev mode is ideal.
Keep in mind that, since everything is in-memory, when you kill the process,
everything you did in dev mode will be lost. This is nice for development but
not so nice for production environments.

If you've used `make build` to build the Kolide binary, you can also run the
following to launch an in-memory instance of the server:

```
make run
```