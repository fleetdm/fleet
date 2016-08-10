# Kolide [![CircleCI](https://circleci.com/gh/kolide/kolide-ose.svg?style=svg&circle-token=2573c239b7f18967040d2dec95ca5f71cfc90693)](https://circleci.com/gh/kolide/kolide-ose)

### Contents

- [Development Environment](#development-environment)
  - [Installing dependencies](#installing-dependencies)
  - [Building](#building)
  - [Testing](#testing)
    - [Viewing test coverage](#viewing-test-coverage)
  - [Starting the local development environment](#starting-the-local-development-environment)
  - [Setting up the database tables](#setting-up-the-database-tables)
  - [Running Kolide](#running-kolide)
  - [Automatic recompilation](#automatic-recompilation)
  - [Stopping the local development environment](#stopping-the-local-development-environment)

## Development Environment

### Installing dependencies

To setup a working local development environment, you must install the following
minimum toolset:

* [Docker](https://www.docker.com/products/overview#/install_the_platform)
* [Go](https://golang.org/dl/) (1.6 or greater)
* [Node.js](https://nodejs.org/en/download/current/) (and npm)
* [GNU Make](https://www.gnu.org/software/make/)

Once you have those minimum requirements, to install build dependencies, run the following:

```
make deps
```

### Building

To generate all necessary code (bundling JavaScript into Go, etc), run the
following:

```
go generate
```

On UNIX (OS X, Linux, etc), run the following to compile the application:

```
go build -o kolide
```

On Windows, run the following to compile the application:

```
go build -o kolide.exe
```

### Testing

To run the application's tests, run the following from the root of the
repository:

```
go test
```

From the root, `go test` will run a test launcher that executes `go test` and 
`go vet` in the appropriate subpackages, etc. If you're working in a specific
subpackage, it's likely that you'll just want to iteratively run `go test` in
that subpackage directly until you are ready to run the full test suite.

#### Viewing test coverage

When you run `go test` from the root of the repository, test coverage reports
are generated in every subpackage. For example, the `sessions` subpackage will
have a coverage report generated in `./sessions/sessions.cover`

To explore a test coverage report on a line-by-line basis in the browser, run 
the following:

```bash
# substitute ./errors/errors.cover, ./app/app.cover, etc
go tool cover -html=./sessions/sessions.cover
```

To view test a test coverage report in a terminal, run the following:

```bash
# substitute ./errors/errors.cover, app/app.cover, etc
go tool cover -func=./sessions/sessions.cover
```

### Starting the local development environment

To set up a canonical development environment via docker,
run the following from the root of the repository:

```
docker-compose up
```

This requires that you have docker installed. At this point in time,
automatic configuration tools are not included with this project.

### Setting up the database tables

Once you `docker-compose up` and are running the databases, you can build
the code and run the following command to create the database tables:

```
kolide prepare-db
```

### Running Kolide

Now you are prepared to run a Kolide development environment. Run the following:

```
kolide serve
```

If you're running the binary from the root of the repository, where it is built
by default, then the binary will automatically use the provided example
configuration file, which assumes that you are running docker locally, on
`localhost` via a native engine.

You may have to edit the example configuration file to use the output of 
`docker-machine ip` instead of `localhost` if you're using Docker via 
[Docker Toolbox](https://www.docker.com/products/docker-toolbox).

### Automatic recompilation

If you're editing mostly frontend JavaScript, you may want the Go binary to be
automatically recompiled with a new JavaScript bundle and restarted whenever 
you save a JavaScript file. To do this, instead of running `kolide serve`, run
the following:

```
make watch
```

This is only supported on OS X and Linux.

### Stopping the local development environment

If you'd like to shut down the virtual infrastructure created by docker, run
the following from the root of the repository:

```
docker-compose down
```