# Kolide [![CircleCI](https://circleci.com/gh/kolide/kolide-ose.svg?style=svg&circle-token=2573c239b7f18967040d2dec95ca5f71cfc90693)](https://circleci.com/gh/kolide/kolide-ose)

## Building

To build the code, run the following from the root of the repository:

```
# On Linux, OS X, etc:
go build -o kolide

# On Windows:
go build -o kolide.exe
```

## Testing

To run the application's tests, run the following from the root of the
repository:

```
go vet ./... && go test -v ./...
```

## Development Environment

To set up a canonical development environment via docker,
run the following from the root of the repository:

```
docker-compose up
```

This requires that you have docker installed. At this point in time,
automatic configuration tools are not included with this project.

Once you `docker-compose up` and are running the databases, you can build
the code and run the following command to create the database tables:

```
kolide prepare-db
```

Now you are prepared to run a Kolide development environment. Run the following:

```
kolide serve
```

You may have to edit the example configuration file to reflect `localhost` if
you're using Docker via a native docker engine or the output of 
`docker-machine ip` if you're using Docker via `docker-toolbox`.

If you'd like to shut down the virtual infrastructure created by docker, run
the following from the root of the repository:

```
docker-compose down
```
