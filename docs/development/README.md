Development Documentation
=========================

The Fleet application is a Go API server which serves a React/Redux single-page application for the frontend. The development documentation contains documents on the following topics:

## Building and contributing code

- For documentation on building the Fleet source code, see the [Building The Code](./building-the-code.md) guide.
- To learn about database migrations and populating the application with default seed data, see the [Database Migrations](./database-migrations.md) document.

## Running tests

For information on running the various tests that Fleet application contains (JavaScript unit tests, Go unit tests, linters, integration tests, etc), see the [Testing](./testing.md) guide.

## Using development infrastructure and tooling

The Fleet application uses a lot of docker tooling to make setting up a development environment quick and easy. For information on this, see the [Development Infrastructure](./development-infrastructure.md) document.

#### Setting up a Launcher environment

It's helpful to have a local build of the Launcher and it's included package building tools when reasoning about connecting the Launcher to Fleet. Both Launcher and Fleet have a similar repository interface that should be familiar.

If you have installed Go, but have never used it before (ie: you have not configured a `$GOPATH` environment variable), then there's good news: you don't need to do this anymore. By default, the Go compiler now looks in `~/go` for your Go source code. So, let's clone the launcher directory where it's supposed to go:

```
mkdir -p $GOPATH/src/github.com/kolide
cd $GOPATH/src/github.com/kolide
git clone git@github.com:kolide/launcher.git
cd launcher
```

Once you're in the root of the repository (and you have a recent Go toolchain installed), you can follow the directions included with the Launcher repository:

```
make deps
make generate
make test
make
./build/launcher --help
```